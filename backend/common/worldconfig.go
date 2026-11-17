package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
)

// TierConfig describes one league tier's shape: how many parallel groups it splits into (e.g.
// tier 2 might be 4 separate "II.1".."II.4" groups feeding the single tier-1 group - see
// engine.ComputePromotionRelegation) and how many teams fill each group's league. See
// GetWorldConfig/SetWorldConfig and handlers.createCountryLeagues.
type TierConfig struct {
	Tier           int    `json:"tier"`
	Name           string `json:"name"`
	Groups         int    `json:"groups"`
	TeamsPerLeague int    `json:"teamsPerLeague"`
}

// DefaultWorldTiers is the league pyramid used until an admin saves something different via
// SetWorldConfig: five tiers of 1/2/4/8/16 groups, each with TeamsPerLeague teams, so every tier
// has PyramidGroupRatio times the groups of the one above (see openspec/specs/league-pyramid).
var DefaultWorldTiers = []TierConfig{
	{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: TeamsPerLeague},
	{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: TeamsPerLeague},
	{Tier: 3, Name: "III", Groups: 4, TeamsPerLeague: TeamsPerLeague},
	{Tier: 4, Name: "IV", Groups: 8, TeamsPerLeague: TeamsPerLeague},
	{Tier: 5, Name: "V", Groups: 16, TeamsPerLeague: TeamsPerLeague},
}

// Bounds for ValidateWorldConfig. TeamsPerLeague must be even because fixture generation
// (engine.randomWeekMatches2) uses a standard round-robin circle method with no bye slot for an
// odd team out. The group/tier caps and the aggregate team-count cap below keep a single
// recreate/add-country request (which creates every team, player and lineup synchronously) from
// taking an impractically long time.
const (
	MinWorldTiers           = 1
	MaxWorldTiers           = 6
	MinGroupsPerTier        = 1
	MaxGroupsPerTier        = 32
	MinTeamsPerLeague       = MinPyramidTeamsPerLeague
	MaxTeamsPerLeague       = 24
	MaxTeamsPerCountryTotal = 1500
)

const getWorldConfigStmtSql = "SELECT tiers FROM world_config WHERE id = 1"
const upsertWorldConfigStmtSql = "INSERT INTO world_config (id, tiers) VALUES (1, $1) " +
	"ON CONFLICT (id) DO UPDATE SET tiers = EXCLUDED.tiers"

// GetWorldConfig returns the tier structure used the next time leagues are (re)created for a
// country (see handlers.InitializeGame, handlers.RecreateLeaguesKeepingUsers,
// handlers.AdminAddCountryHandler) - falls back to DefaultWorldTiers if nothing has been saved
// yet, or the saved value is somehow unusable.
func GetWorldConfig(dbCtx *DatabaseContext) []TierConfig {
	var raw []byte
	if err := dbCtx.Db.QueryRow(getWorldConfigStmtSql).Scan(&raw); err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("error reading world config, using defaults: %s", err)
		}
		return DefaultWorldTiers
	}
	var tiers []TierConfig
	if err := json.Unmarshal(raw, &tiers); err != nil || len(tiers) == 0 {
		log.Errorf("error parsing saved world config, using defaults: %v", err)
		return DefaultWorldTiers
	}
	return tiers
}

// SetWorldConfig persists tiers for the next time leagues are (re)created - it does not itself
// touch any existing league/team. Callers must run ValidateWorldConfig first.
func SetWorldConfig(dbCtx *DatabaseContext, tiers []TierConfig) error {
	raw, err := json.Marshal(tiers)
	if err != nil {
		return err
	}
	_, err = dbCtx.Db.Exec(upsertWorldConfigStmtSql, raw)
	return err
}

// ValidateWorldConfig checks tier/group/team-count bounds and the pyramid rule (see
// openspec/specs/league-pyramid "Pyramid structure rule") before SetWorldConfig persists them:
// tiers numbered 1..T with no gaps, tier 1 a single group, every other tier exactly
// PyramidGroupRatio times the groups of the tier above, and the same even TeamsPerLeague (at least
// MinPyramidTeamsPerLeague, room for every zone) in every tier. Pure so it's testable without a
// database.
func ValidateWorldConfig(tiers []TierConfig) error {
	if len(tiers) < MinWorldTiers || len(tiers) > MaxWorldTiers {
		return fmt.Errorf("must have between %d and %d tiers", MinWorldTiers, MaxWorldTiers)
	}
	seenTiers := make(map[int]bool, len(tiers))
	totalTeams := 0
	for _, t := range tiers {
		if t.Tier < 1 || seenTiers[t.Tier] {
			return fmt.Errorf("tier numbers must be positive and unique")
		}
		seenTiers[t.Tier] = true
		if t.Name == "" {
			return fmt.Errorf("tier %d needs a name", t.Tier)
		}
		if t.Groups < MinGroupsPerTier || t.Groups > MaxGroupsPerTier {
			return fmt.Errorf("tier %d groups must be between %d and %d", t.Tier, MinGroupsPerTier, MaxGroupsPerTier)
		}
		if t.TeamsPerLeague < MinTeamsPerLeague || t.TeamsPerLeague > MaxTeamsPerLeague || t.TeamsPerLeague%2 != 0 {
			return fmt.Errorf("tier %d teams per league must be an even number between %d and %d (room for 2 promotion places, 4 relegation places and 2 safe teams)", t.Tier, MinTeamsPerLeague, MaxTeamsPerLeague)
		}
		totalTeams += t.Groups * t.TeamsPerLeague
	}
	byTier := make(map[int]TierConfig, len(tiers))
	for _, t := range tiers {
		byTier[t.Tier] = t
	}
	for tier := 1; tier <= len(tiers); tier++ {
		if _, ok := byTier[tier]; !ok {
			return fmt.Errorf("tiers must be numbered 1 to %d with no gaps", len(tiers))
		}
	}
	if byTier[1].Groups != 1 {
		return fmt.Errorf("tier 1 must have exactly 1 group")
	}
	for tier := 2; tier <= len(tiers); tier++ {
		want := byTier[tier-1].Groups * PyramidGroupRatio
		if byTier[tier].Groups != want {
			return fmt.Errorf("tier %d must have exactly %d groups (twice tier %d)", tier, want, tier-1)
		}
		if byTier[tier].TeamsPerLeague != byTier[1].TeamsPerLeague {
			return fmt.Errorf("every tier must have the same number of teams per group")
		}
	}
	if totalTeams > MaxTeamsPerCountryTotal {
		return fmt.Errorf("this structure creates %d teams per country, which is more than the %d limit", totalTeams, MaxTeamsPerCountryTotal)
	}
	return nil
}

// WorldConfigPlaces is how many teams one country holds under tiers.
func WorldConfigPlaces(tiers []TierConfig) int {
	places := 0
	for _, t := range tiers {
		places += t.Groups * t.TeamsPerLeague
	}
	return places
}

// CountTeamsPerActiveCountryStmtSql counts each active country's club teams (national teams have
// no league).
// CountTeamsPerActiveCountryStmtSql counts each active country's club teams in its leagues (a
// dissolved team has left its league, league_id = -1) and how many of them a user owns.
const CountTeamsPerActiveCountryStmtSql = "SELECT l.country, count(t.team_id), count(t.owner_id) FROM teams t " +
	"JOIN leagues l ON l.league_id = t.league_id JOIN countries c ON c.name = l.country " +
	"WHERE t.dissolved_season IS NULL AND NOT t.is_national_team " +
	"GROUP BY l.country ORDER BY l.country"

// CountryTeams is an active country's club teams and how many of them users own.
type CountryTeams struct {
	Teams     int
	UserTeams int
}

// CountTeamsPerActiveCountry returns each active country's club team counts.
func CountTeamsPerActiveCountry(dbCtx *DatabaseContext) (map[string]CountryTeams, error) {
	rows, err := dbCtx.Db.Query(CountTeamsPerActiveCountryStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]CountryTeams)
	for rows.Next() {
		var country string
		var c CountryTeams
		if err := rows.Scan(&country, &c.Teams, &c.UserTeams); err != nil {
			return nil, err
		}
		counts[country] = c
	}
	return counts, rows.Err()
}

// PlanStructureChange checks tiers against the countries it would apply to (see
// openspec/specs/league-pyramid "Structure fits existing teams"): it fails when a country has more
// user-owned teams than tiers has places, naming the country and both numbers, and otherwise
// returns the CPU teams each country would lose at its next rollover (countries losing none are
// left out). Pure.
func PlanStructureChange(tiers []TierConfig, teamsByCountry map[string]CountryTeams) (map[string]int, error) {
	places := WorldConfigPlaces(tiers)
	countries := make([]string, 0, len(teamsByCountry))
	for country := range teamsByCountry {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	dissolve := make(map[string]int)
	for _, country := range countries {
		c := teamsByCountry[country]
		if c.UserTeams > places {
			return nil, fmt.Errorf("country %s has %d user teams but this structure only has %d places", country, c.UserTeams, places)
		}
		if c.Teams > places {
			dissolve[country] = c.Teams - places
		}
	}
	return dissolve, nil
}

const getCountryStructureStmtSql = "SELECT structure FROM countries WHERE name = $1"
const setCountryStructureStmtSql = "UPDATE countries SET structure = $2 WHERE name = $1"
const getCountryStructuresStmtSql = "SELECT name, structure FROM countries WHERE structure IS NOT NULL ORDER BY name"

// ErrUnknownCountry is returned when a country structure is saved for a country that does not
// exist.
var ErrUnknownCountry = errors.New("unknown country")

func parseTiers(raw []byte) ([]TierConfig, error) {
	var tiers []TierConfig
	if err := json.Unmarshal(raw, &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

// GetCountryStructure returns country's own structure, or nil when it follows the global one.
func GetCountryStructure(dbCtx *DatabaseContext, country string) ([]TierConfig, error) {
	var raw []byte
	err := dbCtx.Db.QueryRow(getCountryStructureStmtSql, country).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && raw == nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseTiers(raw)
}

// GetCountryStructures returns every country that has its own structure.
func GetCountryStructures(dbCtx *DatabaseContext) (map[string][]TierConfig, error) {
	rows, err := dbCtx.Db.Query(getCountryStructuresStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	structures := make(map[string][]TierConfig)
	for rows.Next() {
		var country string
		var raw []byte
		if err := rows.Scan(&country, &raw); err != nil {
			return nil, err
		}
		tiers, err := parseTiers(raw)
		if err != nil {
			log.Errorf("ignoring %s's unreadable own structure: %s", country, err)
			continue
		}
		structures[country] = tiers
	}
	return structures, rows.Err()
}

// SetCountryStructure saves country's own structure; nil tiers removes it, putting the country
// back on the global structure. Validation is the caller's (see ValidateWorldConfig and
// PlanStructureChange).
func SetCountryStructure(dbCtx *DatabaseContext, country string, tiers []TierConfig) error {
	var raw any
	if tiers != nil {
		b, err := json.Marshal(tiers)
		if err != nil {
			return err
		}
		raw = b
	}
	res, err := dbCtx.Db.Exec(setCountryStructureStmtSql, country, raw)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUnknownCountry
	}
	return nil
}

// EffectiveStructure is the structure country is reshaped into at its next rollover: its own if it
// has one, otherwise the global one.
func EffectiveStructure(dbCtx *DatabaseContext, country string) []TierConfig {
	own, err := GetCountryStructure(dbCtx, country)
	if err != nil {
		log.Errorf("error reading %s's own structure, using the global one: %s", country, err)
	}
	if len(own) > 0 {
		return own
	}
	return GetWorldConfig(dbCtx)
}
