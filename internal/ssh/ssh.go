package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
)

func GetPublicKey(keyFile string) (ssh.PublicKey, error) {
	pubKeyFile := fmt.Sprintf("%s.pub", keyFile)
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
	HostIdent string
	session   *ssh.Session
}

func (s *SshSession) Close() {
	_ = s.session.Close()
}

func NewSsh(host string, username string, keyFile string, ignoreHostKeyChange bool, allowUnknownHosts bool, passphraseProvider PassphraseProvider) (*SshSession, error) {
	var signer ssh.Signer
	var authMethod ssh.AuthMethod

	sshAgent, err := GetAgentInstance()
	if err != nil {
		// this means there's no ssh agent available
		fmt.Printf("warning: %s\n", err.Error())
		signer, _, err = GetPrivateKeySigner(keyFile, passphraseProvider)
		if err != nil {
			return nil, err
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		pubKey, err := GetPublicKey(keyFile)
		if err != nil {
			return nil, err
		}
		keyInAgent, err := sshAgent.KeyExists(pubKey)
		if keyInAgent {
			authMethod, err = sshAgent.GetAuthMethod()
			if err != nil {
				return nil, err
			}
		} else {
			var rawSigner any
			signer, rawSigner, err = GetPrivateKeySigner(keyFile, passphraseProvider)
			if err != nil {
				return nil, err
			}
			authMethod = ssh.PublicKeys(signer)
			err = sshAgent.AddPrivateKey(rawSigner)
			if err != nil {
				fmt.Printf("unable to add private key to ssh agent, but continuing\n%s\n", err.Error())
			}
		}
	}

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			kh, err := GetKnownHostsInstance()
			if err != nil {
				return err
			}

			kh.Lock()
			defer kh.Unlock()

			err = kh.Kh.HostKeyCallback()(hostname, remote, key)
			if knownhosts.IsHostKeyChanged(err) {
				if ignoreHostKeyChange {
					return nil
				}
				return fmt.Errorf("host key has changed for %s", hostname)
			}

			if knownhosts.IsHostUnknown(err) {
				if allowUnknownHosts {
					f, ferr := os.OpenFile(kh.filename, os.O_APPEND|os.O_WRONLY, 0600)
					if ferr != nil {
						return fmt.Errorf("problem opening known_hosts file at %s\n%w", kh.filename, err)
					}
					defer f.Close()

					ferr = knownhosts.WriteKnownHost(f, hostname, remote, key)
					if ferr != nil {
						return fmt.Errorf("unable to append host %s to known_hosts\n%w", hostname, err)
					}

					return nil
				}

				return fmt.Errorf("host %s is not known in your known_hosts file, to remedy, ssh into the host manually", host)
			}

			return err
		},
		Timeout: 10 * time.Second,
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to establish ssh connection for %s\n%w", host, err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("unable to establish ssh session for %s\n%w", host, err)
	}

	return &SshSession{
		session: session,
	}, nil
}
