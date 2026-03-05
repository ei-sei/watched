package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenDuration  = 2 * time.Hour
	RefreshTokenDuration = 90 * 24 * time.Hour
)

type Claims struct {
	UserID    int    `json:"uid"`
	Username  string `json:"usr"`
	IsAdmin   bool   `json:"adm"`
	IsPremium bool   `json:"prm"`
	Kind      string `json:"knd"` // "access" | "refresh"
	jwt.RegisteredClaims
}

func NewAccessToken(secret, username string, userID int, isAdmin, isPremium bool) (string, error) {
	return sign(secret, username, userID, isAdmin, isPremium, "access", AccessTokenDuration)
}

func NewRefreshToken(secret, username string, userID int, isAdmin, isPremium bool) (string, error) {
	return sign(secret, username, userID, isAdmin, isPremium, "refresh", RefreshTokenDuration)
}

func sign(secret, username string, userID int, isAdmin, isPremium bool, kind string, dur time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		IsPremium: isPremium,
		Kind:      kind,
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
