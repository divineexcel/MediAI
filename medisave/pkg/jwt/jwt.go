package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/medisave/app/internal/domain/entity"
)

type Claims struct {
	UserID uint        `json:"user_id"`
	Email  string      `json:"email"`
	Role   entity.Role `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewManager(accessSecret, refreshSecret string, accessHours, refreshDays int) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  time.Duration(accessHours) * time.Hour,
		refreshExpiry: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

func (m *Manager) GenerateTokenPair(userID uint, email string, role entity.Role) (*TokenPair, error) {
	accessExp := time.Now().Add(m.accessExpiry)

	accessClaims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "medisave",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString(m.accessSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "medisave",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString(m.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  signedAccess,
		RefreshToken: signedRefresh,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}

func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, m.accessSecret)
}

func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, m.refreshSecret)
}

func (m *Manager) validateToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
