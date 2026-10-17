package engine

import (
	"context"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// AdvanceSeasonsIfComplete checks every active country independently (see
// init75_season_per_country.sql) and rolls over whichever ones have finished their own season -
// see advanceCountrySeasonIfComplete for the actual per-country rollover. One country with a
// backlog of matches (or just more leagues) no longer holds up any other country's season.
func AdvanceSeasonsIfComplete(dbCtx *common.DatabaseContext) {
	countries, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		log.Errorf("error listing active countries for season rollover: %s", err)
		return
	}
	for _, country := range countries {
		advanceCountrySeasonIfComplete(dbCtx, country)
	}
}

// advanceCountrySeasonIfComplete checks whether every match of country's current season has
// been played and has finished its live broadcast (see common.LiveBroadcastWindow) and, if so,
// rolls that one country forward: it ages every one of its rostered players a year (see
// ProgressPlayers), returns any of its loans whose duration has elapsed, promotes/relegates its
// teams based on the season's final standings, starts a new domestic cup from the
// post-promotion tier-1 teams, resets its leagues' classification for the new season, bumps its
// own season counter, and regenerates its fixtures. See ResetClassificationForSeason for why
// classification is reset in place rather than archived per season.
//
// Regional/international cups and the World Cup are deliberately NOT started here - they used to
// assume every country finished together, which is no longer true once each country advances on
// its own clock; they now run on their own independent, decoupled cadence instead (see
// StartInternationalCupsJob in engine/cups.go), reading each country's most recently completed
// champion rather than waiting for every country to roll over at once.
//
// Guarded by a non-blocking, session-level advisory lock on a dedicated connection, held for the
// whole function body - the same shape advanceCup uses for the same reason (see its own doc
// comment): this is a long multi-step check-then-act with no transaction of its own (player
// aging, promotion/relegation, cup/classification/fixture writes each run their own separate
// statements), and two --engine replicas (a real, supported deployment shape - see
// docker-compose.yml) both ticking AdvanceSeasonJob at once could otherwise both see the season
// complete and both roll country over - double player aging, double promotion/relegation moves,
// double prize money, a duplicate cup. Keyed on hashtext(country) rather than a bare int, since
// country is a string.
func advanceCountrySeasonIfComplete(dbCtx *common.DatabaseContext, country string) {
	ctx := context.Background()
	conn, err := dbCtx.Db.Conn(ctx)
	if err != nil {
		log.Errorf("error acquiring a connection for %s's season advance lock: %s", country, err)
		return
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1)::bigint)", country).Scan(&acquired); err != nil {
		log.Errorf("error acquiring season advance lock for %s: %s", country, err)
		return
	}
	if !acquired {
		// Another replica is already rolling this country's season over this tick - whatever it
		// leaves behind, the next tick picks up correctly.
		return
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext($1)::bigint)", country); err != nil {
			log.Errorf("error releasing season advance lock for %s: %s", country, err)
		}
	}()

	currentSeason, err := common.GetCurrentSeasonForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error reading current season for %s: %s", country, err)
		return
	}

	total, unplayed, err := common.GetSeasonMatchCounts(dbCtx, country, currentSeason)
	if err != nil {
		log.Errorf("error checking %s season %d completion: %s", country, currentSeason, err)
		return
	}
	if total == 0 || unplayed > 0 {
		// season never started yet, or still has matches left to play
		return
	}
	// A match flips to played = true when it is simulated, up to LiveBroadcastWindow before its
	// live broadcast finishes. Closing the season now would send the prize, promotion and
	// relegation notifications (and compute the final standings) while the last round is still
	// being watched, spoiling its result - so wait until every match has finished broadcasting.
	// The next scheduler tick picks the rollover up.
	broadcasting, err := common.CountSeasonMatchesStillBroadcasting(dbCtx, country, currentSeason)
	if err != nil {
		log.Errorf("error checking live broadcasts for %s season %d: %s", country, currentSeason, err)
		return
	}
	if broadcasting > 0 {
		log.Infof("%s season %d has all matches played but %d still broadcasting, postponing rollover", country, currentSeason, broadcasting)
		return
	}

	newSeason := currentSeason + 1
	log.Infof("%s season %d complete (%d matches played), advancing to season %d", country, currentSeason, total, newSeason)
	common.SeasonRolloversCounter.Add(context.Background(), 1)

	teamIds, err := common.GetTeamIdsForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error listing teams for %s season rollover: %s", country, err)
		return
	}
	ProgressPlayers(dbCtx, teamIds)
	ReturnExpiredLoans(dbCtx, country, newSeason)
	applyPromotionRelegation(dbCtx, country, currentSeason)
	if err := StartCupForCountry(dbCtx, country, newSeason); err != nil {
		log.Errorf("error starting cup for %s season %d: %s", country, newSeason, err)
	}

	leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error listing leagues for %s season rollover: %s", country, err)
		return
	}

	for _, leagueId := range leagueIds {
		if err := common.ResetClassificationForSeason(dbCtx, leagueId, newSeason); err != nil {
			log.Errorf("error resetting classification for league %d season %d: %s", leagueId, newSeason, err)
		}
	}

	if err := common.SetCurrentSeasonForCountry(dbCtx, country, newSeason); err != nil {
		log.Errorf("error advancing %s's current season to %d: %s", country, newSeason, err)
		return
	}
	// GetCurrentSeason (this package) caches the season for 30 minutes; refresh it in place
	// so GenerateMatchesForLeagues below picks up the new season immediately instead of
	// regenerating fixtures under the season we just closed out.
	ResetCurrentSeasonCache(dbCtx, country, newSeason)

	if err := GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
		log.Errorf("error generating fixtures for %s season %d: %s", country, newSeason, err)
	}
}

// leagueFinalStandings pairs one league's final ranked standings with what season_history needs
// to describe it - see recordSeasonHistory.
type leagueFinalStandings struct {
	leagueId   int64
	leagueName string
	standings  []common.TeamStanding
}

// applyPromotionRelegation ranks country's own leagues' final standings, computes its
// promotion/relegation moves, applies them by repointing the moved teams' league_id - both on
// teams and on their existing classification row, so the per-league reset that follows resets
// the right rows - and records every team's final position and promotion/relegation outcome for
// `season` into its season_history (see recordSeasonHistory) before that classification data is
// gone for good.
func applyPromotionRelegation(dbCtx *common.DatabaseContext, country string, season int) {
	leagues, err := common.GetLeaguesWithTierForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error listing leagues for %s promotion/relegation: %s", country, err)
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
