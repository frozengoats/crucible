package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualsOperator(t *testing.T) {
	result, err := EqualsOp("a", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = EqualsOp("a", "b")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = EqualsOp(1., 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = EqualsOp(1., 1.1)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = EqualsOp(1., "a")
	assert.Error(t, err)
}

func TestUnEqualsOperator(t *testing.T) {
	result, err := UnequalsOp("a", "a")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = UnequalsOp("a", "b")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = UnequalsOp(1., 1.)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = UnequalsOp(1., 1.1)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = UnequalsOp(1., "a")
	assert.Error(t, err)
}

func TestGreaterThanOperator(t *testing.T) {
	result, err := GreaterThanOp("a", "a")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanOp("a", "b")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanOp("b", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanOp(1., 1.)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanOp(1., 1.1)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanOp(1.1, 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanOp(1., "a")
	assert.Error(t, err)
}

func TestGreaterThanEqualsOperator(t *testing.T) {
	result, err := GreaterThanEqualsOp("a", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanEqualsOp("a", "b")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanEqualsOp("b", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanEqualsOp(1., 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanEqualsOp(1., 1.1)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = GreaterThanEqualsOp(1.1, 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = GreaterThanEqualsOp(1., "a")
	assert.Error(t, err)
}

func TestLessThanOperator(t *testing.T) {
	result, err := LessThanOp("a", "a")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanOp("a", "b")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanOp("b", "a")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanOp(1., 1.)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanOp(1., 1.1)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanOp(1.1, 1.)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanOp(1., "a")
	assert.Error(t, err)
}

func TestLessThanEqualsOperator(t *testing.T) {
	result, err := LessThanEqualsOp("a", "a")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanEqualsOp("a", "b")
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanEqualsOp("b", "a")
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanEqualsOp(1., 1.)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanEqualsOp(1., 1.1)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = LessThanEqualsOp(1.1, 1.)
	assert.NoError(t, err)
	assert.False(t, result.(bool))

	result, err = LessThanEqualsOp(1., "a")
	assert.Error(t, err)
}

func TestAndOperator(t *testing.T) {
	result, err := AndOp("a", "a")
	assert.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = AndOp("a", "b")
	assert.NoError(t, err)
	assert.Equal(t, "b", result)

	result, err = AndOp("a", 1.)
	assert.NoError(t, err)
	assert.Equal(t, 1., result)

	result, err = AndOp("", 1.)
	assert.NoError(t, err)
	assert.Equal(t, "", result)

	result, err = AndOp("a", 0.)
	assert.NoError(t, err)
	assert.Equal(t, 0., result)

	result, err = AndOp(1.1, 1.)
	assert.NoError(t, err)
	assert.Equal(t, 1., result)

	result, err = AndOp(true, false)
	assert.NoError(t, err)
	assert.Equal(t, false, result)

	result, err = AndOp(false, false)
	assert.NoError(t, err)
	assert.Equal(t, false, result)

	result, err = AndOp(true, true)
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = AndOp(true, 0.)
	assert.NoError(t, err)
	assert.Equal(t, 0., result)

	result, err = AndOp(false, true)
	assert.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestOrOperator(t *testing.T) {
	result, err := OrOp("a", "a")
	assert.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = OrOp("a", "b")
	assert.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = OrOp("a", 1.)
	assert.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = OrOp("", 1.)
	assert.NoError(t, err)
	assert.Equal(t, 1., result)

	result, err = OrOp("a", 0.)
	assert.NoError(t, err)
	assert.Equal(t, "a", result)

	result, err = OrOp(1.1, 1.)
	assert.NoError(t, err)
	assert.Equal(t, 1.1, result)

	result, err = OrOp(true, false)
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = OrOp(false, false)
	assert.NoError(t, err)
	assert.Equal(t, false, result)

	result, err = OrOp(true, true)
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = OrOp(true, 0.)
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = OrOp(false, true)
	assert.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestPlusOperator(t *testing.T) {
	result, err := PlusOp("a", "a")
	assert.NoError(t, err)
	assert.Equal(t, "aa", result)

	result, err = PlusOp(2.0, 2.2)
	assert.NoError(t, err)
	assert.Equal(t, 4.2, result)

	result, err = PlusOp([]any{1, 2, 3}, []any{4, 5, 6})
	assert.NoError(t, err)
	assert.Equal(t, []any{1, 2, 3, 4, 5, 6}, result)
}

func TestMinusOperator(t *testing.T) {
	result, err := MinusOp("a", "a")
	assert.Error(t, err)

	result, err = MinusOp(2.0, 2.5)
	assert.NoError(t, err)
	assert.Equal(t, -0.5, result)
}

func TestMultiplyOperator(t *testing.T) {
	result, err := MultiplyOp(1.5, 2.0)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, result)
}

func TestDivideOperator(t *testing.T) {
	result, err := DivideOp(9.0, 3.0)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, result)

	result, err = DivideOp(9.0, 0.0)
	assert.Error(t, err)
}
