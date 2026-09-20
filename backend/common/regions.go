package common

// countryRegions groups countries by continent for cross-country features - the international
// cup (issue #64, see engine.StartInternationalCupsForNewSeason) groups countries this way to
// decide who plays whom. Mirrors the frontend's identical COUNTRY_REGIONS (see
// frontend/src/components/avatar/traits.ts, added for issue #71's face-trait skin tone
// weighting) - kept in sync by hand since one is Go and the other TypeScript. Every code in
// KnownCountries should have an entry here.
var countryRegions = map[string]string{
	"es": "europe",
	"en": "europe",
	"fr": "europe",
	"de": "europe",
	"it": "europe",
	"pt": "europe",
	"br": "americas",
	"jp": "asia",
	"ng": "africa",
}

// RegionForCountry returns country's region, defaulting to "europe" for an unrecognized code -
// mirrors the frontend's regionForCountry.
func RegionForCountry(country string) string {
	if region, ok := countryRegions[country]; ok {
		return region
	}
	return "europe"
}
