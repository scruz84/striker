package common

// countryRegions groups countries by continent for cross-country features - the international
// cup (issue #64, see engine.StartInternationalCupsForNewSeason) groups countries this way to
// decide who plays whom. Mirrors the frontend's identical COUNTRY_REGIONS (see
// frontend/src/components/avatar/traits.ts, added for issue #71's face-trait skin tone
// weighting) - kept in sync by hand since one is Go and the other TypeScript. Every code in
// KnownCountries should have an entry here.
var countryRegions = map[string]string{
	// Europe
	"es":  "europe",
	"en":  "europe",
	"fr":  "europe",
	"de":  "europe",
	"it":  "europe",
	"pt":  "europe",
	"nl":  "europe",
	"se":  "europe",
	"pl":  "europe",
	"gr":  "europe",
	"be":  "europe",
	"tr":  "europe",
	"at":  "europe",
	"sco": "europe",
	"wal": "europe",
	"ie":  "europe",
	"ru":  "europe",
	"rs":  "europe",
	"hr":  "europe",
	"ba":  "europe",
	"si":  "europe",
	"mk":  "europe",
	"me":  "europe",
	"xk":  "europe",
	// Americas
	"br": "americas",
	"ar": "americas",
	"mx": "americas",
	"co": "americas",
	"cl": "americas",
	"pe": "americas",
	"uy": "americas",
	"us": "americas",
	"ca": "americas",
	"ec": "americas",
	"ve": "americas",
	"py": "americas",
	"bo": "americas",
	// Asia
	"jp": "asia",
	"kr": "asia",
	"cn": "asia",
	"in": "asia",
	"th": "asia",
	"vn": "asia",
	"id": "asia",
	"ph": "asia",
	"sa": "asia",
	"ir": "asia",
	"qa": "asia",
	"ae": "asia",
	"uz": "asia",
	// Africa
	"ng": "africa",
	"eg": "africa",
	"gh": "africa",
	"ci": "africa",
	"cm": "africa",
	"sn": "africa",
	"za": "africa",
	"ma": "africa",
	"dz": "africa",
	"ke": "africa",
	"tn": "africa",
	"tz": "africa",
	"ml": "africa",
}

// RegionForCountry returns country's region, defaulting to "europe" for an unrecognized code -
// mirrors the frontend's regionForCountry.
func RegionForCountry(country string) string {
	if region, ok := countryRegions[country]; ok {
		return region
	}
	return "europe"
}
