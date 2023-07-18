package _jwt

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type jwtCertificatePayload struct {
	jwt.StandardClaims
	CustomClaims map[string]int
}

func getSecretKey() []byte {
	return []byte("supersecret") // IMPORTANT: replace it after the development
}

func Token(claims map[string]int, exp time.Duration) (string, error) {
	_claims := jwtCertificatePayload{
		StandardClaims: jwt.StandardClaims{
			// set token lifetime in timestamp
			ExpiresAt: time.Now().Add(exp).Unix(),
	},
	// add custom claims like user_id or email, 
	// it can vary according to requirements
	CustomClaims: claims,
	}

	// generate a string using claims and HS256 algorithm
	tokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, _claims)

	// sign the generated key using secretKey
	token, err := tokenString.SignedString(getSecretKey()) 

	return token, err
}

func Claims(token string) (*jwtCertificatePayload, error) {
	claims := &jwtCertificatePayload{}

	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return getSecretKey(), nil
	})

	if err != nil { // expired or invalid token
		return nil, err
	}

	return claims, nil
}