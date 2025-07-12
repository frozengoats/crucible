# eval
evaluate complex expressions and return a single value

eval is an expression evaluation framework which can deal with the following characteristics
- variables (with retrieval callbacks for arbitrary data sources)
- basic mathematical and boolean logic operators
- functions (with lookup callbacks designed for complete extensibility - no builtins)
- parenthesized evaluation groups
- standard order of operations (exponents, division, multiplication first)
- numbers are treated always treated as floating point

## supported operators
| operator    | description |
| -------- | ------- |
| `==`  | perform a strict equality evaluation (boolean output) on same types only, does not work with arrays/mappings |
| `!=` | perform strict inequality evaluation (same rules as `==`) |
| `>` | greater than, applies to numbers and strings, where ASCII value determines weight |
| `>=` | greater than or equal to (same rules as `>`) |
| `<` | less than (same rules as `>`) |
| `<=` | less than or equal to (same rules as `>`) |
| `&&` | logical and, if both values evaluate to non-empty, last occurrence will be selected, otherwise, first empty occurrence will be selected.  applies to numbers, strings, booleans, arrays and mappings |
| `\|\|` | logical or, same rules apply as `&&` |
| `+` | addition/concatenation. performs addition on numbers and will concatenate both strings and arrays |
| `-` | subtraction, applies to numbers only |
| `*` | multiplication, applies to numbers only |
| `/` | division, applies to numbers only |
| `**` | exponent, applies to numbers only |

## type inference and strings
eval has strict and predictable rules when it comes to type inference.

- quoted strings (single or double quotes) will always be interpreted as string literals (example: `'hello world'`, or `"hello world"`).  this includes strings which may appear as variables.  quotation precludes them as being interpreted as anything but string literals.
- unquoted strings beginning with `.` followed by an alpha/underscore (regardless of case) will be interpreted as a variable.  (example: `.my.variable`, `.some_variable`, `.__my_var`).  hyphens are not supported in variable names due to the fact that they will be interpreted as a minus operator.  generally speaking one should adhere to the rule of alpha-numeric and underscore naming, so long as the first.  this protects against potential future adoption of other symbol characters such as `$` etc. which at the time of initial writing, hold no special representation.
- unquoted strings equalling (strict case sensitivity) `true` or `false` are treated as boolean values.
- unquoted strings followed by a parenthesis group are treated as functions.  functions can accept one or more arguments but must return a single value.  example `myFunc(abc, def, ghi)`, `my_func(.my_var, 123, abc)`.
- unquoted strings which contain only numerically valid characters, will be interpreted as floating point numbers (example `123`, `33.0`, `0`)

## basic example
```
import (
  "github.com/frozengoats/eval"
)

result, err := eval.Evaluate("(10 + 1) * 3 / 100", nil, nil)
if err != nil {
  log.Fatal(err)
}

fmt.Println(result)
```

`Evaluate` will evaluate the provided expression and return a single `any` type interface, representing the final value of the expression.  any error occurring along the way, will be returned in the error variable.  the two `nil` values being supplied here, are callbacks for variable lookup, and function lookup respectively.

## variable and function lookup
because eval is a generic evaluation framework, it does not define any variable storage mechanisms, nor does it define any builtin functions.  both variable data store, and function implementation are left to the implementer, allowing for maximum flexibility.

here is an example using variable lookup, using [kvstore](https://github.com/frozengoats/kvstore) as the variable storage/lookup backend:
```
import (
  "github.com/frozengoats/eval"
  "github.com/frozengoats/kvstore"
)

store := kvstore.NewStore()
store.Set(101, "first_key", "second_key")
store.Set("hello world", "first_key", "third_key")

// above could just as easily be a loaded YAML/JSON file, but sets up a structure as follows (illustrated using YAML notation):
//
// first_key:
//   second_key: 101
//   third_key: 'hello world'

// next a lookup function is implemented to faciliate retrieval
vLookup := func(key string) (any, error) {
  // remove the leading ".", leaving us with "first_key.second_key"
  key = strings.TrimPrefix(key, ".")

  // then use that key to access the value on the kvstore object
  return store.Get(ParseNamespaceString(key)), nil
}

result, err := eval.Evaluate(".first_key.second_key * 33 + 1", vLookup, nil)
// result should be of type float64 and equal to (101 * 33) + 1
```

the above example uses `kvstore`, however the implementor is welcome to use any means of variable value retrieval, which will in turn be substituted for the variable name during evaluation.

function lookup is performed in a very similar fashion.  when function lookup occurs, variables, etc. have already been evaluated by `eval`, thus the implementor need not perform any additional lookup, just locate, validate, and execute the function.  below is a rudimentary example of a function lookup to illustrate its usage in the simplest form - building on the previous example:

```
fLookup := func(name string, args ...any) (any, error) {
  switch name {
  case "len":
    if len(args) != 1 {
      return nil, fmt.Errorf("len expects a single argument")
    }

    arg := args[0]
    switch t := arg.(type) {
    case string:
      return len(t), nil
    case []any:
      return len(t), nil
    default:
      return nil, fmt.Errorf("unsupported argument type")
    }
  default:
    return nil, fmt.Errorf("unknown function named %s", name)
  }

  result, err := eval.Evaluate("len(.first_key.third_key) * 5", vLookup, fLookup)
  // result should be equal to the length of 'hello world' * 5, which should be 55
}
```

## helpers
`eval` comes with a few utility functions to aid in processing of evaluated results
| function   | description |
| -------- | ------- |
| IsTruthy | given an `any` interface, returns `true` if the value evaluates to truthy |
| AsString | given an `any` interface, returns a `string` cast or empty string if not castable
| AsNumber | given an `any` interface, returns a `float64` cast or `0` value if not castable
| AsBool | given an `any` interface, returns a `bool` cast or `false` value if not castable
| AsArray | given an `any` interface, returns a `[]any` cast or `nil` value if not castable
| AsMapping | given an `any` interface, returns a `map[string]any` cast or `nil` value if not castable