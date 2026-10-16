package common

import (
	"encoding/json"
	"sort"
	"time"
)

// ScorerStanding is one row of a league's top-scorers leaderboard - see GetTopScorers.
type ScorerStanding struct {
	PlayerId  int64  `json:"playerId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	TeamId    int64  `json:"teamId"`
	TeamName  string `json:"teamName"`
	Goals     int    `json:"goals"`
}

// A non-NULL $4 excludes goals from a match whose match_date is after it - the public leaderboard
// passes LiveBroadcastCutoff() (see LiveBroadcastWindow), since match_events rows exist the
// instant a match is simulated, well before its broadcast finishes, so without it a scorer's
// tally would jump (and reveal that their team just scored) before that match's own row shows
// anything but "LIVE" (issue: match results review). A NULL $4 counts every simulated match,
// which season awards need (see GetTopScorersFinal).
const GetGoalEventScorersStmtSql = "SELECT me.team_players FROM match_events me " +
	"JOIN matches m ON m.match_id = me.match_id " +
	"WHERE m.league_id = $1 AND m.season = $2 AND me.event_type = $3 " +
	"AND ($4::timestamptz IS NULL OR m.match_date <= $4)"

const GetPlayerTeamForScorerStmtSql = "SELECT p.first_name, p.last_name, p.team_id, t.name " +
	"FROM players p JOIN teams t ON t.team_id = p.team_id WHERE p.player_id = $1"

// tallyGoalsByPlayer parses each match_events.team_players value (JSON-encoded by
// engine/scorer_simulation.go - always a single-element array holding the scoring player's id)
// and counts goals per player. Pure so it's testable without a database; a row that fails to
// parse or is empty is silently skipped rather than failing the whole leaderboard.
func tallyGoalsByPlayer(teamPlayersJsonRows []string) map[int64]int {
	goals := make(map[int64]int)
	for _, raw := range teamPlayersJsonRows {
		var playerIds []int64
		if err := json.Unmarshal([]byte(raw), &playerIds); err != nil || len(playerIds) == 0 {
			continue
		}
		goals[playerIds[0]]++
	}
	return goals
}

// rankScorerStandings sorts by goals descending, tie-broken by player id for a stable order,
// and caps the result at limit (0 or negative means "no cap"). Pure so it's testable without a
// database.
func rankScorerStandings(standings []ScorerStanding, limit int) []ScorerStanding {
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Goals != standings[j].Goals {
			return standings[i].Goals > standings[j].Goals
		}
		return standings[i].PlayerId < standings[j].PlayerId
	})
	if limit > 0 && len(standings) > limit {
		return standings[:limit]
	}
	return standings
}

// GetTopScorers returns a league's top `limit` goalscorers for the given season, leaving out
// matches still inside their live-broadcast window (see LiveBroadcastWindow) so the public
// leaderboard doesn't reveal a result before its broadcast finishes. Only players still findable
// on a team's current roster are included - a scorer who has since retired (or otherwise left the
// game entirely) has no row left to look their name/team up from, so this is a "current squads"
// leaderboard rather than a full historical career record.
func GetTopScorers(dbCtx *DatabaseContext, leagueId int64, season int, limit int) ([]ScorerStanding, error) {
	cutoff := LiveBroadcastCutoff()
	return getTopScorers(dbCtx, leagueId, season, limit, &cutoff)
}

// GetTopScorersFinal is GetTopScorers counting every simulated match of the season, including ones
// whose live broadcast is still running. For season awards (see engine.awardSeasonHonors): the
// season closes as soon as the last match is simulated, which is before that match's broadcast
// ends, so the live-broadcast cutoff would drop the final round's goals and could hand the Golden
// Boot to the wrong player - and GetPlayerOfTheSeason counts those same matches.
func GetTopScorersFinal(dbCtx *DatabaseContext, leagueId int64, season int, limit int) ([]ScorerStanding, error) {
	return getTopScorers(dbCtx, leagueId, season, limit, nil)
}

// getTopScorers is the shared body of GetTopScorers/GetTopScorersFinal; a nil cutoff counts every
// match (see GetGoalEventScorersStmtSql).
func getTopScorers(dbCtx *DatabaseContext, leagueId int64, season int, limit int, cutoff *time.Time) ([]ScorerStanding, error) {
	rows, err := dbCtx.Db.Query(GetGoalEventScorersStmtSql, leagueId, season, GOAL, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rawTeamPlayers := make([]string, 0)
	for rows.Next() {
		var teamPlayersJson string
		if err := rows.Scan(&teamPlayersJson); err != nil {
			return nil, err
		}
		rawTeamPlayers = append(rawTeamPlayers, teamPlayersJson)
	}

	goalsByPlayer := tallyGoalsByPlayer(rawTeamPlayers)
	standings := make([]ScorerStanding, 0, len(goalsByPlayer))
	for playerId, goals := range goalsByPlayer {
		var s ScorerStanding
		s.PlayerId = playerId
		s.Goals = goals
		if err := dbCtx.Db.QueryRow(GetPlayerTeamForScorerStmtSql, playerId).Scan(&s.FirstName, &s.LastName, &s.TeamId, &s.TeamName); err != nil {
			continue
		}
		standings = append(standings, s)
	}

	return rankScorerStandings(standings, limit), nil
}
