package common

import (
	"fmt"
	"math/rand"
)

// Invented place-like names per seeded country (see handlers.seededCountries) - deliberately
// not real cities or club names, just plausible-sounding ones in that country's style, combined
// with a locally-flavored naming pattern in RandomTeamName.
var placeNamesByCountry = map[string][]string{
	"es": {
		"Villanueva", "Montoro", "Peñarroya", "Arandilla", "Bellavista", "Torreblanca",
		"Puigverd", "Sanlázaro", "Almendral", "Riofrío", "Casasola", "Montseny",
		"Valdemora", "Fuenteagria", "Sierraverde", "Costamar",
	},
	"en": {
		"Ashford", "Millbrook", "Kingsley", "Thornbury", "Whitfield", "Eastwood",
		"Broadfield", "Rushmoor", "Hartwell", "Denholm", "Overbridge", "Fenwick",
		"Radcombe", "Marston", "Aldergate", "Ravensworth",
	},
	"de": {
		"Nordhausen", "Falkenberg", "Rheinbach", "Bergstadt", "Waldheim", "Steinfurt",
		"Lindenau", "Grunwald", "Eisenbach", "Hohenfeld", "Rotenburg", "Dornstadt",
		"Kaltenbrunn", "Sonnenberg", "Talheim", "Bruckstadt",
	},
	"fr": {
		"Montval", "Beaulieu", "Clairefontaine", "Vernoux", "Roussel", "Charmont",
		"Belrive", "Fontenay", "Lauris", "Valcourt", "Aubignac", "Montrose",
		"Rochebrune", "Grandchamp", "Servignac", "Aiglemont",
	},
	"it": {
		"Montefiore", "Vallescura", "Torrenova", "Campolungo", "Riobianco", "Castelmare",
		"Pratolungo", "Collegrande", "Fontanalta", "Serrapiana", "Piandoro", "Montesanto",
		"Casalvento", "Bellaripa",
	},
	"pt": {
		"Vilanova", "Montesinho", "Ribalonga", "Casaltorto", "Fontelas", "Serracosta",
		"Valmonte", "Pedraverde", "Riodouro", "Montebranco", "Campolide", "Alvorada",
		"Bragamonte", "Ponteverde",
	},
}

// %s is replaced with a place name from placeNamesByCountry.
var teamNamePatternsByCountry = map[string][]string{
	"es": {"Real %s", "Atlético %s", "%s CF", "%s FC", "Deportivo %s", "Unión %s", "%s Balompié"},
	"en": {"%s United", "%s City", "%s Town", "%s Rovers", "%s Wanderers", "%s Athletic", "%s Rangers"},
	"de": {"%s FC", "SV %s", "TSV %s", "FC %s", "%s Sportfreunde", "Borussia %s"},
	"fr": {"%s FC", "AS %s", "Olympique %s", "Racing %s", "Stade %s", "%s United"},
	"it": {"%s FC", "AC %s", "AS %s", "US %s", "%s Calcio", "Virtus %s", "Polisportiva %s"},
	"pt": {"%s FC", "Sporting %s", "%s SC", "Uniao %s", "Atletico %s", "%s Futebol Clube", "Vitoria %s"},
}

// RandomTeamName generates a realistic-sounding club name for country - an invented place name
// combined with a naming pattern typical of that country's football culture (e.g. "Real
// Villanueva", "Ashford United", "SV Nordhausen"), replacing the old placeholder scheme of just
// appending "Team" to a random person's first name. Falls back to the "en" pools for an
// unrecognized country rather than producing an empty name.
func RandomTeamName(country string) string {
	places, ok := placeNamesByCountry[country]
	if !ok {
		places = placeNamesByCountry["en"]
	}
	patterns, ok := teamNamePatternsByCountry[country]
	if !ok {
		patterns = teamNamePatternsByCountry["en"]
	}
	place := places[rand.Intn(len(places))]
	pattern := patterns[rand.Intn(len(patterns))]
	return fmt.Sprintf(pattern, place)
}
