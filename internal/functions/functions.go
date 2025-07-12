package functions

import (
	"fmt"
	"strings"
)

var lookup = map[string]func(args ...any) (any, error){
	"len":  length,
	"trim": trim,
}

func length(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case string:
		return len(t), nil
	case []any:
		return len(t), nil
	default:
		return nil, fmt.Errorf("invalid argument type")
	}
}

func trim(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case string:
		return strings.Trim(t, "\n "), nil
	default:
		return nil, fmt.Errorf("invalid argument type")
	}
}

func Call(name string, args ...any) (any, error) {
	f, ok := lookup[name]
	if !ok {
		return nil, fmt.Errorf("unknown function \"%s\"", name)
	}

	return f(args...)
}
