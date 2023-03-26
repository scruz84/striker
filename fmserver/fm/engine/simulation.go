package engine

import (
	"footmanager/fmcommon"
)

func simulateMatch(dbCtx *fmcommon.DatabaseContext, match *fmcommon.WeekMatch) (*MatchSimulation, error) {
	// get the teams lineups
	lineups, err := fmcommon.GetLineupsForMatch(dbCtx, match.MatchId, &[]int64{match.Home, match.Away})
	if err != nil {
		return nil, err
	}

	// initializes the simulation data
	homePlayers, err := GetPlayersFromTeam(dbCtx, match.Home)
	if err != nil {
		return nil, err
	}
	awayPlayers, err := GetPlayersFromTeam(dbCtx, match.Away)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0)
	matchSimulation := &MatchSimulation{
		Id:                    match.MatchId,
		DivisionId:            match.DivisionId,
		Home:                  match.Home,
		Away:                  match.Away,
		Minute:                0,
		HomeLineup:            (*lineups)[match.Home],
		AwayLineup:            (*lineups)[match.Away],
		HomeGoals:             0,
		AwayGoals:             0,
		HomeQua:               &TeamQualification{},
		AwayQua:               &TeamQualification{},
		HomePlayers:           playersToMapById(homePlayers),
		AwayPlayers:           playersToMapById(awayPlayers),
		HomePlayersByPosition: playersByPosition((*lineups)[match.Home]),
		AwayPlayersByPosition: playersByPosition((*lineups)[match.Away]),
		Events:                &events,
	}
	events = nil

	simulations := make([]Simulation, 0)
	simulations = append(simulations, InitialSimulation{}, NewScorerSimulation())
	pipeline := SimulationPipelineImpl{
		simulations: simulations,
	}

	//simulation by minute
	endMinute := 90
	for min := 1; min <= endMinute; min++ {
		matchSimulation.Minute = min
		pipeline.Simulate(dbCtx, matchSimulation)
	}

	return matchSimulation, nil
}

func playersToMapById(players *[]*fmcommon.Player) *map[int64]fmcommon.Player {
	playersMap := make(map[int64]fmcommon.Player)
	for _, p := range *players {
		playersMap[p.Id] = *p
	}
	return &playersMap
}

func playersByPosition(lineup *fmcommon.Lineup) *map[int][]int64 {
	players := make(map[int][]int64)
	for _, position := range lineup.Positions {
		currentPlayers := players[position.Position]
		if currentPlayers == nil {
			currentPlayers = make([]int64, 0)
		}
		currentPlayers = append(currentPlayers, position.PlayerId)
		players[position.Position] = currentPlayers
	}
	return &players
}
