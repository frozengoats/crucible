package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualsOperator(t *testing.T) {
	result, err := EqualsOp("a", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = EqualsOp(1., 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = EqualsOp(1., "a")
	assert.Error(t, err)
}

func TestGreaterThanOperator(t *testing.T) {
	result, err := GreaterThanOp("b", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))
}
