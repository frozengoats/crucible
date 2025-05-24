package cmdsession

import (
	"errors"
	"fmt"
)

type ExecutionClient interface {
	Connect() error
	Close() error
	GetCmdSession() (CmdSession, error)
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
	Execute(cmd string) ([]byte, error)
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
