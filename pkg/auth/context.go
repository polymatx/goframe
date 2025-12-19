package auth

import (
	"context"
)

type contextKey string

const claimsKey contextKey = "jwt_claims"

// WithClaims adds claims to context
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims retrieves claims from context
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// MustGetClaims retrieves claims from context or panics
func MustGetClaims(ctx context.Context) *Claims {
	claims, ok := GetClaims(ctx)
	if !ok {
		panic("claims not found in context")
	}
	return claims
}
