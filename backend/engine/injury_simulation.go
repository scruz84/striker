package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"time"
)

const ProbInjuryIncident = 0.005
const MinInjuryMatches = 1
const MaxInjuryMatches = 5

// InjurySimulation gives each minute a small chance of an injury, independent of cards. Like
// DisciplineSimulation, it excludes goalkeepers (no substitute keeper is ever tracked) and
// removes the injured player from their team's active position groups, on top of recording a
// persistent unavailability for their team's next matches (see engine.UpdatePlayedMatch).
// Unlike a card, an injury isn't necessarily the end of that shirt's involvement: if the team
// still has one of its common.MaxScheduledSubstitutions left this match, the best available
// bench player comes on in the injured player's place (see bringOnInjuryReplacement) - the same
// real-football distinction between a forced substitution and simply playing a man down.
type InjurySimulation struct {
	r *rand.Rand
}

func NewInjurySimulation() *InjurySimulation {
	return &InjurySimulation{r: rand.New(rand.NewSource(time.Now().UnixMilli()))}
}

func (s InjurySimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if s.r.Float32() >= ProbInjuryIncident {
		return
	}

	isHome := s.r.Float32() < 0.5
	side := sideOf(match, isHome)

	player, position := pickOutfieldPlayer(s.r, side.playersByPosition)
	if player == -1 {
		// no eligible outfield player left on this team
		return
	}

	removePlayerFromPosition(side.playersByPosition, position, player)
	matches := MinInjuryMatches + s.r.Intn(MaxInjuryMatches-MinInjuryMatches+1)
	match.PendingUnavailabilities = append(match.PendingUnavailabilities, common.PendingUnavailability{
		PlayerId: player,
		Reason:   common.UNAVAILABLE_INJURY,
		Matches:  matches,
	})

	playerName, _ := GetPlayerFullName(dbCtx, player)
	*match.Events = append(*match.Events, common.Event{
		EventType:             common.INJURY,
		Minute:                match.Minute,
		TeamOwner:             side.teamId,
		TeamOwnerPlayers:      []int64{player},
		TeamOwnerPlayersNames: []string{playerName},
		OtherTeam:             side.otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})

	bringOnInjuryReplacement(match, side, position, player)
}

// bringOnInjuryReplacement brings the best eligible bench player on for someone just injured, in
// their vacated position - but only if the team hasn't already used every one of its
// common.MaxScheduledSubstitutions this match (between manager-scheduled swaps and earlier
// injury replacements - see engine.SubstitutionSimulation). With none left, the injured player
// simply isn't replaced and the team plays on a man short, same as before this existed. Like any
// substitution, it marks the injured player substituted off, and the replacement is picked by
// bestEligibleBenchPlayer, so nobody already substituted off comes back.
func bringOnInjuryReplacement(match *common.MatchSimulation, side teamSide, position int, injuredPlayer int64) {
	if *side.substitutionsUsed >= common.MaxScheduledSubstitutions {
		return
	}

	replacement, found := bestEligibleBenchPlayer(match, side, position, injuredPlayer)
	if !found {
		// squad has nobody left eligible to come on
		return
	}

	(*side.playersByPosition)[position] = append((*side.playersByPosition)[position], replacement)
	side.substitutedOff[injuredPlayer] = true
	*side.substitutionsUsed++

	injuredName := (*side.players)[injuredPlayer].FirstName + " " + (*side.players)[injuredPlayer].LastName
	replacementName := (*side.players)[replacement].FirstName + " " + (*side.players)[replacement].LastName
	*match.Events = append(*match.Events, common.Event{
		EventType:             common.SUBSTITUTION,
		Minute:                match.Minute,
		TeamOwner:             side.teamId,
		TeamOwnerPlayers:      []int64{injuredPlayer, replacement},
		TeamOwnerPlayersNames: []string{injuredName, replacementName},
		OtherTeam:             side.otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
}
