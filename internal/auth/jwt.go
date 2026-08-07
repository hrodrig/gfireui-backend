package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

// Claims are the authenticated user identity encoded in JWTs.
type Claims struct {
	Sub   uuid.UUID   `json:"-"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// IssueToken signs a short-lived HS256 token for the given user.
func IssueToken(secret []byte, ttl time.Duration, u *domain.User) (string, error) {
	if u == nil {
		return "", errors.New("user is required")
	}

	now := time.Now().UTC()
	claims := &Claims{
		Sub:   u.ID,
		Email: u.Email,
		Role:  u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates a signed token and returns its claims.
func ParseToken(secret []byte, token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %T", t.Method)
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}
	if claims.Subject == "" {
		return nil, errors.New("token subject is required")
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("parse token subject: %w", err)
	}
	claims.Sub = id

	return claims, nil
}
