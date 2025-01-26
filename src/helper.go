package src

import "strings"

func trimPrefix(name, prefix string) string {
	trimmed := strings.TrimPrefix(name, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")

	if trimmed == "" {
		trimmed = "."
	}

	return trimmed
}
