package engine

import "github.com/scruz84/striker/common"

type SimulationPipelineImpl struct {
	simulations []Simulation
}

func (s SimulationPipelineImpl) Simulate(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	for _, sim := range s.simulations {
		sim.Apply(dbCtx, match)
	}
}

////////////////////////////// Simulations

// InitialSimulation calculates initial weights based on players.
type InitialSimulation struct{}

func (s InitialSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	calculateQua(match.HomeQua, match.HomeLineup, match.HomePlayersByPosition, match.HomePlayers)
	calculateQua(match.AwayQua, match.AwayLineup, match.AwayPlayersByPosition, match.AwayPlayers)

	calculateDiffs(match)
	//Append a new match instant
	*match.MatchInstants = append(*match.MatchInstants, common.MatchInstant{
		MatchId:              match.Id,
		Minute:               match.Minute,
		HomeGoals:            match.HomeGoals,
		AwayGoals:            match.AwayGoals,
		HomeQua:              *match.HomeQua,
		AwayQua:              *match.AwayQua,
		MidfieldDiff:         match.MidfieldDiff,
		HomeAttackRightDiff:  match.HomeAttackRightDiff,
		HomeAttackLeftDiff:   match.HomeAttackLeftDiff,
		HomeAttackCenterDiff: match.HomeAttackCenterDiff,
		AwayAttackRightDiff:  match.AwayAttackRightDiff,
		AwayAttackLeftDiff:   match.AwayAttackLeftDiff,
		AwayAttackCenterDiff: match.AwayAttackCenterDiff,
	})
}

func calculateDiffs(match *common.MatchSimulation) {
	match.MidfieldDiff = calculateMidfieldDiff(match)
	match.HomeAttackCenterDiff = calculateHomeAttackDiff(match)
	match.AwayAttackCenterDiff = calculateAwayAttackDiff(match)
}

func calculateQua(teamQua *common.TeamQualification, lineup *common.Lineup, playersByPosition *map[int][]int64,
	players *map[int64]common.Player) {
	//defense
	teamQua.LeftDefense = calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players)
	teamQua.RightDefense = calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players)
	teamQua.Defense = calculateDefense(
		(*playersByPosition)[common.DEFENSE], (*playersByPosition)[common.MIDFIELD], players)
	//midfield
	teamQua.LeftMidfield = calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players)
	teamQua.RightMidfield = calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players)
	teamQua.Midfield = calculateMidfield(
		(*playersByPosition)[common.DEFENSE], (*playersByPosition)[common.MIDFIELD], players)
	//attack
	teamQua.LeftAttack = calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], (*playersByPosition)[common.LATERAL_STRIKER_LEFT], players)
	teamQua.RightAttach = calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], (*playersByPosition)[common.LATERAL_STRIKER_RIGHT], players)
	teamQua.Attack = calculateAttack(
		(*playersByPosition)[common.STRIKER], (*playersByPosition)[common.MIDFIELD], players)
}

func calculateLateralDefense(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32((*players)[def[0]].Lateral)*0.5 + float32((*players)[def[0]].Defense)*0.5
	return value
}

func calculateDefense(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range def {
		value += float32((*players)[d].Defense)
	}
	return value
}

func calculateLateralMidfield(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32((*players)[def[0]].Lateral)*0.5 + float32((*players)[def[0]].Midfield)*0.5
	return value
}

func calculateMidfield(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range def {
		value += float32((*players)[d].Midfield)
	}
	return value
}

func calculateLateralAttack(striker []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32((*players)[striker[0]].Lateral)*0.5 + float32((*players)[striker[0]].Shot)*0.5
	return value
}

func calculateAttack(striker []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range striker {
		value += float32((*players)[d].Shot)
	}
	return value
}

func calculateMidfieldDiff(match *common.MatchSimulation) float32 {
	return (match.HomeQua.Midfield*0.7 + match.HomeQua.LeftMidfield*0.15 + match.HomeQua.RightMidfield*0.15) -
		(match.AwayQua.Midfield*0.7 + match.AwayQua.LeftMidfield*0.15 + match.AwayQua.RightMidfield*0.15)
}

func calculateHomeAttackDiff(match *common.MatchSimulation) float32 {
	return (match.HomeQua.Attack*0.7 + match.HomeQua.LeftAttack*0.15 + match.HomeQua.RightAttach*0.15) -
		(match.AwayQua.Defense*0.7 + match.AwayQua.LeftDefense*0.15 + match.AwayQua.RightDefense*0.15)
}

func calculateAwayAttackDiff(match *common.MatchSimulation) float32 {
	return (match.AwayQua.Attack*0.7 + match.AwayQua.LeftAttack*0.15 + match.AwayQua.RightAttach*0.15) -
		(match.HomeQua.Defense*0.7 + match.HomeQua.LeftDefense*0.15 + match.HomeQua.RightDefense*0.15)
}
