package eval

import (
	"fmt"
	"strings"
)

type FunctionArgType int

const (
	FunctionArgTypeString FunctionArgType = iota
	FunctionArgTypeNumber
	FunctionArgTypeBool
	FunctionArgTypeAny
)

type Function struct {
	Name       string
	Call       func(...any) (any, error)
	Args       []FunctionArgType
	ReturnType FunctionArgType
}

var functions = map[string]*Function{
	"len": &Function{
		Name: "len",
		Call: Len,
		Args: []FunctionArgType{
			FunctionArgTypeAny,
		},
		ReturnType: FunctionArgTypeNumber,
	},
	"strip": &Function{
		Name: "strip",
		Call: Strip,
		Args: []FunctionArgType{
			FunctionArgTypeAny,
		},
		ReturnType: FunctionArgTypeNumber,
	},
}

func Len(args ...any) (any, error) {
	switch t := args[0].(type) {
	case string:
		return float64(len(t)), nil
	case map[string]any:
		return float64(len(t)), nil
	case []any:
		return float64(len(t)), nil
	default:
		return nil, fmt.Errorf("data type is not supported for len")
	}
}

func Strip(args ...any) (any, error) {
	switch t := args[0].(type) {
	case string:
		return strings.Trim(t, "\n\r "), nil
	default:
		return nil, fmt.Errorf("data type is not supported for strip")
	}
}

// Call executes an arbitary function call and returns the result
func Call(name string, args []any) (any, error) {
	f, ok := functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function %s", name)
	}

	expectedLen := len(f.Args)
	actualLen := len(args)
	if expectedLen != actualLen {
		return nil, fmt.Errorf("%s takes %d arguments but %d were supplied", name, expectedLen, actualLen)
	}

	return f.Call(args...)
}
