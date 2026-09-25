package common

import "time"

// PlayerMatchSummary is one row in a player's "recent matches" list on their profile page -
// enough to see the result, their own goals/cards, and (when available) their match rating
// without loading the full match report. Restricted to league matches: WeekMatchById (and the
// Ratings it computes via ComputeMatchRatings) only knows about the `matches` table, and cup/
// friendly fixtures don't carry a comparable per-player rating.
type PlayerMatchSummary struct {
	MatchId      int64     `json:"matchId"`
	MatchDate    time.Time `json:"matchDate"`
	OpponentId   int64     `json:"opponentId"`
	OpponentName string    `json:"opponentName"`
	IsHome       bool      `json:"isHome"`
	HomeGoals    int       `json:"homeGoals"`
	AwayGoals    int       `json:"awayGoals"`
	Goals        int       `json:"goals"`
	YellowCards  int       `json:"yellowCards"`
	RedCards     int       `json:"redCards"`
	// Rating is this player's Hattrick-style match rating (see ComputeMatchRatings), nil when the
	// match has no lineup data to derive one from (see WeekMatch.Ratings).
	Rating *float64 `json:"rating,omitempty"`
}

const getRecentPlayedLeagueMatchIdsStmtSql = "SELECT match_id FROM matches " +
	"WHERE (home_team_id = $1 OR away_team_id = $1) AND played = true ORDER BY match_date DESC LIMIT $2"

// GetPlayerRecentMatches lists teamId's most recent played league matches (most recent first,
// capped at limit), each annotated with this player's own goals/cards for that match and, when a
// lineup was recorded for it, their match rating - so a player's page can show "how has this
// player actually been playing" rather than only career totals. Returns an empty slice for a free
// agent (teamId == 0), who has no current team's fixtures to show.
func GetPlayerRecentMatches(dbCtx *DatabaseContext, playerId int64, teamId int64, limit int) ([]PlayerMatchSummary, error) {
	summaries := make([]PlayerMatchSummary, 0)
	if teamId == 0 {
		return summaries, nil
	}

	rows, err := dbCtx.Db.Query(getRecentPlayedLeagueMatchIdsStmtSql, teamId, limit)
	if err != nil {
		return nil, err
	}
	matchIds := make([]int64, 0)
	for rows.Next() {
		var matchId int64
		if err := rows.Scan(&matchId); err != nil {
			rows.Close()
			return nil, err
		}
		matchIds = append(matchIds, matchId)
	}
	rows.Close()

	for _, matchId := range matchIds {
		match, err := WeekMatchById(dbCtx, matchId)
		if err != nil {
			return nil, err
		}
		if match == nil {
			continue
		}

		summary := PlayerMatchSummary{
			MatchId:   match.MatchId,
			MatchDate: match.MatchDate,
			HomeGoals: match.HomeGoals,
			AwayGoals: match.AwayGoals,
		}
		if match.Home == teamId {
			summary.IsHome = true
			summary.OpponentId, summary.OpponentName = match.Away, match.AwayName
		} else {
			summary.OpponentId, summary.OpponentName = match.Home, match.HomeName
		}

		if match.Events != nil {
			for _, event := range *match.Events {
				if !containsPlayerId(event.TeamOwnerPlayers, playerId) {
					continue
				}
				switch event.EventType {
				case GOAL:
					summary.Goals++
				case YELLOW_CARD:
					summary.YellowCards++
				case RED_CARD:
					summary.RedCards++
				}
			}
		}

		if match.Ratings != nil {
			if rating, ok := (*match.Ratings)[playerId]; ok {
				summary.Rating = &rating
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func containsPlayerId(playerIds []int64, playerId int64) bool {
	for _, id := range playerIds {
		if id == playerId {
			return true
		}
	}
	return false
}
