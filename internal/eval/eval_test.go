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

func TestTokenizerSimple(t *testing.T) {
	expression := ".Values.abc.def==123"
	tokenGroup, err := tokenize(expression)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeGroup, tokenGroup.Type)
	tokens := tokenGroup.Tokens
	assert.Len(t, tokens, 3)
	assert.Equal(t, tokens[0].Type, TokenTypeVariable)
	assert.Equal(t, tokens[1].Type, TokenTypeOperator)
	assert.Equal(t, tokens[2].Type, TokenTypeNumber)
}

func TestTokenizerNestedInQuotes(t *testing.T) {
	expression := ".Values.abc.def==\"(hello==one)\""
	tokenGroup, err := tokenize(expression)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeGroup, tokenGroup.Type)
	tokens := tokenGroup.Tokens
	assert.Len(t, tokens, 3)
	assert.Equal(t, tokens[0].Type, TokenTypeVariable)
	assert.Equal(t, tokens[1].Type, TokenTypeOperator)
	assert.Equal(t, tokens[2].Type, TokenTypeString)
	assert.Equal(t, tokens[2].Text, "(hello==one)")
}

func TestTokenizerNestedInSingleQuotes(t *testing.T) {
	expression := ".Values.abc.def=='(hello==\"one\")'"
	tokenGroup, err := tokenize(expression)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeGroup, tokenGroup.Type)
	tokens := tokenGroup.Tokens
	assert.Len(t, tokens, 3)
	assert.Equal(t, TokenTypeVariable, tokens[0].Type)
	assert.Equal(t, TokenTypeOperator, tokens[1].Type)
	assert.Equal(t, TokenTypeString, tokens[2].Type)
	assert.Equal(t, "(hello==\"one\")", tokens[2].Text)
}

func TestTokenizerParenthGroup(t *testing.T) {
	expression := ".Values.ent.value > (.Values.ent2.value || (.Values.ent3.value + 2))"
	tokenGroup, err := tokenize(expression)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeGroup, tokenGroup.Type)
	tokens := tokenGroup.Tokens
	assert.Len(t, tokens, 3)
	assert.Equal(t, tokens[0].Type, TokenTypeVariable)
	assert.Equal(t, tokens[1].Type, TokenTypeOperator)
	assert.Equal(t, tokens[2].Type, TokenTypeGroup)

	subTok := tokens[2].Tokens
	assert.Len(t, subTok, 3)
	assert.Equal(t, TokenTypeVariable, subTok[0].Type)
	assert.Equal(t, TokenTypeOperator, subTok[1].Type)
	assert.Equal(t, TokenTypeGroup, subTok[2].Type)

	subTok = subTok[2].Tokens
	assert.Len(t, subTok, 3)

	assert.Equal(t, TokenTypeVariable, subTok[0].Type)
	assert.Equal(t, TokenTypeOperator, subTok[1].Type)
	assert.Equal(t, TokenTypeNumber, subTok[2].Type)
}
