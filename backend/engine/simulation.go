package engine

import (
	"context"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

func simulateMatch(dbCtx *common.DatabaseContext, match *common.WeekMatch) (*common.MatchSimulation, error) {
	lineups, err := common.GetLineupsForMatch(dbCtx, match.MatchId, &[]int64{match.Home, match.Away})
	if err != nil {
		return nil, err
	}
	matchSimulation, err := runMatchSimulation(dbCtx, match.MatchId, match.DivisionId, match.Home, match.Away, lineups)
	if err != nil {
		return nil, err
	}

	UpdatePlayedMatch(dbCtx, matchSimulation)
	UpdateClassification(dbCtx, matchSimulation)
	UpdatePlayerAvailability(dbCtx, matchSimulation)
	UpdateTeamFinancesAfterMatch(dbCtx, matchSimulation)
	recordPlayerMinutes(dbCtx, matchSimulation)
	NotifyMatchResult(dbCtx, matchSimulation)
	BroadcastMatchLive(dbCtx, common.LeagueMatchChannel(matchSimulation.Id), matchSimulation)
	common.RecordMatchSimulated(context.Background(), "league", matchSimulation.HomeGoals, matchSimulation.AwayGoals)

	return matchSimulation, nil
}

// recordPlayerMinutes persists a just-simulated competitive match's per-player-per-position
// minutes into the season-long accumulator engine.ProgressPlayers reads at season rollover (see
// common.RecordPlayerPositionMinutes) - shared by league matches (here) and cup ties (see
// engine/cups.go's simulateCupTie). Deliberately NOT called for friendly matches - see
// engine/friendlies.go's simulateFriendly and init26_friendlies.sql's own rationale for why
// friendlies have no effect beyond the match itself.
func recordPlayerMinutes(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if err := common.RecordPlayerPositionMinutes(dbCtx, match.HomeMinutesByPosition); err != nil {
		log.Errorf("error recording home player minutes for match %d: %s", match.Id, err)
	}
	if err := common.RecordPlayerPositionMinutes(dbCtx, match.AwayMinutesByPosition); err != nil {
		log.Errorf("error recording away player minutes for match %d: %s", match.Id, err)
	}
}

// runMatchSimulation builds both teams' qualifications from the given lineups and runs the full
// 90-minute simulation pipeline (goals, cards, injuries). It does NOT persist anything - callers
// decide what to do with the result, since a league match updates classification differently
// than a cup tie does (see simulateMatch vs engine/cups.go's simulateCupTie).
//
// lineups is resolved by the caller rather than looked up here, so each match kind can use its
// own resolution: simulateMatch/simulateCupTie both use common.GetLineupsForMatch (a
// match-specific override, keyed to matches(match_id), falling back to the team's default
// lineup - nothing currently ever sets such an override, so a cup tie's id here, drawn from a
// different id sequence than real matches, is safe today but would stop being safe if that ever
// changed), while simulateFriendly uses common.GetLineupsForFriendly - its own tables, entirely
// separate from the competitive-match id space.
func runMatchSimulation(dbCtx *common.DatabaseContext, matchId int64, divisionId int64, homeTeamId int64, awayTeamId int64, lineups *map[int64]*common.Lineup) (*common.MatchSimulation, error) {
	homePlayers, err := GetPlayersFromTeam(dbCtx, homeTeamId)
	if err != nil {
		return nil, err
	}
	awayPlayers, err := GetPlayersFromTeam(dbCtx, awayTeamId)
	if err != nil {
		return nil, err
	}
	homeSubstitutions, err := common.GetDefaultLineupSubstitutions(dbCtx, homeTeamId)
	if err != nil {
		return nil, err
	}
	awaySubstitutions, err := common.GetDefaultLineupSubstitutions(dbCtx, awayTeamId)
	if err != nil {
		return nil, err
	}
	events := make([]common.Event, 0)
	matchInstants := make([]common.MatchInstant, 0)
	matchSimulation := &common.MatchSimulation{
		Id:                    matchId,
		DivisionId:            divisionId,
		Home:                  homeTeamId,
		Away:                  awayTeamId,
		Minute:                0,
		HomeLineup:            (*lineups)[homeTeamId],
		AwayLineup:            (*lineups)[awayTeamId],
		HomeGoals:             0,
		AwayGoals:             0,
		HomeQua:               &common.TeamQualification{},
		AwayQua:               &common.TeamQualification{},
		HomePlayers:           playersToMapById(homePlayers),
		AwayPlayers:           playersToMapById(awayPlayers),
		HomePlayersByPosition: playersByPosition((*lineups)[homeTeamId]),
		AwayPlayersByPosition: playersByPosition((*lineups)[awayTeamId]),
		Events:                &events,
		MatchInstants:         &matchInstants,
		YellowCards:           make(map[int64]bool),
		SentOff:               make(map[int64]bool),
		HomeSubstitutions:     homeSubstitutions,
		AwaySubstitutions:     awaySubstitutions,
		HomeMinutesByPosition: make(map[int64]map[int]int),
		AwayMinutesByPosition: make(map[int64]map[int]int),
	}
	events = nil
	matchInstants = nil

	// Suspended/injured starters must not play even if they're still in the saved lineup - and
	// rather than leaving their slot empty for the match, the next-best available squad player
	// steps in, same as a real substitute (see backfillEmptyPositions).
	if err := excludeUnavailablePlayers(dbCtx, homeTeamId, matchSimulation.HomePlayersByPosition, *homePlayers); err != nil {
		return nil, err
	}
	if err := excludeUnavailablePlayers(dbCtx, awayTeamId, matchSimulation.AwayPlayersByPosition, *awayPlayers); err != nil {
		return nil, err
	}

	simulations := make([]Simulation, 0)
	simulations = append(simulations, SubstitutionSimulation{}, InitialSimulation{}, NewScorerSimulation(), NewDisciplineSimulation(), NewInjurySimulation())
	pipeline := SimulationPipelineImpl{
		simulations: simulations,
	}

	//simulation by minute
	endMinute := 90
	for min := 1; min <= endMinute; min++ {
		matchSimulation.Minute = min
		pipeline.Simulate(dbCtx, matchSimulation)
		// Tallied after this minute's stages have run, so a player substituted/sent off/injured
		// this exact minute doesn't get it counted, and whoever replaced them does - see
		// SubstitutionSimulation, DisciplineSimulation, InjurySimulation, all of which mutate
		// *PlayersByPosition before this point in the same minute.
		trackMinutesByPosition(matchSimulation.HomePlayersByPosition, matchSimulation.HomeMinutesByPosition)
		trackMinutesByPosition(matchSimulation.AwayPlayersByPosition, matchSimulation.AwayMinutesByPosition)
	}

	return matchSimulation, nil
}

// trackMinutesByPosition credits every player currently occupying a position slot with one more
// minute played at that exact position - called once per simulated minute (see
// runMatchSimulation). A player only ever appears in one position's slice at a time (kickoff
// lineup, or wherever a later substitution places them - see SubstitutionSimulation), so this
// naturally accumulates correct per-position minutes even across a mid-match position change.
func trackMinutesByPosition(playersByPosition *map[int][]int64, minutesByPosition map[int64]map[int]int) {
	for position, ids := range *playersByPosition {
		for _, id := range ids {
			if minutesByPosition[id] == nil {
				minutesByPosition[id] = make(map[int]int)
			}
			minutesByPosition[id][position]++
		}
	}
}

// excludeUnavailablePlayers removes suspended/injured players from a team's active position
// groups before kickoff, then backfills each thinned position back up to its original headcount
// from the rest of the squad (see backfillEmptyPositions) - the closest thing this simulation
// has to a substitute coming off the bench. Goalkeepers are exempt from the exclusion itself
// even though nothing in this codebase currently can flag one unavailable (see
// DisciplineSimulation/InjurySimulation) - if it ever did, an empty GOALKEEPER slice would
// panic elsewhere (e.g. ScorerSimulation's goalkeeper value lookup), so this is cheap,
// deliberate insurance; the backfill pass, unlike the exclusion, does cover every position.
func excludeUnavailablePlayers(dbCtx *common.DatabaseContext, teamId int64, playersByPosition *map[int][]int64, squad []*common.Player) error {
	unavailable, err := common.GetUnavailablePlayerIds(dbCtx, teamId)
	if err != nil {
		return err
	}

	originalCounts := make(map[int]int, len(common.ALL_OUTFIELD_POSITIONS))
	for _, position := range common.ALL_OUTFIELD_POSITIONS {
		originalCounts[position] = len((*playersByPosition)[position])
	}

	filterOutfieldPlayers(playersByPosition, unavailable)
	backfillEmptyPositions(playersByPosition, originalCounts, squad, unavailable)
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

// backfillEmptyPositions restores each position in wantCounts back to its original headcount,
// using the next-best available (not already fielded elsewhere, not unavailable) squad player
// for that specific position each time - see common.PositionValue. Works regardless of how many
// players a position started with: an auto-generated lineup (common.SelectBestLineup) can double
// up DEFENSE/MIDFIELD/STRIKER, while a human-edited one (handlers.validateLineup) always has
// exactly one player per position - this only ever restores what a position already had, never
// invents extra slots. If the squad genuinely runs out of eligible players, a position is left
// short rather than fielding someone twice or someone unavailable - the simulation already
// tolerates an understaffed position group.
func backfillEmptyPositions(playersByPosition *map[int][]int64, wantCounts map[int]int, squad []*common.Player, unavailable map[int64]bool) {
	used := make(map[int64]bool, len(squad))
	for _, ids := range *playersByPosition {
		for _, id := range ids {
			used[id] = true
		}
	}

	for position, wantCount := range wantCounts {
		for len((*playersByPosition)[position]) < wantCount {
			bestId, bestValue := int64(-1), -1
			for _, p := range squad {
				if used[p.Id] || unavailable[p.Id] {
					continue
				}
				if v := common.PositionValue(position, *p); v > bestValue {
					bestId, bestValue = p.Id, v
				}
			}
			if bestId == -1 {
				break
			}
			(*playersByPosition)[position] = append((*playersByPosition)[position], bestId)
			used[bestId] = true
		}
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
