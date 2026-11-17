package common

import (
	"database/sql"
	"errors"
	"fmt"
)

// Dissolving a CPU club (openspec/specs/league-pyramid "Dissolved team"): a restructure with
// fewer places than teams dissolves the surplus CPU clubs at the country's rollover. A dissolved
// club keeps its row, past matches and history, but leaves its league (league_id = -1) and never
// plays, trades or is assigned again: every club listing skips dissolved_season IS NOT NULL.

// LockDissolvableTeamStmtSql locks a CPU club that is not dissolved yet: a user's club, a national
// team or an already dissolved club is never dissolved.
const LockDissolvableTeamStmtSql = "SELECT team_id FROM teams WHERE team_id = $1 AND owner_id IS NULL " +
	"AND NOT is_national_team AND dissolved_season IS NULL FOR UPDATE"

// ReleaseDissolvedSquadStmtSql frees the senior squad like ReleaseSeniorSquadStmtSql, but with no
// last club: a club that no longer exists earns no free-agent premium when they sign elsewhere.
const ReleaseDissolvedSquadStmtSql = "UPDATE players SET last_team_id = NULL, team_id = NULL, contract_years_remaining = 0, " +
	"shirt_number = NULL, free_agent_seasons = 0 WHERE team_id = $1 AND in_academy = false"

// Unplayed friendlies of the club go with their lineups (friendly_lineup rows of the club itself
// are already gone, see clearClubStateTx; these also remove the opponent's).
const (
	DeleteDissolvedFriendlyLineupPositionsStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id IN (SELECT l.lineup_id FROM friendly_lineup l " +
		"JOIN friendly_matches f ON f.match_id = l.match_id WHERE NOT f.played AND (f.home_team_id = $1 OR f.away_team_id = $1))"
	DeleteDissolvedFriendlyLineupsStmtSql = "DELETE FROM friendly_lineup WHERE match_id IN (SELECT match_id FROM friendly_matches " +
		"WHERE NOT played AND (home_team_id = $1 OR away_team_id = $1))"
	DeleteDissolvedFriendlySubstitutionsStmtSql = "DELETE FROM friendly_lineup_substitutions WHERE match_id IN (SELECT match_id FROM friendly_matches " +
		"WHERE NOT played AND (home_team_id = $1 OR away_team_id = $1))"
	DeleteDissolvedFriendlyOrdersStmtSql = "DELETE FROM friendly_lineup_orders WHERE match_id IN (SELECT match_id FROM friendly_matches " +
		"WHERE NOT played AND (home_team_id = $1 OR away_team_id = $1))"
	DeleteDissolvedFriendliesStmtSql = "DELETE FROM friendly_matches WHERE NOT played AND (home_team_id = $1 OR away_team_id = $1)"
)

// DeleteTeamClassificationStmtSql removes the club from its league's standings table.
const DeleteTeamClassificationStmtSql = "DELETE FROM classification WHERE team_id = $1"

// MarkTeamDissolvedStmtSql takes the club out of play for good.
const MarkTeamDissolvedStmtSql = "UPDATE teams SET dissolved_season = $2, league_id = -1, ai_level = 0, " +
	"reset_pending_since = NULL, reset_reason = NULL WHERE team_id = $1"

// DissolveTeam dissolves a CPU club in one transaction, as the restructure of season's rollover
// does with its surplus clubs: everything pending around it is cleared (see clearClubStateTx), its
// senior players become free agents, its academy prospects retire, its unplayed friendlies are
// cancelled and it leaves its league. It reports false, changing nothing, for a club that is
// user-owned, a national team or already dissolved, so a resumed rollover can call it again. The
// loans of players sent home are returned so the caller can notify their owners after the commit.
func DissolveTeam(dbCtx *DatabaseContext, teamId int64, season int) (bool, []PlayerLoan, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return false, nil, err
	}
	defer tx.Rollback()

	var locked int64
	err = tx.QueryRow(LockDissolvableTeamStmtSql, teamId).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	returned, err := clearClubStateTx(tx, teamId)
	if err != nil {
		return false, nil, err
	}
	for _, stmt := range []string{
		ReleaseDissolvedSquadStmtSql,
		DeleteDissolvedFriendlyLineupPositionsStmtSql, DeleteDissolvedFriendlyLineupsStmtSql,
		DeleteDissolvedFriendlySubstitutionsStmtSql, DeleteDissolvedFriendlyOrdersStmtSql, DeleteDissolvedFriendliesStmtSql,
		DeleteTeamClassificationStmtSql,
	} {
		if _, err := tx.Exec(stmt, teamId); err != nil {
			return false, nil, fmt.Errorf("dissolving team %d: %w (%.60s)", teamId, err, stmt)
		}
	}
	if _, err := tx.Exec(MarkTeamDissolvedStmtSql, teamId, season); err != nil {
		return false, nil, err
	}
	if err := tx.Commit(); err != nil {
		return false, nil, err
	}
	return true, returned, nil
}

// GetProtectedTeamsStmtSql picks, among $1, the clubs a restructure never dissolves: a user's
// club, or one still due to play a cup tie.
const GetProtectedTeamsStmtSql = "SELECT t.team_id FROM teams t WHERE t.team_id = ANY($1) AND (t.owner_id IS NOT NULL " +
	"OR EXISTS (SELECT 1 FROM cup_ties ct WHERE NOT ct.played AND (ct.home_team_id = t.team_id OR ct.away_team_id = t.team_id)))"

// GetProtectedTeams returns which of teamIds a restructure must keep (see GetProtectedTeamsStmtSql).
func GetProtectedTeams(dbCtx *DatabaseContext, teamIds []int64) (map[int64]bool, error) {
	protected := make(map[int64]bool)
	if len(teamIds) == 0 {
		return protected, nil
	}
	rows, err := dbCtx.Db.Query(GetProtectedTeamsStmtSql, teamIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		protected[id] = true
	}
	return protected, rows.Err()
}

const IsTeamDissolvedStmtSql = "SELECT dissolved_season IS NOT NULL FROM teams WHERE team_id = $1"

// IsTeamDissolved reports whether teamId was dissolved (false for an unknown team).
func IsTeamDissolved(dbCtx *DatabaseContext, teamId int64) (bool, error) {
	var dissolved bool
	err := dbCtx.Db.QueryRow(IsTeamDissolvedStmtSql, teamId).Scan(&dissolved)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return dissolved, err
}

const GetDissolvedTeamStmtSql = "SELECT country, name, dissolved_season FROM teams WHERE team_id = $1 AND dissolved_season IS NOT NULL"

// getDissolvedTeamDetails is a dissolved club's page (league-pyramid "Dissolved team"): its name,
// country and the last season it played, with no squad, division or fixtures. Its history and past
// matches come from their own endpoints as for any club. It returns nil for a club still playing.
func getDissolvedTeamDetails(dbCtx *DatabaseContext, teamId int64) (*TeamDetails, error) {
	team := Team{Id: teamId}
	err := dbCtx.Db.QueryRow(GetDissolvedTeamStmtSql, teamId).Scan(&team.Country, &team.TeamName, &team.DissolvedSeason)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &TeamDetails{Team: &team, Players: []*Player{}, LastMatches: []WeekMatch{}, NextMatches: []WeekMatch{}}, nil
}
