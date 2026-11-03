package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"time"
)

const ProbCardIncident = 0.03

// The per-minute card probability of a match between rivals (see openspec match-simulation
// "Discipline").
const ProbCardIncidentRival = 0.036
const ProbCardIncidentClasico = 0.042

// cardIncidentProbability is the per-minute chance of a card at rivalryLevel.
func cardIncidentProbability(rivalryLevel int) float32 {
	switch rivalryLevel {
	case common.RIVALRY_CLASICO:
		return ProbCardIncidentClasico
	case common.RIVALRY_RIVAL:
		return ProbCardIncidentRival
	default:
		return ProbCardIncident
	}
}

const ProbStraightRed = 0.06
const StraightRedSuspensionMatches = 2
const SecondYellowSuspensionMatches = 1

// DisciplineSimulation generates yellow/red card events. A red card - straight, or a second
// yellow - sends the player off: they're removed from their team's active position groups,
// so every subsequent minute's quality calculation (see InitialSimulation) reflects the team
// playing a man down for the rest of the match. Goalkeepers are excluded from the card pool -
// this codebase has no substitution model, so sending one off with no replacement would leave
// GOALKEEPER position lookups (e.g. in ScorerSimulation) with nothing to index.
type DisciplineSimulation struct {
	r *rand.Rand
}

func NewDisciplineSimulation() *DisciplineSimulation {
	return &DisciplineSimulation{r: rand.New(rand.NewSource(time.Now().UnixMilli()))}
}

func (s DisciplineSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if s.r.Float32() >= cardIncidentProbability(match.RivalryLevel) {
		return
	}

	isHome := s.r.Float32() < 0.5
	teamId := match.Away
	otherTeam := match.Home
	playersByPosition := match.AwayPlayersByPosition
	if isHome {
		teamId = match.Home
		otherTeam = match.Away
		playersByPosition = match.HomePlayersByPosition
	}

	player, position := pickOutfieldPlayer(s.r, playersByPosition)
	if player == -1 {
		// no eligible outfield player left on this team
		return
	}
	if captainCancelsCard(s.r, sideRoles(match, teamId), playersByPosition, player) {
		return
	}

	eventType := common.YELLOW_CARD
	wasSecondYellow := match.YellowCards[player]
	isStraightRed := !wasSecondYellow && s.r.Float32() < ProbStraightRed
	if wasSecondYellow || isStraightRed {
		eventType = common.RED_CARD
		match.SentOff[player] = true
		removePlayerFromPosition(playersByPosition, position, player)

		// A red card in a friendly carries no suspension at all - not into other friendlies
		// (already excluded by GetUnavailablePlayerIdsForFriendly) and not into league/cup
		// matches either, since a friendly has no competitive stakes to sanction.
		if !match.IsFriendly {
			suspensionMatches := SecondYellowSuspensionMatches
			if isStraightRed {
				suspensionMatches = StraightRedSuspensionMatches
			}
			match.PendingUnavailabilities = append(match.PendingUnavailabilities, common.PendingUnavailability{
				PlayerId: player,
				Reason:   common.UNAVAILABLE_SUSPENSION,
				Matches:  suspensionMatches,
			})
		}
	} else {
		match.YellowCards[player] = true
	}

	playerName, _ := GetPlayerFullName(dbCtx, player)

	*match.Events = append(*match.Events, common.Event{
		EventType:             eventType,
		Minute:                match.Minute,
		TeamOwner:             teamId,
		TeamOwnerPlayers:      []int64{player},
		TeamOwnerPlayersNames: []string{playerName},
		OtherTeam:             otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
}

// pickOutfieldPlayer picks a random player from any non-goalkeeper position group that still
// has someone in it, and returns their id and the position group they were picked from. It
// returns (-1, 0) if no eligible player is left.
func pickOutfieldPlayer(r *rand.Rand, playersByPosition *map[int][]int64) (int64, int) {
	eligiblePositions := make([]int, 0, len(common.ALL_OUTFIELD_POSITIONS))
	for _, position := range common.ALL_OUTFIELD_POSITIONS {
		if len((*playersByPosition)[position]) > 0 {
			eligiblePositions = append(eligiblePositions, position)
		}
	}
	if len(eligiblePositions) == 0 {
		return -1, 0
	}

	position := eligiblePositions[r.Intn(len(eligiblePositions))]
	players := (*playersByPosition)[position]
	return players[r.Intn(len(players))], position
}

func removePlayerFromPosition(playersByPosition *map[int][]int64, position int, playerId int64) {
	players := (*playersByPosition)[position]
	for i, id := range players {
		if id == playerId {
			(*playersByPosition)[position] = append(players[:i], players[i+1:]...)
			return
		}
	}
}

// captainCancelsCard reports whether the side's captain steadies a team-mate about to be booked
// (see openspec match-simulation "Captain's calming effect"): only while the captain is on the
// pitch (goalkeeper included), never for the captain himself, with probability
// common.CaptainCardCancelProbability. A side without a captain on the pitch draws nothing, so its
// random stream - and its cards - are exactly as before.
func captainCancelsCard(r *rand.Rand, roles common.LineupRoles, playersByPosition *map[int][]int64, player int64) bool {
	if roles.CaptainId == nil || *roles.CaptainId == player || !isOnPitch(playersByPosition, *roles.CaptainId) {
		return false
	}
	return r.Float64() < common.CaptainCardCancelProbability
}

// isOnPitch reports whether playerId currently holds any position slot, goalkeeper included.
func isOnPitch(playersByPosition *map[int][]int64, playerId int64) bool {
	for _, ids := range *playersByPosition {
		for _, id := range ids {
			if id == playerId {
				return true
			}
		}
	}
	return false
}
