package engine

import (
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

// createRoundTies persists one round's pairings as cup_ties. A pairing with a bye (one side is
// ByeTeamId) is recorded as already resolved, with no match_date - see common.CreateCupTie.
func createRoundTies(dbCtx *common.DatabaseContext, cupId int64, round int, pairs [][2]int64) error {
	matchDate := time.Now().Add(15 * time.Minute)
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
		if err := advanceCup(dbCtx, cup.CupId); err != nil {
			log.Errorf("error advancing cup %d: %s", cup.CupId, err)
		}
	}
}

func advanceCup(dbCtx *common.DatabaseContext, cupId int64) error {
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
		return common.SetCupCompleted(dbCtx, cupId, winners[0])
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
	matchSimulation, err := runMatchSimulation(dbCtx, tie.TieId, 0, *tie.HomeTeamId, *tie.AwayTeamId)
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
	return nil
}
