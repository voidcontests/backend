package jwt

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type jwtCustomClaims struct {
	Address string `json:"address"`
	jwt.StandardClaims
}

func GenerateToken(address, secret string) (string, error) {
	claims := &jwtCustomClaims{
		address,
		jwt.StandardClaims{
			ExpiresAt: time.Now().AddDate(100, 0, 0).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
