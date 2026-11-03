package engine

import (
	"math"

	"github.com/scruz84/striker/common"
)

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
// resolves any pre-kickoff refill of the owner's side). homeAdvantage is the home side's zone-rating
// multiplier (see ExpectedHomeAdvantage; 0 or 1 for none, as in a neutral fixture) and ownIsHome
// says which of the two sides gets it - applied after the zone blend exactly as InitialSimulation
// does, so a projection can never drift from the engine.
func ProjectZoneDuel(own ZoneDuelSide, opponent ZoneDuelSide, homeAdvantage float32, ownIsHome bool) ZoneDuelProjection {
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
	// Own is simulated in the home slot whichever side it really is, so the multiplier is applied
	// here, to the side that is really at home, rather than through MatchSimulation.HomeAdvantage.
	if ownIsHome {
		applyHomeAdvantage(match.HomeQua, homeAdvantage)
	} else {
		applyHomeAdvantage(match.AwayQua, homeAdvantage)
	}

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

// ExpectedHomeAdvantage is the home side's zone-rating multiplier a preview should assume for a
// match at homeTeamId's ground: the one the match will draw (see homeAdvantageFor), but from the
// expected crowd - the modelled demand in average weather, without the random swing - since the
// real crowd is only drawn at kickoff. 1 for a neutral fixture (a World Cup match) and, as when the
// stadium cannot be read, an empty house. cupRound is 0 for a league match or friendly.
func ExpectedHomeAdvantage(dbCtx *common.DatabaseContext, homeTeamId int64, awayTeamId int64, cupRound int, neutral bool) float32 {
	if neutral {
		return 1
	}
	inputs, ok := loadCrowdInputs(dbCtx, homeTeamId, matchdayContext{awayTeamId: awayTeamId, cupRound: cupRound})
	if !ok {
		return homeAdvantageMultiplier(0, 0)
	}
	demand := attendanceDemand(inputs.socios, inputs.stadium.TicketPrice, inputs.formPoints, inputs.formMatches, inputs.rivalFanbase, cupRound, common.RIVALRY_NONE, "") * meanWeatherAttendanceFactor
	attendance := int(math.Max(0, math.Min(float64(inputs.stadium.Capacity), demand)))
	return homeAdvantageMultiplier(attendance, inputs.stadium.Capacity)
}
