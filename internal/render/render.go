package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/frozengoats/eval"
)

var templateFinder = regexp.MustCompile(`<!\s*.*?\s*!>`)

func Render(template string, varLookup eval.VariableLookup, funcCall eval.FunctionCall) (any, error) {
	isEncompassed := strings.HasPrefix(template, "<!") && strings.HasSuffix(template, "!>")
	matches := templateFinder.FindAllStringSubmatchIndex(template, -1)

	if len(matches) == 1 && isEncompassed {
		// this is a single template and should not undergo any string concatenation,
		// only the raw value is used
		match := matches[0]
		start := match[0]
		end := match[1]
		result, err := eval.Evaluate(template[start+2:end-2], varLookup, funcCall)
		if err != nil {
			return "", err
		}
		return result, nil
	}

	var newParts []string
	lastEnd := 0
	for _, match := range matches {
		start := match[0]
		end := match[1]

		if lastEnd < start {
			newParts = append(newParts, template[lastEnd:start])
		}

		result, err := eval.Evaluate(template[start+2:end-2], varLookup, funcCall)
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

func ToString(value any) string {
	return fmt.Sprintf("%v", value)
}
