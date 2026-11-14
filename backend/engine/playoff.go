package engine

import (
	"strconv"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PlayoffMinimumLead is the least time between creating a country's playoff ties and their kickoff
// (see openspec/specs/league-pyramid "Playoff phase"); a kickoff closer than that moves a day.
const PlayoffMinimumLead = time.Hour

// playoffKickoff dates a playoff tie between upperTier and the tier below it: the day after the
// last league match's local date, at upperTier's league kickoff time on the country's own clock
// (kickoffMinute is the country's tier-1 time, see common.TierKickoffMinute), and a day later for
// as long as that is less than PlayoffMinimumLead after now. Per-league schedule overrides do not
// apply to the playoff. Pure.
func playoffKickoff(loc *time.Location, kickoffMinute int, upperTier int, lastLeagueMatch time.Time, now time.Time) time.Time {
	return common.LocalKickoffOnDayAfter(loc, lastLeagueMatch, common.TierKickoffMinute(kickoffMinute, upperTier), now.Add(PlayoffMinimumLead))
}

// countryHasPlayoff reports whether a country with this active pyramid plays a promotion playoff:
// more than one tier, and the ratio rule the pairings need (a legacy pyramid plays the old swap
// instead - see computeLegacyMoves).
func countryHasPlayoff(shape common.PyramidShape) bool {
	return shape.TierCount() > 1 && shape.FollowsRatioRule()
}

// rankedLeaguesByTier reads every league of shape's final standings, keyed by tier.
func rankedLeaguesByTier(dbCtx *common.DatabaseContext, shape common.PyramidShape) (map[int][]LeagueStandings, error) {
	byTier := make(map[int][]LeagueStandings)
	for _, league := range shape.Leagues {
		standings, err := common.GetRankedClassification(dbCtx, league.LeagueId)
		if err != nil {
			return nil, err
		}
		byTier[league.Tier] = append(byTier[league.Tier], LeagueStandings{LeagueId: league.LeagueId, Standings: standings})
	}
	return byTier, nil
}

// startPlayoff creates country's promotion playoff for season from its leagues' final standings
// (see ComputePlayoffPairings): one cup of kind playoff holding every tier pair's ties, each
// dated by playoffKickoff. It runs under the country's rollover lock (see
// advanceCountrySeasonIfComplete) and the insert is a no-op if the season already has a playoff
// (cups_playoff_once), so the ties and their notifications are created once even with several
// engine replicas. A pyramid with no pairings (groups too small for the playoff places) still
// gets its empty playoff, so the rollover does not wait for one forever.
func startPlayoff(dbCtx *common.DatabaseContext, country string, season int, shape common.PyramidShape) error {
	leaguesByTier, err := rankedLeaguesByTier(dbCtx, shape)
	if err != nil {
		return err
	}
	pairings := ComputePlayoffPairings(leaguesByTier)

	lastMatch, ok, err := common.GetLastLeagueMatchDate(dbCtx, country, season)
	if err != nil {
		return err
	}
	now := time.Now()
	if !ok {
		lastMatch = now
	}
	loc := common.CountryLocation(country)
	kickoffMinute := common.GetCountryLeagueKickoff(dbCtx, country)
	ties := make([]common.PlayoffTieInput, len(pairings))
	for i, p := range pairings {
		ties[i] = common.PlayoffTieInput{
			HomeTeamId: p.HomeTeamId,
			AwayTeamId: p.AwayTeamId,
			MatchDate:  playoffKickoff(loc, kickoffMinute, p.UpperTier, lastMatch, now),
		}
	}

	cupId, tieIds, created, err := common.CreatePlayoff(dbCtx, country, season, ties)
	if err != nil || !created {
		return err
	}
	log.Infof("created %s's season %d playoff (cup %d) with %d ties", country, season, cupId, len(tieIds))
	for i, tieId := range tieIds {
		notifyPlayoffQualified(dbCtx, tieId, ties[i].HomeTeamId, ties[i].AwayTeamId)
		notifyPlayoffQualified(dbCtx, tieId, ties[i].AwayTeamId, ties[i].HomeTeamId)
	}
	return nil
}

// notifyPlayoffQualified tells teamId's owner who it plays in the playoff, linking to the tie.
func notifyPlayoffQualified(dbCtx *common.DatabaseContext, tieId, teamId, opponentId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error getting team %d for its playoff notification: %s", teamId, err)
		return
	}
	opponent, err := common.GetTeamById(opponentId, dbCtx)
	if err != nil {
		log.Errorf("error getting team %d for a playoff notification: %s", opponentId, err)
		return
	}
	msg := common.NotificationMessage{
		Text:   team.TeamName + " will play the promotion playoff against " + opponent.TeamName + ".",
		Key:    "notifications.messages.playoffQualified",
		Params: map[string]any{"team": team.TeamName, "opponent": opponent.TeamName},
	}
	link := "/cup/ties/" + strconv.FormatInt(tieId, 10)
	if err := common.NotifyTeamOwnerWithLink(dbCtx, teamId, common.NOTIFICATION_CUP, msg, link); err != nil {
		log.Errorf("error notifying team %d of its playoff tie %d: %s", teamId, tieId, err)
	}
}

// playoffResults reads a playoff's ties as results, each side's league taken from the final
// standings (teams do not change league before the rollover applies the outcomes).
func playoffResults(dbCtx *common.DatabaseContext, cupId int64, leagueOfTeam map[int64]int64, tierOfLeague map[int64]int) ([]PlayoffResult, error) {
	ties, err := common.GetCupTies(dbCtx, cupId)
	if err != nil {
		return nil, err
	}
	results := make([]PlayoffResult, 0, len(ties))
	for _, tie := range ties {
		if tie.HomeTeamId == nil || tie.AwayTeamId == nil || tie.WinnerTeamId == nil {
			continue
		}
		home, away := *tie.HomeTeamId, *tie.AwayTeamId
		homeLeague, okHome := leagueOfTeam[home]
		awayLeague, okAway := leagueOfTeam[away]
		if !okHome || !okAway {
			log.Errorf("playoff tie %d: a side is in none of the country's leagues, skipping its outcome", tie.TieId)
			continue
		}
		results = append(results, PlayoffResult{
			PlayoffPairing: PlayoffPairing{
				UpperTier:    tierOfLeague[homeLeague],
				HomeTeamId:   home,
				HomeLeagueId: homeLeague,
				AwayTeamId:   away,
				AwayLeagueId: awayLeague,
			},
			WinnerTeamId: *tie.WinnerTeamId,
		})
	}
	return results, nil
}
