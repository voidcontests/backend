package jwt

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID int32 `json:"id"`
}

func GenerateToken(id int32, secret string) (string, error) {
	claims := &CustomClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(100, 0, 0)),
		},
		id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func Parse(token, secret string) (id int32, err error) {
	jsonwebtoken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := jsonwebtoken.Claims.(CustomClaims)
	if !ok || !jsonwebtoken.Valid {
		return 0, fmt.Errorf("token.Parse: can't parse invalid jsonwebtoken")
	}

	return claims.UserID, nil
}
