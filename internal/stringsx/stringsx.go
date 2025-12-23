package stringsx

import "strings"

// Clip returns at most max characters of s.
// If max <= 0, an empty string is returned.
func Clip(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// Normalize trims spaces and converts a string to lower case.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// IsEmpty reports whether s is empty after trimming spaces.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
