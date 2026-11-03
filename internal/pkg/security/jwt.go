package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("token inválido ou expirado")

const issuer = "neonflow-api"

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenProvider struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenProvider(secret string, ttl time.Duration) *TokenProvider {
	return &TokenProvider{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Generate emite um JWT HS256 com o usuário e o papel.
func (p *TokenProvider) Generate(userID, role string) (string, time.Time, error) {
	now := p.now()
	expiresAt := now.Add(p.ttl)
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(p.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse valida assinatura, algoritmo, emissor e expiração.
func (p *TokenProvider) Parse(token string) (Claims, error) {
	var claims Claims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		return p.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithTimeFunc(p.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid || claims.UserID == "" || claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
