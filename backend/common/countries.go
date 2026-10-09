package common

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CountryInfo is one entry in KnownCountries - a country the game engine can generate a world
// for (it has a names_<code>/lastnames_<code> table pair and team-naming data, see
// GetRandomName and teamnames.go), whether or not it's currently active (present in the
// `countries` table).
type CountryInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// KnownCountries lists every country the game engine knows how to generate a world for. Not all
// of these are necessarily active - es/en/fr/de are seeded by a fresh --initialize (see
// SeededCountries), while the rest stay inactive until an admin activates one via
// POST /v1/admin/countries (see handlers.AdminAddCountryHandler) - adding a new entry here still
// requires its own names_<code>/lastnames_<code> table pair (see resources/sql/init.sql), an
// entry in nameTableSuffixByCountry below, and teamnames.go/stadiumnames.go entries first.
//
// br/jp/ng (issue #63) are the game's first non-European countries - added specifically so a
// region grouping (see frontend regionForCountry, and the backend counterpart common.RegionForCountry
// for the international competition) has more than one continent's worth of countries to actually
// group. The rest below round each of the four regions (see common.RegionForCountry) out to at
// least ten countries.
var KnownCountries = []CountryInfo{
	// Europe
	{Code: "es", Name: "Spain"},
	{Code: "en", Name: "England"},
	{Code: "fr", Name: "France"},
	{Code: "de", Name: "Germany"},
	{Code: "it", Name: "Italy"},
	{Code: "pt", Name: "Portugal"},
	{Code: "nl", Name: "Netherlands"},
	{Code: "se", Name: "Sweden"},
	{Code: "pl", Name: "Poland"},
	{Code: "gr", Name: "Greece"},
	{Code: "be", Name: "Belgium"},
	{Code: "tr", Name: "Turkey"},
	{Code: "at", Name: "Austria"},
	// Americas
	{Code: "br", Name: "Brazil"},
	{Code: "ar", Name: "Argentina"},
	{Code: "mx", Name: "Mexico"},
	{Code: "co", Name: "Colombia"},
	{Code: "cl", Name: "Chile"},
	{Code: "pe", Name: "Peru"},
	{Code: "uy", Name: "Uruguay"},
	{Code: "us", Name: "United States"},
	{Code: "ca", Name: "Canada"},
	{Code: "ec", Name: "Ecuador"},
	{Code: "ve", Name: "Venezuela"},
	{Code: "py", Name: "Paraguay"},
	{Code: "bo", Name: "Bolivia"},
	// Asia
	{Code: "jp", Name: "Japan"},
	{Code: "kr", Name: "South Korea"},
	{Code: "cn", Name: "China"},
	{Code: "in", Name: "India"},
	{Code: "th", Name: "Thailand"},
	{Code: "vn", Name: "Vietnam"},
	{Code: "id", Name: "Indonesia"},
	{Code: "ph", Name: "Philippines"},
	{Code: "sa", Name: "Saudi Arabia"},
	{Code: "ir", Name: "Iran"},
	{Code: "qa", Name: "Qatar"},
	{Code: "ae", Name: "United Arab Emirates"},
	{Code: "uz", Name: "Uzbekistan"},
	// Africa
	{Code: "ng", Name: "Nigeria"},
	{Code: "eg", Name: "Egypt"},
	{Code: "gh", Name: "Ghana"},
	{Code: "ci", Name: "Ivory Coast"},
	{Code: "cm", Name: "Cameroon"},
	{Code: "sn", Name: "Senegal"},
	{Code: "za", Name: "South Africa"},
	{Code: "ma", Name: "Morocco"},
	{Code: "dz", Name: "Algeria"},
	{Code: "ke", Name: "Kenya"},
	{Code: "tn", Name: "Tunisia"},
	{Code: "tz", Name: "Tanzania"},
	{Code: "ml", Name: "Mali"},
}

// IsKnownCountry reports whether code has the name-pool/team-naming data needed to generate a
// world for it (see KnownCountries) - checked before an admin can activate it.
func IsKnownCountry(code string) bool {
	for _, c := range KnownCountries {
		if c.Code == code {
			return true
		}
	}
	return false
}

// DefaultSeededCountries mirrors the game's original hardcoded set - used until SEEDED_COUNTRIES
// says otherwise.
var DefaultSeededCountries = []string{"es", "en", "fr", "de"}

// SeededCountries lists every country a fresh --initialize creates a world for (see
// handlers.InitializeGame), and that a new Google sign-in can be randomly assigned to (see
// handlers.randomSeededCountry) - configurable via the comma-separated SEEDED_COUNTRIES
// environment variable, same pattern as CorsAllowedOrigins/IsConfiguredAdminEmail. Falls back to
// DefaultSeededCountries if unset, or if nothing configured survives the IsKnownCountry check
// (each needs its own names_<code>/lastnames_<code> table pair - see KnownCountries).
func SeededCountries() []string {
	configured := os.Getenv("SEEDED_COUNTRIES")
	if configured == "" {
		return DefaultSeededCountries
	}
	countries := make([]string, 0)
	for _, candidate := range strings.Split(configured, ",") {
		code := strings.ToLower(strings.TrimSpace(candidate))
		if code == "" {
			continue
		}
		if !IsKnownCountry(code) {
			log.Errorf("SEEDED_COUNTRIES: %q is not a known country, skipping", code)
			continue
		}
		countries = append(countries, code)
	}
	if len(countries) == 0 {
		log.Error("SEEDED_COUNTRIES had no usable country codes, using defaults")
		return DefaultSeededCountries
	}
	return countries
}

// GetActiveCountries returns every country currently in the `countries` table - the ones a
// world has actually been generated for, as opposed to KnownCountries' full list of what's
// possible.
func GetActiveCountries(dbCtx *DatabaseContext) ([]string, error) {
	rows, err := dbCtx.Db.Query("SELECT name FROM countries ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countries := make([]string, 0)
	for rows.Next() {
		var country string
		if err := rows.Scan(&country); err != nil {
			return nil, err
		}
		countries = append(countries, country)
	}
	return countries, rows.Err()
}

// InsertCountry activates one country - the caller (handlers.AdminAddCountryHandler) is
// responsible for checking IsKnownCountry and that it isn't already active first.
func InsertCountry(dbCtx *DatabaseContext, code string) error {
	_, err := dbCtx.Db.Exec("INSERT INTO countries (name) VALUES ($1)", code)
	return err
}
