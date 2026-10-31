package common

import (
	"database/sql"
	"time"
)

// National-team matches live in three places: weekly friendlies and World Cup group matches in
// friendly_matches (a group match is linked through world_cup_group_matches) and World Cup
// knockout ties in cup_ties. NationalMatch flattens all three into the one row the
// national-teams page and a national team's own page render.
const (
	NationalMatchKindFriendly         = "friendly"
	NationalMatchKindWorldCupGroup    = "worldCupGroup"
	NationalMatchKindWorldCupKnockout = "worldCupKnockout"

	NationalMatchScheduled = "scheduled"
	NationalMatchLive      = "live"
	NationalMatchFinished  = "finished"
)

// NationalTeamResultsPageSize is how many results GetNationalTeamResults returns per page.
const NationalTeamResultsPageSize = 25

// nationalTeamRoundRecentWindow is how long after kickoff a played match stays in the current
// round feed (see GetNationalTeamRound) before it only appears in the results history.
const nationalTeamRoundRecentWindow = 24 * time.Hour

// NationalMatch is one national-team fixture, broadcast-safe: Status is "scheduled" (no score),
// "live" (Minute and the score as of that minute, never the final one) or "finished" (the final
// score) - see applyNationalMatchStatus. Id is a friendly match id, or a cup tie id for a
// knockout tie. GroupName/Round label a World Cup group match (Round = matchday); Round/
// TotalRounds label a knockout tie. MyPlayers is how many of the requesting owner's players are
// in a saved call-up for it (see GetMyPlayersInNationalMatches).
type NationalMatch struct {
	Kind         string    `json:"kind"`
	Id           int64     `json:"id"`
	HomeTeamId   int64     `json:"homeTeamId"`
	HomeTeamName string    `json:"homeTeamName"`
	HomeCountry  string    `json:"homeCountry"`
	AwayTeamId   int64     `json:"awayTeamId"`
	AwayTeamName string    `json:"awayTeamName"`
	AwayCountry  string    `json:"awayCountry"`
	Kickoff      time.Time `json:"kickoff"`
	Status       string    `json:"status"`
	Minute       *int      `json:"minute,omitempty"`
	HomeGoals    *int      `json:"homeGoals,omitempty"`
	AwayGoals    *int      `json:"awayGoals,omitempty"`
	GroupName    string    `json:"groupName,omitempty"`
	Round        int       `json:"round,omitempty"`
	TotalRounds  int       `json:"totalRounds,omitempty"`
	MyPlayers    int       `json:"myPlayers,omitempty"`
}

// liveScoreLookup is liveScoreAsOfNow's shape, injectable so applyNationalMatchStatus can be
// unit-tested without a database or Redis.
type liveScoreLookup func(kind string, id int64, kickoff time.Time) (minute, homeGoals, awayGoals int, complete bool, err error)

// applyNationalMatchStatus is the one place deciding what a national-team feed may reveal about a
// match: nothing but the fixture until it is simulated; while its kickoff is after cutoff (see
// LiveBroadcastCutoff - simulated up to 15 minutes early, broadcast for ~105) the minute and the
// score a live viewer sees, never the final one; afterwards the final score. A broadcast that
// already published its "complete" message is finished even inside the window.
func applyNationalMatchStatus(m *NationalMatch, played bool, finalHome, finalAway int, cutoff time.Time, lookup liveScoreLookup) error {
	m.Minute, m.HomeGoals, m.AwayGoals = nil, nil, nil
	if !played {
		m.Status = NationalMatchScheduled
		return nil
	}
	if m.Kickoff.After(cutoff) {
		minute, home, away, complete, err := lookup(liveMatchKindForNational(m.Kind), m.Id, m.Kickoff)
		if err != nil {
			return err
		}
		if !complete {
			m.Status = NationalMatchLive
			m.Minute, m.HomeGoals, m.AwayGoals = &minute, &home, &away
			return nil
		}
	}
	m.Status = NationalMatchFinished
	m.HomeGoals, m.AwayGoals = &finalHome, &finalAway
	return nil
}

// liveMatchKindForNational maps a NationalMatch kind onto the live-broadcast kind its channel and
// event table are keyed by: a group match is a friendly under the hood, a knockout tie a cup tie.
func liveMatchKindForNational(kind string) string {
	if kind == NationalMatchKindWorldCupKnockout {
		return MatchKindCup
	}
	return MatchKindFriendly
}

// nationalMatchesSql is every national-team match in one relation. Postgres pushes the outer
// filters of the statements below into each UNION ALL branch, so each branch still walks its own
// indexes. Columns: kind, id, home id/name/country, away id/name/country, match_date, played,
// home_goals, away_goals, group_name, round, total_rounds.
const nationalMatchesSql = "(" +
	"SELECT '" + NationalMatchKindFriendly + "' AS kind, fm.match_id AS id, fm.home_team_id, ht.name AS home_name, ht.country AS home_country, " +
	"fm.away_team_id, at.name AS away_name, at.country AS away_country, fm.match_date, fm.played, fm.home_goals, fm.away_goals, " +
	"'' AS group_name, 0 AS round, 0 AS total_rounds " +
	"FROM friendly_matches fm " +
	"JOIN teams ht ON ht.team_id = fm.home_team_id AND ht.is_national_team = true " +
	"JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE NOT EXISTS (SELECT 1 FROM world_cup_group_matches wcgm WHERE wcgm.match_id = fm.match_id) " +
	"UNION ALL " +
	"SELECT '" + NationalMatchKindWorldCupGroup + "', fm.match_id, fm.home_team_id, ht.name, ht.country, " +
	"fm.away_team_id, at.name, at.country, fm.match_date, fm.played, fm.home_goals, fm.away_goals, " +
	"g.group_name, wcgm.round, 0 " +
	"FROM world_cup_group_matches wcgm " +
	"JOIN world_cup_groups g ON g.group_id = wcgm.group_id " +
	"JOIN friendly_matches fm ON fm.match_id = wcgm.match_id " +
	"JOIN teams ht ON ht.team_id = fm.home_team_id " +
	"JOIN teams at ON at.team_id = fm.away_team_id " +
	"UNION ALL " +
	"SELECT '" + NationalMatchKindWorldCupKnockout + "', ct.tie_id, ct.home_team_id, ht.name, ht.country, " +
	"ct.away_team_id, at.name, at.country, ct.match_date, ct.played, ct.home_goals, ct.away_goals, " +
	"'', ct.round, " +
	"(SELECT CEIL(LOG(2, GREATEST(COUNT(r1.home_team_id) + COUNT(r1.away_team_id), 2)))::int FROM cup_ties r1 " +
	"WHERE r1.cup_id = ct.cup_id AND r1.round = (SELECT MIN(r0.round) FROM cup_ties r0 WHERE r0.cup_id = ct.cup_id)) " +
	"FROM world_cups wc " +
	"JOIN cup_ties ct ON ct.cup_id = wc.knockout_cup_id " +
	"JOIN teams ht ON ht.team_id = ct.home_team_id " +
	"JOIN teams at ON at.team_id = ct.away_team_id " +
	"WHERE ct.match_date IS NOT NULL" +
	") nm "

const nationalMatchColumnsSql = "SELECT nm.kind, nm.id, nm.home_team_id, nm.home_name, nm.home_country, nm.away_team_id, nm.away_name, nm.away_country, " +
	"nm.match_date, nm.played, nm.home_goals, nm.away_goals, nm.group_name, nm.round, nm.total_rounds FROM "

// getNationalTeamRoundStmtSql: every unplayed national-team match plus those that kicked off
// after $1 (now - nationalTeamRoundRecentWindow), soonest first. At most one unplayed weekly
// fixture per country exists (see ScheduleNationalTeamFriendlies), which keeps this bounded.
const getNationalTeamRoundStmtSql = nationalMatchColumnsSql + nationalMatchesSql +
	"WHERE nm.played = false OR nm.match_date > $1 ORDER BY nm.match_date, nm.kind, nm.id"

// getNationalTeamResultsStmtSql: played matches whose broadcast has finished (kickoff at or
// before $1, see LiveBroadcastCutoff), newest first, optionally one team's ($2, 0 = every team).
const getNationalTeamResultsStmtSql = nationalMatchColumnsSql + nationalMatchesSql +
	"WHERE nm.played = true AND nm.match_date <= $1 AND ($2::bigint = 0 OR nm.home_team_id = $2::bigint OR nm.away_team_id = $2::bigint) " +
	"ORDER BY nm.match_date DESC, nm.kind, nm.id DESC LIMIT $3 OFFSET $4"

// NationalTeamResults is one page of GetNationalTeamResults.
type NationalTeamResults struct {
	Matches []NationalMatch `json:"matches"`
	HasMore bool            `json:"hasMore"`
}

// GetNationalTeamRound lists the current international round - see getNationalTeamRoundStmtSql -
// each match broadcast-safe (see applyNationalMatchStatus).
func GetNationalTeamRound(dbCtx *DatabaseContext) ([]NationalMatch, error) {
	rows, err := dbCtx.Db.Query(getNationalTeamRoundStmtSql, time.Now().Add(-nationalTeamRoundRecentWindow))
	if err != nil {
		return nil, err
	}
	return scanNationalMatches(dbCtx, rows)
}

// GetNationalTeamResults is the paginated results history (page 0 is the newest
// NationalTeamResultsPageSize), optionally limited to teamId's matches (0 = every national team).
// A match still inside its broadcast window is never included - see LiveBroadcastCutoff.
func GetNationalTeamResults(dbCtx *DatabaseContext, page int, teamId int64) (NationalTeamResults, error) {
	rows, err := dbCtx.Db.Query(getNationalTeamResultsStmtSql, LiveBroadcastCutoff(), teamId,
		NationalTeamResultsPageSize+1, page*NationalTeamResultsPageSize)
	if err != nil {
		return NationalTeamResults{}, err
	}
	matches, err := scanNationalMatches(dbCtx, rows)
	if err != nil {
		return NationalTeamResults{}, err
	}
	hasMore := len(matches) > NationalTeamResultsPageSize
	if hasMore {
		matches = matches[:NationalTeamResultsPageSize]
	}
	return NationalTeamResults{Matches: matches, HasMore: hasMore}, nil
}

func scanNationalMatches(dbCtx *DatabaseContext, rows *sql.Rows) ([]NationalMatch, error) {
	type scanned struct {
		match                NationalMatch
		played               bool
		homeGoals, awayGoals int
	}
	all := make([]scanned, 0)
	for rows.Next() {
		var s scanned
		m := &s.match
		if err := rows.Scan(&m.Kind, &m.Id, &m.HomeTeamId, &m.HomeTeamName, &m.HomeCountry, &m.AwayTeamId, &m.AwayTeamName, &m.AwayCountry,
			&m.Kickoff, &s.played, &s.homeGoals, &s.awayGoals, &m.GroupName, &m.Round, &m.TotalRounds); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Live scores are looked up only after the rows are closed: the lookup may itself query.
	cutoff := LiveBroadcastCutoff()
	lookup := func(kind string, id int64, kickoff time.Time) (int, int, int, bool, error) {
		return liveScoreAsOfNow(dbCtx, kind, id, kickoff)
	}
	matches := make([]NationalMatch, 0, len(all))
	for _, s := range all {
		m := s.match
		if err := applyNationalMatchStatus(&m, s.played, s.homeGoals, s.awayGoals, cutoff, lookup); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// The starting XI a national side saves (see SetNationalTeamLineup/SetNationalTeamCupTieLineup),
// counted by the club each called-up player currently belongs to.
const getMyPlayersInNationalFriendliesStmtSql = "SELECT fl.match_id, COUNT(*) FROM friendly_lineup fl " +
	"JOIN teams nt ON nt.team_id = fl.team_id AND nt.is_national_team = true " +
	"JOIN friendly_lineup_positions flp ON flp.lineup_id = fl.lineup_id " +
	"JOIN players p ON p.player_id = flp.player_id " +
	"WHERE p.team_id = $1 AND fl.match_id = ANY($2) GROUP BY fl.match_id"

const getMyPlayersInNationalCupTiesStmtSql = "SELECT ctl.tie_id, COUNT(*) FROM cup_tie_lineup ctl " +
	"JOIN teams nt ON nt.team_id = ctl.team_id AND nt.is_national_team = true " +
	"JOIN cup_tie_lineup_positions ctlp ON ctlp.lineup_id = ctl.lineup_id " +
	"JOIN players p ON p.player_id = ctlp.player_id " +
	"WHERE p.team_id = $1 AND ctl.tie_id = ANY($2) GROUP BY ctl.tie_id"

// SetMyPlayersInNationalMatches fills each match's MyPlayers with how many of clubTeamId's
// players are in a saved national call-up for it. No lineup is saved until 24 hours before
// kickoff (see CallUpNationalTeamSquads), so a fixture further away always counts 0.
func SetMyPlayersInNationalMatches(dbCtx *DatabaseContext, clubTeamId int64, matches []NationalMatch) error {
	friendlyIds := make([]int64, 0)
	tieIds := make([]int64, 0)
	for _, m := range matches {
		if m.Kind == NationalMatchKindWorldCupKnockout {
			tieIds = append(tieIds, m.Id)
		} else {
			friendlyIds = append(friendlyIds, m.Id)
		}
	}
	friendlyCounts, err := countByMatch(dbCtx, getMyPlayersInNationalFriendliesStmtSql, clubTeamId, friendlyIds)
	if err != nil {
		return err
	}
	tieCounts, err := countByMatch(dbCtx, getMyPlayersInNationalCupTiesStmtSql, clubTeamId, tieIds)
	if err != nil {
		return err
	}
	for i := range matches {
		if matches[i].Kind == NationalMatchKindWorldCupKnockout {
			matches[i].MyPlayers = tieCounts[matches[i].Id]
		} else {
			matches[i].MyPlayers = friendlyCounts[matches[i].Id]
		}
	}
	return nil
}

func countByMatch(dbCtx *DatabaseContext, stmtSql string, clubTeamId int64, ids []int64) (map[int64]int, error) {
	counts := make(map[int64]int)
	if len(ids) == 0 {
		return counts, nil
	}
	rows, err := dbCtx.Db.Query(stmtSql, clubTeamId, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}
