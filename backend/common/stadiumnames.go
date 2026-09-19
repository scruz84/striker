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
