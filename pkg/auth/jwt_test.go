package auth

import (
	"testing"
	"time"
)

func TestNewJWTManager(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}
}

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	manager := NewJWTManager("test-secret-key-12345", time.Hour)

	t.Run("generate and validate token", func(t *testing.T) {
		token, err := manager.GenerateToken("user-123", "john", "admin", nil)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		claims, err := manager.ValidateToken(token)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if claims.UserID != "user-123" {
			t.Errorf("expected UserID 'user-123', got '%s'", claims.UserID)
		}
		if claims.Username != "john" {
			t.Errorf("expected Username 'john', got '%s'", claims.Username)
		}
		if claims.Role != "admin" {
			t.Errorf("expected Role 'admin', got '%s'", claims.Role)
		}
	})

	t.Run("with extra claims", func(t *testing.T) {
		extra := map[string]interface{}{
			"department": "engineering",
			"level":      5,
		}
		token, err := manager.GenerateToken("user-456", "jane", "user", extra)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		claims, err := manager.ValidateToken(token)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if claims.Extra["department"] != "engineering" {
			t.Errorf("expected department 'engineering', got '%v'", claims.Extra["department"])
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := manager.ValidateToken("invalid-token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		otherManager := NewJWTManager("different-secret", time.Hour)
		token, _ := otherManager.GenerateToken("user", "name", "role", nil)

		_, err := manager.ValidateToken(token)
		if err == nil {
			t.Error("expected error for token signed with different secret")
		}
	})
}

func TestJWTManager_RefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key-12345", time.Hour)

	token, err := manager.GenerateToken("user-123", "john", "admin", nil)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	newToken, err := manager.RefreshToken(token)
	if err != nil {
		t.Fatalf("failed to refresh token: %v", err)
	}

	if newToken == "" {
		t.Fatal("expected non-empty refreshed token")
	}

	// Validate the new token
	claims, err := manager.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("failed to validate refreshed token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID preserved, got '%s'", claims.UserID)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	// Create a manager with very short expiration
	manager := NewJWTManager("test-secret", -time.Hour) // Already expired

	token, err := manager.GenerateToken("user", "name", "role", nil)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
