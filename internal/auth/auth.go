package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret []byte, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: secret,
		ttl:    ttl,
	}
}

func (t *TokenManager) Generate(userID int) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.ttl)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return tokenString, nil
}

func (t *TokenManager) Parse(tokenString string) (int, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			return t.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return 0, fmt.Errorf("token invalid: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("token invalid")
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, fmt.Errorf("invalid user id in claims: %w", err)
	}

	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id in claims")
	}

	return userID, nil
}
