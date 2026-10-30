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

// The statements below give a fixture back its attempts when it has used them all but its kickoff
// is still more than RedatedFixtureMargin away - it was re-dated after failing (e.g. by
// engine.RescheduleUnplayedMatches, which moves every unplayed match). Neither the selectors
// (attempts left) nor the default result (kickoff passed) would ever take such a fixture, so it
// sat in the admin overview as stuck or failing with nothing able to clear it. Released, it is
// simulated normally at its new kickoff and settled 0-0 only if it fails 3 more times after that.
// A live claim (younger than $1 seconds) is left alone. Each is one statement, so several replicas
// running it at once release each fixture once.

// RedatedFixtureMargin is how far ahead a fixture's kickoff must be for its used-up attempts to
// count as belonging to an earlier date: a fixture is claimed at most 15 minutes before kickoff
// (see GetSimulableMatchesStmtSql), so on its current date it cannot have used any attempt yet.
const RedatedFixtureMargin = 15 * time.Minute

const releaseRedatedSetSql = "in_simulation = false, simulation_attempts = 0, last_simulation_error = NULL, last_simulation_attempt_at = NULL "

const releaseRedatedLeagueMatchesStmtSql = "UPDATE matches SET " + releaseRedatedSetSql +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date > now() + ($2 * interval '1 second') " +
	"AND (in_simulation = false OR in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second'))"

const releaseRedatedFriendliesStmtSql = "UPDATE friendly_matches SET " + releaseRedatedSetSql +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date > now() + ($2 * interval '1 second') " +
	"AND (in_simulation = false OR in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second'))"

// A cup tie has no claim of its own (see GetCupTiesForDefault), so there is no live claim to skip.
const releaseRedatedCupTiesStmtSql = "UPDATE cup_ties SET " + releaseRedatedSetSql +
	"WHERE played = false AND " + simulationAttemptsUsedSql + " AND match_date > now() + ($1 * interval '1 second')"

// ReleaseRedatedFixtures gives back their attempts to every unplayed league match, friendly and cup
// tie that has used them all but kicks off more than RedatedFixtureMargin from now, and returns how
// many of each it released.
func ReleaseRedatedFixtures(dbCtx *DatabaseContext) (FixtureKindCounts, error) {
	var counts FixtureKindCounts
	margin, claimAge := RedatedFixtureMargin.Seconds(), StuckSimulationAge.Seconds()
	for _, release := range []struct {
		count *int
		stmt  string
		args  []any
	}{
		{&counts.LeagueMatches, releaseRedatedLeagueMatchesStmtSql, []any{claimAge, margin}},
		{&counts.Friendlies, releaseRedatedFriendliesStmtSql, []any{claimAge, margin}},
		{&counts.CupTies, releaseRedatedCupTiesStmtSql, []any{margin}},
	} {
		result, err := dbCtx.Db.Exec(release.stmt, release.args...)
		if err != nil {
			return counts, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return counts, err
		}
		*release.count = int(affected)
	}
	return counts, nil
}

// The statements below back the admin "finish stuck matches" action for fixtures that already
// failed but are not stuck: they were released with their error stored and are waiting out the
// retry backoff, so neither the stuck selectors (which want a live claim) nor the default result
// (which wants all attempts used) would touch them, and the action did nothing for them. Each
// leaves the fixture with exactly one attempt left and no backoff, so the next simulation pass
// picks it up at once and, if that attempt fails too, the attempt count reaches
// MaxSimulationAttempts and the default result settles it 0-0 - the admin does not have to press
// the action three more times. A failed attempt caused by an unreachable database is still given
// back (see RecordSimulationFailure), so an outage cannot default a fixture this way. Only fixtures
// past kickoff are touched: an earlier one is still simulated on its own schedule. A live claim is
// left alone. Each is one statement, so several replicas running it at once act on each fixture once.

const lastAttemptWhereSql = "played = false AND simulation_attempts > 0 AND " + simulationAttemptsLeftSql +
	" AND last_simulation_error IS NOT NULL AND match_date <= now()"

const lastAttemptSetSql = "simulation_attempts = $1, last_simulation_attempt_at = NULL "

const giveLeagueMatchesLastAttemptStmtSql = "UPDATE matches SET " + lastAttemptSetSql +
	"WHERE " + lastAttemptWhereSql + " AND in_simulation = false RETURNING match_id"

const giveFriendliesLastAttemptStmtSql = "UPDATE friendly_matches SET " + lastAttemptSetSql +
	"WHERE " + lastAttemptWhereSql + " AND in_simulation = false RETURNING match_id"

const giveCupTiesLastAttemptStmtSql = "UPDATE cup_ties SET " + lastAttemptSetSql +
	"WHERE " + lastAttemptWhereSql + " AND home_team_id IS NOT NULL AND away_team_id IS NOT NULL RETURNING tie_id"

// FixtureKindIds is one list of fixture ids per kind of fixture a simulation can fail on.
type FixtureKindIds struct {
	LeagueMatches []int64
	Friendlies    []int64
	CupTies       []int64
}

// GiveFailingFixturesLastAttempt leaves every unplayed league match, friendly and cup tie that has
// failed before and whose kickoff has passed with one simulation attempt left and no retry
// backoff, and returns their ids.
func GiveFailingFixturesLastAttempt(dbCtx *DatabaseContext) (FixtureKindIds, error) {
	var ids FixtureKindIds
	for _, give := range []struct {
		out  *[]int64
		stmt string
	}{
		{&ids.LeagueMatches, giveLeagueMatchesLastAttemptStmtSql},
		{&ids.Friendlies, giveFriendliesLastAttemptStmtSql},
		{&ids.CupTies, giveCupTiesLastAttemptStmtSql},
	} {
		rows, err := dbCtx.Db.Query(give.stmt, MaxSimulationAttempts-1)
		if err != nil {
			return ids, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return ids, err
			}
			*give.out = append(*give.out, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return ids, err
		}
	}
	return ids, nil
}
