package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// AdvanceSeasonIfComplete checks whether every match of the current season has been played
// and, if so, rolls the game forward: it ages every player a year (see ProgressPlayers),
// promotes/relegates teams based on the season's final standings, starts a new cup for each
// country from the post-promotion tier-1 teams, resets each league's classification for the
// new season, bumps the season counter, and regenerates fixtures. See
// ResetClassificationForSeason for why classification is reset in place rather than archived
// per season.
func AdvanceSeasonIfComplete(dbCtx *common.DatabaseContext) {
	currentSeason := common.GetCurrentSeason(dbCtx)

	total, unplayed, err := common.GetSeasonMatchCounts(dbCtx, currentSeason)
	if err != nil {
		log.Errorf("error checking season %d completion: %s", currentSeason, err)
		return
	}
	if total == 0 || unplayed > 0 {
		// season never started yet, or still has matches left to play
		return
	}

	newSeason := currentSeason + 1
	log.Infof("season %d complete (%d matches played), advancing to season %d", currentSeason, total, newSeason)

	ProgressPlayers(dbCtx)
	applyPromotionRelegation(dbCtx)
	StartCupsForNewSeason(dbCtx, newSeason)

	leagueIds, err := common.GetAllLeagueIds(dbCtx)
	if err != nil {
		log.Errorf("error listing leagues for season rollover: %s", err)
		return
	}

	for _, leagueId := range leagueIds {
		if err := common.ResetClassificationForSeason(dbCtx, leagueId, newSeason); err != nil {
			log.Errorf("error resetting classification for league %d season %d: %s", leagueId, newSeason, err)
		}
	}

	if err := common.SetCurrentSeason(dbCtx, newSeason); err != nil {
		log.Errorf("error advancing current season to %d: %s", newSeason, err)
		return
	}
	// GetCurrentSeason (this package) caches the season for 30 minutes; refresh it in place
	// so GenerateMatches below picks up the new season immediately instead of regenerating
	// fixtures under the season we just closed out.
	dbCtx.ObjectCache.SetValueLongDuration(CurrentSeasonKey, strconv.Itoa(newSeason))

	GenerateMatches(dbCtx)
}

// applyPromotionRelegation ranks every league's final standings, computes each country's
// promotion/relegation moves independently (a team never crosses countries), and applies
// them by repointing the moved teams' league_id - both on teams and on their existing
// classification row, so the per-league reset that follows resets the right rows.
func applyPromotionRelegation(dbCtx *common.DatabaseContext) {
	leagues, err := common.GetLeaguesWithTier(dbCtx)
	if err != nil {
		log.Errorf("error listing leagues for promotion/relegation: %s", err)
		return
	}

	leaguesByCountryAndTier := make(map[string]map[int][]LeagueStandings)
	for _, league := range leagues {
		standings, err := common.GetRankedClassification(dbCtx, league.LeagueId)
		if err != nil {
			log.Errorf("error getting standings for league %d: %s", league.LeagueId, err)
			continue
		}
		if leaguesByCountryAndTier[league.Country] == nil {
			leaguesByCountryAndTier[league.Country] = make(map[int][]LeagueStandings)
		}
		leaguesByCountryAndTier[league.Country][league.Tier] = append(
			leaguesByCountryAndTier[league.Country][league.Tier],
			LeagueStandings{LeagueId: league.LeagueId, Standings: standings},
		)
	}

	for country, leaguesByTier := range leaguesByCountryAndTier {
		moves := ComputePromotionRelegation(leaguesByTier)
		if len(moves) == 0 {
			continue
		}
		log.Infof("applying %d promotion/relegation moves for %s", len(moves), country)
		for _, move := range moves {
			if err := common.UpdateTeamLeague(dbCtx, move.TeamId, move.NewLeagueId); err != nil {
				log.Errorf("error moving team %d to league %d: %s", move.TeamId, move.NewLeagueId, err)
				continue
			}
			if err := common.UpdateClassificationLeague(dbCtx, move.TeamId, move.NewLeagueId); err != nil {
				log.Errorf("error repointing classification for team %d to league %d: %s", move.TeamId, move.NewLeagueId, err)
			}
		}
	}
}
