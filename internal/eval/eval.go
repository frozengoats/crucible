package eval

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/frozengoats/kvstore"
)

const (
	OperatorEquals        string = "=="
	OperatorUnequals      string = "!="
	OperatorGreaterEquals string = ">="
	OperatorLessEquals    string = "<="
	OperatorGreater       string = ">"
	OperatorLess          string = "<"
	OperatorAnd           string = "&&"
	OperatorOr            string = "||"
	OperatorPlus          string = "+"
	OperatorMinus         string = "-"
	OperatorMultiply      string = "*"
	OperatorDivide        string = "/"
	Separator             string = ","
)

var operators = map[string]struct{}{
	OperatorEquals:        {},
	OperatorUnequals:      {},
	OperatorGreaterEquals: {},
	OperatorLessEquals:    {},
	OperatorGreater:       {},
	OperatorLess:          {},
	OperatorAnd:           {},
	OperatorOr:            {},
	OperatorPlus:          {},
	OperatorMinus:         {},
	OperatorMultiply:      {},
	OperatorDivide:        {},
	Separator:             {},
}

const (
	OpenParenthesis   byte = 40
	ClosedParenthesis byte = 41
	DoubleQuote       byte = 34
	SingleQuote       byte = 39
	Equals            byte = 61
	Exclamation       byte = 33
	GreaterThan       byte = 62
	LessThan          byte = 60
	Ampersand         byte = 38
	Pipe              byte = 124
	Plus              byte = 43
	Minus             byte = 45
	Multiply          byte = 42
	Divide            byte = 47
	Comma             byte = 44
)

var operatorChars = map[byte]struct{}{
	Equals:      struct{}{},
	Exclamation: struct{}{},
	GreaterThan: struct{}{},
	LessThan:    struct{}{},
	Ampersand:   struct{}{},
	Pipe:        struct{}{},
	Plus:        struct{}{},
	Minus:       struct{}{},
	Multiply:    struct{}{},
	Divide:      struct{}{},
	Comma:       struct{}{},
}

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

func (g *Group) EmitTokens() ([]*Token, error) {
	var tokens []*Token

	// if this is a group of sub-groups, go recurive
	if g.Type == GroupTypeParenthesis {
		subGroups, err := getGroups(g.Text)
		if err != nil {
			return nil, err
		}

		for _, g := range subGroups {
			subTokens, err := g.EmitTokens()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, subTokens...)
		}

		return []*Token{
			{
				Text:   g.Text,
				Type:   TokenTypeGroup,
				Tokens: tokens,
			},
		}, nil
	}

	if g.Type == GroupTypeString {
		return []*Token{
			{
				Text: g.Text,
				Type: TokenTypeString,
			},
		}, nil
	}

	// this is an unqualified group which can be broken down into tokens
	tokenStart := 0
	var prevToken byte = 0
	var c byte
	for i := range len(g.Text) + 1 {
		if i < len(g.Text) {
			c = g.Text[i]
		} else {
			c = 32
		}

		if prevToken != 0 {
			_, isPrevOperator := operatorChars[prevToken]
			_, isCurrentOperator := operatorChars[c]

			if isPrevOperator != isCurrentOperator || i == len(g.Text) {
				text := strings.Trim(g.Text[tokenStart:i], " ")
				toks := strings.Split(text, " ")
				tokenStart = i
				for _, tok := range toks {
					if len(tok) == 0 {
						continue
					}

					var tokenType TokenType
					_, isOperator := operators[tok]
					if isOperator {
						if text == Separator {
							tokenType = TokenTypeSeparator
						} else {
							tokenType = TokenTypeOperator
						}
					} else if isPrevOperator {
						// this had operator characters but didn't match any known operator
						return nil, fmt.Errorf("unrecognized operator %s", tok)
					} else if strings.HasPrefix(tok, ".Values.") || strings.HasPrefix(tok, ".Context.") {
						tokenType = TokenTypeVariable
					} else {
						// is it a number
						_, err := strconv.ParseFloat(tok, 64)
						if err != nil {
							tokenType = TokenTypeInferredString
						} else {
							tokenType = TokenTypeNumber
						}
					}

					tokens = append(tokens, &Token{
						Text: text,
						Type: tokenType,
					})
				}
			}
		}
		prevToken = c
	}

	return tokens, nil
}

type TokenType string

const (
	TokenTypeString         TokenType = "STRING"
	TokenTypeInferredString TokenType = "INFERRED_STRING"
	TokenTypeNumber         TokenType = "NUMBER"
	TokenTypeGroup          TokenType = "GROUP"
	TokenTypeOperator       TokenType = "OPERATOR"
	TokenTypeVariable       TokenType = "VARIABLE"
	TokenTypeFunction       TokenType = "FUNCTION"
	TokenTypeSeparator      TokenType = "SEPARATOR"
)

type Token struct {
	Text   string
	Type   TokenType
	Tokens []*Token
}

// simplify traverses the token in a depth-first order and evaluates the result
func (t *Token) evaluate() (any, error) {
	var curVal any
	var prevToken = &Token{
		Type: TokenTypeOperator,
	}

	if t.Type == TokenTypeFunction {
		var args []any
		for _, token := range t.Tokens {
			// these are function arguments in this case, they need to be simplified but
			v, err := token.evaluate()
			if err != nil {
				return nil, err
			}

			args = append(args, v)
		}

		// execute the function call with the supplied arguments
		return Call(t.Text, args)
	}

	for _, token := range t.Tokens {
		var value any
		if prevToken.Type == TokenTypeOperator && token.Type == TokenTypeOperator {
			return nil, fmt.Errorf("bad expression, multiple adjacent operators")
		}

		if token.Type == TokenTypeOperator {
			prevToken = token
			continue
		}

		if token.Type == TokenTypeGroup || token.Type == TokenTypeFunction {
			v, err := token.evaluate()
			if err != nil {
				return nil, err
			}
			value = v
		}

		if curVal == nil {
			curVal = value
			continue
		}

		if prevToken.Type != TokenTypeOperator {
			return nil, fmt.Errorf("bad expression, values must be separated by operators")
		}

		var err error
		switch prevToken.Text {
		case OperatorEquals:
			curVal, err = EqualsOp(curVal, value)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
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

func tokenize(expression string) (*Token, error) {
	groups, err := getGroups(expression)
	if err != nil {
		return nil, err
	}

	var tokens []*Token
	for _, g := range groups {
		toks, err := g.EmitTokens()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, toks...)
	}

	var rectifiedTokens []*Token
	var prevToken *Token
	for _, t := range tokens {
		if prevToken != nil && prevToken.Type == TokenTypeInferredString && t.Type == TokenTypeGroup {
			// this is likely a function call, look up the call on the function board
			_, isFunc := functions[prevToken.Text]
			if !isFunc {
				return nil, fmt.Errorf("unknown function %s", prevToken.Text)
			}

			// a function call will have one or more arguments, thus this token list needs to be converted into a series of groups, one per arg
			var newTok *Token = &Token{
				Type: TokenTypeGroup,
			}
			for _, subTok := range t.Tokens {
				if subTok.Type == TokenTypeSeparator {
					prevToken.Tokens = append(prevToken.Tokens, newTok)
					newTok = &Token{
						Type: TokenTypeGroup,
					}
					continue
				}

				newTok.Tokens = append(newTok.Tokens, subTok)
			}
			prevToken.Tokens = append(prevToken.Tokens, newTok)
			prevToken.Type = TokenTypeFunction
			// don't reassign prevToken here since this has just swallowed the next token
			continue
		}

		rectifiedTokens = append(rectifiedTokens, t)
		prevToken = t
	}

	return &Token{
		Type:   TokenTypeGroup,
		Tokens: tokens,
	}, nil
}

// // Evaluate evaluates an expression to either true or false, or returns an error if the expression cannot
// // be evaluated.
// func Evaluate(expression string) (bool, error) {
// 	tokenGroup, err := tokenize(expression)
// 	if err != nil {
// 		return false, err
// 	}
// }
