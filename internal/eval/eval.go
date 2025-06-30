package eval

import (
	"fmt"
	"strings"

	"github.com/frozengoats/kvstore"
)

var operators = map[string]struct{}{
	"==": struct{}{},
	"!=": struct{}{},
	">=": struct{}{},
	"<=": struct{}{},
	"<":  struct{}{},
	">":  struct{}{},
	"&&": struct{}{},
	"||": struct{}{},
	"+":  struct{}{},
	"-":  struct{}{},
	"*":  struct{}{},
	"/":  struct{}{},
}

const (
	OpenParenthesis   byte = 40
	ClosedParenthesis byte = 41
	DoubleQuote       byte = 34
	SingleQuote       byte = 39
)

type GroupType string

const (
	GroupTypeString      GroupType = "STRING"
	GroupTypeParenthesis GroupType = "PARENTHESIS"
	GroupTypeUnqualified GroupType = "UNQUALIFIED"
)

type Group struct {
	Text string
	Type GroupType
}

type TokenType string

const (
	TokenTypeString   TokenType = "STRING"
	TokenTypeNumber   TokenType = "NUMBER"
	TokenTypeGroup    TokenType = "GROUP"
	TokenTypeOperator TokenType = "OPERATOR"
	TokenTypeVariable TokenType = "VARIABLE"
	TokenTypeFunction TokenType = "FUNCTION"
)

type Token struct {
	Text   string
	Type   TokenType
	Tokens []Token
}

// GetValue returns a value from either of the context stores using identifier
func GetValue(valuesStore *kvstore.Store, contextStore *kvstore.Store, identifier string) (any, error) {
	var store *kvstore.Store
	if strings.HasPrefix(identifier, ".Values.") {
		store = valuesStore
		identifier = strings.TrimPrefix(identifier, ".Values.")
	} else if strings.HasPrefix(identifier, ".Context.") {
		store = contextStore
		identifier = strings.TrimPrefix(identifier, ".Context.")
	} else {
		return nil, fmt.Errorf("unknown store source (must begin with .Values. or .Context.): %s", identifier)
	}

	// return the value, it may or may not exist, that is not our concern
	return store.Get(store.ParseNamespaceString(identifier)...), nil
}

// getGroups returns a list of groups, whereby each group is either a quoted string, parenthesis group,
// or unqualified.
func getGroups(expression string) ([]*Group, error) {
	var groups []*Group
	var quoteChar byte
	parenthCount := 0
	groupStart := 0

	for i := range len(expression) {
		c := expression[i]
		if quoteChar == 0 && (c == DoubleQuote || c == SingleQuote) {
			quoteChar = c
			if i-groupStart > 0 {
				text := strings.Trim(expression[groupStart:i], " ")
				if len(text) > 0 {
					groups = append(groups, &Group{
						Text: text,
						Type: GroupTypeUnqualified,
					})
				}
			}
			groupStart = i
			continue
		}

		if quoteChar == DoubleQuote && c == DoubleQuote {
			groups = append(groups, &Group{
				Text: expression[groupStart+1 : i],
				Type: GroupTypeString,
			})
			quoteChar = 0
			groupStart = i + 1
			continue
		}

		if quoteChar == SingleQuote && c == SingleQuote {
			groups = append(groups, &Group{
				Text: expression[groupStart+1 : i],
				Type: GroupTypeString,
			})
			quoteChar = 0
			groupStart = i + 1
			continue
		}

		if quoteChar != 0 {
			continue
		}

		if parenthCount == 0 && c == OpenParenthesis {
			parenthCount++
			if i-groupStart > 0 {
				text := strings.Trim(expression[groupStart:i], " ")
				if len(text) > 0 {
					groups = append(groups, &Group{
						Text: text,
						Type: GroupTypeUnqualified,
					})
				}
			}
			groupStart = i
			continue
		}

		if parenthCount == 1 && c == ClosedParenthesis {
			parenthCount--
			text := strings.Trim(expression[groupStart+1:i], " ")
			if len(text) == 0 {
				return nil, fmt.Errorf("empty parenthesis group contained no contents")
			}
			groups = append(groups, &Group{
				Text: text,
				Type: GroupTypeParenthesis,
			})
			groupStart = i + 1
			continue
		}

		if c == OpenParenthesis {
			parenthCount++
			continue
		}

		if c == ClosedParenthesis {
			parenthCount--
			continue
		}
	}

	if parenthCount != 0 {
		return nil, fmt.Errorf("unclosed parenthesis group")
	}

	if quoteChar != 0 {
		return nil, fmt.Errorf("unclosed quotation mark")
	}

	if groupStart < len(expression)-1 {
		text := strings.Trim(expression[groupStart:], " ")
		if len(text) > 0 {
			groups = append(groups, &Group{
				Text: text,
				Type: GroupTypeUnqualified,
			})
		}
	}

	return groups, nil
}

// // Evaluate evaluates an expression to either true or false, or returns an error if the expression cannot
// // be evaluated.
// func Evaluate(expression string) (bool, error) {

// }
