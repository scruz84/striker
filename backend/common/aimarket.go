package common

import (
	"fmt"
	"strings"
)

// skillColumns are the players-table columns of the seven skills, in SkillWeights order
// ("lateral" is quoted: LATERAL is a reserved word in Postgres).
var skillColumns = [7]string{"p.goal_keeper", "p.defense", `p."lateral"`, "p.pass", "p.midfield", "p.dribbling", "p.shot"}

// slotValueSQL is SkillWeights.Value as a SQL expression over the players table: the weighted sum
// plus 50, integer-divided by 100, exactly as Go rounds it.
func slotValueSQL(weights SkillWeights) string {
	terms := make([]string, 0, len(weights))
	for i, w := range weights {
		if w != 0 {
			terms = append(terms, fmt.Sprintf("%s * %d", skillColumns[i], w))
		}
	}
	return "((" + strings.Join(terms, " + ") + " + 50) / 100)"
}

// positionSkillSQL is PositionValue for each core position as a SQL expression over the players
// table, generated from SlotWeights - how a market-wide search ranks players for a position
// without loading every player into memory. The match is only a pre-selection: the candidates it
// returns are hydrated and judged properly in Go afterwards.
var positionSkillSQL = func() map[int]string {
	out := make(map[int]string, len(CorePositions))
	for _, position := range CorePositions {
		out[position] = slotValueSQL(SlotWeights[position])
	}
	return out
}()

// marketCandidateFilter keeps a market-wide search to players another team could actually be
// asked for: on a real club's senior squad (not an academy prospect, not a national team's), not
// out on loan, not already on the transfer list (those are bought at their asking price, a
// different path), and not on the searching team itself.
const marketCandidateFilter = "p.team_id <> $1 AND p.in_academy = false AND t.is_national_team = false " +
	"AND NOT EXISTS (SELECT 1 FROM player_loans l WHERE l.player_id = p.player_id) " +
	"AND NOT EXISTS (SELECT 1 FROM transfer_listings tl WHERE tl.player_id = p.player_id) " +
	"AND NOT EXISTS (SELECT 1 FROM pre_contract_offers pc WHERE pc.player_id = p.player_id AND pc.signed_at IS NOT NULL)"

// GetMarketCandidatesForPosition returns the ids of up to limit players, from any other club in
// the world, whose rating for position is at least minValue and who are no older than maxAge,
// best first - a Director's search for an upgrade at a position, human managers' squads included.
func GetMarketCandidatesForPosition(dbCtx *DatabaseContext, excludeTeamId int64, position int, minValue int, maxAge int, limit int) ([]int64, error) {
	expr, ok := positionSkillSQL[position]
	if !ok {
		return nil, fmt.Errorf("no market search for position %d", position)
	}
	query := "SELECT p.player_id FROM players p JOIN teams t ON t.team_id = p.team_id WHERE " + marketCandidateFilter +
		" AND p.age <= $2 AND " + expr + " >= $3 ORDER BY " + expr + " DESC, p.player_id LIMIT $4"
	return queryIds(dbCtx, query, excludeTeamId, maxAge, minValue, limit)
}

// GetExpiringCandidatesForPosition is GetMarketCandidatesForPosition narrowed to players in the
// last season of their contract (and without a signed pre-contract) - the CPU's pre-contract
// targets (openspec/specs/cpu-transfer-market "Pre-contract targets").
func GetExpiringCandidatesForPosition(dbCtx *DatabaseContext, excludeTeamId int64, position int, minValue int, maxAge int, limit int) ([]int64, error) {
	expr, ok := positionSkillSQL[position]
	if !ok {
		return nil, fmt.Errorf("no market search for position %d", position)
	}
	query := "SELECT p.player_id FROM players p JOIN teams t ON t.team_id = p.team_id WHERE " + marketCandidateFilter +
		" AND p.contract_years_remaining = 1 AND p.age <= $2 AND " + expr + " >= $3 ORDER BY " + expr + " DESC, p.player_id LIMIT $4"
	return queryIds(dbCtx, query, excludeTeamId, maxAge, minValue, limit)
}

// GetYoungScoutingPool returns the ids of up to limit randomly chosen players aged maxAge or
// younger from other clubs - the pool a Director scouts for prospects. It deliberately does not
// rank by hidden potential: which of them is worth a project is for the scorer to decide, through
// its noisy estimate of their ceiling.
func GetYoungScoutingPool(dbCtx *DatabaseContext, excludeTeamId int64, maxAge int, limit int) ([]int64, error) {
	query := "SELECT p.player_id FROM players p JOIN teams t ON t.team_id = p.team_id WHERE " + marketCandidateFilter +
		" AND p.age <= $2 ORDER BY random() LIMIT $3"
	return queryIds(dbCtx, query, excludeTeamId, maxAge, limit)
}

func queryIds(dbCtx *DatabaseContext, query string, args ...any) ([]int64, error) {
	rows, err := dbCtx.Db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
