package ssh

import (
	"io"
	"strings"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"golang.org/x/crypto/ssh"
)

type SshCmdSession struct {
	client *ssh.Client
}

// Execute executes a remote command session
func (s *SshCmdSession) Execute(stdin io.Reader, cmd ...string) ([]byte, error) {
	sess, err := s.client.NewSession()
	if err != nil {
		return nil, cmdsession.NewSessionError("unable to initiate new session: %", err.Error())
	}

	if stdin != nil {
		sess.Stdin = stdin
	}

	output, err := sess.Output(strings.Join(cmd, " "))
	if err != nil {
		exitErr, ok := err.(*ssh.ExitError)
		if ok {
			return nil, cmdsession.NewExitCodeError(exitErr.ExitStatus())
		}

		return nil, err
	}

	return output, nil
}
