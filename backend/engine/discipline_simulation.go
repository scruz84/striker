package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"time"
)

const ProbCardIncident = 0.03
const ProbStraightRed = 0.15

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
	if s.r.Float32() >= ProbCardIncident {
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

	eventType := common.YELLOW_CARD
	isRed := match.YellowCards[player] || s.r.Float32() < ProbStraightRed
	if isRed {
		eventType = common.RED_CARD
		match.SentOff[player] = true
		removePlayerFromPosition(playersByPosition, position, player)
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
