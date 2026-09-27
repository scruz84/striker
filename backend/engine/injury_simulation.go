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
	teamId := match.Away
	otherTeam := match.Home
	playersByPosition := match.AwayPlayersByPosition
	players := match.AwayPlayers
	unavailable := match.AwayUnavailablePlayers
	substitutionsUsed := &match.AwaySubstitutionsUsed
	if isHome {
		teamId = match.Home
		otherTeam = match.Away
		playersByPosition = match.HomePlayersByPosition
		players = match.HomePlayers
		unavailable = match.HomeUnavailablePlayers
		substitutionsUsed = &match.HomeSubstitutionsUsed
	}

	player, position := pickOutfieldPlayer(s.r, playersByPosition)
	if player == -1 {
		// no eligible outfield player left on this team
		return
	}

	removePlayerFromPosition(playersByPosition, position, player)
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
		TeamOwner:             teamId,
		TeamOwnerPlayers:      []int64{player},
		TeamOwnerPlayersNames: []string{playerName},
		OtherTeam:             otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})

	bringOnInjuryReplacement(match, playersByPosition, players, unavailable, substitutionsUsed,
		position, player, teamId, otherTeam)
}

// bringOnInjuryReplacement brings the best eligible bench player on for someone just injured, in
// their vacated position - but only if the team hasn't already used every one of its
// common.MaxScheduledSubstitutions this match (between manager-scheduled swaps and earlier
// injury replacements - see engine.SubstitutionSimulation). With none left, the injured player
// simply isn't replaced and the team plays on a man short, same as before this existed.
func bringOnInjuryReplacement(match *common.MatchSimulation, playersByPosition *map[int][]int64,
	players *map[int64]common.Player, unavailable map[int64]bool, substitutionsUsed *int, position int,
	injuredPlayer int64, teamId int64, otherTeam int64) {
	if *substitutionsUsed >= common.MaxScheduledSubstitutions {
		return
	}

	replacement, found := pickInjuryReplacement(match, playersByPosition, players, unavailable, position, injuredPlayer)
	if !found {
		// squad has nobody left eligible to come on
		return
	}

	(*playersByPosition)[position] = append((*playersByPosition)[position], replacement)
	*substitutionsUsed++

	injuredName := (*players)[injuredPlayer].FirstName + " " + (*players)[injuredPlayer].LastName
	replacementName := (*players)[replacement].FirstName + " " + (*players)[replacement].LastName
	*match.Events = append(*match.Events, common.Event{
		EventType:             common.SUBSTITUTION,
		Minute:                match.Minute,
		TeamOwner:             teamId,
		TeamOwnerPlayers:      []int64{injuredPlayer, replacement},
		TeamOwnerPlayersNames: []string{injuredName, replacementName},
		OtherTeam:             otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
}

// pickInjuryReplacement finds the best squad player, by common.PositionValue at the vacated
// position, who is neither currently on the pitch (any position, either team's players map is
// scoped to their own squad so no cross-team collision) nor already unavailable for this match -
// the just-injured player themselves, a pre-existing suspension/injury (unavailable), sent off
// earlier this match, or already sidelined by an earlier injury this same match
// (match.PendingUnavailabilities). Mirrors engine.backfillEmptyPositions' selection, just picking
// one player instead of filling a count.
func pickInjuryReplacement(match *common.MatchSimulation, playersByPosition *map[int][]int64,
	players *map[int64]common.Player, unavailable map[int64]bool, position int, injuredPlayer int64) (int64, bool) {
	ineligible := make(map[int64]bool, len(unavailable))
	ineligible[injuredPlayer] = true
	for id := range unavailable {
		ineligible[id] = true
	}
	for id := range match.SentOff {
		ineligible[id] = true
	}
	for _, pending := range match.PendingUnavailabilities {
		ineligible[pending.PlayerId] = true
	}
	for _, ids := range *playersByPosition {
		for _, id := range ids {
			ineligible[id] = true
		}
	}

	bestId, bestValue := int64(-1), -1
	for id, p := range *players {
		if ineligible[id] {
			continue
		}
		if v := common.PositionValue(position, p); v > bestValue {
			bestId, bestValue = id, v
		}
	}
	return bestId, bestId != -1
}
