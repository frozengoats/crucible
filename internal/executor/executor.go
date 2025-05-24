package executor

import (
	"fmt"
	"os/user"
	"sync"

	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/sequence"
	"github.com/frozengoats/crucible/internal/ssh"
)

type Executor struct {
	Config         *config.Config
	HostConfig     *config.HostConfig
	HostConfigName string
	SequencePath   string

	sshSession *ssh.SshSession
	sequence   *sequence.Sequence
}

var passphraseLock sync.Mutex

// NewExecutor creates an executor instance for dealing with a specific host and sequence
func NewExecutor(cfg *config.Config, hostConfigName string, sequencePath string) (*Executor, error) {
	hostConfig, ok := cfg.Hosts[hostConfigName]
	if !ok {
		return nil, fmt.Errorf("no host key \"%s\" exists", hostConfigName)
	}

	sshKeyPath := cfg.Executor.Ssh.KeyPath
	if hostConfig.SshKeyPath != "" {
		sshKeyPath = hostConfig.SshKeyPath
	}

	if sshKeyPath == "" {
		return nil, fmt.Errorf("no ssh key was specified both at top level or for host key \"%s\"", hostConfigName)
	}

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to ascertain current user\n%w", err)
	}

	sshSession := ssh.NewSsh(hostConfig.Host, u.Username, sshKeyPath,
		ssh.WithIgnoreHostKeyChangeOption(cfg.Executor.Ssh.IgnoreHostKeyChange),
		ssh.WithAllowUnknownHostsOption(cfg.Executor.Ssh.AllowUnknownHosts),
		ssh.WithPassphraseProviderOption(ssh.NewTypedPassphraseProvider()),
	)

	s, err := sequence.LoadSequence(sequencePath)
	if err != nil {
		return nil, err
	}

	ex := &Executor{
		Config:         cfg,
		HostConfig:     hostConfig,
		HostConfigName: hostConfigName,
		SequencePath:   sequencePath,

		sshSession: sshSession,
		sequence:   s,
	}
	return ex, nil
}

func (ex *Executor) RunOne() error {
	return nil
}

func (ex *Executor) RunAll() error {
	return nil
}
