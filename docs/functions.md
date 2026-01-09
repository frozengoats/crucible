# Complete function list

| name | description | args | return |
|--|--|--|--|
| b64decode  | decodes a base64 encoded string  | value: string  | string |
| b64encode | encodes a string to a base64 encoded string | value: string | string |
| b64decodeUrl  | decodes a url safe base64 encoded string  | value: string  | string |
| b64encodeUrl | encodes a string to a url safe base64 encoded string | value: string | string |
| hash | returns a hex encoded hash sum of the input string<br/>valid algorithms are `sha256`, `md5` | value: string<br/>algo: string | string |
| json  | returns a json string representation of any data | value: any  | string |
| keys  | returns an array of keys | value: map[string]any  | []any |
| len | returns the length of an array, string, or mapping | value: any | float64 |
| line | returns the first line of a string, removing anything after and including the first newline | value: string | string |
| lines | returns an array of lines from a multiline string, removing all newline characters | value: string | []any |
| lower | returns the lowercase version of the provided string | value: string | string |
| map | iterates an array or map values and executes a function against each element, returning an array<br/>example: `map(.Values.myArray, "b64encode")` | value: any<br/> funcName: string | []any |
| semver | returns an alpha comparable string representation of a semver string, allowing it to be properly compared.  prefix of v is considered irrelevant in the comparison
| split | returns an array, resulting from splitting a string by a separator value<br/>example: `split(.Context.myString, ",")` | value: string<br/>separator: string | []any |
| string | returns a string representation of the input data | value: any | string |
| trim | trims leading and trailing whitespace from string, returning new string | value: string | string |
| upper | returns the uppercase version of the provided string | value: string | string |
| values | returns an array of values from a mapping | value: map[string]any | []any |
| yaml | returns a yaml string representation of the input data | value: any | string |
