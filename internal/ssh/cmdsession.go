package ssh

import (
	"github.com/frozengoats/crucible/internal/cmdsession"
	"golang.org/x/crypto/ssh"
)

type SshCmdSession struct {
	client *ssh.Client
}

// Execute executes a
func (s *SshCmdSession) Execute(cmd string) ([]byte, error) {
	sess, err := s.client.NewSession()
	if err != nil {
		return nil, cmdsession.NewSessionError("unable to initiate new session: %", err.Error())
	}

	output, err := sess.Output(cmd)
	if err != nil {
		exitErr, ok := err.(*ssh.ExitError)
		if ok {
			return nil, cmdsession.NewExitCodeError(exitErr.ExitStatus())
		}

		return nil, err
	}

	return output, nil
}
