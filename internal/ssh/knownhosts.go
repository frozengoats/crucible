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

func GetKnownHostsInstance() (*SshKnownHosts, error) {
	if knownhostsInstance == nil {
		knownhostsLock.Lock()
		defer knownhostsLock.Unlock()

		if knownhostsInstance != nil {
			return knownhostsInstance, nil
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("unable to establish user's home directory\n%w", err)
		}
		knownHostsFile := filepath.Join(homeDir, ".ssh", "known_hosts")
		kh, err := knownhosts.NewDB(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load known_hosts file from %s\n%w", knownHostsFile, err)
		}

		knownhostsInstance = &SshKnownHosts{
			Kh:       kh,
			filename: knownHostsFile,
		}
	}

	return knownhostsInstance, nil
}
