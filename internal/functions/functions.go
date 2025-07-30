package functions

import (
	"fmt"
	"strings"
)

var lookup = map[string]func(args ...any) (any, error){
	"len":       length,
	"trim":      trim,
	"line":      line,
	"lines":     lines,
	"to_string": to_string,
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

func line(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case string:
		return strings.Split(t, "\n")[0], nil
	default:
		return nil, fmt.Errorf("invalid argument type")
	}
}

func lines(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case string:
		return strings.Split(strings.TrimSuffix(t, "\n"), "\n"), nil
	default:
		return nil, fmt.Errorf("invalid argument type")
	}
}

func to_string(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	return fmt.Sprintf("%v", arg), nil
}

func Call(name string, args ...any) (any, error) {
	f, ok := lookup[name]
	if !ok {
		return nil, fmt.Errorf("unknown function \"%s\"", name)
	}

	return f(args...)
}
