package engine

import "github.com/scruz84/striker/common"

func simulateMatch(dbCtx *common.DatabaseContext, match *common.WeekMatch) (*common.MatchSimulation, error) {
	// get the teams lineups
	lineups, err := common.GetLineupsForMatch(dbCtx, match.MatchId, &[]int64{match.Home, match.Away})
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
	events := make([]common.Event, 0)
	matchInstants := make([]common.MatchInstant, 0)
	matchSimulation := &common.MatchSimulation{
		Id:                    match.MatchId,
		DivisionId:            match.DivisionId,
		Home:                  match.Home,
		Away:                  match.Away,
		Minute:                0,
		HomeLineup:            (*lineups)[match.Home],
		AwayLineup:            (*lineups)[match.Away],
		HomeGoals:             0,
		AwayGoals:             0,
		HomeQua:               &common.TeamQualification{},
		AwayQua:               &common.TeamQualification{},
		HomePlayers:           playersToMapById(homePlayers),
		AwayPlayers:           playersToMapById(awayPlayers),
		HomePlayersByPosition: playersByPosition((*lineups)[match.Home]),
		AwayPlayersByPosition: playersByPosition((*lineups)[match.Away]),
		Events:                &events,
		MatchInstants:         &matchInstants,
		YellowCards:           make(map[int64]bool),
		SentOff:               make(map[int64]bool),
	}
	events = nil
	matchInstants = nil

	// Suspended/injured players must not play even if they're still in the saved lineup -
	// the manager is expected to work around it, but the simulation itself must not field them.
	if err := excludeUnavailablePlayers(dbCtx, match.Home, matchSimulation.HomePlayersByPosition); err != nil {
		return nil, err
	}
	if err := excludeUnavailablePlayers(dbCtx, match.Away, matchSimulation.AwayPlayersByPosition); err != nil {
		return nil, err
	}

	simulations := make([]Simulation, 0)
	simulations = append(simulations, InitialSimulation{}, NewScorerSimulation(), NewDisciplineSimulation(), NewInjurySimulation())
	pipeline := SimulationPipelineImpl{
		simulations: simulations,
	}

	//simulation by minute
	endMinute := 90
	for min := 1; min <= endMinute; min++ {
		matchSimulation.Minute = min
		pipeline.Simulate(dbCtx, matchSimulation)
	}

	UpdatePlayedMatch(dbCtx, matchSimulation)
	UpdateClassification(dbCtx, matchSimulation)
	UpdatePlayerAvailability(dbCtx, matchSimulation)
	UpdateTeamFinancesAfterMatch(dbCtx, matchSimulation)

	return matchSimulation, nil
}

// excludeUnavailablePlayers removes suspended/injured players from a team's active position
// groups before kickoff. Goalkeepers are exempt even if the data somehow flags one - nothing
// in this codebase currently can (see DisciplineSimulation/InjurySimulation), but if it ever
// did, an empty GOALKEEPER slice would panic elsewhere (e.g. ScorerSimulation's goalkeeper
// value lookup), so this is cheap, deliberate insurance.
func excludeUnavailablePlayers(dbCtx *common.DatabaseContext, teamId int64, playersByPosition *map[int][]int64) error {
	unavailable, err := common.GetUnavailablePlayerIds(dbCtx, teamId)
	if err != nil {
		return err
	}
	filterOutfieldPlayers(playersByPosition, unavailable)
	return nil
}

// filterOutfieldPlayers removes every player in `exclude` from a team's non-goalkeeper
// position groups, in place.
func filterOutfieldPlayers(playersByPosition *map[int][]int64, exclude map[int64]bool) {
	if len(exclude) == 0 {
		return
	}
	for _, position := range common.ALL_OUTFIELD_POSITIONS {
		ids := (*playersByPosition)[position]
		kept := ids[:0]
		for _, id := range ids {
			if !exclude[id] {
				kept = append(kept, id)
			}
		}
		(*playersByPosition)[position] = kept
	}
}

func playersToMapById(players *[]*common.Player) *map[int64]common.Player {
	playersMap := make(map[int64]common.Player)
	for _, p := range *players {
		playersMap[p.Id] = *p
	}
	return &playersMap
}

func playersByPosition(lineup *common.Lineup) *map[int][]int64 {
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
