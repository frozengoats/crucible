package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
)

type SshOptions struct {
	ignoreHostKeyChange bool
	allowUnknownHosts   bool
	passphraseProvider  PassphraseProvider
}

type SshConfigOption func(*SshOptions)

func WithAllowUnknownHostsOption(allow bool) SshConfigOption {
	return func(o *SshOptions) {
		o.allowUnknownHosts = allow
	}
}

func WithIgnoreHostKeyChangeOption(ignore bool) SshConfigOption {
	return func(o *SshOptions) {
		o.ignoreHostKeyChange = ignore
	}
}

func WithPassphraseProviderOption(provider PassphraseProvider) SshConfigOption {
	return func(s *SshOptions) {
		s.passphraseProvider = provider
	}
}

func GetPublicKey(keyFile string) (ssh.PublicKey, error) {
	pubKeyFile := fmt.Sprintf("%s.pub", keyFile)
	_, err := os.Stat(pubKeyFile)
	if err != nil {
		return nil, fmt.Errorf("a public key named %s could not be located", pubKeyFile)
	}

	key, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read public key %s\n%w", pubKeyFile, err)
	}

	key = bytes.TrimSpace(key)
	key = bytes.SplitN(key, []byte("\n"), 2)[0]

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ssh public key from %s\n%w", pubKeyFile, err)
	}

	return pubKey, nil
}

func GetPrivateKeySigner(keyFile string, passphraseProvider PassphraseProvider) (ssh.Signer, any, error) {
	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read key %s\n%w", keyFile, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return signer, nil, nil
	}

	if _, ok := err.(*ssh.PassphraseMissingError); !ok {
		return nil, nil, err
	}

	// this is now an indication that this key is locked with a passphrase
	passPhrase, err := passphraseProvider.GetPassphrase()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to obtain passphrase\n%w", err)
	}

	signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passPhrase))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get signer from private key\n%w", err)
	}

	rawKey, err := ssh.ParseRawPrivateKeyWithPassphrase(key, []byte(passPhrase))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse raw key from private key\n%w", err)
	}

	return signer, rawKey, nil
}

type SshSession struct {
	options        *SshOptions
	client         *ssh.Client
	hostname       string
	port           int
	username       string
	keyFile        string
	knownHostsFile string
}

func NewSsh(hostname string, port int, username string, keyFile string, knownHostsFile string, configOptions ...SshConfigOption) *SshSession {
	options := &SshOptions{}
	for _, o := range configOptions {
		o(options)
	}

	return &SshSession{
		hostname:       hostname,
		port:           port,
		username:       username,
		keyFile:        keyFile,
		knownHostsFile: knownHostsFile,
		options:        options,
	}
}

func (s *SshSession) Close() error {
	if s.client != nil {
		result := s.client.Close()
		s.client = nil
		return result
	}

	return nil
}

func (s *SshSession) hostKeyCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	kh, err := GetKnownHostsInstance(s.knownHostsFile)
	if err != nil {
		return err
	}

	kh.Lock()
	defer kh.Unlock()

	err = kh.Kh.HostKeyCallback()(hostname, remote, key)
	if knownhosts.IsHostKeyChanged(err) {
		if s.options.ignoreHostKeyChange {
			return nil
		}
		return fmt.Errorf("host key has changed for %s", hostname)
	}

	if knownhosts.IsHostUnknown(err) {
		if s.options.allowUnknownHosts {
			ferr := kh.WriteKnownHost(hostname, remote, key)
			if ferr != nil {
				return ferr
			}

			return nil
		}

		return fmt.Errorf("host %s is not known in your known_hosts file, to remedy, ssh into the host manually", s.hostname)
	}

	return err
}

func (s *SshSession) Connect() error {
	if s.client != nil {
		return nil
	}

	var signer ssh.Signer
	var authMethod ssh.AuthMethod

	_, err := os.Stat(s.keyFile)
	if err != nil {
		return fmt.Errorf("keyfile %s was not found - if you did not explicitly provide this keyfile, please specify the correct one in the configuration", s.keyFile)
	}

	sshAgent, err := GetAgentInstance()
	if err != nil {

		// this means there's no ssh agent available
		signer, _, err = GetPrivateKeySigner(s.keyFile, s.options.passphraseProvider)
		if err != nil {
			return err
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		pubKey, err := GetPublicKey(s.keyFile)
		if err != nil {
			return err
		}
		keyInAgent, err := sshAgent.KeyExists(pubKey)
		if err != nil {
			return err
		}
		if keyInAgent {
			authMethod, err = sshAgent.GetAuthMethod()
			if err != nil {
				return err
			}
		} else {
			var rawSigner any
			signer, rawSigner, err = GetPrivateKeySigner(s.keyFile, s.options.passphraseProvider)
			if err != nil {
				return err
			}
			authMethod = ssh.PublicKeys(signer)
			if rawSigner != nil {
				err = sshAgent.AddPrivateKey(rawSigner)
				if err != nil {
					fmt.Printf("unable to add private key to ssh agent, but continuing\n%s\n", err.Error())
				}
			}
		}
	}

	sshConfig := &ssh.ClientConfig{
		User: s.username,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: s.hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	host := s.hostname
	host = fmt.Sprintf("%s:%d", host, s.port)
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return fmt.Errorf("unable to establish ssh connection for %s\n%w", host, err)
	}

	s.client = client
	return nil
}

func (s *SshSession) NewCmdSession() (cmdsession.CmdSession, error) {
	return &SshCmdSession{
		client: s.client,
	}, nil
}
