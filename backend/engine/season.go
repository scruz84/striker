package engine

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

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
// its own clock. A region's cup starts from its own check once enough of its countries have new
// champions (see StartRegionalCupsIfDue), and the World Cup runs on its own clock (see
// StartWorldCupIfDue).
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
//
// The lock does not cover a run that stops halfway (a restart, an error between steps), so each
// step is recorded in the country's progress marker as it completes and a later run resumes at
// the next step - see runSeasonRollover and openspec/specs/engine-replicas "Resumable season
// rollover".
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
	progress, err := common.GetRolloverProgress(dbCtx, country)
	if err != nil {
		log.Errorf("error reading %s's rollover progress: %s", country, err)
		return
	}
	// The season already moved but its fixtures were never generated: that is all that is left.
	if progress.Step == rolloverStepSeasonAdvanced && progress.Season == currentSeason-1 {
		ResetCurrentSeasonCache(dbCtx, country, currentSeason)
		finishRolloverFixtures(dbCtx, country, currentSeason)
		return
	}
	// A marker for the season being closed means an earlier run stopped halfway: the season was
	// complete then, so the checks below are skipped and the rollover goes on from the next step.
	// Any other marker is stale (another season's) and is ignored.
	resuming := progress.Season == currentSeason && progress.Step > rolloverStepNone && progress.Step < rolloverStepSeasonAdvanced
	if !resuming {
		progress = common.RolloverProgress{}
	}

	if !resuming {
		total, unplayed, err := common.GetSeasonMatchCounts(dbCtx, country, currentSeason)
		if err != nil {
			log.Errorf("error checking %s season %d completion: %s", country, currentSeason, err)
			return
		}
		if total == 0 || unplayed > 0 {
			// season never started yet, or still has matches left to play
			return
		}
		// A match flips to played = true when it is simulated, up to LiveBroadcastWindow before
		// its live broadcast finishes. Closing the season now would send the prize, promotion and
		// relegation notifications (and compute the final standings) while the last round is
		// still being watched, spoiling its result - so wait until every match has finished
		// broadcasting. The next scheduler tick picks the rollover up.
		broadcasting, err := common.CountSeasonMatchesStillBroadcasting(dbCtx, country, currentSeason)
		if err != nil {
			log.Errorf("error checking live broadcasts for %s season %d: %s", country, currentSeason, err)
			return
		}
		if broadcasting > 0 {
			log.Infof("%s season %d has all matches played but %d still broadcasting, postponing rollover", country, currentSeason, broadcasting)
			return
		}
	} else {
		log.Infof("%s season %d rollover resuming after step %d", country, currentSeason, progress.Step)
	}

	// The promotion playoff (see openspec/specs/league-pyramid "Playoff phase"): once the league
	// is over and broadcast, the first tick creates the season's playoff ties and returns; the
	// rollover then waits until every tie has been played and its broadcast has finished. Both are
	// re-read here, inside the country's lock, so the ties are created once across replicas.
	shape, err := common.GetCountryPyramidShape(dbCtx, country)
	if err != nil {
		log.Errorf("error reading %s's league pyramid: %s", country, err)
		return
	}
	var playoff common.PlayoffState
	if countryHasPlayoff(shape) {
		if playoff, err = common.GetCountryPlayoff(dbCtx, country, currentSeason); err != nil {
			log.Errorf("error reading %s's season %d playoff: %s", country, currentSeason, err)
			return
		}
		if !resuming {
			if !playoff.Exists {
				if err := startPlayoff(dbCtx, country, currentSeason, shape); err != nil {
					log.Errorf("error starting %s's season %d playoff: %s", country, currentSeason, err)
				}
				return
			}
			if !playoff.Done() {
				return
			}
		}
	}

	runSeasonRollover(dbCtx, country, currentSeason, shape, playoff, progress)
}

// The steps of a country's season rollover, in order. Each is recorded in the country's progress
// marker (see common.SetRolloverProgress) as soon as it completes, so a rollover that stops
// halfway resumes at the next step instead of repeating aging, prizes, the cup or
// promotion/relegation (openspec/specs/engine-replicas "Resumable season rollover").
const (
	rolloverStepNone              = 0
	rolloverStepPlayersProgressed = 1 // ProgressPlayers: age, contracts, retirements
	rolloverStepLoansReturned     = 2 // ReturnExpiredLoans
	rolloverStepPromotionApplied  = 3 // applyPromotionRelegation: prizes, income, awards, moves, history
	rolloverStepRestructured      = 4 // restructureIfNeeded
	rolloverStepCupStarted        = 5 // StartCupForCountry
	rolloverStepStandingsReset    = 6 // classification reset and starting order
	rolloverStepSeasonAdvanced    = 7 // current_season + 1; only the fixtures are left
)

// rolloverState is what later steps need from earlier ones, kept with the marker so a resumed
// rollover uses the standings and moves computed before the classification moved on.
type rolloverState struct {
	FinalStandings []LeagueStandings `json:"finalStandings"`
	Moves          []LeagueMove      `json:"moves"`
	// RestructureAttempts counts the restructures of this rollover that left the country differing
	// from its structure (see maxRestructureAttempts).
	RestructureAttempts int `json:"restructureAttempts,omitempty"`
}

// maxRestructureAttempts is how many restructures one rollover runs, one per tick, before it
// moves on with a country that still differs from its structure (see
// openspec/specs/league-pyramid "Restructure result is checked").
const maxRestructureAttempts = 3

// rolloverInterruptAfter, when set (tests only), stops a rollover right after step has been
// recorded - what a restart at that point does.
var rolloverInterruptAfter func(step int) bool

// runSeasonRollover runs the steps of country's season rollover after progress.Step, recording
// each one. It stops, leaving the marker for the next tick, as soon as a step cannot be recorded or
// an error leaves the rollover unable to go on.
func runSeasonRollover(dbCtx *common.DatabaseContext, country string, currentSeason int, shape common.PyramidShape, playoff common.PlayoffState, progress common.RolloverProgress) {
	newSeason := currentSeason + 1
	var state rolloverState
	if len(progress.State) > 0 {
		if err := json.Unmarshal(progress.State, &state); err != nil {
			log.Errorf("error reading %s's season %d rollover state: %s", country, currentSeason, err)
			return
		}
	}
	step := progress.Step
	record := func(done int) bool {
		var raw json.RawMessage
		if state.FinalStandings != nil || state.Moves != nil {
			encoded, err := json.Marshal(state)
			if err != nil {
				log.Errorf("error encoding %s's season %d rollover state: %s", country, currentSeason, err)
				return false
			}
			raw = encoded
		}
		if err := common.SetRolloverProgress(dbCtx, country, currentSeason, done, raw); err != nil {
			log.Errorf("error recording step %d of %s's season %d rollover: %s", done, country, currentSeason, err)
			return false
		}
		step = done
		return rolloverInterruptAfter == nil || !rolloverInterruptAfter(done)
	}

	if step == rolloverStepNone {
		log.Infof("%s season %d complete, advancing to season %d", country, currentSeason, newSeason)
		common.SeasonRolloversCounter.Add(context.Background(), 1)
	}
	if step < rolloverStepPlayersProgressed {
		teamIds, err := common.GetTeamIdsForCountry(dbCtx, country)
		if err != nil {
			log.Errorf("error listing teams for %s season rollover: %s", country, err)
			return
		}
		ProgressPlayers(dbCtx, teamIds)
		if !record(rolloverStepPlayersProgressed) {
			return
		}
	}
	if step < rolloverStepLoansReturned {
		ReturnExpiredLoans(dbCtx, country, newSeason)
		// A returning loanee leaves his borrower's saved lineup (see common.ReturnLoan), after
		// ProgressPlayers already refreshed it - re-pick every unowned club's XI so a CPU side
		// does not start the season with a lineup short of the players it just sent home.
		refreshUnownedLineupsForCountry(dbCtx, country)
		if !record(rolloverStepLoansReturned) {
			return
		}
	}
	if step < rolloverStepPromotionApplied {
		state.FinalStandings, state.Moves = applyPromotionRelegation(dbCtx, country, currentSeason, shape, playoff)
		if !record(rolloverStepPromotionApplied) {
			return
		}
	}
	if step < rolloverStepRestructured {
		// A country whose leagues differ from the saved structure is reshaped now, so the new
		// season's cup, classification and fixtures see the new leagues; its starting order is
		// then built from where every team ends up.
		restructureMoves, diffs := restructureIfNeeded(dbCtx, country, currentSeason, state.FinalStandings)
		if restructureMoves != nil {
			state.Moves = restructureMoves
		}
		if len(diffs) > 0 {
			// The restructure did not reach the structure. Keep the step where it was, so the next
			// tick runs it again from what is there now, unless it has failed too often: then move
			// on, and the country stays pending for its next rollover.
			state.RestructureAttempts++
			if state.RestructureAttempts < maxRestructureAttempts {
				log.Warnf("%s's restructure attempt %d of %d left it differing from its structure, retrying next tick: %s",
					country, state.RestructureAttempts, maxRestructureAttempts, common.DescribeShapeDifferences(diffs))
				record(rolloverStepPromotionApplied)
				return
			}
			log.Errorf("%s's restructure still differs from its structure after %d attempts, advancing the season anyway: %s",
				country, state.RestructureAttempts, common.DescribeShapeDifferences(diffs))
		}
		if !record(rolloverStepRestructured) {
			return
		}
	}
	if step < rolloverStepCupStarted {
		if err := StartCupForCountry(dbCtx, country, newSeason); err != nil {
			log.Errorf("error starting cup for %s season %d: %s", country, newSeason, err)
		}
		if !record(rolloverStepCupStarted) {
			return
		}
	}

	leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error listing leagues for %s season rollover: %s", country, err)
		return
	}
	if step < rolloverStepStandingsReset {
		for _, leagueId := range leagueIds {
			if err := common.ResetClassificationForSeason(dbCtx, leagueId, newSeason); err != nil {
				log.Errorf("error resetting classification for league %d season %d: %s", leagueId, newSeason, err)
			}
		}
		writeSeasonSeeds(dbCtx, leagueIds, state.FinalStandings, state.Moves)
		if !record(rolloverStepStandingsReset) {
			return
		}
	}

	// The season moves and the bump is recorded in one statement: after a stop here the next tick
	// sees the new season and only generates its fixtures.
	if err := common.AdvanceSeasonRecordingStep(dbCtx, country, newSeason, rolloverStepSeasonAdvanced); err != nil {
		log.Errorf("error advancing %s's current season to %d: %s", country, newSeason, err)
		return
	}
	// GetCurrentSeason (this package) caches the season for 30 minutes; refresh it in place
	// so GenerateMatchesForLeagues below picks up the new season immediately instead of
	// regenerating fixtures under the season we just closed out.
	ResetCurrentSeasonCache(dbCtx, country, newSeason)
	if rolloverInterruptAfter != nil && rolloverInterruptAfter(rolloverStepSeasonAdvanced) {
		return
	}
	finishRolloverFixtures(dbCtx, country, newSeason)
}

// finishRolloverFixtures generates country's fixtures for newSeason, the last step of its
// rollover, and clears the progress marker. On failure the marker stays, so the next tick
// retries; generating is safe to repeat (a league that already has upcoming fixtures is skipped).
func finishRolloverFixtures(dbCtx *common.DatabaseContext, country string, newSeason int) {
	leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, country)
	if err != nil {
		log.Errorf("error listing leagues for %s season %d fixtures: %s", country, newSeason, err)
		return
	}
	if err := GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
		log.Errorf("error generating fixtures for %s season %d: %s", country, newSeason, err)
		return
	}
	if err := common.ClearRolloverProgress(dbCtx, country); err != nil {
		log.Errorf("error clearing %s's rollover progress: %s", country, err)
	}
}

// writeSeasonSeeds stores every league's starting order for the new season (see
// ComputeSeasonSeeds) once its classification has been reset. Runs under the country's rollover
// lock; a league's seeds are replaced in one transaction, so a rerun writes the same values. A
// league that got no seeds (its standings could not be read) is left as it was.
func writeSeasonSeeds(dbCtx *common.DatabaseContext, leagueIds []int64, finalStandings []LeagueStandings, moves []LeagueMove) {
	if len(finalStandings) == 0 {
		return
	}
	seeds := ComputeSeasonSeeds(finalStandings, moves)
	for _, leagueId := range leagueIds {
		leagueSeeds, ok := seeds[leagueId]
		if !ok {
			continue
		}
		if err := common.SetLeagueSeeds(dbCtx, leagueId, leagueSeeds); err != nil {
			log.Errorf("error writing the starting order of league %d: %s", leagueId, err)
		}
	}
}

// leagueFinalStandings pairs one league's final ranked standings with what season_history needs
// to describe it - see recordSeasonHistory. lines holds each team's full final table line, read
// before any promotion/relegation move repoints the classification rows.
type leagueFinalStandings struct {
	leagueId   int64
	leagueName string
	standings  []common.TeamStanding
	lines      map[int64]common.ClassificationPosition
}

// applyPromotionRelegation ranks country's own leagues' final standings, computes its
// promotion/relegation moves (see ComputeSeasonMoves: the direct moves plus the outcomes of the
// season's playoff, or the legacy swap for a pyramid that breaks the ratio rule), applies them by
// repointing the moved teams' league_id - both on teams and on their existing classification row,
// so the per-league reset that follows resets the right rows - marks the playoff completed, and
// records every team's final position and promotion/relegation outcome for `season` into its
// season_history (see recordSeasonHistory) before that classification data is gone for good. It
// returns every league's final standings and the moves applied, from which the new season's
// starting order is built (see ComputeSeasonSeeds).
func applyPromotionRelegation(dbCtx *common.DatabaseContext, country string, season int, shape common.PyramidShape, playoff common.PlayoffState) ([]LeagueStandings, []LeagueMove) {
	results := make([]leagueFinalStandings, 0, len(shape.Leagues))
	leaguesByTier := make(map[int][]LeagueStandings)
	leagueOfTeam := make(map[int64]int64)
	tierOfLeague := make(map[int64]int)
	for _, league := range shape.Leagues {
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
		finalLines, err := common.GetFinalClassification(dbCtx, league.LeagueId)
		if err != nil {
			log.Errorf("error getting the final table of league %d: %s", league.LeagueId, err)
		}
		lines := make(map[int64]common.ClassificationPosition, len(finalLines))
		for _, line := range finalLines {
			lines[line.TeamId] = line
		}
		results = append(results, leagueFinalStandings{leagueId: league.LeagueId, leagueName: leagueName, standings: standings, lines: lines})
		leaguesByTier[league.Tier] = append(leaguesByTier[league.Tier], LeagueStandings{LeagueId: league.LeagueId, Standings: standings})
		tierOfLeague[league.LeagueId] = league.Tier
		for _, st := range standings {
			leagueOfTeam[st.TeamId] = league.LeagueId
		}
	}

	var playoffOutcomes []PlayoffResult
	if playoff.Exists {
		outcomes, err := playoffResults(dbCtx, playoff.CupId, leagueOfTeam, tierOfLeague)
		if err != nil {
			log.Errorf("error reading %s's season %d playoff results: %s", country, season, err)
		}
		playoffOutcomes = outcomes
	}
	moves := ComputeSeasonMoves(leaguesByTier, !shape.FollowsRatioRule(), playoffOutcomes)
	outcomeByTeam := make(map[int64]LeagueMove, len(moves))
	for _, move := range moves {
		outcomeByTeam[move.TeamId] = move
	}
	if len(moves) > 0 {
		log.Infof("applying %d promotion/relegation moves for %s", len(moves), country)
	}
	for _, move := range moves {
		if err := common.UpdateTeamLeague(dbCtx, move.TeamId, move.NewLeagueId); err != nil {
			log.Errorf("error moving team %d to league %d: %s", move.TeamId, move.NewLeagueId, err)
			continue
		}
		if err := common.UpdateClassificationLeague(dbCtx, move.TeamId, move.NewLeagueId); err != nil {
			log.Errorf("error repointing classification for team %d to league %d: %s", move.TeamId, move.NewLeagueId, err)
		}
		notifyPromotionRelegation(dbCtx, move)
		redrawAiLevelAfterMove(dbCtx, move.TeamId, move.NewLeagueId, rand.New(rand.NewSource(time.Now().UnixNano()+move.TeamId)))
	}
	if playoff.Exists && !playoff.Completed {
		if err := common.SetCupCompletedWithoutChampion(dbCtx, playoff.CupId); err != nil {
			log.Errorf("error completing %s's season %d playoff (cup %d): %s", country, season, playoff.CupId, err)
		}
	}

	recordSeasonHistory(dbCtx, season, results, outcomeByTeam)
	applyTitleRaceRivalries(dbCtx, country, results)

	finalStandings := make([]LeagueStandings, 0, len(results))
	for _, result := range results {
		finalStandings = append(finalStandings, LeagueStandings{LeagueId: result.leagueId, Standings: result.standings})
	}
	return finalStandings, moves
}

// applyTitleRaceRivalries adds the season's title race to each division's champion-runner-up pair
// (see openspec rivalries "Season and transfer rivalry events"), then applies the country's yearly
// decay to every pair, so a title race counts before the decay. A level the title race raised is
// announced only if it is still raised once the decay is applied; a fall is never announced.
// Runs inside advanceCountrySeasonIfComplete, under the country's rollover lock, so the decay is
// applied once per rollover even with several engine replicas.
func applyTitleRaceRivalries(dbCtx *common.DatabaseContext, country string, results []leagueFinalStandings) {
	type raised struct {
		a, b int64
		old  float64
	}
	races := make([]raised, 0, len(results))
	for _, result := range results {
		if len(result.standings) < 2 {
			continue
		}
		champion, runnerUp := result.standings[0].TeamId, result.standings[1].TeamId
		if old, _, ok := common.AddRivalryEventQuietly(dbCtx, champion, runnerUp, common.RivalryTitleRacePoints); ok {
			races = append(races, raised{champion, runnerUp, old})
		}
	}
	if err := common.DecayCountryRivalries(dbCtx, country); err != nil {
		log.Errorf("error decaying %s's rivalries: %s", country, err)
		return
	}
	for _, race := range races {
		score, err := common.GetRivalryScore(dbCtx, race.a, race.b)
		if err != nil {
			log.Errorf("error reading rivalry %d-%d after the decay: %s", race.a, race.b, err)
			continue
		}
		common.NotifyRivalryRise(dbCtx, race.a, race.b, race.old, score)
	}
}

// recordSeasonHistory upserts every team's final league position, final table line and
// promotion/relegation outcome for `season` - see common.RecordSeasonHistoryLeagueResult. A team
// in outcomeByTeam was promoted or relegated (through the playoff when its move says so); every
// other team neither, including a team that won its playoff and stayed put.
func recordSeasonHistory(dbCtx *common.DatabaseContext, season int, results []leagueFinalStandings, outcomeByTeam map[int64]LeagueMove) {
	for _, result := range results {
		for i, standing := range result.standings {
			move, moved := outcomeByTeam[standing.TeamId]
			promoted := moved && move.Promoted
			relegated := moved && !move.Promoted
			viaPlayoff := moved && move.ViaPlayoff
			var line *common.ClassificationPosition
			if l, ok := result.lines[standing.TeamId]; ok {
				line = &l
			}
			if err := common.RecordSeasonHistoryLeagueResult(dbCtx, standing.TeamId, season, result.leagueId, result.leagueName, i+1, line, promoted, relegated, viaPlayoff); err != nil {
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
	suffix := "!"
	if move.ViaPlayoff {
		key += "ViaPlayoff"
		suffix = " through the playoff!"
	}
	msg := common.NotificationMessage{
		Text:   team.TeamName + " was " + verb + suffix,
		Key:    key,
		Params: map[string]any{"team": team.TeamName},
	}
	if err := common.NotifyTeamOwner(dbCtx, move.TeamId, notificationType, msg); err != nil {
		log.Errorf("error notifying team %d of promotion/relegation: %s", move.TeamId, err)
	}
}
