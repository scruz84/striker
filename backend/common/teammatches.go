package common

import (
	"sort"
	"time"
)

// Match kind discriminators for TeamMatchSummary.Kind - together with MatchId this tells the
// frontend which detail/lineup route to use, since a league match_id, a cup_ties.tie_id, and a
// friendly_matches.match_id are independent id sequences that can collide numerically (see
// resources/sql/history/init38_match_lineup_overrides.sql for why that matters for lineup overrides too).
const (
	MatchKindLeague   = "league"
	MatchKindCup      = "cup"
	MatchKindFriendly = "friendly"
)

// TeamMatchSummary is one row in a team's combined fixture/results list (see GetTeamMatches) -
// enough to render a list entry and route a click, not the full per-kind detail (events,
// lineups, etc.), which stays behind the existing per-kind detail endpoints.
type TeamMatchSummary struct {
	Kind         string    `json:"kind"`
	MatchId      int64     `json:"matchId"`
	OpponentId   int64     `json:"opponentId"`
	OpponentName string    `json:"opponentName"`
	IsHome       bool      `json:"isHome"`
	MatchDate    time.Time `json:"matchDate"`
	Played       bool      `json:"played"`
	HomeGoals    int       `json:"homeGoals"`
	AwayGoals    int       `json:"awayGoals"`
	// Round is only meaningful for a cup tie (MatchKindCup) - the round number within that cup's
	// bracket. Zero, and omitted, for league/friendly entries.
	Round int `json:"round,omitempty"`
	// Competition is a cup tie's cup kind (CupKindCup or CupKindPlayoff); omitted for league and
	// friendly entries.
	Competition string `json:"competition,omitempty"`
	// Rivalry is the fixture's rivalry level (RIVALRY_*, see common/rivalries.go): the one it was
	// played at, or the pair's current one for a fixture still to come. Always 0 for a friendly.
	Rivalry int `json:"rivalry"`
}

var GetTeamLeagueMatchesStmtSql = "SELECT m.match_id, m.home_team_id, ht.name, m.away_team_id, at.name, m.match_date, m.played, m.home_goals, m.away_goals, " + fixtureRivalrySql("m") + " " +
	"FROM matches m JOIN teams ht ON ht.team_id = m.home_team_id JOIN teams at ON at.team_id = m.away_team_id " +
	"WHERE m.home_team_id = $1 OR m.away_team_id = $1 ORDER BY m.match_date"

func getTeamLeagueMatches(dbCtx *DatabaseContext, teamId int64) ([]TeamMatchSummary, error) {
	rows, err := dbCtx.Db.Query(GetTeamLeagueMatchesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]TeamMatchSummary, 0)
	for rows.Next() {
		var matchId, homeId, awayId int64
		var homeName, awayName string
		var s TeamMatchSummary
		if err := rows.Scan(&matchId, &homeId, &homeName, &awayId, &awayName, &s.MatchDate, &s.Played, &s.HomeGoals, &s.AwayGoals, &s.Rivalry); err != nil {
			return nil, err
		}
		s.Kind = MatchKindLeague
		s.MatchId = matchId
		if homeId == teamId {
			s.IsHome = true
			s.OpponentId, s.OpponentName = awayId, awayName
		} else {
			s.OpponentId, s.OpponentName = homeId, homeName
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// getTeamCupTies lists teamId's real cup ties (an INNER JOIN on both teams naturally excludes
// bye rounds, which have no away_team_id and thus nothing to click into or field a lineup for).
var GetTeamCupTiesStmtSql = "SELECT ct.tie_id, ct.round, ct.home_team_id, th.name, ct.away_team_id, ta.name, ct.match_date, ct.played, ct.home_goals, ct.away_goals, " + fixtureRivalrySql("ct") + ", c.kind " +
	"FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id JOIN teams th ON th.team_id = ct.home_team_id JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE ct.home_team_id = $1 OR ct.away_team_id = $1 ORDER BY ct.match_date"

func getTeamCupTies(dbCtx *DatabaseContext, teamId int64) ([]TeamMatchSummary, error) {
	rows, err := dbCtx.Db.Query(GetTeamCupTiesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]TeamMatchSummary, 0)
	for rows.Next() {
		var tieId, homeId, awayId int64
		var round int
		var homeName, awayName string
		var s TeamMatchSummary
		if err := rows.Scan(&tieId, &round, &homeId, &homeName, &awayId, &awayName, &s.MatchDate, &s.Played, &s.HomeGoals, &s.AwayGoals, &s.Rivalry, &s.Competition); err != nil {
			return nil, err
		}
		s.Kind = MatchKindCup
		s.MatchId = tieId
		s.Round = round
		if homeId == teamId {
			s.IsHome = true
			s.OpponentId, s.OpponentName = awayId, awayName
		} else {
			s.OpponentId, s.OpponentName = homeId, homeName
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func friendlyToSummary(f FriendlyMatch, teamId int64) TeamMatchSummary {
	s := TeamMatchSummary{
		Kind:      MatchKindFriendly,
		MatchId:   f.MatchId,
		MatchDate: f.MatchDate,
		Played:    f.Played,
		HomeGoals: f.HomeGoals,
		AwayGoals: f.AwayGoals,
	}
	if f.HomeTeamId == teamId {
		s.IsHome = true
		s.OpponentId, s.OpponentName = f.AwayTeamId, f.AwayTeamName
	} else {
		s.OpponentId, s.OpponentName = f.HomeTeamId, f.HomeTeamName
	}
	return s
}

// GetTeamMatches lists every league match, cup tie, and friendly teamId has ever played or has
// scheduled, oldest first - the "see all my matches, past and future, marked by competition"
// view (issue request), merging three otherwise-separate per-kind queries since matches,
// cup_ties, and friendly_matches are different enough in shape (nullable team ids/bye rounds for
// cup, no round/slot concept for league or friendly) that a single SQL UNION isn't a clean fit.
func GetTeamMatches(dbCtx *DatabaseContext, teamId int64) ([]TeamMatchSummary, error) {
	summaries := make([]TeamMatchSummary, 0)

	leagueMatches, err := getTeamLeagueMatches(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, leagueMatches...)

	cupTies, err := getTeamCupTies(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, cupTies...)

	friendlies, err := GetTeamFriendlyMatches(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	for _, f := range friendlies {
		summaries = append(summaries, friendlyToSummary(f, teamId))
	}

	sort.Slice(summaries, func(i, j int) bool { return summaries[i].MatchDate.Before(summaries[j].MatchDate) })
	return summaries, nil
}
