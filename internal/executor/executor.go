package executor

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/crucible/internal/sequence"
	"github.com/frozengoats/crucible/internal/ssh"
)

type ResultObj struct {
	Values       json.RawMessage `json:"values"`
	SuccessCount int             `json:"successCount"`
	FailCount    int             `json:"failCount"`
	SuccessHosts []string        `json:"successHosts"`
	FailHosts    []*FailedHost   `json:"failHosts"`
}

type FailedHost struct {
	Identity    string `json:"identity"`
	Error       string
	Contexts    []*sequence.ActionContext
	FullContext json.RawMessage
}

type Executor struct {
	Config       *config.Config
	HostConfig   *config.HostConfig
	HostIdent    string
	SequencePath string

	executionClient   cmdsession.ExecutionClient
	sequence          *sequence.Sequence
	ExecutionInstance *sequence.ExecutionInstance
	sequenceIndex     int
}

var passphraseLock sync.Mutex

// NewExecutor creates an executor instance for dealing with a specific host and sequence
func NewExecutor(cfg *config.Config, hostIdent string, sequencePath string) (*Executor, error) {
	hostConfig, ok := cfg.Hosts[hostIdent]
	if !ok {
		return nil, fmt.Errorf("no host identity \"%s\" exists", hostIdent)
	}

	isLoopback := false
	addrs, err := net.LookupIP(hostConfig.Host)
	if err == nil {
		// this logic is a little weak - though is there a chance of having multiple ips that are both loopback and non-loopback?
		for _, addr := range addrs {
			if addr.IsLoopback() {
				isLoopback = true
				break
			}
		}
	}

	var executionClient cmdsession.ExecutionClient
	if isLoopback {
		executionClient = cmdsession.NewLocalExecutionClient()
	} else {
		executionClient = ssh.NewSsh(
			cfg.Hostname(hostIdent),
			cfg.Port(hostIdent),
			cfg.Username(hostIdent),
			cfg.KeyPath(hostIdent),
			cfg.KnownHostsFile(hostIdent),
			ssh.WithIgnoreHostKeyChangeOption(cfg.IgnoreHostKeyChange(hostIdent)),
			ssh.WithAllowUnknownHostsOption(cfg.AllowUnknownHost(hostIdent)),
			ssh.WithPassphraseProviderOption(ssh.NewTypedPassphraseProvider()),
		)
	}

	s, err := sequence.LoadSequence(cfg.CwdPath, sequencePath)
	if err != nil {
		return nil, err
	}

	err = executionClient.Connect()
	if err != nil {
		return nil, err
	}

	ex := &Executor{
		Config:            cfg,
		HostConfig:        hostConfig,
		HostIdent:         hostIdent,
		SequencePath:      sequencePath,
		executionClient:   executionClient,
		sequence:          s,
		sequenceIndex:     0,
		ExecutionInstance: s.NewExecutionInstance(executionClient, cfg, hostIdent),
	}
	return ex, nil
}

// RunConcurrentExecutionGroup creates and runs concurrent execution groups
func RunConcurrentExecutionGroup(sequencePath string, configObj *config.Config, hostIdents []string) ([]byte, error) {
	maxConcurrentHosts := configObj.Executor.MaxConcurrentHosts
	if len(hostIdents) < maxConcurrentHosts {
		maxConcurrentHosts = len(hostIdents)
	}

	// iterate the selected hosts
	executors := []*Executor{}
	for _, hostIdent := range hostIdents {
		e, err := NewExecutor(configObj, hostIdent, sequencePath)
		if err != nil {
			return nil, fmt.Errorf("unable to create executor\n%w", err)
		}
		defer e.ExecutionInstance.Close()
		executors = append(executors, e)
	}

	syncExecutionSteps := configObj.Executor.SyncExecutionSteps

	// start up the executing threads and standby until executions are queued below
	execWaitGroup := &sync.WaitGroup{}
	execChan := make(chan *Executor, maxConcurrentHosts)
	for range maxConcurrentHosts {
		go func() {
			// channel is closed when all executions have completed or an error is encountered, loop will
			// exit automatically on last item
			for e := range execChan {
				func() {
					// closure allows this block to execute and signal completion using the wait group which
					// is incremented for every executor being enqueued (once per action in the case of sync)
					defer execWaitGroup.Done()
					for {
						action, err := e.ExecutionInstance.Next()
						if err != nil {
							e.ExecutionInstance.SetError(err)
							log.Error([]any{"host", e.HostIdent}, "execution terminated due to error: %s", err.Error())
							break
						}

						if action == nil {
							// no more actions, process the next thing
							break
						}

						// execute the action here, first clearing the immediate context from any previous run
						e.ExecutionInstance.ExecContext.Set(map[string]any{}, sequence.ImmediateKey)
						err = e.ExecutionInstance.Execute(action)
						if syncExecutionSteps || err != nil {
							if err != nil {
								e.ExecutionInstance.SetError(err)
								log.Error([]any{"host", e.HostIdent}, "execution terminated due to error: %s", err.Error())
							}

							// break after this execution if
							break
						}
					}
				}()
			}
		}()
	}

	// start queueing executions
	hasMore := true
	for hasMore {
		hasMore = false
		for _, e := range executors {
			if e.ExecutionInstance.HasMore() {
				execWaitGroup.Add(1)
				hasMore = true
				execChan <- e
			}
		}
		// the wait group ensures that IF this is operating in sync mode, that the next wave of processing
		// will not start for any execution instance until the previous wave is completed.  in the case of
		// non sync mode, a single loop will indicate completion of all hosts.
		execWaitGroup.Wait()

		if syncExecutionSteps {
			for _, e := range executors {
				if e.ExecutionInstance.GetError() != nil {
					hasMore = false
					break
				}
			}
		}
	}
	close(execChan)

	valuesBytes, err := json.Marshal(configObj.ValuesStore.GetMapping())
	if err != nil {
		return nil, fmt.Errorf("unable to marshal values mapping to json: %w", err)
	}
	resultObj := &ResultObj{
		SuccessHosts: []string{},
		FailHosts:    []*FailedHost{},
		Values:       valuesBytes,
	}

	for _, e := range executors {
		if e.ExecutionInstance.GetError() != nil {
			resultObj.FailCount++
			fh := &FailedHost{Identity: e.HostIdent, Error: e.ExecutionInstance.GetError().Error()}
			if e.Config.Debug {
				jBytes, err := json.Marshal(e.ExecutionInstance.ExecContext.GetMapping())
				if err != nil {
					return nil, fmt.Errorf("unable to export final execution context: %w", err)
				}
				fh.FullContext = jBytes
				fh.Contexts = e.ExecutionInstance.ImmediateContexts
			}
			resultObj.FailHosts = append(resultObj.FailHosts, fh)
		} else {
			resultObj.SuccessCount++
			resultObj.SuccessHosts = append(resultObj.SuccessHosts, e.HostIdent)
		}
	}

	resultObjBytes, err := json.Marshal(resultObj)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal result object to JSON: %w", err)
	}

	return resultObjBytes, nil
}
