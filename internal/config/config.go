package config

import (
	"fmt"
	"path/filepath"

	"github.com/frozengoats/crucible/internal/defaults"
	"github.com/frozengoats/crucible/internal/yamlstack"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

type SshConfig struct {
	AllowUnknownHosts   bool   `yaml:"allowUnknownHosts"`
	IgnoreHostKeyChange bool   `yaml:"ignoreHostKeyChange"`
	KeyPath             string `yaml:"keyPath"` // the main ssh key path, expected to be able to access all hosts except those with overrides
}

type Executor struct {
	MaxConcurrentHosts int       `yaml:"maxConcurrentHosts" default:"10"`
	Ssh                SshConfig `yaml:"ssh"`
	SyncExecutionSteps bool      `yaml:"syncExecutionSteps"` // if true, execution step must complete on all hosts before advancing
}

type HostConfig struct {
	Host       string         `yaml:"host"`
	Group      string         `yaml:"group"`      // optional group key, which must be uniquely identifiable and different than any host key name
	SshKeyPath string         `yaml:"sshKeyPath"` // optional lookup to the SSH private key, if not for some reason, the master key
	Context    map[string]any `yaml:"context"`    // generic k/v storage for data to be referenced later
}

type Config struct {
	// keys are unique host identifiers, though they themselves have no meaning
	Executor    Executor               `yaml:"executor"`
	Hosts       map[string]*HostConfig `yaml:"hosts"`
	ValuesStore *kvstore.Store
}

func FromFilePaths(cwd string, stackPaths ...string) (*Config, error) {
	var err error
	absPaths := []string{}
	for _, stackPath := range stackPaths {
		if !filepath.IsAbs(stackPath) {
			stackPath, err = filepath.Abs(filepath.Join(cwd, stackPath))
			if err != nil {
				return nil, fmt.Errorf("problem interpreting path %s\n%w", stackPath, err)
			}
		}

		absPaths = append(absPaths, stackPath)
	}

	b, err := yamlstack.StackYaml(absPaths...)
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

	return c, nil
}
