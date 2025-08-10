package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type AgentOptions struct {
	sshAuthSock string
}

type AgentOption func(*AgentOptions)

func WithSshAuthSock(sshAuthSock string) AgentOption {
	return func(o *AgentOptions) {
		o.sshAuthSock = sshAuthSock
	}
}

const (
	SshAuthSockVar = "SSH_AUTH_SOCK"
)

var (
	agentInstance *SshAgent
	agentLock     sync.Mutex
)

type SshAgent struct {
	lock   sync.Mutex
	client agent.ExtendedAgent
	conn   net.Conn
}

func (a *SshAgent) Close() {
	_ = a.conn.Close()
}

func (a *SshAgent) HasKeys() (bool, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	keys, err := a.client.List()
	if err != nil {
		return false, fmt.Errorf("unable to retrieve key list\n%w", err)
	}

	return len(keys) > 0, nil
}

func (a *SshAgent) GetAuthMethod() (ssh.AuthMethod, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	signers, err := a.client.Signers()
	if err != nil {
		return nil, fmt.Errorf("unable to generate list of signers\n%w", err)
	}
	return ssh.PublicKeys(signers...), nil
}

func (a *SshAgent) AddPrivateKey(key any) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	err := a.client.Add(agent.AddedKey{
		PrivateKey: key,
	})

	if err != nil {
		return fmt.Errorf("unable to add signer to agent\n%w", err)
	}

	return nil
}

func (a *SshAgent) KeyExists(pubKey ssh.PublicKey) (bool, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	kBytes := pubKey.Marshal()

	keys, err := a.client.List()
	if err != nil {
		return false, fmt.Errorf("unable to list keys in ssh agent\n%w", err)
	}

	for _, key := range keys {
		if bytes.Equal(key.Marshal(), kBytes) {
			return true, nil
		}
	}

	return false, nil
}

func InitAgentInstance(options ...AgentOption) error {
	agentOptions := &AgentOptions{}
	for _, opt := range options {
		opt(agentOptions)
	}

	agentLock.Lock()
	defer agentLock.Unlock()

	if agentOptions.sshAuthSock == "" {
		agentOptions.sshAuthSock = os.Getenv(SshAuthSockVar)
		if agentOptions.sshAuthSock == "" {
			return fmt.Errorf("ssh agent auth sock was not specified in the environment")
		}
	}

	conn, err := net.Dial("unix", agentOptions.sshAuthSock)
	if err != nil {
		return fmt.Errorf("problem connecting to ssh agent\n%w", err)
	}

	client := agent.NewClient(conn)
	agentInstance = &SshAgent{
		client: client,
		conn:   conn,
	}

	return nil
}

func GetAgentInstance() (*SshAgent, error) {
	if agentInstance == nil {
		err := InitAgentInstance()
		if err != nil {
			return nil, err
		}
	}

	return agentInstance, nil
}
