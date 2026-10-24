package common

import "time"

// The statements below pick the fixtures that have used all their simulation attempts without
// being played, so engine.SettleDefaultResults can settle them 0-0. Only fixtures past kickoff
// count. A league match or friendly is claimed by the statement itself (without counting another
// attempt), skipping one a live simulation still holds: its claim is younger than $1 seconds. That
// makes several replicas settling at once safe - each fixture goes to exactly one of them.

const claimLeagueMatchesForDefaultStmtSql = "UPDATE matches SET in_simulation = true, in_simulation_since = now() " +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date <= now() " +
	"AND (in_simulation = false OR in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second')) " +
	"RETURNING match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals"

// ClaimLeagueMatchesForDefault claims and returns every league match that has used its
// MaxSimulationAttempts attempts and is still unplayed.
func ClaimLeagueMatchesForDefault(dbCtx *DatabaseContext) ([]WeekMatch, error) {
	rows, err := dbCtx.Db.Query(claimLeagueMatchesForDefaultStmtSql, StuckSimulationAge.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]WeekMatch, 0)
	for rows.Next() {
		var m WeekMatch
		if err := rows.Scan(&m.MatchId, &m.DivisionId, &m.Season, &m.WeekNumber, &m.MatchDate,
			&m.Home, &m.Away, &m.HomeGoals, &m.AwayGoals); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

const claimFriendliesForDefaultStmtSql = "UPDATE friendly_matches SET in_simulation = true, in_simulation_since = now() " +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date <= now() " +
	"AND (in_simulation = false OR in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second')) " +
	"RETURNING match_id, home_team_id, away_team_id, match_date"

// ClaimFriendliesForDefault is ClaimLeagueMatchesForDefault's friendly counterpart.
func ClaimFriendliesForDefault(dbCtx *DatabaseContext) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(claimFriendliesForDefaultStmtSql, StuckSimulationAge.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]FriendlyMatch, 0)
	for rows.Next() {
		var m FriendlyMatch
		if err := rows.Scan(&m.MatchId, &m.HomeTeamId, &m.AwayTeamId, &m.MatchDate); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

const getCupTiesForDefaultStmtSql = "SELECT tie_id, cup_id, round, home_team_id, away_team_id, match_date FROM cup_ties " +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date IS NOT NULL AND match_date <= now() " +
	"AND home_team_id IS NOT NULL AND away_team_id IS NOT NULL"

// GetCupTiesForDefault lists the cup ties that have used their attempts and are still unplayed. A
// cup tie has no claim of its own, so the caller settles each one under the tie's simulation lock
// (see engine.settleCupTieByDefault), which also keeps it clear of a simulation still running.
func GetCupTiesForDefault(dbCtx *DatabaseContext) ([]CupTie, error) {
	rows, err := dbCtx.Db.Query(getCupTiesForDefaultStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ties := make([]CupTie, 0)
	for rows.Next() {
		var tie CupTie
		var homeTeamId, awayTeamId int64
		var matchDate time.Time
		if err := rows.Scan(&tie.TieId, &tie.CupId, &tie.Round, &homeTeamId, &awayTeamId, &matchDate); err != nil {
			return nil, err
		}
		tie.HomeTeamId = &homeTeamId
		tie.AwayTeamId = &awayTeamId
		tie.MatchDate = &matchDate
		ties = append(ties, tie)
	}
	return ties, rows.Err()
}
