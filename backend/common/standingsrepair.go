package common

import "database/sql"

// standingsFromPlayedMatchesSql aggregates, for every classification row, what its team's played
// league matches of the row's own league and season add up to. A classification row is only ever
// changed by incrementing it per match (see UpdateTeamClassificationStmtSql) or by resetting it at
// rollover, so a row can be rebuilt exactly from those matches.
const standingsFromPlayedMatchesSql = "SELECT c.league_id, c.team_id, count(m.match_id) AS matches, " +
	"COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 3 WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS points, " +
	"COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 1 ELSE 0 END), 0) AS wins, " +
	"COALESCE(SUM(CASE WHEN g.gf < g.ga THEN 1 ELSE 0 END), 0) AS losses, " +
	"COALESCE(SUM(CASE WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS draws, " +
	"COALESCE(SUM(g.gf), 0) AS goals_scored, COALESCE(SUM(g.ga), 0) AS goals_received " +
	"FROM classification c " +
	"LEFT JOIN matches m ON m.league_id = c.league_id AND m.season = c.season AND m.played = true " +
	"AND (m.home_team_id = c.team_id OR m.away_team_id = c.team_id) " +
	"LEFT JOIN LATERAL (SELECT CASE WHEN m.home_team_id = c.team_id THEN m.home_goals ELSE m.away_goals END AS gf, " +
	"CASE WHEN m.home_team_id = c.team_id THEN m.away_goals ELSE m.home_goals END AS ga) g ON true "

const countStandingsMismatchesStmtSql = "SELECT count(*) FROM classification c JOIN (" +
	standingsFromPlayedMatchesSql + "GROUP BY c.league_id, c.team_id) agg " +
	"ON agg.league_id = c.league_id AND agg.team_id = c.team_id WHERE c.matches <> agg.matches"

// CountStandingsMismatches counts the classification rows whose matches count differs from the
// number of played league matches of that team in that league and season - standings drift left by
// a result that was booked twice. The admin overview shows it, and engine.RepairStandingsOnce
// rebuilds these rows.
func CountStandingsMismatches(dbCtx *DatabaseContext) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(countStandingsMismatchesStmtSql).Scan(&count)
	return count, err
}

const getLeaguesWithStandingsMismatchStmtSql = "SELECT DISTINCT c.league_id FROM classification c JOIN (" +
	standingsFromPlayedMatchesSql + "GROUP BY c.league_id, c.team_id) agg " +
	"ON agg.league_id = c.league_id AND agg.team_id = c.team_id WHERE c.matches <> agg.matches ORDER BY c.league_id"

// GetLeaguesWithStandingsMismatch lists the leagues that have at least one mismatched row.
func GetLeaguesWithStandingsMismatch(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(getLeaguesWithStandingsMismatchStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	leagueIds := make([]int64, 0)
	for rows.Next() {
		var leagueId int64
		if err := rows.Scan(&leagueId); err != nil {
			return nil, err
		}
		leagueIds = append(leagueIds, leagueId)
	}
	return leagueIds, rows.Err()
}

const rebuildLeagueStandingsStmtSql = "UPDATE classification c SET matches = agg.matches, points = agg.points, wins = agg.wins, " +
	"losses = agg.losses, draws = agg.draws, goals_scored = agg.goals_scored, goals_received = agg.goals_received FROM (" +
	"SELECT c.league_id, c.team_id, count(m.match_id) AS matches, " +
	"COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 3 WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS points, " +
	"COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 1 ELSE 0 END), 0) AS wins, " +
	"COALESCE(SUM(CASE WHEN g.gf < g.ga THEN 1 ELSE 0 END), 0) AS losses, " +
	"COALESCE(SUM(CASE WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS draws, " +
	"COALESCE(SUM(g.gf), 0) AS goals_scored, COALESCE(SUM(g.ga), 0) AS goals_received " +
	"FROM classification c " +
	"LEFT JOIN matches m ON m.league_id = c.league_id AND m.season = c.season AND m.played = true " +
	"AND (m.home_team_id = c.team_id OR m.away_team_id = c.team_id) " +
	"LEFT JOIN LATERAL (SELECT CASE WHEN m.home_team_id = c.team_id THEN m.home_goals ELSE m.away_goals END AS gf, " +
	"CASE WHEN m.home_team_id = c.team_id THEN m.away_goals ELSE m.home_goals END AS ga) g ON true " +
	"WHERE c.league_id = $1 GROUP BY c.league_id, c.team_id) agg " +
	"WHERE c.league_id = agg.league_id AND c.team_id = agg.team_id AND c.matches <> agg.matches"

// RebuildLeagueStandings rebuilds every mismatched classification row of leagueId from its played
// matches, and returns how many rows it changed. It holds the league's standings lock
// (LockNamespaceStandings) for the whole statement, so it never interleaves with a result save
// incrementing the same rows, and rows that already agree with their matches are left untouched.
func RebuildLeagueStandings(dbCtx *DatabaseContext, leagueId int64) (int64, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", NamespacedLockKey(LockNamespaceStandings, leagueId)); err != nil {
		return 0, err
	}
	var result sql.Result
	if result, err = tx.Exec(rebuildLeagueStandingsStmtSql, leagueId); err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, tx.Commit()
}
