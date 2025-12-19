package validator

import (
	"encoding/json"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// IsEmail validates email format
func IsEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsURL validates URL format
func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// IsPhone validates phone number format
func IsPhone(phone string) bool {
	re := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return re.MatchString(phone)
}

// IsStrongPassword checks password strength
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// IsIPv4 validates IPv4 address
func IsIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && strings.Contains(ip, ".")
}

// IsIPv6 validates IPv6 address
func IsIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && strings.Contains(ip, ":")
}

// IsJSON validates JSON string
func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// IsAlpha checks if string contains only letters
func IsAlpha(str string) bool {
	for _, c := range str {
		if !unicode.IsLetter(c) {
			return false
		}
	}
	return len(str) > 0
}

// IsAlphanumeric checks if string contains only letters and digits
func IsAlphanumeric(str string) bool {
	for _, c := range str {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return len(str) > 0
}

// IsNumeric checks if string contains only digits
func IsNumeric(str string) bool {
	for _, c := range str {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(str) > 0
}

// IsUUID validates UUID format
func IsUUID(str string) bool {
	re := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	return re.MatchString(str)
}

// IsEmpty checks if string is empty or whitespace only
func IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

// MinLength checks minimum string length
func MinLength(str string, min int) bool {
	return len(str) >= min
}

// MaxLength checks maximum string length
func MaxLength(str string, max int) bool {
	return len(str) <= max
}

// InRange checks if number is in range
func InRange(n, min, max int) bool {
	return n >= min && n <= max
}
