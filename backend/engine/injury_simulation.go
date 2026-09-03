package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"time"
)

const ProbInjuryIncident = 0.015
const MinInjuryMatches = 1
const MaxInjuryMatches = 5

// InjurySimulation gives each minute a small chance of an injury, independent of cards. Like
// DisciplineSimulation, it excludes goalkeepers (no substitution model) and removes the
// injured player from their team's active position groups for the rest of the match, on top
// of recording a persistent unavailability for their team's next matches (see
// engine.UpdatePlayedMatch).
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
}
