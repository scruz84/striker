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

// StartCupsForNewSeason creates a fresh single-elimination cup for every country, entrants
// drawn from that country's tier-1 teams, and generates its first round. Called once per
// season, alongside league fixtures - see AdvanceSeasonIfComplete.
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
	pairs := BuildFirstRoundPairings(teamIds, r)
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
// ByeTeamId) is recorded as already resolved, with no match_date - see common.CreateCupTie.
func createRoundTies(dbCtx *common.DatabaseContext, cupId int64, round int, pairs [][2]int64) error {
	matchDate := time.Now().Add(cupRoundGap)
	for i, pair := range pairs {
		home, away := asTeamIdPtr(pair[0]), asTeamIdPtr(pair[1])
		var scheduledAt *time.Time
		if home != nil && away != nil {
			scheduledAt = &matchDate
		}
		if err := common.CreateCupTie(dbCtx, cupId, round, i+1, home, away, scheduledAt); err != nil {
			return err
		}
	}
	return nil
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

func advanceCup(dbCtx *common.DatabaseContext, cupId int64, season int) error {
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
			if err := common.NotifyTeamOwner(dbCtx, winners[0], common.NOTIFICATION_CUP, team.TeamName+" won the cup!"); err != nil {
				log.Errorf("error notifying cup champion team %d: %s", winners[0], err)
			}
		}
		return nil
	}

	pairs := BuildNextRoundPairings(winners)
	return createRoundTies(dbCtx, cupId, round+1, pairs)
}

// SimulateCupTies simulates every cup tie whose match_date has arrived.
func SimulateCupTies(dbCtx *common.DatabaseContext) {
	ties, err := common.GetSimulableCupTies(dbCtx)
	if err != nil {
		log.Errorf("error getting simulable cup ties: %s", err)
		return
	}

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
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("panic simulating cup tie %d: %v", tie.TieId, r)
				}
			}()
			if err := simulateCupTie(dbCtx, &tie); err != nil {
				log.Errorf("error simulating cup tie %d: %s", tie.TieId, err)
			}
		}(tie)
	}
	wg.Wait()
}

func simulateCupTie(dbCtx *common.DatabaseContext, tie *common.CupTie) error {
	lineups, err := common.GetLineupsForCupTie(dbCtx, tie.TieId, []int64{*tie.HomeTeamId, *tie.AwayTeamId})
	if err != nil {
		return err
	}
	matchSimulation, err := runMatchSimulation(dbCtx, tie.TieId, 0, *tie.HomeTeamId, *tie.AwayTeamId, lineups)
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

	if err := common.SetCupTiePlayed(dbCtx, tie.TieId, matchSimulation.HomeGoals, matchSimulation.AwayGoals,
		wentToPenalties, winnerTeamId, *matchSimulation.Events); err != nil {
		return err
	}

	UpdatePlayerAvailability(dbCtx, matchSimulation)
	UpdateTeamFinancesAfterMatch(dbCtx, matchSimulation)
	recordPlayerMinutes(dbCtx, matchSimulation)
	awardCupRoundPrize(dbCtx, winnerTeamId, tie.Round)
	BroadcastMatchLive(dbCtx, common.CupTieChannel(matchSimulation.Id), matchSimulation, func() {
		notifyCupTieResult(dbCtx, matchSimulation, wentToPenalties)
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
	if wentToPenalties {
		message += " (penalties)"
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_CUP, message); err != nil {
		log.Errorf("error notifying home team owner of cup tie result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_CUP, message); err != nil {
		log.Errorf("error notifying away team owner of cup tie result: %s", err)
	}
}
