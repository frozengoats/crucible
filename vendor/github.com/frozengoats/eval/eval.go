package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type VariableLookup func(key string) (any, error)
type FunctionCall func(name string, args ...any) (any, error)

var variableFinder = regexp.MustCompile(`^\.[a-zA-Z_]`)
var templateFinder = regexp.MustCompile(`{{\s+.*?\s+}}`)

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
	OperatorExponent      string = "**"
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
	OperatorExponent:      {},
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
	Equals:      {},
	Exclamation: {},
	GreaterThan: {},
	LessThan:    {},
	Ampersand:   {},
	Pipe:        {},
	Plus:        {},
	Minus:       {},
	Multiply:    {},
	Divide:      {},
	Comma:       {},
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

func CastToNumberIfApplicable(value any) any {
	switch t := value.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case float32:
		return float64(t)
	case int16:
		return float64(t)
	case int8:
		return float64(t)
	case uint:
		return float64(t)
	case uint64:
		return float64(t)
	case uint32:
		return float64(t)
	case uint16:
		return float64(t)
	case uint8:
		return float64(t)
	default:
		return value
	}
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
				Tokens: organizeTokens(tokens),
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

					matchesVariable := variableFinder.MatchString(tok)

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
					} else if matchesVariable {
						tokenType = TokenTypeVariable
					} else {
						// is it a number
						_, err := strconv.ParseFloat(tok, 64)
						if err != nil {
							if tok == "true" || tok == "false" {
								tokenType = TokenTypeBoolean
							} else {
								tokenType = TokenTypeInferredString
							}
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
	TokenTypeBoolean        TokenType = "BOOLEAN"
)

type Token struct {
	Text   string
	Type   TokenType
	Tokens []*Token
}

// simplify traverses the token in a depth-first order and evaluates the result
func (t *Token) evaluate(varLookup VariableLookup, funcCall FunctionCall) (any, error) {
	var curVal any
	var prevToken = &Token{
		Type: TokenTypeOperator,
	}

	switch t.Type {
	case TokenTypeFunction:
		var args []any
		for _, token := range t.Tokens {
			// these are function arguments in this case, they need to be simplified but
			v, err := token.evaluate(varLookup, funcCall)
			if err != nil {
				return nil, err
			}

			args = append(args, v)
		}

		// execute the function call with the supplied arguments
		v, err := funcCall(t.Text, args...)
		if err != nil {
			return nil, err
		}
		return CastToNumberIfApplicable(v), nil
	case TokenTypeInferredString:
		return t.Text, nil
	case TokenTypeBoolean:
		if t.Text == "true" {
			return true, nil
		}
		return false, nil
	case TokenTypeString:
		return t.Text, nil
	case TokenTypeNumber:
		fl, _ := strconv.ParseFloat(t.Text, 64)
		return fl, nil
	case TokenTypeVariable:
		varValue, err := varLookup(t.Text)
		if err != nil {
			return nil, err
		}
		return CastToNumberIfApplicable(varValue), nil
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

		v, e := token.evaluate(varLookup, funcCall)
		if e != nil {
			return nil, e
		}
		value = v

		if curVal == nil {
			curVal = value
			prevToken = token
			continue
		}

		if prevToken.Type != TokenTypeOperator {
			return nil, fmt.Errorf("bad expression, values must be separated by operators")
		}

		var err error
		switch prevToken.Text {
		case OperatorEquals:
			curVal, err = EqualsOp(curVal, value)
		case OperatorUnequals:
			curVal, err = UnequalsOp(curVal, value)
		case OperatorGreater:
			curVal, err = GreaterThanOp(curVal, value)
		case OperatorGreaterEquals:
			curVal, err = GreaterThanEqualsOp(curVal, value)
		case OperatorLess:
			curVal, err = LessThanOp(curVal, value)
		case OperatorLessEquals:
			curVal, err = LessThanEqualsOp(curVal, value)
		case OperatorAnd:
			curVal, err = AndOp(curVal, value)
		case OperatorOr:
			curVal, err = OrOp(curVal, value)
		case OperatorPlus:
			curVal, err = PlusOp(curVal, value)
		case OperatorMinus:
			curVal, err = MinusOp(curVal, value)
		case OperatorMultiply:
			curVal, err = MultiplyOp(curVal, value)
		case OperatorExponent:
			curVal, err = ExponentOp(curVal, value)
		case OperatorDivide:
			curVal, err = DivideOp(curVal, value)
		default:
			return nil, fmt.Errorf("unknown operator %s", prevToken.Text)
		}
		if err != nil {
			return nil, err
		}

		prevToken = token
	}

	return curVal, nil
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
		if quoteChar == 0 && (c == DoubleQuote || c == SingleQuote) && parenthCount == 0 {
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

func organizeTokens(tokens []*Token) []*Token {
	var rectifiedTokens []*Token
	var prevToken *Token
	for _, t := range tokens {
		if prevToken != nil && prevToken.Type == TokenTypeInferredString && t.Type == TokenTypeGroup {
			// a function call will have one or more arguments, thus this token list needs to be converted into a series of groups, one per arg
			var newTok = &Token{
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

	var orderedTokens []*Token
	for _, oper := range []string{
		OperatorExponent, OperatorDivide, OperatorMultiply,
	} {
		i := 0
		for i < len(rectifiedTokens) {
			j := i + 1
			if j < len(rectifiedTokens)-1 {
				if rectifiedTokens[j].Type == TokenTypeOperator && rectifiedTokens[j].Text == oper {
					orderedTokens = append(orderedTokens, &Token{
						Type:   TokenTypeGroup,
						Tokens: rectifiedTokens[i : i+3],
					})
					i += 3
					continue
				}
			}

			orderedTokens = append(orderedTokens, rectifiedTokens[i])
			i++
		}

		// swap them and start on the next loop
		rectifiedTokens = orderedTokens
		orderedTokens = nil
	}

	return rectifiedTokens
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

	return &Token{
		Type:   TokenTypeGroup,
		Tokens: organizeTokens(tokens),
	}, nil
}

// Evaluate evaluates an expression to either true or false, or returns an error if the expression cannot
// be evaluated.
func Evaluate(expression string, varLookup VariableLookup, funcCall FunctionCall) (any, error) {
	tokenGroup, err := tokenize(expression)
	if err != nil {
		return false, err
	}
	return tokenGroup.evaluate(varLookup, funcCall)
}

func IsTruthy(value any) bool {
	switch t := value.(type) {
	case string:
		return len(t) > 0
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	case bool:
		return t
	default:
		return false
	}
}

func AsNumber(value any) float64 {
	switch t := value.(type) {
	case float64:
		return t
	default:
		return 0.
	}
}

func AsString(value any) string {
	switch t := value.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func AsBoolean(value any) bool {
	switch t := value.(type) {
	case bool:
		return t
	default:
		return false
	}
}

func AsArray(value any) []any {
	switch t := value.(type) {
	case []any:
		return t
	default:
		return nil
	}
}

func AsMapping(value any) map[string]any {
	switch t := value.(type) {
	case map[string]any:
		return t
	default:
		return nil
	}
}

func Template(template string, varLookup VariableLookup, funcCall FunctionCall) (string, error) {
	matches := templateFinder.FindAllStringSubmatchIndex(template, -1)
	var newParts []string
	lastEnd := 0
	for _, match := range matches {
		start := match[0]
		end := match[1]

		if lastEnd < start {
			newParts = append(newParts, template[lastEnd:start])
		}

		result, err := Evaluate(template[start+2:end-2], varLookup, funcCall)
		if err != nil {
			return "", err
		}

		lastEnd = end
		if result == nil {
			result = ""
		}
		newParts = append(newParts, fmt.Sprintf("%v", result))
	}
	if lastEnd < len(template) {
		newParts = append(newParts, template[lastEnd:])
	}
	finalString := strings.Join(newParts, "")
	return finalString, nil
}
