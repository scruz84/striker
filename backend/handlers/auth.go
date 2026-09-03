package handlers

import (
	"github.com/golang-jwt/jwt/v4"
)

func TokenClaimsNoValidation(jwtToken string) (*jwt.MapClaims, error) {
	jwtParser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _ := jwtParser.Parse(jwtToken, nil)
	mapClaims := token.Claims.(jwt.MapClaims)
	return &mapClaims, nil
}
