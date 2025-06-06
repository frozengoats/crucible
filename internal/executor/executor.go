package executor

import (
	"fmt"
	"net"
	"os/user"
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

	executionClient cmdsession.ExecutionClient
	sequence        *sequence.Sequence
	sequenceIndex   int
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

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to ascertain current user\n%w", err)
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
		executionClient = ssh.NewSsh(hostConfig.Host, u.Username, sshKeyPath,
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
		Config:       cfg,
		HostConfig:   hostConfig,
		HostIdent:    hostIdent,
		SequencePath: sequencePath,

		executionClient: executionClient,
		sequence:        s,
		sequenceIndex:   0,
	}
	return ex, nil
}

func (ex *Executor) RunOne() error {
	return nil
}

func (ex *Executor) RunAll() error {
	return nil
}

func (ex *Executor) HasMore() bool {
	return ex.sequenceIndex <= ex.sequence.Len()
}
