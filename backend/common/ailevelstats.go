package common

// AI level buckets of the CPU level gauges and the admin overview (see
// openspec/specs/observability "CPU level gauges"): owned clubs are "human" whatever level they
// carried before, the rest by ai_level.
const (
	AiLevelBucketHuman     = "human"
	AiLevelBucketCaretaker = "caretaker"
	AiLevelBucketManager   = "manager"
	AiLevelBucketDirector  = "director"
)

// AiLevelBuckets lists the buckets in display order.
var AiLevelBuckets = []string{AiLevelBucketHuman, AiLevelBucketCaretaker, AiLevelBucketManager, AiLevelBucketDirector}

// AiLevelStats is one bucket's standing: how many clubs, their average budget, how many are
// insolvent (budget below -MaxDebt) and the league points per match they have taken this season.
type AiLevelStats struct {
	Level          string  `json:"level"`
	Teams          int64   `json:"teams"`
	AverageBudget  int64   `json:"averageBudget"`
	Insolvent      int64   `json:"insolvent"`
	PointsPerMatch float64 `json:"pointsPerMatch"`
	// Points and Matches are the season totals PointsPerMatch is computed from.
	Points  int64 `json:"-"`
	Matches int64 `json:"-"`
}

// GetAiLevelStatsStmtSql groups every club (national teams and clubs waiting for their reset
// left out) by ownership and AI level, with its current-season league row - classification holds
// one row per team for the season it is playing, so no season filter is needed.
const GetAiLevelStatsStmtSql = "SELECT t.ai_level, t.owner_id IS NOT NULL, count(*), COALESCE(SUM(t.budget), 0), " +
	"count(*) FILTER (WHERE t.budget < -$1::bigint), COALESCE(SUM(c.points), 0), COALESCE(SUM(c.matches), 0) " +
	"FROM teams t LEFT JOIN classification c ON c.team_id = t.team_id " +
	"WHERE t.is_national_team = false AND t.reset_pending_since IS NULL GROUP BY 1, 2"

// aiLevelBucket names the bucket of a club with aiLevel that is (or is not) owned; "" for an
// unowned club with no AI level (it is not managed at all).
func aiLevelBucket(aiLevel int, owned bool) string {
	switch {
	case owned:
		return AiLevelBucketHuman
	case aiLevel >= AiLevelDirector:
		return AiLevelBucketDirector
	case aiLevel == AiLevelManager:
		return AiLevelBucketManager
	case aiLevel == AiLevelCaretaker:
		return AiLevelBucketCaretaker
	}
	return ""
}

// aiLevelRow is one GROUP BY row of GetAiLevelStatsStmtSql.
type aiLevelRow struct {
	level                                     int
	owned                                     bool
	teams, budget, insolvent, points, matches int64
}

// GetAiLevelStats returns one AiLevelStats per bucket, in AiLevelBuckets order (a bucket with no
// club reads all zero).
func GetAiLevelStats(dbCtx *DatabaseContext) ([]AiLevelStats, error) {
	if dbCtx == nil || dbCtx.Db == nil {
		return nil, errNoDatabase
	}
	rows, err := dbCtx.Db.Query(GetAiLevelStatsStmtSql, MaxDebt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scanned := make([]aiLevelRow, 0, 8)
	for rows.Next() {
		var r aiLevelRow
		if err := rows.Scan(&r.level, &r.owned, &r.teams, &r.budget, &r.insolvent, &r.points, &r.matches); err != nil {
			return nil, err
		}
		scanned = append(scanned, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return foldAiLevelStats(scanned), nil
}

// foldAiLevelStats folds the grouped rows into the four buckets (an unowned club with no AI level
// is in none) and works out each bucket's averages. Pure.
func foldAiLevelStats(rows []aiLevelRow) []AiLevelStats {
	totals := map[string]*AiLevelStats{}
	budgets := map[string]int64{}
	for _, b := range AiLevelBuckets {
		totals[b] = &AiLevelStats{Level: b}
	}
	for _, r := range rows {
		bucket := aiLevelBucket(r.level, r.owned)
		if bucket == "" {
			continue
		}
		s := totals[bucket]
		s.Teams += r.teams
		s.Insolvent += r.insolvent
		s.Points += r.points
		s.Matches += r.matches
		budgets[bucket] += r.budget
	}
	stats := make([]AiLevelStats, 0, len(AiLevelBuckets))
	for _, b := range AiLevelBuckets {
		s := totals[b]
		if s.Teams > 0 {
			s.AverageBudget = budgets[b] / s.Teams
		}
		if s.Matches > 0 {
			s.PointsPerMatch = float64(s.Points) / float64(s.Matches)
		}
		stats = append(stats, *s)
	}
	return stats
}
