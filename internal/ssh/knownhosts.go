package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/skeema/knownhosts"
)

var (
	knownhostsInstance *SshKnownHosts
	knownhostsLock     sync.Mutex
)

type KnownHostOptions struct {
	knownHostsFile string
}

type KnownHostOption func(*KnownHostOptions)

func WithKnownHostsFile(filename string) KnownHostOption {
	return func(o *KnownHostOptions) {
		o.knownHostsFile = filename
	}
}

type SshKnownHosts struct {
	lock     sync.Mutex
	Kh       *knownhosts.HostKeyDB
	filename string
}

func (kh *SshKnownHosts) Lock() {
	kh.lock.Lock()
}

func (kh *SshKnownHosts) Unlock() {
	kh.lock.Unlock()
}

func InitializeKnownHosts(options ...KnownHostOption) error {
	knownHostOptions := &KnownHostOptions{}
	for _, opt := range options {
		opt(knownHostOptions)
	}

	knownhostsLock.Lock()
	defer knownhostsLock.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to establish user's home directory\n%w", err)
	}

	if knownHostOptions.knownHostsFile == "" {
		knownHostOptions.knownHostsFile = filepath.Join(homeDir, ".ssh", "known_hosts")
	}

	kh, err := knownhosts.NewDB(knownHostOptions.knownHostsFile)
	if err != nil {
		return fmt.Errorf("unable to load known_hosts file from %s\n%w", knownHostOptions.knownHostsFile, err)
	}

	knownhostsInstance = &SshKnownHosts{
		Kh:       kh,
		filename: knownHostOptions.knownHostsFile,
	}

	return nil
}

func GetKnownHostsInstance() (*SshKnownHosts, error) {
	if knownhostsInstance == nil {
		err := InitializeKnownHosts()
		if err != nil {
			return nil, err
		}
	}

	return knownhostsInstance, nil
}
