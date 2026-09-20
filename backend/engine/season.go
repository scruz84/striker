package engine

import (
	"context"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// AdvanceSeasonIfComplete checks whether every match of the current season has been played
// and, if so, rolls the game forward: it ages every rostered player a year (see ProgressPlayers)
// and every unowned one too (see ProgressFreeAgents), returns any loan whose duration has
// elapsed, promotes/relegates teams based on the season's final standings, starts a new cup for
// each country from the post-promotion tier-1 teams plus a new international cup per region from
// each of its countries' just-finished tier-1 champions (see StartInternationalCupsForNewSeason),
// resets each league's classification for the new season, bumps the season counter, and
// regenerates fixtures. See ResetClassificationForSeason for why classification is reset in
// place rather than archived per season.
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
	common.SeasonRolloversCounter.Add(context.Background(), 1)

	ProgressPlayers(dbCtx, currentSeason)
	ProgressFreeAgents(dbCtx)
	ReturnExpiredLoans(dbCtx, newSeason)
	applyPromotionRelegation(dbCtx, currentSeason)
	StartCupsForNewSeason(dbCtx, newSeason)
	// Must run before the classification reset below - it reads the season just finished, not
	// the one just started, to find each country's tier-1 champion (see
	// StartInternationalCupsForNewSeason).
	StartInternationalCupsForNewSeason(dbCtx, newSeason)

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

// leagueFinalStandings pairs one league's final ranked standings with what season_history needs
// to describe it - see recordSeasonHistory.
type leagueFinalStandings struct {
	leagueId   int64
	leagueName string
	standings  []common.TeamStanding
}

// applyPromotionRelegation ranks every league's final standings, computes each country's
// promotion/relegation moves independently (a team never crosses countries), applies them by
// repointing the moved teams' league_id - both on teams and on their existing classification
// row, so the per-league reset that follows resets the right rows - and records every team's
// final position and promotion/relegation outcome for `season` into its season_history (see
// recordSeasonHistory) before that classification data is gone for good.
func applyPromotionRelegation(dbCtx *common.DatabaseContext, season int) {
	leagues, err := common.GetLeaguesWithTier(dbCtx)
	if err != nil {
		log.Errorf("error listing leagues for promotion/relegation: %s", err)
		return
	}

	results := make([]leagueFinalStandings, 0, len(leagues))
	leaguesByCountryAndTier := make(map[string]map[int][]LeagueStandings)
	for _, league := range leagues {
		standings, err := common.GetRankedClassification(dbCtx, league.LeagueId)
		if err != nil {
			log.Errorf("error getting standings for league %d: %s", league.LeagueId, err)
			continue
		}
		awardLeaguePrizeMoney(dbCtx, league.Tier, standings)
		awardSeasonTicketsAndMerchandising(dbCtx, league.Tier, standings)
		awardSeasonHonors(dbCtx, league.LeagueId, season)

		leagueName, err := common.GetDivisionName(dbCtx, league.LeagueId)
		if err != nil {
			log.Errorf("error getting name for league %d: %s", league.LeagueId, err)
		}
		results = append(results, leagueFinalStandings{leagueId: league.LeagueId, leagueName: leagueName, standings: standings})

		if leaguesByCountryAndTier[league.Country] == nil {
			leaguesByCountryAndTier[league.Country] = make(map[int][]LeagueStandings)
		}
		leaguesByCountryAndTier[league.Country][league.Tier] = append(
			leaguesByCountryAndTier[league.Country][league.Tier],
			LeagueStandings{LeagueId: league.LeagueId, Standings: standings},
		)
	}

	config := common.GetPromotionRelegationConfig(dbCtx)
	promotedByTeam := make(map[int64]bool)
	for country, leaguesByTier := range leaguesByCountryAndTier {
		moves := ComputePromotionRelegation(leaguesByTier, config)
		for _, move := range moves {
			promotedByTeam[move.TeamId] = move.Promoted
		}
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
			notifyPromotionRelegation(dbCtx, move)
		}
	}

	recordSeasonHistory(dbCtx, season, results, promotedByTeam)
}

// recordSeasonHistory upserts every team's final league position and promotion/relegation
// outcome for `season` - see common.RecordSeasonHistoryLeagueResult. moved is true (and its
// value meaningful) only for teams that actually appear in promotedByTeam; everyone else neither
// promoted nor relegated.
func recordSeasonHistory(dbCtx *common.DatabaseContext, season int, results []leagueFinalStandings, promotedByTeam map[int64]bool) {
	for _, result := range results {
		for i, standing := range result.standings {
			promoted, moved := promotedByTeam[standing.TeamId]
			relegated := moved && !promoted
			if err := common.RecordSeasonHistoryLeagueResult(dbCtx, standing.TeamId, season, result.leagueId, result.leagueName, i+1, promoted, relegated); err != nil {
				log.Errorf("error recording season history for team %d season %d: %s", standing.TeamId, season, err)
			}
		}
	}
}

func notifyPromotionRelegation(dbCtx *common.DatabaseContext, move LeagueMove) {
	team, err := common.GetTeamById(move.TeamId, dbCtx)
	if err != nil {
		log.Errorf("error getting team %d for promotion/relegation notification: %s", move.TeamId, err)
		return
	}
	notificationType := common.NOTIFICATION_RELEGATION
	verb := "relegated"
	key := "notifications.messages.relegated"
	if move.Promoted {
		notificationType = common.NOTIFICATION_PROMOTION
		verb = "promoted"
		key = "notifications.messages.promoted"
	}
	msg := common.NotificationMessage{
		Text:   team.TeamName + " was " + verb + "!",
		Key:    key,
		Params: map[string]any{"team": team.TeamName},
	}
	if err := common.NotifyTeamOwner(dbCtx, move.TeamId, notificationType, msg); err != nil {
		log.Errorf("error notifying team %d of promotion/relegation: %s", move.TeamId, err)
	}
}
