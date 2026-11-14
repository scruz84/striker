package common

import (
	"database/sql"
	"encoding/json"
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
const CountTeamsPerActiveCountryStmtSql = "SELECT l.country, count(t.team_id) FROM teams t " +
	"JOIN leagues l ON l.league_id = t.league_id JOIN countries c ON c.name = l.country " +
	"GROUP BY l.country ORDER BY l.country"

// CountTeamsPerActiveCountry returns each active country's club team count.
func CountTeamsPerActiveCountry(dbCtx *DatabaseContext) (map[string]int, error) {
	rows, err := dbCtx.Db.Query(CountTeamsPerActiveCountryStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var country string
		var n int
		if err := rows.Scan(&country, &n); err != nil {
			return nil, err
		}
		counts[country] = n
	}
	return counts, rows.Err()
}

// ValidateWorldConfigFitsTeams rejects tiers that hold fewer teams than an active country already
// has - a restructure never deletes a team (see openspec/specs/league-pyramid "Structure fits
// existing teams"). Pure.
func ValidateWorldConfigFitsTeams(tiers []TierConfig, teamsByCountry map[string]int) error {
	places := WorldConfigPlaces(tiers)
	countries := make([]string, 0, len(teamsByCountry))
	for country := range teamsByCountry {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	for _, country := range countries {
		if teamsByCountry[country] > places {
			return fmt.Errorf("country %s has %d teams but this structure only has %d places", country, teamsByCountry[country], places)
		}
	}
	return nil
}
