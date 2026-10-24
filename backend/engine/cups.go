package engine

import (
	"context"
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

// CupTier is the league tier a cup draws its entrants from (1 = top division).
const CupTier = 1

// InternationalCupCadence is how often, in real time, StartInternationalCupsIfDue actually
// starts a new regional/international cup round - since countries no longer all finish their
// domestic season together (see init75_season_per_country.sql), there's no single shared
// rollover moment left to hang this on, so it runs on its own clock instead, standing in for
// "roughly how long a domestic season takes" to play out.
const InternationalCupCadence = 60 * 24 * time.Hour

// StartInternationalCupsIfDue starts a new regional/international cup round (see
// StartInternationalCupsForNewSeason) and, every WorldCupFrequencySeasons rounds, a new World
// Cup (see StartWorldCupForNewSeason) - gated on InternationalCupCadence via
// season.last_international_cup_check (see common.ClaimInternationalCupSeason), so this only
// actually fires roughly that often even though the job polling it (see
// StartInternationalCupScheduler) ticks far more frequently. Entrants are each country's most
// recently completed season's tier-1 champion (see common.GetMostRecentTier1Champions), grouped
// by region (see common.RegionForCountry) - not any one moment's live standings, since countries
// can now be mid-season at wildly different points when this runs.
func StartInternationalCupsIfDue(dbCtx *common.DatabaseContext) {
	newSeason, claimed, err := common.ClaimInternationalCupSeason(dbCtx, InternationalCupCadence)
	if err != nil {
		log.Errorf("error claiming the next international cup round: %s", err)
		return
	}
	if !claimed {
		// Not due yet, or another replica claimed this round at the same instant.
		return
	}

	championsByCountry, err := common.GetMostRecentTier1Champions(dbCtx, InternationalCupTier)
	if err != nil {
		log.Errorf("error finding each country's most recent tier-1 champion for international cups: %s", err)
		return
	}
	championsByRegion := make(map[string][]int64, len(championsByCountry))
	for country, teamId := range championsByCountry {
		region := common.RegionForCountry(country)
		championsByRegion[region] = append(championsByRegion[region], teamId)
	}

	StartInternationalCupsForNewSeason(dbCtx, newSeason, championsByRegion)
	StartWorldCupForNewSeason(dbCtx, newSeason)
}

// StartCupsForNewSeason creates a fresh single-elimination cup for every country, entrants
// drawn from that country's tier-1 teams, and generates its first round. Used for the
// every-country-at-once case (a freshly initialized world - see handlers.InitializeGame); a
// single country's own rollover calls StartCupForCountry directly instead (see
// advanceCountrySeasonIfComplete).
func StartCupsForNewSeason(dbCtx *common.DatabaseContext, season int) {
	leagues, err := common.GetLeaguesWithTier(dbCtx)
	if err != nil {
		log.Errorf("error listing leagues to start season cups: %s", err)
		return
	}

	countries := make(map[string]bool)
	for _, league := range leagues {
		if league.Tier == CupTier {
			countries[league.Country] = true
		}
	}

	for country := range countries {
		if err := startCupForCountry(dbCtx, country, season); err != nil {
			log.Errorf("error starting cup for %s season %d: %s", country, season, err)
		}
	}
}

// StartCupForCountry starts this season's cup for exactly one country - used when a country is
// activated mid-season (see handlers.AdminAddCountryHandler), where calling
// StartCupsForNewSeason again would duplicate every other country's already-started cup (cups
// has no unique constraint on (country, season) to stop that).
func StartCupForCountry(dbCtx *common.DatabaseContext, country string, season int) error {
	return startCupForCountry(dbCtx, country, season)
}

func startCupForCountry(dbCtx *common.DatabaseContext, country string, season int) error {
	leagues, err := common.GetLeaguesWithTier(dbCtx)
	if err != nil {
		return err
	}

	teamIds := make([]int64, 0)
	for _, league := range leagues {
		if league.Country != country || league.Tier != CupTier {
			continue
		}
		teamIds = append(teamIds, common.GetDivisionTeams(dbCtx, league.LeagueId)...)
	}
	if len(teamIds) < 2 {
		return nil // not enough tier-1 teams yet to run a cup
	}

	cupId, err := common.CreateCup(dbCtx, country, season, "Cup")
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pairs := BuildFirstRoundPairings(teamIds, r, classificationPointsForSeeding(dbCtx, teamIds))
	return createRoundTies(dbCtx, cupId, 1, pairs)
}

// classificationPointsForSeeding fetches each of teamIds' current league classification points,
// for BuildFirstRoundPairings/BuildNextRoundPairings to use as their "which entrant(s) get the
// bye" tiebreak - a real tournament hands a walkover to its higher seeds rather than drawing one
// at random. common.GetTeamClassificationPoints already maps a team with no classification row
// (a national team, in a World Cup knockout bracket - see startWorldCupKnockout) to 0 rather than
// an error, so this never needs to special-case that itself; a cup whose every entrant is a
// national team (the normal case there) just ends up with an all-zero map, which
// splitByeCandidates/pairSequentially already treat the same as no seeding data at all.
func classificationPointsForSeeding(dbCtx *common.DatabaseContext, teamIds []int64) map[int64]int {
	points := make(map[int64]int, len(teamIds))
	for _, teamId := range teamIds {
		p, err := common.GetTeamClassificationPoints(dbCtx, teamId)
		if err != nil {
			log.Errorf("error reading classification points for team %d, seeding without it: %s", teamId, err)
			continue
		}
		points[teamId] = p
	}
	return points
}

// InternationalCupTier is the league tier whose champion represents a country in its region's
// international cup (issue #64) - same tier the domestic cup draws its entrants from.
const InternationalCupTier = CupTier

// internationalCupNames gives each region's cup a name distinct from any country's domestic
// "Cup" - falls back to a generic name for a region with no dedicated one.
var internationalCupNames = map[string]string{
	"europe":   "European Cup",
	"americas": "Copa America",
	"asia":     "Asian Cup",
	"africa":   "African Cup",
}

func internationalCupName(region string) string {
	if name, ok := internationalCupNames[region]; ok {
		return name
	}
	return "International Cup"
}

// StartInternationalCupsForNewSeason creates a fresh international cup for every region with at
// least two countries that have a recorded tier-1 champion, entrants being one team per country
// - that country's most recently completed season's tier-1 champion. `champions` comes from
// common.GetMostRecentTier1Champions, grouped by region (see StartInternationalCupsIfDue) -
// reads persisted season_history rather than any one moment's live standings, since countries no
// longer all finish their season together (see init75_season_per_country.sql). Mirrors
// StartCupsForNewSeason's per-country cup, one level up.
func StartInternationalCupsForNewSeason(dbCtx *common.DatabaseContext, season int, champions map[string][]int64) {
	for region, teamIds := range champions {
		if len(teamIds) < 2 {
			continue // not enough countries in this region yet for a cup
		}
		if err := startInternationalCupForRegion(dbCtx, region, season, teamIds); err != nil {
			log.Errorf("error starting international cup for region %s season %d: %s", region, season, err)
		}
	}
}

func startInternationalCupForRegion(dbCtx *common.DatabaseContext, region string, season int, teamIds []int64) error {
	cupId, err := common.CreateInternationalCup(dbCtx, region, season, internationalCupName(region))
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pairs := BuildFirstRoundPairings(teamIds, r, classificationPointsForSeeding(dbCtx, teamIds))
	return createRoundTies(dbCtx, cupId, 1, pairs)
}

// cupRoundGap is how far ahead of "now" a round's ties are scheduled - both for a fresh cup's
// first round and for each next round AdvanceCups generates once the previous one resolves. A
// round only counts as "resolved" once every tie is simulated (see IsCupRoundResolved), which
// happens well before that tie's live broadcast finishes pacing out to viewers (~90 real minutes,
// see engine.liveMatchMinuteDelay) - scheduling the next round only 15 minutes later (this file's
// original delay) meant a manager could still be watching one round's tie live while the next
// round's was already kicking off (issue #55). A full day comfortably clears that broadcast
// window and, as a side effect, spreads rounds across different days rather than back-to-back.
const cupRoundGap = 24 * time.Hour

// createRoundTies persists one round's pairings as cup_ties. A pairing with a bye (one side is
// ByeTeamId) is recorded as already resolved, with no match_date - see common.CreateCupTie. For a
// real tie, any side that turns out to be a national team (true only for a World Cup's knockout
// bracket - see engine.startWorldCupKnockout) immediately gets its lineup auto-selected, the same
// way SetNationalTeamLineup does for a weekly friendly - a national team has no manager to set one
// otherwise, and this covers every round a World Cup's knockout stage generates, not just its
// first.
func createRoundTies(dbCtx *common.DatabaseContext, cupId int64, round int, pairs [][2]int64) error {
	matchDate := time.Now().Add(cupRoundGap)
	for i, pair := range pairs {
		home, away := asTeamIdPtr(pair[0]), asTeamIdPtr(pair[1])
		var scheduledAt *time.Time
		if home != nil && away != nil {
			scheduledAt = &matchDate
		}
		tieId, err := common.CreateCupTie(dbCtx, cupId, round, i+1, home, away, scheduledAt)
		if err != nil {
			return err
		}
		if home != nil && away != nil {
			setNationalTeamLineupForTieIfNeeded(dbCtx, tieId, *home)
			setNationalTeamLineupForTieIfNeeded(dbCtx, tieId, *away)
		}
	}
	return nil
}

// setNationalTeamLineupForTieIfNeeded is createRoundTies' per-side hook: a no-op for every club
// team (the overwhelming majority of cup ties), and otherwise sets teamId's lineup for tieId via
// common.SetNationalTeamCupTieLineup.
func setNationalTeamLineupForTieIfNeeded(dbCtx *common.DatabaseContext, tieId int64, teamId int64) {
	country, isNationalTeam, err := common.GetNationalTeamCountry(dbCtx, teamId)
	if err != nil {
		log.Errorf("error checking whether team %d is a national team for tie %d: %s", teamId, tieId, err)
		return
	}
	if !isNationalTeam {
		return
	}
	if err := common.SetNationalTeamCupTieLineup(dbCtx, tieId, teamId, country); err != nil {
		log.Errorf("error setting national-team lineup for team %d tie %d: %s", teamId, tieId, err)
	}
}

func asTeamIdPtr(teamId int64) *int64 {
	if teamId == ByeTeamId {
		return nil
	}
	return &teamId
}

// AdvanceCups checks every active cup: if its current round is fully resolved (played, or
// byes), it either generates the next round's ties or, if that was the final, crowns the
// champion and marks the cup complete.
func AdvanceCups(dbCtx *common.DatabaseContext) {
	cups, err := common.GetActiveCups(dbCtx)
	if err != nil {
		log.Errorf("error listing active cups: %s", err)
		return
	}

	for _, cup := range cups {
		if err := advanceCup(dbCtx, cup.CupId, cup.Season); err != nil {
			log.Errorf("error advancing cup %d: %s", cup.CupId, err)
		}
	}
}

// advanceCup checks whether cupId's current round is fully resolved and, if so, either completes
// the cup (a single winner left) or creates the next round's ties - a multi-step check-then-act
// with no transaction of its own, since its steps (GetCupRoundWinners, SetCupCompleted,
// createRoundTies, ...) each run their own separate statements against dbCtx.Db. Two --engine
// replicas (a real, supported deployment shape - see docker-compose.yml) both running CupJob's
// per-minute tick could otherwise both see the round resolved and both create a duplicate round
// (or both complete the cup and double-award/double-notify its champion) before either one's
// writes are visible to the other. Guarded by a non-blocking, session-level advisory lock on a
// dedicated connection (same shape common.EnsureSchema's cross-process migration lock uses,
// pg_try_ rather than blocking since a losing replica should just skip this tick, not wait) held
// for the whole function body - unlike the single-transaction pg_try_advisory_xact_lock pattern
// used elsewhere (see generateMatchesForOneLeague, common.SetDefaultLineup), this critical
// section spans several independent statements/implicit transactions, so the lock needs to
// outlive any one of them. Keyed on cupId alone, the same shared single-int-key convention
// common.SetDefaultLineup (teamId) and generateMatchesForOneLeague (leagueId) already use.
func advanceCup(dbCtx *common.DatabaseContext, cupId int64, season int) error {
	ctx := context.Background()
	conn, err := dbCtx.Db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", cupId).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		// Another replica is already advancing this cup this tick - whatever it leaves behind,
		// the next tick picks up correctly.
		return nil
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", cupId); err != nil {
			log.Errorf("error releasing cup advance lock for cup %d: %s", cupId, err)
		}
	}()

	round, err := common.GetMaxCupRound(dbCtx, cupId)
	if err != nil || round == 0 {
		return err
	}

	resolved, err := common.IsCupRoundResolved(dbCtx, cupId, round)
	if err != nil || !resolved {
		return err
	}

	winners, err := common.GetCupRoundWinners(dbCtx, cupId, round)
	if err != nil {
		return err
	}

	if len(winners) == 1 {
		log.Infof("cup %d complete, champion is team %d", cupId, winners[0])
		if err := common.SetCupCompleted(dbCtx, cupId, winners[0]); err != nil {
			return err
		}
		awardCupChampionBonus(dbCtx, winners[0])
		if err := common.RecordSeasonHistoryCupResult(dbCtx, winners[0], season, "Champion"); err != nil {
			log.Errorf("error recording cup champion history for team %d season %d: %s", winners[0], season, err)
		}
		if team, err := common.GetTeamById(winners[0], dbCtx); err == nil {
			msg := common.NotificationMessage{
				Text:   team.TeamName + " won the cup!",
				Key:    "notifications.messages.cupChampion",
				Params: map[string]any{"team": team.TeamName},
			}
			if err := common.NotifyTeamOwner(dbCtx, winners[0], common.NOTIFICATION_CUP, msg); err != nil {
				log.Errorf("error notifying cup champion team %d: %s", winners[0], err)
			}
		}
		return nil
	}

	pairs := BuildNextRoundPairings(winners, classificationPointsForSeeding(dbCtx, winners))
	return createRoundTies(dbCtx, cupId, round+1, pairs)
}

// SimulateCupTies simulates every cup tie whose match_date has arrived.
func SimulateCupTies(dbCtx *common.DatabaseContext) {
	ties, err := common.GetSimulableCupTies(dbCtx)
	if err != nil {
		log.Errorf("error getting simulable cup ties: %s", err)
		return
	}
	simulateCupTiesNow(dbCtx, ties)
}

// FinishStuckCupTies simulates every due cup tie that is past its retry backoff right away, for the
// admin action - a cup tie has no claim to get stuck on, but a failed one waits out the backoff,
// and this is what moves it along on demand. Returns how many of them ended up played.
func FinishStuckCupTies(dbCtx *common.DatabaseContext) int {
	ties, err := common.GetSimulableCupTies(dbCtx)
	if err != nil {
		log.Errorf("error getting simulable cup ties: %s", err)
		return 0
	}
	simulateCupTiesNow(dbCtx, ties)
	ids := make([]int64, len(ties))
	for i, tie := range ties {
		ids[i] = tie.TieId
	}
	return countPlayed(dbCtx, common.SimulationKindCup, ids)
}

func simulateCupTiesNow(dbCtx *common.DatabaseContext, ties []common.CupTie) {
	// Same bounded-concurrency + panic-recovery approach as SimulateMatches (engine.go) - a
	// bracket round produces far fewer simultaneous ties than a league's matchday, but the
	// failure mode (an unbounded goroutine burst, or one panic taking down the whole process)
	// is identical, so it gets the same guard.
	sem := make(chan struct{}, maxConcurrentSimulations)
	var wg sync.WaitGroup
	for _, tie := range ties {
		log.Infof("simulating cup tie %d", tie.TieId)
		wg.Add(1)
		sem <- struct{}{}
		go func(tie common.CupTie) {
			defer wg.Done()
			defer func() { <-sem }()
			runClaimedSimulation(dbCtx, common.SimulationKindCup, tie.TieId, func() error {
				return simulateCupTieOnce(dbCtx, &tie)
			})
		}(tie)
	}
	wg.Wait()
}

// simulateCupTieOnce simulates tie unless another --engine replica is already simulating it or
// has already finished it. GetSimulableCupTies does not claim the ties it returns, and every
// replica runs CupJob at the same instant, so both used to simulate the same tie: two sets of
// events, double injuries and skill changes, and possibly two different winners. The tie is
// simulated under a session-level advisory lock keyed to it, and the claim of its attempt
// (common.ClaimCupTieAttempt) re-checks inside the lock that it is still unplayed, so the replica
// that loses the race skips it. A crash mid-simulation releases the lock with the session, so the
// next tick simulates the tie again instead of leaving it stuck. A simulation that returns an error
// is returned to the caller, which records the failure (see runClaimedSimulation).
func simulateCupTieOnce(dbCtx *common.DatabaseContext, tie *common.CupTie) error {
	var simulateErr error
	_, err := common.TryWithSessionAdvisoryLock(dbCtx, common.NamespacedLockKey(common.LockNamespaceCupTieSimulation, tie.TieId), func() error {
		// A cup tie has no in_simulation claim, so this is where its attempt is counted. Inside the
		// lock, it re-checks that the tie is still unplayed, has attempts left and is past its
		// backoff - the replica that waited for the lock may be acting on a stale selection.
		claimed, err := common.ClaimCupTieAttempt(dbCtx, tie.TieId)
		if err != nil || !claimed {
			return err
		}
		simulateErr = simulateCupTie(dbCtx, tie)
		return nil
	})
	// A failure to take or release the lock or to count the attempt is not a failed simulation: the
	// tie is simply tried again next tick, with no attempt used.
	if err != nil {
		log.Errorf("error claiming cup tie %d for simulation: %s", tie.TieId, err)
	}
	return simulateErr
}

// persistCupTieLineupSnapshot is persistResolvedLineupSnapshot's cup-tie equivalent, writing into
// cup_tie_lineup/cup_tie_lineup_positions (via common.SetCupTieLineup) instead of the league-match
// tables, since a tie's id space is its own (see SetCupTieLineup's doc comment).
func persistCupTieLineupSnapshot(dbCtx *common.DatabaseContext, tieId int64, teamId int64, lineup *common.Lineup, substitutions []common.LineupSubstitution) {
	if lineup == nil || len(lineup.Positions) == 0 {
		return
	}
	if err := common.SetCupTieLineup(dbCtx, tieId, teamId, lineup.Formation, lineup.Positions, substitutions); err != nil {
		log.Errorf("error persisting resolved lineup snapshot for cup tie %d team %d: %s", tieId, teamId, err)
	}
}

func simulateCupTie(dbCtx *common.DatabaseContext, tie *common.CupTie) error {
	lineups, err := common.GetLineupsForCupTie(dbCtx, tie.TieId, []int64{*tie.HomeTeamId, *tie.AwayTeamId})
	if err != nil {
		return err
	}
	// A team's own per-tie lineup override can schedule its own substitutions (see
	// common.SetCupTieLineup/GetSubstitutionsForCupTie); a team without one still draws from its
	// team-wide default, same as it always has.
	homeSubstitutions, err := common.GetSubstitutionsForCupTie(dbCtx, tie.TieId, *tie.HomeTeamId)
	if err != nil {
		return err
	}
	awaySubstitutions, err := common.GetSubstitutionsForCupTie(dbCtx, tie.TieId, *tie.AwayTeamId)
	if err != nil {
		return err
	}
	// Freeze today's resolution as this tie's own permanent override - see
	// persistResolvedLineupSnapshot's doc comment for why GetLineupsForCupTie's fallback alone
	// isn't enough to keep a played tie's lineup showing correctly forever.
	persistCupTieLineupSnapshot(dbCtx, tie.TieId, *tie.HomeTeamId, (*lineups)[*tie.HomeTeamId], homeSubstitutions)
	persistCupTieLineupSnapshot(dbCtx, tie.TieId, *tie.AwayTeamId, (*lineups)[*tie.AwayTeamId], awaySubstitutions)
	matchSimulation, err := runMatchSimulation(dbCtx, tie.TieId, 0, *tie.HomeTeamId, *tie.AwayTeamId, lineups, homeSubstitutions, awaySubstitutions, false)
	if err != nil {
		return err
	}

	winnerTeamId := *tie.HomeTeamId
	wentToPenalties := false
	switch {
	case matchSimulation.AwayGoals > matchSimulation.HomeGoals:
		winnerTeamId = *tie.AwayTeamId
	case matchSimulation.AwayGoals == matchSimulation.HomeGoals:
		// A knockout tie needs a winner - resolved by a penalty shootout, modeled here as a
		// straight coin flip rather than another simulated contest.
		wentToPenalties = true
		if rand.Float32() < 0.5 {
			winnerTeamId = *tie.AwayTeamId
		}
	}

	// The result, events, instants and pending post-match effects are stored in one transaction that
	// only applies while the tie is still unplayed - see simulateMatch.
	if err := common.SaveCupTieResultTx(dbCtx, tie.TieId, wentToPenalties, winnerTeamId, common.MatchResult{
		HomeGoals: matchSimulation.HomeGoals,
		AwayGoals: matchSimulation.AwayGoals,
		Events:    *matchSimulation.Events,
		Instants:  *matchSimulation.MatchInstants,
		Effects:   newPostMatchPayload(matchSimulation, wentToPenalties),
	}); err != nil {
		return err
	}

	// Drawn once, so the same weather both shapes the crowd and is what the tie's report shows -
	// see simulateMatch, which does the same for a league match.
	weather := common.RandomWeather()
	attendance := UpdateTeamFinancesAfterMatch(dbCtx, matchSimulation, tie.Round, weather)
	updateMatchReportFlavor(dbCtx, "cup tie", tie.TieId, attendance, weather,
		func(weather string, attendance int, referee string, assistant1 string, assistant2 string) error {
			return common.UpdateCupTieFlavor(dbCtx, tie.TieId, weather, attendance, referee, assistant1, assistant2)
		})
	awardCupRoundPrize(dbCtx, winnerTeamId, tie.Round)
	// UpdatePlayerAvailability and updateSkillsAfterPlayedMatch are deferred to onComplete, same
	// as simulateMatch - see its own comment. A tie is simulated and marked played up to 15
	// minutes before its advertised kickoff, so running these here would leak an injury or skill
	// jump before the tie has even nominally started.
	BroadcastMatchLive(dbCtx, common.CupTieChannel(matchSimulation.Id), matchSimulation, *tie.MatchDate, func() {
		completePostMatchEffects(dbCtx, common.SimulationKindCup, matchSimulation.Id)
	})
	common.RecordMatchSimulated(context.Background(), "cup", matchSimulation.HomeGoals, matchSimulation.AwayGoals)
	return nil
}

func notifyCupTieResult(dbCtx *common.DatabaseContext, match *common.MatchSimulation, wentToPenalties bool) {
	homeTeam, errHome := common.GetTeamById(match.Home, dbCtx)
	awayTeam, errAway := common.GetTeamById(match.Away, dbCtx)
	if errHome != nil || errAway != nil {
		log.Errorf("error getting team names for cup tie notification: %v / %v", errHome, errAway)
		return
	}
	message := fmt.Sprintf("Cup: %s %d - %d %s", homeTeam.TeamName, match.HomeGoals, match.AwayGoals, awayTeam.TeamName)
	key := "notifications.messages.cupTieResult"
	if wentToPenalties {
		message += " (penalties)"
		key = "notifications.messages.cupTieResultPenalties"
	}
	msg := common.NotificationMessage{
		Text: message,
		Key:  key,
		Params: map[string]any{
			"home": homeTeam.TeamName, "homeGoals": match.HomeGoals,
			"away": awayTeam.TeamName, "awayGoals": match.AwayGoals,
		},
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_CUP, msg); err != nil {
		log.Errorf("error notifying home team owner of cup tie result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_CUP, msg); err != nil {
		log.Errorf("error notifying away team owner of cup tie result: %s", err)
	}
}
