package report

import "strings"

func TrimPrefix(name, prefix string) string {
	trimmed := strings.TrimPrefix(name, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")

	if trimmed == "" {
		trimmed = "."
	}

	return trimmed
}
