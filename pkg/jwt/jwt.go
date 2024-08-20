package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	AccessTokenSecret          string
	AccessTokenExpirationTime  time.Duration
	RefreshTokenSecret         string
	RefreshTokenExpirationTime time.Duration
}

type JWT struct {
	accessSecret          []byte
	accessExpirationTime  time.Duration
	refreshSecret         []byte
	refreshExpirationTime time.Duration
}

func New(cfg *Config) *JWT {
	return &JWT{
		accessSecret:          []byte(cfg.AccessTokenSecret),
		accessExpirationTime:  cfg.AccessTokenExpirationTime,
		refreshSecret:         []byte(cfg.RefreshTokenSecret),
		refreshExpirationTime: cfg.RefreshTokenExpirationTime,
	}
}

func (j *JWT) GenerateAccessToken(userID string) (string, error) {
	return j.generateToken(userID, j.accessSecret, j.accessExpirationTime)
}

func (j *JWT) GenerateRefreshToken(userID string) (string, error) {
	return j.generateToken(userID, j.refreshSecret, j.refreshExpirationTime)
}

func (j *JWT) generateToken(userID string, secret []byte, expirationTime time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expirationTime)),
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (j *JWT) ValidateAccessToken(tokenString string) (string, error) {
	return j.validateToken(tokenString, j.accessSecret)
}

func (j *JWT) ValidateRefreshToken(tokenString string) (string, error) {
	return j.validateToken(tokenString, j.refreshSecret)
}

func (j *JWT) validateToken(tokenString string, secret []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("invalid token")
}
