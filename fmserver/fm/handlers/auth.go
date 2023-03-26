package handlers

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
)

var TokenSecret = "xx"

func TokenClaimsWithValidation(jwtToken string, validateWithSecret string) (*jwt.Claims, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return validateWithSecret, nil
	})

	if err != nil {
		return nil, err
	}
	return &token.Claims, nil
}

func TokenClaimsNoValidation(jwtToken string) (*jwt.MapClaims, error) {
	jwtParser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _ := jwtParser.Parse(jwtToken, nil)
	mapClaims := token.Claims.(jwt.MapClaims)
	return &mapClaims, nil
}

func createJWTToken() {

}
