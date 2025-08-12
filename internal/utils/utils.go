package utils

import (
	"fmt"
	"strings"
)

func Quote(parts ...string) string {
	var newParts []string
	for _, p := range parts {
		if strings.Contains(p, " ") {
			if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") {
				newParts = append(newParts, p)
				continue
			}

			if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") {
				newParts = append(newParts, p)
				continue
			}

			if strings.Contains(p, "\"") {
				newParts = append(newParts, fmt.Sprintf("'%s'", p))
				continue
			}

			newParts = append(newParts, fmt.Sprintf("\"%s\"", p))
		} else {
			newParts = append(newParts, p)
		}
	}

	return strings.Join(newParts, " ")
}

func Combine(parts ...string) string {
	return strings.Join(parts, " ")
}
