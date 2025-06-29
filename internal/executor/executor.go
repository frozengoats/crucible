package executor

import (
	"fmt"
	"net"
	"sync"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/sequence"
	"github.com/frozengoats/crucible/internal/ssh"
)

type Executor struct {
	Config       *config.Config
	HostConfig   *config.HostConfig
	HostIdent    string
	SequencePath string

	executionClient   cmdsession.ExecutionClient
	sequence          *sequence.Sequence
	sequenceIndex     int
	ExecutionInstance *sequence.ExecutionInstance
}

var passphraseLock sync.Mutex

// NewExecutor creates an executor instance for dealing with a specific host and sequence
func NewExecutor(cfg *config.Config, hostIdent string, sequencePath string) (*Executor, error) {
	hostConfig, ok := cfg.Hosts[hostIdent]
	if !ok {
		return nil, fmt.Errorf("no host identity \"%s\" exists", hostIdent)
	}

	sshKeyPath := cfg.Executor.Ssh.KeyPath
	if hostConfig.SshKeyPath != "" {
		sshKeyPath = hostConfig.SshKeyPath
	}

	if sshKeyPath == "" {
		return nil, fmt.Errorf("no ssh key was specified both at top level or for host identity \"%s\"", hostIdent)
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
		executionClient = ssh.NewSsh(hostConfig.Host, cfg.User.Username, sshKeyPath,
			ssh.WithIgnoreHostKeyChangeOption(cfg.Executor.Ssh.IgnoreHostKeyChange),
			ssh.WithAllowUnknownHostsOption(cfg.Executor.Ssh.AllowUnknownHosts),
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

	execWaitGroup := &sync.WaitGroup{}
	execChan := make(chan *Executor, maxConcurrentHosts)
	errChan := make(chan error, len(hostIdents))
	for range maxConcurrentHosts {
		go func() {
			for e := range execChan {
				if syncExecutionSteps {
					action := e.ExecutionInstance.Next()
				} else {
					action := e.ExecutionInstance.Next()
				}

				defer execWaitGroup.Done()

				// execute the action

			}
		}()
	}

	if configObj.Executor.SyncExecutionSteps {
		hasMore := true
		for hasMore {
			hasMore = false
			for _, e := range executors {
				if e.ExecutionInstance.HasMore() {
					hasMore = true
					execWaitGroup.Add(1)
					execChan <- e
				}
			}

			// wait until all goroutines are finished
			execWaitGroup.Wait()

			// pull errors from error channel
			errChanLen := len(errChan)
			for range errChanLen {
				err := <-errChan
				fmt.Printf("%s\n", err)
			}
			if errChanLen > 0 {
				close(execChan)
				return fmt.Errorf("sequence aborted due to one or more failures")
			}
		}
		close(execChan)
	} else {
		for _, e := range executors {
			execWaitGroup.Add(1)
			execChan <- e
		}

		// wait until all goroutines are finished
		execWaitGroup.Wait()

		// pull errors from error channel
		errChanLen := len(errChan)
		for range errChanLen {
			err := <-errChan
			fmt.Printf("%s\n", err)
		}
		if errChanLen > 0 {
			close(execChan)
			return fmt.Errorf("sequence aborted due to one or more failures")
		}
	}

	return nil
}
