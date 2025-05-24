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
}

var passphraseLock sync.Mutex

// NewExecutor creates an executor instance for dealing with a specific host and sequence
func NewExecutor(cfg *config.Config, hostConfigName string, sequencePath string) (*Executor, error) {
	hostConfig, ok := cfg.Hosts[hostConfigName]
	if !ok {
		return nil, fmt.Errorf("no host key \"%s\" exists", hostConfigName)
	}
	ex := &Executor{
		Config:         cfg,
		HostConfig:     hostConfig,
		HostConfigName: hostConfigName,
		SequencePath:   sequencePath,
	}
	return ex, nil
}

func (ex *Executor) Run() error {
	sshKeyPath := ex.Config.Executor.Ssh.KeyPath
	if ex.HostConfig.SshKeyPath != "" {
		sshKeyPath = ex.HostConfig.SshKeyPath
	}

	if sshKeyPath == "" {
		return fmt.Errorf("no ssh key was specified both at top level or for host key \"%s\"", ex.HostConfigName)
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to ascertain current user\n%w", err)
	}

	sshSession := ssh.NewSsh(ex.HostConfig.Host, u.Username, sshKeyPath,
		ssh.WithIgnoreHostKeyChangeOption(ex.Config.Executor.Ssh.IgnoreHostKeyChange),
		ssh.WithAllowUnknownHostsOption(ex.Config.Executor.Ssh.AllowUnknownHosts),
		ssh.WithPassphraseProviderOption(ssh.NewTypedPassphraseProvider()),
	)
	if err != nil {
		return fmt.Errorf("unable to create ssh session for host id %s\n%w", ex.HostConfigName, err)
	}

	s, err := sequence.LoadSequence(ex.SequencePath)
	if err != nil {
		return err
	}

	fmt.Println(sshSession)
	fmt.Println(s)
	return nil
}
