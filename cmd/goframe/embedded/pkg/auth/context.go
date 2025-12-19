package auth

import (
	"context"
	"errors"
)

type contextKey string

const claimsKey contextKey = "claims"

// SetClaims sets claims in context
func SetClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims retrieves claims from context
func GetClaims(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	if !ok {
		return nil, errors.New("no claims in context")
	}
	return claims, nil
}

// MustGetClaims retrieves claims from context or panics
func MustGetClaims(ctx context.Context) *Claims {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims
}
