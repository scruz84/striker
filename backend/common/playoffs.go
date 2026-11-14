package common

import (
	"database/sql"
	"errors"
	"time"
)

// A country's end-of-season promotion playoff (see openspec/specs/league-pyramid) is one cup of
// kind CupKindPlayoff per (country, season), holding every tier pair's ties as round 1. The
// upper-tier team is always the home side, so the ties alone say which league each side plays in
// until the rollover applies their outcome (see engine.applyPromotionRelegation).

// PlayoffCupName is the playoff cup's stored name - only logs and admin views show it; the
// frontend labels a playoff tie by its kind.
const PlayoffCupName = "Playoff"

const isPlayoffCupStmtSql = "SELECT kind = 'playoff' FROM cups WHERE cup_id = $1"

// IsPlayoffCup reports whether cupId is a promotion playoff.
func IsPlayoffCup(dbCtx *DatabaseContext, cupId int64) (bool, error) {
	var playoff bool
	err := dbCtx.Db.QueryRow(isPlayoffCupStmtSql, cupId).Scan(&playoff)
	return playoff, err
}

// PlayoffTieInput is one tie to create: the upper-tier team at home against the lower-tier one.
type PlayoffTieInput struct {
	HomeTeamId int64
	AwayTeamId int64
	MatchDate  time.Time
}

// InsertPlayoffCupStmtSql creates a country's playoff for a season, or does nothing when one
// already exists (cups_playoff_once) - RETURNING then yields no row.
const InsertPlayoffCupStmtSql = "INSERT INTO cups (country, region, season, name, kind) VALUES ($1, NULL, $2, $3, 'playoff') " +
	"ON CONFLICT (country, season) WHERE kind = 'playoff' DO NOTHING RETURNING cup_id"

// CreatePlayoff creates country's playoff for season with its ties in one transaction. created is
// false, and nothing is written, when the season already has a playoff - another replica made it.
// The returned tie ids follow ties' order.
func CreatePlayoff(dbCtx *DatabaseContext, country string, season int, ties []PlayoffTieInput) (cupId int64, tieIds []int64, created bool, err error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, nil, false, err
	}
	defer tx.Rollback()
	if err := tx.QueryRow(InsertPlayoffCupStmtSql, country, season, PlayoffCupName).Scan(&cupId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil, false, nil
		}
		return 0, nil, false, err
	}
	tieIds = make([]int64, 0, len(ties))
	for i, tie := range ties {
		var tieId int64
		if err := tx.QueryRow(InsertCupTieStmtSql, cupId, 1, i+1, tie.HomeTeamId, tie.AwayTeamId, tie.MatchDate, false, nil).Scan(&tieId); err != nil {
			return 0, nil, false, err
		}
		tieIds = append(tieIds, tieId)
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, false, err
	}
	return cupId, tieIds, true, nil
}

// PlayoffState is where a country's playoff for one season stands.
type PlayoffState struct {
	Exists    bool
	CupId     int64
	Completed bool
	Ties      int
	// Unresolved counts ties with no winner yet; Broadcasting counts ties whose live broadcast
	// window is still open (or that have not kicked off yet).
	Unresolved   int
	Broadcasting int
}

// Done reports whether every tie has a winner and its broadcast is over.
func (s PlayoffState) Done() bool {
	return s.Exists && s.Unresolved == 0 && s.Broadcasting == 0
}

const GetCountryPlayoffStmtSql = "SELECT c.cup_id, c.completed, count(ct.tie_id), " +
	"count(ct.tie_id) FILTER (WHERE ct.winner_team_id IS NULL), " +
	"count(ct.tie_id) FILTER (WHERE ct.match_date > $3) " +
	"FROM cups c LEFT JOIN cup_ties ct ON ct.cup_id = c.cup_id " +
	"WHERE c.country = $1 AND c.season = $2 AND c.kind = 'playoff' GROUP BY c.cup_id, c.completed"

// GetCountryPlayoff reads country's playoff for season; Exists is false when there is none.
func GetCountryPlayoff(dbCtx *DatabaseContext, country string, season int) (PlayoffState, error) {
	var s PlayoffState
	err := dbCtx.Db.QueryRow(GetCountryPlayoffStmtSql, country, season, LiveBroadcastCutoff()).
		Scan(&s.CupId, &s.Completed, &s.Ties, &s.Unresolved, &s.Broadcasting)
	if errors.Is(err, sql.ErrNoRows) {
		return PlayoffState{}, nil
	}
	if err != nil {
		return PlayoffState{}, err
	}
	s.Exists = true
	return s, nil
}

const GetLastLeagueMatchDateStmtSql = "SELECT MAX(m.match_date) FROM matches m JOIN leagues l ON l.league_id = m.league_id " +
	"WHERE l.country = $1 AND m.season = $2"

// GetLastLeagueMatchDate is the kickoff of country's last league match of season - the playoff is
// played the day after it (see engine.playoffKickoff). ok is false when the season has no match.
func GetLastLeagueMatchDate(dbCtx *DatabaseContext, country string, season int) (time.Time, bool, error) {
	var last sql.NullTime
	if err := dbCtx.Db.QueryRow(GetLastLeagueMatchDateStmtSql, country, season).Scan(&last); err != nil {
		return time.Time{}, false, err
	}
	return last.Time, last.Valid, nil
}

const GetLeaguePlayoffTiesStmtSql = "SELECT ct.tie_id, ct.cup_id, ct.round, ct.slot_number, ct.home_team_id, th.name, " +
	"ct.away_team_id, ta.name, ct.match_date, ct.played, ct.home_goals, ct.away_goals, ct.went_to_penalties, ct.winner_team_id " +
	"FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id " +
	"JOIN leagues l ON l.league_id = $1 JOIN countries co ON co.name = l.country " +
	"LEFT JOIN teams th ON th.team_id = ct.home_team_id LEFT JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE c.kind = 'playoff' AND c.country = l.country AND c.season = co.current_season " +
	"AND (th.league_id = $1 OR ta.league_id = $1) ORDER BY ct.slot_number"

// GetLeaguePlayoffTies returns the current season's playoff ties with a side in leagueId (see
// openspec/specs/league-page "Playoff section"), empty before the ties exist and once the season
// has rolled over. A tie still inside its live broadcast window shows no result yet - played,
// score, penalties and winner are cleared so the league page never spoils it.
func GetLeaguePlayoffTies(dbCtx *DatabaseContext, leagueId int64) ([]CupTie, error) {
	rows, err := dbCtx.Db.Query(GetLeaguePlayoffTiesStmtSql, leagueId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cutoff := LiveBroadcastCutoff()
	ties := make([]CupTie, 0)
	for rows.Next() {
		tie, err := scanCupTie(rows)
		if err != nil {
			return nil, err
		}
		if tie.MatchDate == nil || tie.MatchDate.After(cutoff) {
			tie.Played = false
			tie.HomeGoals, tie.AwayGoals = 0, 0
			tie.WentToPenalties = false
			tie.WinnerTeamId = nil
		}
		ties = append(ties, *tie)
	}
	return ties, rows.Err()
}
