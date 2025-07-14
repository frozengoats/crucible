package executor

import (
	"fmt"
	"net"
	"sync"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/crucible/internal/sequence"
	"github.com/frozengoats/crucible/internal/ssh"
	"github.com/frozengoats/kvstore"
)

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

	addrs, err := net.LookupIP(hostConfig.Host)
	if err != nil {
		return nil, fmt.Errorf("problem resolving hostname: %s\n%w", hostConfig.Host, err)
	}

	// this logic is a little weak - though is there a chance of having multiple ips that are both loopback and non-loopback?
	isLoopback := false
	for _, addr := range addrs {
		if addr.IsLoopback() {
			isLoopback = true
			break
		}
	}

	var executionClient cmdsession.ExecutionClient
	if isLoopback {
		executionClient = cmdsession.NewLocalExecutionClient()
	} else {
		executionClient = ssh.NewSsh(
			cfg.Hostname(hostIdent),
			cfg.Port(hostIdent),
			cfg.KnownHostsFile(hostIdent),
			cfg.Username(hostIdent),
			cfg.KeyPath(hostIdent),
			ssh.WithIgnoreHostKeyChangeOption(cfg.IgnoreHostKeyChange(hostIdent)),
			ssh.WithAllowUnknownHostsOption(cfg.AllowUnknownHost(hostIdent)),
			ssh.WithPassphraseProviderOption(ssh.NewTypedPassphraseProvider()),
		)
	}

	s, err := sequence.LoadSequence(sequencePath)
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
func RunConcurrentExecutionGroup(sequencePath string, configObj *config.Config, hostIdents []string) error {
	maxConcurrentHosts := configObj.Executor.MaxConcurrentHosts
	if len(hostIdents) < maxConcurrentHosts {
		maxConcurrentHosts = len(hostIdents)
	}

	// iterate the selected hosts
	executors := []*Executor{}
	for _, hostIdent := range hostIdents {
		e, err := NewExecutor(configObj, hostIdent, sequencePath)
		if err != nil {
			return fmt.Errorf("unable to create executor\n%w", err)
		}
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
						action := e.ExecutionInstance.Next()
						if action == nil {
							// no more actions, process the next thing
							break
						}

						// execute the action here
						err := e.ExecutionInstance.Execute(action, kvstore.NewStore())
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
	}
	close(execChan)

	return nil
}
