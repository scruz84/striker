package engine

import (
	"footmanager/fmcommon"
	"math/rand"
	"time"
)

const ProbChance = 0.14
const ProbCenterChance = 0.50
const ProbLateralChance = 0.25

// ScorerSimulation calculates initial weights based on players.
type ScorerSimulation struct {
	r *rand.Rand
}

func NewScorerSimulation() *ScorerSimulation {
	return &ScorerSimulation{r: rand.New(rand.NewSource(time.Now().UnixMilli()))}
}

func (s ScorerSimulation) Apply(dbCtx *fmcommon.DatabaseContext, match *fmcommon.MatchSimulation) {
	if rand.Float32() < ProbChance {
		teamChance := getTeamChance(s, match)
		processChance(s, dbCtx, match, teamChance)
	}
}

func processChance(s ScorerSimulation, dbCtx *fmcommon.DatabaseContext, match *fmcommon.MatchSimulation, teamChance int64) {
	isRight := false
	isLeft := false
	//decide if is right, left or center chance
	randValue := rand.Float32()
	if randValue > ProbCenterChance {
		if randValue < (ProbLateralChance + ProbCenterChance) {
			isRight = true
		} else {
			isLeft = true
		}
	}
	var otherTeam int64
	//get the attack and defense values
	var attack float32
	var defense float32
	var goalKeeperValue int
	var attackPlayers *map[int][]int64
	var players *map[int64]fmcommon.Player
	if teamChance == match.Home {
		players = match.HomePlayers
		attackPlayers = match.HomePlayersByPosition
		otherTeam = match.Away
		goalKeeperValue = (*match.AwayPlayers)[(*match.AwayPlayersByPosition)[fmcommon.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.HomeQua.RightAttach
			defense = match.AwayQua.LeftDefense
		} else if isLeft {
			attack = match.HomeQua.LeftAttack
			defense = match.AwayQua.RightDefense
		} else {
			attack = match.HomeQua.Attack
			defense = match.AwayQua.Defense
		}
	} else {
		players = match.AwayPlayers
		attackPlayers = match.AwayPlayersByPosition
		otherTeam = match.Home
		goalKeeperValue = (*match.HomePlayers)[(*match.HomePlayersByPosition)[fmcommon.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.AwayQua.RightAttach
			defense = match.HomeQua.LeftDefense
		} else if isLeft {
			attack = match.AwayQua.LeftAttack
			defense = match.HomeQua.RightDefense
		} else {
			attack = match.AwayQua.Attack
			defense = match.HomeQua.Defense
		}
	}
	//probability of success chance
	attackPlayer := getPlayerForChance(s.r, attackPlayers)
	attackPlayerValue := getPlayerScorerValue(isLeft || isRight, (*players)[attackPlayer])
	probSuccess := (attack + attackPlayerValue) / (defense + float32(goalKeeperValue) + attack + attackPlayerValue)
	//calculates if the chance was a success!
	successfullyChance := s.r.Float32() < probSuccess

	var eventType int
	if successfullyChance {
		eventType = fmcommon.GOAL
		if teamChance == match.Home {
			match.HomeGoals++
		} else {
			match.AwayGoals++
		}
	} else {
		eventType = fmcommon.NO_GOAL
	}

	attackPlayerName, _ := GetPlayerFullName(dbCtx, attackPlayer)

	*match.Events = append(*match.Events, fmcommon.Event{
		EventType:             eventType,
		Minute:                match.Minute,
		TeamOwner:             teamChance,
		TeamOwnerPlayers:      []int64{attackPlayer},
		TeamOwnerPlayersNames: []string{attackPlayerName},
		OtherTeam:             otherTeam,
		OtherTeamPlayers:      nil,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
}

func getTeamChance(s ScorerSimulation, match *fmcommon.MatchSimulation) int64 {
	homeMidfieldAverage := match.HomeQua.Midfield / (match.HomeQua.Midfield + match.AwayQua.Midfield)
	if s.r.Float32() < homeMidfieldAverage {
		return match.Home
	}
	return match.Away
}

func getPlayerForChance(r *rand.Rand, playersByPosition *map[int][]int64) int64 {
	//70% striker
	//20% midfield
	//10% defense
	v := r.Float32()
	var playersToSelect []int64
	if v <= 70 {
		playersToSelect = (*playersByPosition)[fmcommon.STRIKER]
	} else if v > 70 && v <= 90 {
		playersToSelect = (*playersByPosition)[fmcommon.MIDFIELD]
	} else {
		playersToSelect = (*playersByPosition)[fmcommon.DEFENSE]
	}
	//Next two if for cases when there are no more players on the positions
	if len(playersToSelect) == 0 {
		playersToSelect = (*playersByPosition)[fmcommon.STRIKER]
	}
	if len(playersToSelect) == 0 {
		playersToSelect = (*playersByPosition)[fmcommon.MIDFIELD]
	}
	return playersToSelect[r.Int()%len(playersToSelect)]
}

func getPlayerScorerValue(isLateral bool, player fmcommon.Player) float32 {
	if isLateral {
		return float32(player.Shot)*0.5 + float32(player.Lateral)*0.2 + float32(player.Pass)*0.15 + float32(player.Dribbling)*0.15
	}
	return float32(player.Shot)*0.7 + float32(player.Pass)*0.15 + float32(player.Dribbling)*0.15
}
