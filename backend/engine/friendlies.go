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
// UpdateTeamFinancesAfterMatch and any per-match AI-team management hook (a friendly result
// itself is not something an AI manager needs to react to - see RunAiTeamDailyActivity in
// aiteams.go for the separate, match-independent scheduler that now covers CPU market upkeep
// instead). It still credits gate receipts, split evenly between both sides (see
// creditFriendlyTicketRevenue) - unlike a competitive match's ticket revenue (home side only, see
// creditTicketRevenue), which this deliberately does not reuse - but there's still no
// sponsorship income, since that's tied to goals scored in a result that actually matters (see
// creditSponsorship), which a friendly's never is.
// Injuries still apply via UpdatePlayerAvailability, same as any other simulated match, and so
// does skill growth (updateSkillsAfterPlayedMatch) - a friendly still develops the players who
// feature in it, even with no competitive stakes attached. Red cards, however, carry no
// suspension at all when picked up in a friendly (see DisciplineSimulation.Apply) - a friendly
// has nothing competitive to sanction.
// persistFriendlyLineupSnapshot is persistResolvedLineupSnapshot's friendly equivalent, writing
// into friendly_lineup/friendly_lineup_positions (via common.SetFriendlyLineup) instead of the
// league-match tables, since a friendly's id space is kept deliberately separate (see
// GetLineupsForFriendly's doc comment).
func persistFriendlyLineupSnapshot(dbCtx *common.DatabaseContext, matchId int64, teamId int64, lineup *common.Lineup, substitutions []common.LineupSubstitution) {
	if lineup == nil || len(lineup.Positions) == 0 {
		return
	}
	if err := common.SetFriendlyLineup(dbCtx, matchId, teamId, lineup.Formation, lineup.Positions, substitutions); err != nil {
		log.Errorf("error persisting resolved lineup snapshot for friendly %d team %d: %s", matchId, teamId, err)
	}
}

func simulateFriendly(dbCtx *common.DatabaseContext, friendly *common.FriendlyMatch) error {
	lineups, err := common.GetLineupsForFriendly(dbCtx, friendly.MatchId, []int64{friendly.HomeTeamId, friendly.AwayTeamId})
	if err != nil {
		return err
	}
	// A friendly's scheduled substitutions are saved specifically for that match (see
	// common.SetFriendlyLineup/GetFriendlyLineupSubstitutions), not the team's standing default
	// lineup subs - see runMatchSimulation's own doc comment for why.
	homeSubstitutions, err := common.GetFriendlyLineupSubstitutions(dbCtx, friendly.MatchId, friendly.HomeTeamId)
	if err != nil {
		return err
	}
	awaySubstitutions, err := common.GetFriendlyLineupSubstitutions(dbCtx, friendly.MatchId, friendly.AwayTeamId)
	if err != nil {
		return err
	}
	// Freeze today's resolution as this friendly's own permanent override - see
	// persistResolvedLineupSnapshot's doc comment for why GetLineupsForFriendly's fallback alone
	// isn't enough to keep a played friendly's lineup showing correctly forever. A no-op for a
	// national-team friendly, which already has an explicit override saved at call-up time (see
	// common.SetNationalTeamLineup).
	persistFriendlyLineupSnapshot(dbCtx, friendly.MatchId, friendly.HomeTeamId, (*lineups)[friendly.HomeTeamId], homeSubstitutions)
	persistFriendlyLineupSnapshot(dbCtx, friendly.MatchId, friendly.AwayTeamId, (*lineups)[friendly.AwayTeamId], awaySubstitutions)
	matchSimulation, err := runMatchSimulation(dbCtx, friendly.MatchId, 0, friendly.HomeTeamId, friendly.AwayTeamId, lineups, homeSubstitutions, awaySubstitutions, true)
	if err != nil {
		return err
	}

	if err := common.SetFriendlyPlayed(dbCtx, friendly.MatchId, matchSimulation.HomeGoals,
		matchSimulation.AwayGoals, *matchSimulation.Events, *matchSimulation.MatchInstants); err != nil {
		return err
	}

	// A no-op for the overwhelming majority of friendlies, which aren't a World Cup group fixture
	// at all - see UpdateWorldCupStandingsAfterMatch.
	if err := common.UpdateWorldCupStandingsAfterMatch(dbCtx, friendly.MatchId, friendly.HomeTeamId,
		friendly.AwayTeamId, matchSimulation.HomeGoals, matchSimulation.AwayGoals); err != nil {
		log.Errorf("error updating World Cup standings for friendly %d: %s", friendly.MatchId, err)
	}

	// Credited immediately rather than deferred to onComplete below, same as
	// UpdateTeamFinancesAfterMatch for a league match/cup tie (see simulateMatch) - a budget
	// change isn't the kind of visible-outside-this-page state (an injury, a skill jump, a
	// notification) that needs to wait for the friendly's advertised kickoff to actually arrive.
	weather := common.RandomWeather()
	attendance := creditFriendlyTicketRevenue(dbCtx, friendly.HomeTeamId, friendly.AwayTeamId, weather)
	updateMatchReportFlavor(dbCtx, "friendly", friendly.MatchId, attendance, weather,
		func(weather string, attendance int, referee string, assistant1 string, assistant2 string) error {
			return common.UpdateFriendlyFlavor(dbCtx, friendly.MatchId, weather, attendance, referee, assistant1, assistant2)
		})

	// UpdatePlayerAvailability and updateSkillsAfterPlayedMatch are deferred to onComplete, same
	// as simulateMatch - see its own comment. A friendly is simulated and marked played up to 15
	// minutes before its advertised kickoff, so running these here would leak an injury or skill
	// jump before the friendly has even nominally started.
	BroadcastMatchLive(dbCtx, common.FriendlyMatchChannel(matchSimulation.Id), matchSimulation, friendly.MatchDate, func() {
		UpdatePlayerAvailability(dbCtx, matchSimulation)
		updateSkillsAfterPlayedMatch(dbCtx, matchSimulation)
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
