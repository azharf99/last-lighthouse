package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrEmptyName    = errors.New("display name cannot be empty")
)

type Claims struct {
	UserID      string `json:"sub"`
	DisplayName string `json:"name"`
	jwt.RegisteredClaims
}

type Authenticator struct {
	secret []byte
}

func NewAuthenticator(secret []byte) *Authenticator {
	if len(secret) == 0 {
		secret = RandomSecret()
	}
	return &Authenticator{secret: secret}
}

func RandomSecret() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

// GenerateGuestToken creates a new UUID guest user and signs a JWT for 30 days.
func (a *Authenticator) GenerateGuestToken(displayName string) (string, string, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return "", "", ErrEmptyName
	}

	userID := "u_" + hex.EncodeToString([]byte(uuid.New().String()[:8]))
	now := time.Now()
	claims := Claims{
		UserID:      userID,
		DisplayName: name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, userID, nil
}

// ValidateToken parses and validates a signed JWT token string.
func (a *Authenticator) ValidateToken(tokenStr string) (*Claims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
