package engine

import (
	"context"
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"sync"
)

// SimulateFriendlies simulates every friendly match whose kickoff has arrived. Same
// bounded-concurrency + panic-recovery shape as SimulateCupTies - see there for why.
func SimulateFriendlies(dbCtx *common.DatabaseContext) {
	friendlies, err := common.GetSimulableFriendlies(dbCtx)
	if err != nil {
		log.Errorf("error getting simulable friendlies: %s", err)
		return
	}

	sem := make(chan struct{}, maxConcurrentSimulations)
	var wg sync.WaitGroup
	for _, friendly := range friendlies {
		log.Infof("simulating friendly %d", friendly.MatchId)
		wg.Add(1)
		sem <- struct{}{}
		go func(friendly common.FriendlyMatch) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("panic simulating friendly %d: %v", friendly.MatchId, r)
				}
			}()
			if err := simulateFriendly(dbCtx, &friendly); err != nil {
				log.Errorf("error simulating friendly %d: %s", friendly.MatchId, err)
			}
		}(friendly)
	}
	wg.Wait()
}

// simulateFriendly runs the shared match engine and persists the result, but deliberately skips
// UpdateTeamFinancesAfterMatch (no ticket revenue/sponsorship for a friendly - see
// init26_friendlies.sql) and any AI-team management hook (a friendly is not a competitive
// result an AI manager needs to react to). Injuries/suspensions still apply via
// UpdatePlayerAvailability, same as any other simulated match.
func simulateFriendly(dbCtx *common.DatabaseContext, friendly *common.FriendlyMatch) error {
	lineups, err := common.GetLineupsForFriendly(dbCtx, friendly.MatchId, []int64{friendly.HomeTeamId, friendly.AwayTeamId})
	if err != nil {
		return err
	}
	matchSimulation, err := runMatchSimulation(dbCtx, friendly.MatchId, 0, friendly.HomeTeamId, friendly.AwayTeamId, lineups)
	if err != nil {
		return err
	}

	if err := common.SetFriendlyPlayed(dbCtx, friendly.MatchId, matchSimulation.HomeGoals,
		matchSimulation.AwayGoals, *matchSimulation.Events); err != nil {
		return err
	}

	UpdatePlayerAvailability(dbCtx, matchSimulation)
	notifyFriendlyResult(dbCtx, matchSimulation)
	BroadcastMatchLive(dbCtx, common.FriendlyMatchChannel(matchSimulation.Id), matchSimulation)
	common.RecordMatchSimulated(context.Background(), "friendly", matchSimulation.HomeGoals, matchSimulation.AwayGoals)
	return nil
}

func notifyFriendlyResult(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	homeTeam, errHome := common.GetTeamById(match.Home, dbCtx)
	awayTeam, errAway := common.GetTeamById(match.Away, dbCtx)
	if errHome != nil || errAway != nil {
		log.Errorf("error getting team names for friendly notification: %v / %v", errHome, errAway)
		return
	}
	message := fmt.Sprintf("Friendly: %s %d - %d %s", homeTeam.TeamName, match.HomeGoals, match.AwayGoals, awayTeam.TeamName)
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_FRIENDLY, message); err != nil {
		log.Errorf("error notifying home team owner of friendly result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_FRIENDLY, message); err != nil {
		log.Errorf("error notifying away team owner of friendly result: %s", err)
	}
}
