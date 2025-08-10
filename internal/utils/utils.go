package utils

import (
	"fmt"
	"strings"
)

func QuoteAndCombine(parts ...string) string {
	needQuotes := false

	if len(parts) == 1 {
		if strings.Contains(parts[0], " ") {
			needQuotes = true
		}
	} else {
		needQuotes = true
	}

	singleString := strings.Join(parts, " ")
	if !needQuotes {
		return singleString
	}

	if strings.Contains(singleString, "\"") {
		return fmt.Sprintf("'%s'", singleString)
	}

	return fmt.Sprintf("\"%s\"", singleString)
}
