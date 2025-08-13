package config

import (
	"fmt"
	"os/user"
	"sync"

	"github.com/frozengoats/crucible/internal/defaults"
	"github.com/frozengoats/crucible/internal/ssh"
	"github.com/frozengoats/crucible/internal/yamlstack"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

var (
	sshInfoCache = map[string]*ssh.SshInfo{}
	sshInfoLock  sync.Mutex
)

type SshConfig struct {
	AllowUnknownHosts           bool    `yaml:"allowUnknownHosts"`
	IgnoreHostKeyChange         bool    `yaml:"ignoreHostKeyChange"`
	KeyPath                     string  `yaml:"keyPath"`        // the main ssh key path, expected to be able to access all hosts except those with overrides
	KnownHostsPath              string  `yaml:"knownHostsPath"` // path to the known_hosts file
	User                        string  `yaml:"user"`
	MaxConnectionAttempts       int     `yaml:"maxConnectionAttempts" default:"20"`        // maximum consecutive connection attempts
	DelayAfterConnectionFailure float64 `yaml:"delayAfterConnectionFailure" default:"5.0"` // number of seconds to wait before retrying
}

type Executor struct {
	MaxConcurrentHosts int       `yaml:"maxConcurrentHosts" default:"10"`
	ShellBinary        string    `yaml:"shellBinary" default:"sh"`
	Ssh                SshConfig `yaml:"ssh"`
	SyncExecutionSteps bool      `yaml:"syncExecutionSteps"` // if true, execution step must complete on all hosts before advancing
}

type HostConfig struct {
	Host    string         `yaml:"host"`
	Group   string         `yaml:"group"`   // optional group key, which must be uniquely identifiable and different than any host key name
	Context map[string]any `yaml:"context"` // generic k/v storage for data to be referenced later
	Ssh     SshConfig      `yaml:"ssh"`
}

type UserConfig struct {
	Username string
	HomeDir  string
}

type Config struct {
	// keys are unique host identifiers, though they themselves have no meaning
	Executor    Executor               `yaml:"executor"`
	Hosts       map[string]*HostConfig `yaml:"hosts"`
	ValuesStore *kvstore.Store
	User        *UserConfig
	Debug       bool
	Json        bool
	CwdPath     string
}

func FromFilePaths(stackPaths ...string) (*Config, error) {
	b, err := yamlstack.StackYaml(stackPaths...)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, fmt.Errorf("yaml provided was incompatible with the config spec\n%w", err)
	}

	err = defaults.ApplyDefaults(c)
	if err != nil {
		return nil, fmt.Errorf("unable to apply defaults to config\n%w", err)
	}

	c.User = &UserConfig{}
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to ascertain current user\n%w", err)
	}
	c.User.Username = u.Username
	c.User.HomeDir = u.HomeDir

	return c, nil
}

func (c *Config) getSshInfo(hostIdent string) *ssh.SshInfo {
	sshInfoLock.Lock()
	defer sshInfoLock.Unlock()

	var err error
	sshInfo, ok := sshInfoCache[hostIdent]
	if !ok {
		sshInfo, err = ssh.GetSshInfo(c.Hosts[hostIdent].Host)
		if err != nil {
			sshInfo = &ssh.SshInfo{}
		}

		sshInfoCache[hostIdent] = sshInfo
	}

	return sshInfo
}

func (c *Config) Username(hostIdent string) string {
	if c.Hosts[hostIdent].Ssh.User != "" {
		return c.Hosts[hostIdent].Ssh.User
	}

	if c.Executor.Ssh.User != "" {
		return c.Executor.Ssh.User
	}

	return c.getSshInfo(hostIdent).User
}

func (c *Config) Hostname(hostIdent string) string {
	return c.getSshInfo(hostIdent).Hostname
}

func (c *Config) Port(hostIdent string) int {
	return c.getSshInfo(hostIdent).Port
}

func (c *Config) KeyPath(hostIdent string) string {
	if c.Hosts[hostIdent].Ssh.KeyPath != "" {
		return c.Hosts[hostIdent].Ssh.KeyPath
	}

	if c.Executor.Ssh.KeyPath != "" {
		return c.Executor.Ssh.KeyPath
	}

	return c.getSshInfo(hostIdent).KeyPath
}

func (c *Config) KnownHostsFile(hostIdent string) string {
	if c.Hosts[hostIdent].Ssh.KnownHostsPath != "" {
		return c.Hosts[hostIdent].Ssh.KnownHostsPath
	}

	if c.Executor.Ssh.KnownHostsPath != "" {
		return c.Executor.Ssh.KnownHostsPath
	}

	return c.getSshInfo(hostIdent).KnownHostsPath
}

func (c *Config) AllowUnknownHosts(hostIdent string) bool {
	if c.Hosts[hostIdent].Ssh.AllowUnknownHosts {
		return true
	}

	if c.Executor.Ssh.AllowUnknownHosts {
		return true
	}

	return false
}

func (c *Config) IgnoreHostKeyChange(hostIdent string) bool {
	if c.Hosts[hostIdent].Ssh.IgnoreHostKeyChange {
		return true
	}

	if c.Executor.Ssh.IgnoreHostKeyChange {
		return true
	}

	return false
}
