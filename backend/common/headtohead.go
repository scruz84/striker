package common

import "time"

// HeadToHeadMeeting is one past meeting between two teams, keeping the actual home/away sides
// that played (not reordered relative to whichever team asked for the summary).
type HeadToHeadMeeting struct {
	MatchId    int64     `json:"matchId"`
	Season     int       `json:"season"`
	MatchDate  time.Time `json:"matchDate"`
	HomeTeamId int64     `json:"homeTeamId"`
	AwayTeamId int64     `json:"awayTeamId"`
	HomeGoals  int       `json:"homeGoals"`
	AwayGoals  int       `json:"awayGoals"`
}

// HeadToHead summarizes every played match between two teams. Wins/goals are reported from
// TeamAId's perspective; TeamBWins/TeamBGoals are simply the complement, so the same struct
// reads correctly regardless of which of the two teams a caller thinks of as "theirs".
type HeadToHead struct {
	TeamAId      int64               `json:"teamAId"`
	TeamBId      int64               `json:"teamBId"`
	TotalMatches int                 `json:"totalMatches"`
	TeamAWins    int                 `json:"teamAWins"`
	TeamBWins    int                 `json:"teamBWins"`
	Draws        int                 `json:"draws"`
	TeamAGoals   int                 `json:"teamAGoals"`
	TeamBGoals   int                 `json:"teamBGoals"`
	Meetings     []HeadToHeadMeeting `json:"meetings"`
}

// HeadToHeadMeetingsLimit caps how many past meetings are returned in full (most recent first) -
// the aggregate win/draw/loss/goal counts above still cover every played match regardless.
const HeadToHeadMeetingsLimit = 10

// match_date <= $3 excludes a match still inside its live-broadcast window (see
// LiveBroadcastWindow) - otherwise this would reveal a result on the two teams' past-meetings
// summary while that same match still shows as "LIVE" everywhere else (issue: match results
// review).
const GetHeadToHeadMatchesStmtSql = "SELECT match_id, season, match_date, home_team_id, away_team_id, home_goals, away_goals " +
	"FROM matches WHERE played = true AND match_date <= $3 AND ((home_team_id = $1 AND away_team_id = $2) OR (home_team_id = $2 AND away_team_id = $1)) " +
	"ORDER BY match_date DESC"

// GetHeadToHead fetches every played match between two teams (excluding one still inside its
// live-broadcast window - see LiveBroadcastWindow) and summarizes it.
func GetHeadToHead(dbCtx *DatabaseContext, teamAId int64, teamBId int64) (*HeadToHead, error) {
	rows, err := dbCtx.Db.Query(GetHeadToHeadMatchesStmtSql, teamAId, teamBId, LiveBroadcastCutoff())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meetings := make([]HeadToHeadMeeting, 0)
	for rows.Next() {
		var m HeadToHeadMeeting
		if err := rows.Scan(&m.MatchId, &m.Season, &m.MatchDate, &m.HomeTeamId, &m.AwayTeamId, &m.HomeGoals, &m.AwayGoals); err != nil {
			return nil, err
		}
		meetings = append(meetings, m)
	}
	return summarizeHeadToHead(teamAId, teamBId, meetings), nil
}

// summarizeHeadToHead tallies wins/draws/goals from teamAId's perspective and keeps the most
// recent HeadToHeadMeetingsLimit meetings (meetings is expected most-recent-first, matching
// GetHeadToHeadMatchesStmtSql's ORDER BY). Pure so it's testable without a database.
func summarizeHeadToHead(teamAId int64, teamBId int64, meetings []HeadToHeadMeeting) *HeadToHead {
	result := &HeadToHead{TeamAId: teamAId, TeamBId: teamBId, Meetings: make([]HeadToHeadMeeting, 0)}
	for i, m := range meetings {
		teamAGoals, teamBGoals := m.HomeGoals, m.AwayGoals
		if m.HomeTeamId == teamBId {
			teamAGoals, teamBGoals = m.AwayGoals, m.HomeGoals
		}
		result.TotalMatches++
		result.TeamAGoals += teamAGoals
		result.TeamBGoals += teamBGoals
		switch {
		case teamAGoals > teamBGoals:
			result.TeamAWins++
		case teamBGoals > teamAGoals:
			result.TeamBWins++
		default:
			result.Draws++
		}
		if i < HeadToHeadMeetingsLimit {
			result.Meetings = append(result.Meetings, m)
		}
	}
	return result
}
