package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// AdvanceSeasonIfComplete checks whether every match of the current season has been played
// and, if so, rolls the game forward: it resets each league's classification for the new
// season, bumps the season counter, and regenerates fixtures.
//
// Promotion/relegation between divisions is NOT implemented - the leagues table has no
// tier/parent-child relationship to model a ladder, so every team simply continues in the
// same league next season. See ResetClassificationForSeason for why classification is reset
// in place rather than archived per season.
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
