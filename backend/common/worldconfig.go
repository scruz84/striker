package common

import (
	"database/sql"
	"encoding/json"
	"fmt"

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

// DefaultWorldTiers mirrors the game's original hardcoded structure (three tiers - 1/4/16
// groups - each with TeamsPerLeague teams) - used until an admin saves something different via
// SetWorldConfig.
var DefaultWorldTiers = []TierConfig{
	{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: TeamsPerLeague},
	{Tier: 2, Name: "II", Groups: 4, TeamsPerLeague: TeamsPerLeague},
	{Tier: 3, Name: "III", Groups: 16, TeamsPerLeague: TeamsPerLeague},
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
	MinTeamsPerLeague       = 4
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

// ValidateWorldConfig checks tier/group/team-count bounds before SetWorldConfig persists them.
// Pure so it's testable without a database.
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
			return fmt.Errorf("tier %d teams per league must be an even number between %d and %d", t.Tier, MinTeamsPerLeague, MaxTeamsPerLeague)
		}
		totalTeams += t.Groups * t.TeamsPerLeague
	}
	if totalTeams > MaxTeamsPerCountryTotal {
		return fmt.Errorf("this structure creates %d teams per country, which is more than the %d limit", totalTeams, MaxTeamsPerCountryTotal)
	}
	return nil
}
