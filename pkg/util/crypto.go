package util

import (
	"crypto/md5" //#nosec G501 -- intentionally provided for legacy compatibility
	"crypto/rand"
	"crypto/sha1" //#nosec G505 -- intentionally provided for legacy compatibility
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if password matches hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MD5 returns MD5 hash of input
// Deprecated: MD5 is cryptographically broken. Use SHA256 for security-sensitive applications.
// This function is provided for legacy compatibility and non-security uses (e.g., checksums).
func MD5(input string) string {
	hash := md5.Sum([]byte(input)) //#nosec G401 -- provided for legacy compatibility
	return hex.EncodeToString(hash[:])
}

// SHA1 returns SHA1 hash of input
// Deprecated: SHA1 is cryptographically weak. Use SHA256 for security-sensitive applications.
// This function is provided for legacy compatibility and non-security uses.
func SHA1(input string) string {
	hash := sha1.Sum([]byte(input)) //#nosec G401 -- provided for legacy compatibility
	return hex.EncodeToString(hash[:])
}

// SHA256 returns SHA256 hash of input
func SHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// RandomBytes generates n random bytes
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RandomString generates random string of length n
func RandomString(n int) (string, error) {
	bytes, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:n], nil
}

// RandomToken generates a random token
func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Base64Encode encodes data to base64
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes base64 string
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// UUIDv4 generates a UUID v4
func UUIDv4() (string, error) {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
