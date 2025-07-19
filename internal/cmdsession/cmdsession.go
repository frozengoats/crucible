package cmdsession

import (
	"errors"
	"fmt"
	"os/exec"
)

type ExecutionClient interface {
	Connect() error
	Close() error
	NewCmdSession() (CmdSession, error)
}

type DummyCmdSession struct {
}

func (cs *DummyCmdSession) Execute(cmd ...string) ([]byte, error) {
	return nil, nil
}

func NewDummyExecutionClient() *DummyExecutionClient {
	return &DummyExecutionClient{}
}

type DummyExecutionClient struct {
}

func (c *DummyExecutionClient) Connect() error {
	return nil
}

func (c *DummyExecutionClient) Close() error {
	return nil
}

func (c *DummyExecutionClient) NewCmdSession() (CmdSession, error) {
	return &DummyCmdSession{}, nil
}

type SessionError struct {
	msg  string
	args []any
}

func NewSessionError(msg string, args ...any) *SessionError {
	return &SessionError{
		msg:  msg,
		args: args,
	}
}

type ExitCodeError struct {
	code int
}

func NewExitCodeError(code int) *ExitCodeError {
	return &ExitCodeError{
		code: code,
	}
}

func (ec *ExitCodeError) Error() string {
	return fmt.Sprintf("exited with a status of %d", ec.code)
}

func (se *SessionError) Error() string {
	return fmt.Sprintf(se.msg, se.args...)
}

type CmdSession interface {
	Execute(cmd ...string) ([]byte, error)
}

func IsSessionError(err error) bool {
	var sessionError *SessionError
	return errors.As(err, &sessionError)
}

func GetExitCode(err error) (int, bool) {
	var exitCodeError *ExitCodeError
	if !errors.As(err, &exitCodeError) {
		return 0, false
	}

	return exitCodeError.code, true
}

type LocalExecutionClient struct {
}

func NewLocalExecutionClient() *LocalExecutionClient {
	return &LocalExecutionClient{}
}

func (c *LocalExecutionClient) Connect() error {
	return nil
}

func (c *LocalExecutionClient) Close() error {
	return nil
}

func (c *LocalExecutionClient) NewCmdSession() (CmdSession, error) {
	return &LocalCmdSession{}, nil
}

type LocalCmdSession struct {
}

func (c *LocalCmdSession) Execute(cmd ...string) ([]byte, error) {
	com := exec.Command(cmd[0], cmd[1:]...)
	outBytes, err := com.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, NewExitCodeError(exitError.ExitCode())
		}

		return nil, err
	}

	return outBytes, nil
}
