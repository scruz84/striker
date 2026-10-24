package engine

import (
	"math/rand"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// DefaultResultsResult reports how many fixtures of each kind SettleDefaultResults settled 0-0.
type DefaultResultsResult struct {
	LeagueMatches int
	Friendlies    int
	CupTies       int
}

// ReleaseRedatedFixtures gives back their simulation attempts to the fixtures that used them all
// but were then re-dated to a kickoff still to come (see common.ReleaseRedatedFixtures), so they
// are simulated normally at their new kickoff instead of sitting stuck until then. Callers run it
// before SettleDefaultResults.
func ReleaseRedatedFixtures(dbCtx *common.DatabaseContext) common.FixtureKindCounts {
	released, err := common.ReleaseRedatedFixtures(dbCtx)
	if err != nil {
		log.Errorf("error releasing re-dated fixtures that used all their simulation attempts: %s", err)
	}
	if released.LeagueMatches+released.Friendlies+released.CupTies > 0 {
		log.Infof("Released re-dated fixtures for a new simulation at their kickoff: %d league matches, %d friendlies, %d cup ties",
			released.LeagueMatches, released.Friendlies, released.CupTies)
	}
	return released
}

// SettleDefaultResults settles, as a 0-0 default result, every league match, friendly and cup tie
// that has used its common.MaxSimulationAttempts attempts without being played and whose kickoff
// has passed - so one fixture that fails for good can no longer block its country's season
// rollover, or every national-team fixture, forever. What a default result books is in
// settleLeagueMatchByDefault, settleFriendlyByDefault and settleCupTieByDefault.
//
// Safe with several --engine replicas running it at once: a league match or friendly is claimed by
// one statement (see common.ClaimLeagueMatchesForDefault), a cup tie by the tie's simulation lock,
// and the result save only applies to an unplayed fixture, so each fixture is settled once.
func SettleDefaultResults(dbCtx *common.DatabaseContext) DefaultResultsResult {
	var result DefaultResultsResult

	matches, err := common.ClaimLeagueMatchesForDefault(dbCtx)
	if err != nil {
		log.Errorf("error claiming league matches for a default result: %s", err)
	}
	for _, m := range matches {
		m := m
		runClaimedSimulation(dbCtx, common.SimulationKindLeague, m.MatchId, func() error {
			if err := settleLeagueMatchByDefault(dbCtx, &m); err != nil {
				return err
			}
			result.LeagueMatches++
			return nil
		})
	}

	friendlies, err := common.ClaimFriendliesForDefault(dbCtx)
	if err != nil {
		log.Errorf("error claiming friendlies for a default result: %s", err)
	}
	for _, f := range friendlies {
		f := f
		runClaimedSimulation(dbCtx, common.SimulationKindFriendly, f.MatchId, func() error {
			if err := settleFriendlyByDefault(dbCtx, &f); err != nil {
				return err
			}
			result.Friendlies++
			return nil
		})
	}

	ties, err := common.GetCupTiesForDefault(dbCtx)
	if err != nil {
		log.Errorf("error listing cup ties for a default result: %s", err)
	}
	for _, tie := range ties {
		tie := tie
		settled := false
		runClaimedSimulation(dbCtx, common.SimulationKindCup, tie.TieId, func() error {
			var err error
			settled, err = settleCupTieByDefault(dbCtx, &tie)
			return err
		})
		if settled {
			result.CupTies++
		}
	}

	if result.LeagueMatches+result.Friendlies+result.CupTies > 0 {
		log.Infof("Settled by default 0-0: %d league matches, %d friendlies, %d cup ties", result.LeagueMatches, result.Friendlies, result.CupTies)
	}
	return result
}

// defaultMatchResult is what every default result stores: 0-0, one MATCH_DEFAULT_RESULT event and
// no instants (nothing was simulated, so there is no replay). Its post-match effects are due at
// once - there is no live window to wait for - and skip skill growth, since nobody played.
func defaultMatchResult(home int64, away int64, wentToPenalties bool) common.MatchResult {
	events := []common.Event{{EventType: common.MATCH_DEFAULT_RESULT, Minute: 0, TeamOwner: home, OtherTeam: away}}
	return common.MatchResult{
		Events: events,
		Effects: &common.PostMatchPayload{
			Home:            home,
			Away:            away,
			WentToPenalties: wentToPenalties,
			SkipSkillGrowth: true,
			Events:          events,
		},
		EffectsDueNow: true,
	}
}

// publishDefaultComplete closes the fixture's live channel with its 0-0, so a viewer or the
// dashboard settles at once instead of waiting for a replay that will never run.
func publishDefaultComplete(dbCtx *common.DatabaseContext, channel string) {
	if dbCtx.LiveHub == nil {
		return
	}
	dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{Type: "complete", HomeGoals: 0, AwayGoals: 0})
}

// settleLeagueMatchByDefault records match as a 0-0 draw (1 point each) and charges both sides
// their matchday wages. No ticket revenue, sponsorship, fanbase drift or report flavor is booked -
// nothing was played - and the pending post-match effects still count the matchday against
// suspended or injured players and send the result notification.
func settleLeagueMatchByDefault(dbCtx *common.DatabaseContext, match *common.WeekMatch) error {
	result := defaultMatchResult(match.Home, match.Away, false)
	if err := common.SaveLeagueMatchResultTx(dbCtx, match.MatchId, match.DivisionId, match.Home, match.Away, result); err != nil {
		return err
	}
	PayMatchdayWages(dbCtx, match.Home)
	PayMatchdayWages(dbCtx, match.Away)
	publishDefaultComplete(dbCtx, common.LeagueMatchChannel(match.MatchId))
	return nil
}

// settleFriendlyByDefault records friendly as a 0-0 draw (a World Cup group fixture gets its group
// draw in the same transaction). No gate receipts are split.
func settleFriendlyByDefault(dbCtx *common.DatabaseContext, friendly *common.FriendlyMatch) error {
	result := defaultMatchResult(friendly.HomeTeamId, friendly.AwayTeamId, false)
	if err := common.SaveFriendlyResultTx(dbCtx, friendly.MatchId, friendly.HomeTeamId, friendly.AwayTeamId, result); err != nil {
		return err
	}
	publishDefaultComplete(dbCtx, common.FriendlyMatchChannel(friendly.MatchId))
	return nil
}

// settleCupTieByDefault records tie as a 0-0 that goes to the usual shoot-out (a coin flip) and
// pays the winner the round prize. It runs under the tie's simulation lock - so it never overlaps a
// simulation of the tie, or another replica's settlement - and skips a tie that is no longer
// unplayed. Reports whether this call actually settled the tie.
func settleCupTieByDefault(dbCtx *common.DatabaseContext, tie *common.CupTie) (bool, error) {
	settled := false
	var settleErr error
	_, err := common.TryWithSessionAdvisoryLock(dbCtx, common.NamespacedLockKey(common.LockNamespaceCupTieSimulation, tie.TieId), func() error {
		played, err := common.IsCupTiePlayed(dbCtx, tie.TieId)
		if err != nil || played {
			return err
		}
		winner := *tie.HomeTeamId
		if rand.Float32() < 0.5 {
			winner = *tie.AwayTeamId
		}
		result := defaultMatchResult(*tie.HomeTeamId, *tie.AwayTeamId, true)
		if settleErr = common.SaveCupTieResultTx(dbCtx, tie.TieId, true, winner, result); settleErr != nil {
			return nil
		}
		awardCupRoundPrize(dbCtx, winner, tie.Round)
		publishDefaultComplete(dbCtx, common.CupTieChannel(tie.TieId))
		settled = true
		return nil
	})
	if err != nil {
		log.Errorf("error settling cup tie %d by default: %s", tie.TieId, err)
	}
	return settled, settleErr
}
