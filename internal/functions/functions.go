package functions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

var lookup map[string]func(args ...any) (any, error)

func init() {
	lookup = map[string]func(args ...any) (any, error){
		"len":          length,
		"trim":         trim,
		"line":         line,
		"lines":        lines,
		"string":       toString,
		"keys":         keys,
		"values":       values,
		"map":          doMap,
		"b64encode":    doB64encode,
		"b64decode":    doB64decode,
		"b64encodeUrl": doUrlSafeB64encode,
		"b64decodeUrl": doUrlSafeB64decode,
		"json":         toJson,
		"yaml":         toYamnl,
	}
}

func from_string_array(v []string) []any {
	conv := make([]any, len(v))
	for i, val := range v {
		conv[i] = val
	}

	return conv
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
		return nil, fmt.Errorf("invalid argument type %T", t)
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
		return nil, fmt.Errorf("invalid argument type %T", t)
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
		return nil, fmt.Errorf("invalid argument type %T", t)
	}
}

func lines(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case string:
		return from_string_array(strings.Split(strings.TrimSuffix(t, "\n"), "\n")), nil
	default:
		return nil, fmt.Errorf("invalid argument type %T", t)
	}
}

func toString(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	return fmt.Sprintf("%v", arg), nil
}

func toJson(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	jBytes, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	return string(jBytes), nil
}

func toYamnl(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	yBytes, err := yaml.Marshal(arg)
	if err != nil {
		return nil, err
	}

	return string(yBytes), nil
}

// keys returns an array of keys from a mapping type
func keys(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case map[string]any:
		var keys []any
		for k := range t {
			keys = append(keys, k)
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("invalid argument type %T", t)
	}
}

// values returns an array of values from a mapping type
func values(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	arg := args[0]
	switch t := arg.(type) {
	case map[string]any:
		var values []any
		for _, v := range t {
			values = append(values, v)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("invalid argument type %T", t)
	}
}

// performs a mapping function against an array or map
// in the case of an array, a new array will be returned using the mapping function to transform the item
// in the case of a map, a new map will be returned, using the mapping function to transform the value
func doMap(args ...any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	object := args[0]
	fName, ok := args[1].(string)

	if !ok {
		return nil, fmt.Errorf("2nd argument must be a function name in string form")
	}

	f, ok := lookup[fName]
	if !ok {
		return nil, fmt.Errorf("function \"%s\" is not unknown", fName)
	}

	switch t := object.(type) {
	case []any:
		var newArr []any
		for _, o := range t {
			nValue, err := f(o)
			if err != nil {
				return nil, fmt.Errorf("unable to apply function \"%s\" to value \"%v\"", fName, o)
			}
			newArr = append(newArr, nValue)
		}
		return newArr, nil
	case map[string]any:
		newMap := map[string]any{}
		for k, v := range t {
			nValue, err := f(v)
			if err != nil {
				return nil, fmt.Errorf("unable to apply function \"%s\" to value \"%v\"", fName, v)
			}
			newMap[k] = nValue
		}
		return newMap, nil
	default:
		return nil, fmt.Errorf("invalid argument type %T for iterable", t)
	}
}

func doB64encode(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid argument type %T", value)
	}

	return base64.StdEncoding.EncodeToString([]byte(value)), nil
}

func doB64decode(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid argument type %T", value)
	}

	b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func doUrlSafeB64encode(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid argument type %T", value)
	}

	return base64.URLEncoding.EncodeToString([]byte(value)), nil
}

func doUrlSafeB64decode(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid argument type %T", value)
	}

	b, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func Call(name string, args ...any) (any, error) {
	f, ok := lookup[name]
	if !ok {
		return nil, fmt.Errorf("unknown function \"%s\"", name)
	}

	v, err := f(args...)
	if err != nil {
		return nil, fmt.Errorf("error with function \"%s\": %w", name, err)
	}

	return v, nil
}
