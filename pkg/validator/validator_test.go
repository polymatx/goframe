package validator

import (
	"strings"
	"testing"
)

func TestIsEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"simple valid email", "user@example.com", true},
		{"valid with dots and plus", "first.last+tag@sub.example.co", true},
		{"valid with digits", "user123@example99.com", true},
		{"missing at sign", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing tld", "user@example", false},
		{"single char tld", "user@example.c", false},
		{"double at sign", "user@@example.com", false},
		{"space in local part", "us er@example.com", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmail(tt.email); got != tt.want {
				t.Errorf("IsEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"http url", "http://example.com", true},
		{"https url", "https://example.com", true},
		{"url with path and query", "https://example.com/path?q=1", true},
		{"url with subdomain", "https://sub.example.com", true},
		{"missing scheme", "example.com", false},
		{"unsupported scheme", "ftp://example.com", false},
		{"missing tld", "http://example", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsURL(tt.url); got != tt.want {
				t.Errorf("IsURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"ten digits", "1234567890", true},
		{"fifteen digits", "123456789012345", true},
		{"with plus prefix", "+905551234567", true},
		{"with spaces", "+90 555 123 4567", true},
		{"nine digits too short", "123456789", false},
		{"sixteen digits too long", "1234567890123456", false},
		{"contains letters", "abc4567890", false},
		{"contains dashes", "123-456-7890", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPhone(tt.phone); got != tt.want {
				t.Errorf("IsPhone(%q) = %v, want %v", tt.phone, got, tt.want)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"letters only", "abcDEF", true},
		{"digits only", "12345", true},
		{"letters and digits", "abc123", true},
		{"unicode letters", "héllo", true},
		{"with space", "abc 123", false},
		{"with punctuation", "abc-123", false},
		{"empty string", "", true}, // current behavior: vacuously true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAlphanumeric(tt.input); got != tt.want {
				t.Errorf("IsAlphanumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsAlpha(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase letters", "abc", true},
		{"mixed case letters", "AbC", true},
		{"unicode letters", "héllo", true},
		{"contains digit", "abc1", false},
		{"contains space", "a b", false},
		{"empty string", "", true}, // current behavior: vacuously true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAlpha(tt.input); got != tt.want {
				t.Errorf("IsAlpha(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"digits only", "0123456789", true},
		{"contains letter", "123a", false},
		{"contains sign", "-123", false},
		{"contains decimal point", "1.5", false},
		{"empty string", "", true}, // current behavior: vacuously true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNumeric(tt.input); got != tt.want {
				t.Errorf("IsNumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		min   int
		want  bool
	}{
		{"longer than min", "hello", 3, true},
		{"exactly min", "hello", 5, true},
		{"shorter than min", "hi", 3, false},
		{"empty with zero min", "", 0, true},
		{"empty with positive min", "", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinLength(tt.input, tt.min); got != tt.want {
				t.Errorf("MinLength(%q, %d) = %v, want %v", tt.input, tt.min, got, tt.want)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  bool
	}{
		{"shorter than max", "hi", 5, true},
		{"exactly max", "hello", 5, true},
		{"longer than max", "hello world", 5, false},
		{"empty with zero max", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxLength(tt.input, tt.max); got != tt.want {
				t.Errorf("MaxLength(%q, %d) = %v, want %v", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		min, max int
		want     bool
	}{
		{"within range", "hello", 3, 10, true},
		{"at lower bound", "abc", 3, 10, true},
		{"at upper bound", "abcdefghij", 3, 10, true},
		{"below range", "ab", 3, 10, false},
		{"above range", "abcdefghijk", 3, 10, false},
		{"empty in zero range", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InRange(tt.input, tt.min, tt.max); got != tt.want {
				t.Errorf("InRange(%q, %d, %d) = %v, want %v", tt.input, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"all requirements met", "Password1!", true},
		{"symbol as special char", "Passw0rd+ok", true},
		{"too short", "Pw1!abc", false},
		{"missing uppercase", "password1!", false},
		{"missing lowercase", "PASSWORD1!", false},
		{"missing digit", "Password!!", false},
		{"missing special char", "Password11", false},
		{"space is not special", "Password1 x", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrongPassword(tt.password); got != tt.want {
				t.Errorf("IsStrongPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"typical address", "192.168.1.1", true},
		{"all zeros", "0.0.0.0", true},
		{"broadcast", "255.255.255.255", true},
		{"octet above 255", "256.1.1.1", false},
		{"last octet above 255", "1.1.1.999", false},
		{"too few octets", "1.2.3", false},
		{"too many octets", "1.2.3.4.5", false},
		{"letters", "a.b.c.d", false},
		{"empty string", "", false},
		{"with port", "192.168.1.1:80", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIPv4(tt.ip); got != tt.want {
				t.Errorf("IsIPv4(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"object", `{"a": 1, "b": "two"}`, true},
		{"array", `[1, 2, 3]`, true},
		{"quoted string", `"hello"`, true},
		{"number", `123`, true},
		{"boolean", `true`, true},
		{"null literal", `null`, true},
		{"nested object", `{"a": {"b": [1, {"c": null}]}}`, true},
		{"unclosed brace", `{"a": 1`, false},
		{"bare word", `hello`, false},
		{"trailing comma", `{"a": 1,}`, false},
		{"empty string", "", false},
		{"large valid array", "[" + strings.Repeat("1,", 999) + "1]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSON(tt.input); got != tt.want {
				t.Errorf("IsJSON(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
