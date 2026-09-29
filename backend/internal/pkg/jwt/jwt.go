package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	OwnerID  uint   `json:"owner_id"`
	Username string `json:"username"`
	TokenTyp string `json:"typ"` // access | refresh
	jwt.RegisteredClaims
}

type Manager struct {
	secret          []byte
	accessExpire    time.Duration
	refreshExpire   time.Duration
}

func NewManager(secret string, accessHours, refreshHours int) *Manager {
	return &Manager{
		secret:        []byte(secret),
		accessExpire:  time.Duration(accessHours) * time.Hour,
		refreshExpire: time.Duration(refreshHours) * time.Hour,
	}
}

func (m *Manager) GenerateAccess(ownerID uint, username string) (string, time.Time, error) {
	exp := time.Now().Add(m.accessExpire)
	claims := Claims{
		OwnerID:  ownerID,
		Username: username,
		TokenTyp: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(m.secret)
	return s, exp, err
}

func (m *Manager) GenerateRefresh(ownerID uint, username string) (string, time.Time, error) {
	exp := time.Now().Add(m.refreshExpire)
	claims := Claims{
		OwnerID:  ownerID,
		Username: username,
		TokenTyp: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(m.secret)
	return s, exp, err
}

func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing tidak valid")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}
