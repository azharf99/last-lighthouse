package auth

import (
	"testing"
)

func TestAuthGenerateAndValidate(t *testing.T) {
	auth := NewAuthenticator(nil)

	token, userID, err := auth.GenerateGuestToken("Alice")
	if err != nil {
		t.Fatalf("GenerateGuestToken failed: %v", err)
	}
	if token == "" || userID == "" {
		t.Fatalf("expected non-empty token and userID")
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != userID || claims.DisplayName != "Alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Invalid token
	if _, err := auth.ValidateToken("invalid.token.here"); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}

	// Empty name rejected
	if _, _, err := auth.GenerateGuestToken("   "); err != ErrEmptyName {
		t.Fatalf("expected ErrEmptyName, got %v", err)
	}
}
