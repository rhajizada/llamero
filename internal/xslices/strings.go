package xslices

import (
	"slices"
	"strings"
)

func UniqueTrimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}
