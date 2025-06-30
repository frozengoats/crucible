package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGroupsSingleQuote(t *testing.T) {
	groups, err := getGroups("hello world && 'hello world'")
	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Equal(t, groups[0].Text, "hello world &&")
	assert.Equal(t, groups[1].Text, "hello world")
}

func TestGetGroupsDoubleQuote(t *testing.T) {
	groups, err := getGroups("hello world && \"hello world\"")
	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Equal(t, groups[0].Text, "hello world &&")
	assert.Equal(t, groups[1].Text, "hello world")
}

func TestGetGroupsParenthesis(t *testing.T) {
	groups, err := getGroups("(hello world) && \"hello world\"")
	assert.NoError(t, err)
	assert.Len(t, groups, 3)
	assert.Equal(t, "hello world", groups[0].Text)
	assert.Equal(t, groups[0].Type, GroupTypeParenthesis)
	assert.Equal(t, "&&", groups[1].Text)
	assert.Equal(t, groups[1].Type, GroupTypeUnqualified)
	assert.Equal(t, "hello world", groups[2].Text)
	assert.Equal(t, groups[2].Type, GroupTypeString)
}
