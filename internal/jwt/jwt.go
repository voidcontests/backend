package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type CustomClaims struct {
	jwt.StandardClaims
	Address string `json:"address"`
	ID      int32  `json:"id"`
}

func GenerateToken(address string, id int32, secret string) (string, error) {
	claims := &CustomClaims{
		jwt.StandardClaims{
			ExpiresAt: time.Now().AddDate(100, 0, 0).Unix(),
		},
		address,
		id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func Parse(token, secret string) (address string, err error) {
	jsonwebtoken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := jsonwebtoken.Claims.(CustomClaims)
	if !ok || !jsonwebtoken.Valid {
		return "", fmt.Errorf("token.ParseToID: can't parse invalid jsonwebtoken")
	}
	address = claims.Address

	return address, nil
}
