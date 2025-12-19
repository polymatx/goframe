package util

import "strings"

// CamelToSnake converts CamelCase to snake_case
func CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// SnakeToCamel converts snake_case to CamelCase
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = Capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

// Truncate truncates a string to maxLen
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Capitalize capitalizes the first letter
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RemoveSpaces removes all spaces
func RemoveSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}
