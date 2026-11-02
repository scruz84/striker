package engine

import "github.com/scruz84/striker/common"

// ZoneDuelSide is one side of a projected zone duel: the XI it would field and the squad that XI
// is drawn from.
type ZoneDuelSide struct {
	Lineup *common.Lineup
	Squad  []*common.Player
}

// ZoneDuelProjection is the minute-1 zone ratings of both sides, in the shape the match viewer's
// zone duel renders (see frontend ZoneComparisonRows): each side's TeamQualification plus its
// goalkeeper's rating, which the defense and attack columns add on top of the zone strengths.
type ZoneDuelProjection struct {
	Own                common.TeamQualification `json:"own"`
	Opponent           common.TeamQualification `json:"opponent"`
	OwnGoalkeeper      float32                  `json:"ownGoalkeeper"`
	OpponentGoalkeeper float32                  `json:"opponentGoalkeeper"`
}

// ProjectZoneDuel rates two XIs against each other exactly as the simulation does at minute 1: it
// builds a MatchSimulation, rebuilds each side's fatigue-adjusted players as StaminaSimulation does
// before the first minute (no minutes played yet) and runs InitialSimulation on it - the same code
// that fills every real match's zone instants, so a projection can never drift from the engine.
// Pure: it reads no database and changes neither lineup. Both lineups are used as given (the caller
// resolves any pre-kickoff refill of the owner's side), and Own is rated as the home side - the
// ratings of a side do not depend on which side it is until home advantage exists.
func ProjectZoneDuel(own ZoneDuelSide, opponent ZoneDuelSide) ZoneDuelProjection {
	ownPlayers := playersToMapById(&own.Squad)
	opponentPlayers := playersToMapById(&opponent.Squad)
	events := make([]common.Event, 0)
	instants := make([]common.MatchInstant, 0, 1)
	match := &common.MatchSimulation{
		Minute:                1,
		HomeLineup:            own.Lineup,
		AwayLineup:            opponent.Lineup,
		HomeQua:               &common.TeamQualification{},
		AwayQua:               &common.TeamQualification{},
		HomePlayers:           ownPlayers,
		AwayPlayers:           opponentPlayers,
		HomePlayersByPosition: playersByPosition(own.Lineup),
		AwayPlayersByPosition: playersByPosition(opponent.Lineup),
		Events:                &events,
		MatchInstants:         &instants,
		HomeMinutesByPosition: make(map[int64]map[int]int),
		AwayMinutesByPosition: make(map[int64]map[int]int),
	}
	StaminaSimulation{}.Apply(nil, match)
	InitialSimulation{}.Apply(nil, match)

	return ZoneDuelProjection{
		Own:                *match.HomeQua,
		Opponent:           *match.AwayQua,
		OwnGoalkeeper:      goalkeeperSkill(own.Lineup, ownPlayers),
		OpponentGoalkeeper: goalkeeperSkill(opponent.Lineup, opponentPlayers),
	}
}

// goalkeeperSkill is the goalkeeper skill of the lineup's keeper (0 when it has none), which is
// what the viewer's zone duel adds to a side's defense column.
func goalkeeperSkill(lineup *common.Lineup, players *map[int64]common.Player) float32 {
	for _, p := range lineup.Positions {
		if p.Position == common.GOALKEEPER {
			return float32((*players)[p.PlayerId].GoalKeeper)
		}
	}
	return 0
}
