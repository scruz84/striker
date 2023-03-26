package engine

import "footmanager/fmcommon"

type SimulationPipeline interface {
	Simulate(dbCtx *fmcommon.DatabaseContext, match *MatchSimulation)
}

type Simulation interface {
	Apply(dbCtx *fmcommon.DatabaseContext, match *MatchSimulation)
}

type MatchSimulation struct {
	Id         int64
	DivisionId int64
	Minute     int
	Home       int64
	Away       int64
	HomeLineup *fmcommon.Lineup
	AwayLineup *fmcommon.Lineup
	HomeGoals  int
	AwayGoals  int
	//Players
	HomePlayers           *map[int64]fmcommon.Player
	AwayPlayers           *map[int64]fmcommon.Player
	HomePlayersByPosition *map[int][]int64
	AwayPlayersByPosition *map[int][]int64
	//Qualifications
	HomeQua *TeamQualification
	AwayQua *TeamQualification
	//Calculations
	MidfieldDiff         float32 // home midfield vs away midfield difference
	HomeAttackRightDiff  float32
	HomeAttackLeftDiff   float32
	HomeAttackCenterDiff float32
	AwayAttackRightDiff  float32
	AwayAttackLeftDiff   float32
	AwayAttackCenterDiff float32
	//Events
	Events *[]Event
}

type TeamQualification struct {
	Goalkeeper    float32
	LeftDefense   float32
	RightDefense  float32
	Defense       float32
	LeftMidfield  float32
	RightMidfield float32
	Midfield      float32
	LeftAttack    float32
	RightAttach   float32
	Attack        float32
}

type Event struct {
	EventType        int
	Minute           int
	TeamOwner        int64
	TeamOwnerPlayers []int64
	OtherTeam        int64
	OtherTeamPlayers []int64
}
