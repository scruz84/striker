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
	simulateFriendliesNow(dbCtx, friendlies)
}

// FinishStuckFriendlies re-simulates every friendly a simulation pass claimed
// (in_simulation = true) but never finished simulating - the case a server crash mid-run leaves
// behind, since GetSimulableFriendliesStmtSql's "and in_simulation = false" filter means the
// normal scheduler never revisits a claimed friendly on its own (see
// common.GetStuckFriendlies). Returns how many friendlies it re-simulated.
func FinishStuckFriendlies(dbCtx *common.DatabaseContext) int {
	friendlies, err := common.GetStuckFriendlies(dbCtx)
	if err != nil {
		log.Errorf("error getting stuck friendlies: %s", err)
		return 0
	}
	simulateFriendliesNow(dbCtx, friendlies)
	return len(friendlies)
}

// simulateFriendliesNow fans out over the given friendlies with the same bounded-concurrency +
// panic-recovery approach as SimulateMatchesNow (engine.go) - shared by SimulateFriendlies and
// FinishStuckFriendlies, which differ only in how they pick which friendlies to simulate.
func simulateFriendliesNow(dbCtx *common.DatabaseContext, friendlies []common.FriendlyMatch) {
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
// init26_friendlies.sql) and any per-match AI-team management hook (a friendly result itself is
// not something an AI manager needs to react to - see RunAiTeamDailyActivity in aiteams.go for the
// separate, match-independent scheduler that now covers CPU market upkeep instead).
// Injuries still apply via UpdatePlayerAvailability, same as any other simulated match, and so
// does skill growth (updateSkillsAfterPlayedMatch) - a friendly still develops the players who
// feature in it, even with no financial stakes attached. Red cards, however, carry no suspension
// at all when picked up in a friendly (see DisciplineSimulation.Apply) - a friendly has nothing
// competitive to sanction.
func simulateFriendly(dbCtx *common.DatabaseContext, friendly *common.FriendlyMatch) error {
	lineups, err := common.GetLineupsForFriendly(dbCtx, friendly.MatchId, []int64{friendly.HomeTeamId, friendly.AwayTeamId})
	if err != nil {
		return err
	}
	matchSimulation, err := runMatchSimulation(dbCtx, friendly.MatchId, 0, friendly.HomeTeamId, friendly.AwayTeamId, lineups, true)
	if err != nil {
		return err
	}

	if err := common.SetFriendlyPlayed(dbCtx, friendly.MatchId, matchSimulation.HomeGoals,
		matchSimulation.AwayGoals, *matchSimulation.Events, *matchSimulation.MatchInstants); err != nil {
		return err
	}

	UpdatePlayerAvailability(dbCtx, matchSimulation)
	updateSkillsAfterPlayedMatch(dbCtx, matchSimulation)
	BroadcastMatchLive(dbCtx, common.FriendlyMatchChannel(matchSimulation.Id), matchSimulation, friendly.MatchDate, func() {
		notifyFriendlyResult(dbCtx, matchSimulation)
	})
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
	msg := common.NotificationMessage{
		Text: fmt.Sprintf("Friendly: %s %d - %d %s", homeTeam.TeamName, match.HomeGoals, match.AwayGoals, awayTeam.TeamName),
		Key:  "notifications.messages.friendlyResult",
		Params: map[string]any{
			"home": homeTeam.TeamName, "homeGoals": match.HomeGoals,
			"away": awayTeam.TeamName, "awayGoals": match.AwayGoals,
		},
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_FRIENDLY, msg); err != nil {
		log.Errorf("error notifying home team owner of friendly result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_FRIENDLY, msg); err != nil {
		log.Errorf("error notifying away team owner of friendly result: %s", err)
	}
}
