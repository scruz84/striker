package common

import (
	"fmt"
	"math/rand"
)

// %s is replaced with a place name from placeNamesByCountry (see teamnames.go) - a stadium takes
// its host town's name the same way a club does, just combined with a stadium-style suffix
// instead of a club-style one.
var stadiumNamePatternsByCountry = map[string][]string{
	"es": {"Estadio %s", "Campo de %s", "Ciudad Deportiva %s"},
	"en": {"%s Stadium", "%s Park", "%s Arena", "%s Ground"},
	"de": {"%s Stadion", "%s Arena", "%s Sportpark"},
	"fr": {"Stade %s", "Parc %s", "Arena %s"},
	"it": {"Stadio %s", "Arena %s", "Campo %s"},
	"pt": {"Estadio %s", "Parque %s", "Arena %s"},
	"nl": {"%s Stadion", "%s Arena", "Sportpark %s"},
	"se": {"%s Arena", "%s IP", "%s Stadion"},
	"pl": {"Stadion %s", "%s Arena"},
	"gr": {"Stadio %s", "%s Arena"},
	"br": {"Estadio %s", "Arena %s", "Parque %s"},
	"ar": {"Estadio %s", "Cancha %s", "Parque %s"},
	"mx": {"Estadio %s", "Cancha %s", "Parque %s"},
	"co": {"Estadio %s", "Cancha %s", "Parque %s"},
	"cl": {"Estadio %s", "Cancha %s", "Parque %s"},
	"pe": {"Estadio %s", "Cancha %s", "Parque %s"},
	"uy": {"Estadio %s", "Cancha %s", "Parque %s"},
	"us": {"%s Stadium", "%s Field", "%s Arena"},
	"ca": {"%s Stadium", "%s Field", "%s Arena"},
	"ec": {"Estadio %s", "Cancha %s", "Parque %s"},
	"jp": {"%s Stadium", "%s Dome", "%s Arena"},
	"kr": {"%s Stadium", "%s Arena"},
	"cn": {"%s Stadium", "%s Arena"},
	"in": {"%s Stadium", "%s Ground"},
	"th": {"%s Stadium", "%s Arena"},
	"vn": {"%s Stadium", "%s Arena"},
	"id": {"Stadion %s", "%s Arena"},
	"ph": {"%s Stadium", "%s Arena"},
	"sa": {"Malaab %s", "%s Stadium"},
	"ir": {"Varzeshgah %s", "%s Stadium"},
	"ng": {"%s Stadium", "%s Sports Complex", "%s Arena"},
	"eg": {"Malaab %s", "%s Stadium"},
	"gh": {"%s Park", "%s Stadium"},
	"ci": {"%s Stadium", "%s Arena"},
	"cm": {"%s Stadium", "%s Arena"},
	"sn": {"Stade %s", "%s Stade"},
	"za": {"%s Stadium", "%s Park"},
	"ma": {"Stade %s", "%s Stade"},
	"dz": {"Stade %s", "%s Stade"},
	"ke": {"%s Stadium", "%s Park"},
}

// RandomStadiumName generates a stadium name for country in the same spirit as RandomTeamName -
// an invented place name (drawn from the same pool RandomTeamName uses, so a club and its ground
// plausibly share a hometown) combined with a stadium-style naming pattern typical of that
// country (e.g. "Estadio Villanueva", "Ashford Park", "Nordhausen Stadion"). Falls back to the
// "en" pools for an unrecognized country, same as RandomTeamName.
func RandomStadiumName(country string) string {
	places, ok := placeNamesByCountry[country]
	if !ok {
		places = placeNamesByCountry["en"]
	}
	patterns, ok := stadiumNamePatternsByCountry[country]
	if !ok {
		patterns = stadiumNamePatternsByCountry["en"]
	}
	place := places[rand.Intn(len(places))]
	pattern := patterns[rand.Intn(len(patterns))]
	return fmt.Sprintf(pattern, place)
}
