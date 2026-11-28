package common

import "time"

// ScorerStanding is one row of a league's top scorers (see GetTopScorers). TeamId is the team of
// his latest league appearance that season; the name resolves for a retired player too.
type ScorerStanding struct {
	PlayerId  int64  `json:"playerId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	TeamId    int64  `json:"teamId"`
	TeamName  string `json:"teamName"`
	Goals     int    `json:"goals"`
	// BirthCountry lets the leaderboard show the scorer's flag; empty for a legacy player.
	BirthCountry string `json:"birthCountry,omitempty"`
}

// GetTopScorers returns a league's top `limit` goalscorers for the given season from its league
// matches' player stats (see GetLeagueLeaders), leaving out matches still inside their
// live-broadcast window (see LiveBroadcastWindow) so the public leaderboard doesn't reveal a
// result before its broadcast finishes. Ties go to the lower player id.
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

func getTopScorers(dbCtx *DatabaseContext, leagueId int64, season int, limit int, cutoff *time.Time) ([]ScorerStanding, error) {
	leaders, err := GetLeagueLeaders(dbCtx, leagueId, season, LeadersGoals, limit, cutoff)
	if err != nil {
		return nil, err
	}
	standings := make([]ScorerStanding, 0, len(leaders))
	for _, l := range leaders {
		standings = append(standings, ScorerStanding{
			PlayerId:     l.PlayerId,
			FirstName:    l.FirstName,
			LastName:     l.LastName,
			TeamId:       l.TeamId,
			TeamName:     l.TeamName,
			Goals:        l.Goals,
			BirthCountry: l.BirthCountry,
		})
	}
	return standings, nil
}
