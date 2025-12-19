package validator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// IsEmail validates email format
func IsEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsURL validates URL format
func IsURL(url string) bool {
	pattern := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}.*$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsPhone validates phone number (basic)
func IsPhone(phone string) bool {
	pattern := `^\+?[0-9]{10,15}$`
	matched, _ := regexp.MatchString(pattern, strings.ReplaceAll(phone, " ", ""))
	return matched
}

// IsAlphanumeric checks if string contains only alphanumeric characters
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlpha checks if string contains only letters
func IsAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsNumeric checks if string contains only digits
func IsNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// MinLength checks minimum length
func MinLength(s string, min int) bool {
	return len(s) >= min
}

// MaxLength checks maximum length
func MaxLength(s string, max int) bool {
	return len(s) <= max
}

// InRange checks if length is in range
func InRange(s string, min, max int) bool {
	l := len(s)
	return l >= min && l <= max
}

// IsStrongPassword checks password strength
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// IsIPv4 validates IPv4 address
func IsIPv4(ip string) bool {
	pattern := `^(\d{1,3}\.){3}\d{1,3}$`
	matched, _ := regexp.MatchString(pattern, ip)
	if !matched {
		return false
	}

	parts := strings.Split(ip, ".")
	for _, part := range parts {
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}
	return true
}

// IsJSON checks if string is valid JSON
func IsJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
