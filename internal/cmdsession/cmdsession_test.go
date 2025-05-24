package cmdsession

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSessionError(t *testing.T) {
	myError := NewSessionError("hello")
	myWrappedError := fmt.Errorf("some new error: %w", myError)
	assert.True(t, IsSessionError(myWrappedError))
}

func TestGetExitCode(t *testing.T) {
	myError := NewExitCodeError(1)
	myWrappedError := fmt.Errorf("some new error: %w", myError)
	ec, ok := GetExitCode(myWrappedError)
	assert.True(t, ok)
	assert.Equal(t, ec, 1)
}
