package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

type Claims struct {
	UserID  int    `json:"uid"`
	IsAdmin bool   `json:"adm"`
	Kind    string `json:"knd"` // "access" | "refresh"
	jwt.RegisteredClaims
}

func NewAccessToken(secret string, userID int, isAdmin bool) (string, error) {
	return sign(secret, userID, isAdmin, "access", AccessTokenDuration)
}

func NewRefreshToken(secret string, userID int, isAdmin bool) (string, error) {
	return sign(secret, userID, isAdmin, "refresh", RefreshTokenDuration)
}

func sign(secret string, userID int, isAdmin bool, kind string, dur time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:  userID,
		IsAdmin: isAdmin,
		Kind:    kind,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(dur)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(secret, tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
