package engine

import (
	"context"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// MinPlayersToContinueMatch is the fewest players (goalkeeper included) a side can have on the
// pitch and still carry on - the real-football "7 players" rule. Dropping to this many or fewer,
// whether that's a pre-kickoff squad crisis (excludeUnavailablePlayers couldn't backfill enough
// eligible players) or a mid-match collapse (unreplaced red cards/injuries - see
// applyForfeitIfUnderstaffed), ends the match immediately as a forfeit rather than letting it play
// out short-handed.
const MinPlayersToContinueMatch = 7

// ForfeitGoals is the scoreline awarded to the side still standing when the other is forfeited
// for falling to MinPlayersToContinueMatch or fewer players - the match's actual score up to that
// point is discarded in favor of this fixed result, same as a real abandoned match.
const ForfeitGoals = 3

// persistResolvedLineupSnapshot writes the lineup a match is actually about to be simulated with
// into that match's own lineup/lineup_positions override row, even when it was only resolved via
// GetLineupsForMatch's fallback to the team's standing default_lineup_positions rather than an
// explicit per-match override. Without this, a past match's "Lineups" tab (WeekMatchById) re-reads
// GetLineupsForMatch fresh every time, so once a transfer/retirement deletes one of these starting
// XI's players from default_lineup_positions (see DeletePlayerLineupPositionsStmtSql), or the
// team's default lineup is simply reconfigured, an already-played match's lineup degrades or goes
// fully blank - there was never a true historical record of who actually played, only whatever the
// default lineup happens to look like right now. Calling this up front freezes today's resolution
// as this match's own permanent override, mirroring the explicit snapshot
// SetNationalTeamLineup/SetNationalTeamCupTieLineup already save for national-team fixtures.
func persistResolvedLineupSnapshot(dbCtx *common.DatabaseContext, matchId int64, teamId int64, lineup *common.Lineup, substitutions []common.LineupSubstitution) {
	if lineup == nil || len(lineup.Positions) == 0 {
		return
	}
	if err := common.SetLineupForMatch(dbCtx, matchId, teamId, lineup.Formation, lineup.Positions, substitutions); err != nil {
		log.Errorf("error persisting resolved lineup snapshot for match %d team %d: %s", matchId, teamId, err)
	}
}

func simulateMatch(dbCtx *common.DatabaseContext, match *common.WeekMatch) (*common.MatchSimulation, error) {
	lineups, err := common.GetLineupsForMatch(dbCtx, match.MatchId, &[]int64{match.Home, match.Away})
	if err != nil {
		return nil, err
	}
	// A team's own per-match lineup override can schedule its own substitutions (see
	// common.SetLineupForMatch/GetSubstitutionsForMatch); a team without one still draws from its
	// team-wide default, same as it always has.
	homeSubstitutions, err := common.GetSubstitutionsForMatch(dbCtx, match.MatchId, match.Home)
	if err != nil {
		return nil, err
	}
	awaySubstitutions, err := common.GetSubstitutionsForMatch(dbCtx, match.MatchId, match.Away)
	if err != nil {
		return nil, err
	}
	persistResolvedLineupSnapshot(dbCtx, match.MatchId, match.Home, (*lineups)[match.Home], homeSubstitutions)
	persistResolvedLineupSnapshot(dbCtx, match.MatchId, match.Away, (*lineups)[match.Away], awaySubstitutions)
	matchSimulation, err := runMatchSimulation(dbCtx, match.MatchId, match.DivisionId, match.Home, match.Away, lineups, homeSubstitutions, awaySubstitutions, false)
	if err != nil {
		return nil, err
	}

	// The result, its events and instants, both sides' standings and the pending post-match effects
	// are stored in one transaction that only applies while the match is still unplayed. Everything
	// below runs only after it commits, so a failed save can never be followed by (or later repeated
	// with) money, a report or a broadcast, and two simulations of one match book them once.
	if err := common.SaveLeagueMatchResultTx(dbCtx, matchSimulation.Id, matchSimulation.DivisionId, matchSimulation.Home,
		matchSimulation.Away, common.MatchResult{
			HomeGoals: matchSimulation.HomeGoals,
			AwayGoals: matchSimulation.AwayGoals,
			Events:    *matchSimulation.Events,
			Instants:  *matchSimulation.MatchInstants,
			Effects:   newPostMatchPayload(matchSimulation, false),
		}); err != nil {
		return nil, err
	}

	// Drawn once, up front, so the same weather both shapes the crowd (see CalculateAttendance) and
	// is what the match report shows.
	weather := common.RandomWeather()
	attendance := UpdateTeamFinancesAfterMatch(dbCtx, matchSimulation, 0, weather)
	// Wages track league match cadence, not cup ties or friendlies (see PayMatchdayWages) - a
	// round-robin schedule gives every team exactly one league match per round, so this fires
	// exactly once per round per team, here rather than inside UpdateTeamFinancesAfterMatch so a
	// cup tie played the same week doesn't charge wages a second time.
	PayMatchdayWages(dbCtx, matchSimulation.Home)
	PayMatchdayWages(dbCtx, matchSimulation.Away)
	updateMatchReportFlavor(dbCtx, "match", matchSimulation.Id, attendance, weather,
		func(weather string, attendance int, referee string, assistant1 string, assistant2 string) error {
			return common.UpdateMatchFlavor(dbCtx, matchSimulation.Id, weather, attendance, referee, assistant1, assistant2)
		})
	// UpdatePlayerAvailability (new injuries/suspensions) and updateSkillsAfterPlayedMatch (skill
	// growth, plus the NOTIFICATION_PROGRESSION push it can fire) both change state a team owner
	// can see outside this match's own live-gated page - the squad list, a player's page, the
	// evolutions report - same as the match result itself. Deferred into onComplete for the exact
	// reason NotifyMatchResult already is (see BroadcastMatchLive's doc comment): a match is
	// simulated and marked played up to 15 minutes before its advertised kickoff, so running these
	// here would leak an injury or a skill jump before the match has even nominally started. They
	// were stored as pending with the result, so an engine restart during the replay doesn't lose
	// them (see ApplyDuePostMatchEffects).
	BroadcastMatchLive(dbCtx, common.LeagueMatchChannel(matchSimulation.Id), matchSimulation, match.MatchDate, func() {
		completePostMatchEffects(dbCtx, common.SimulationKindLeague, matchSimulation.Id)
	})
	common.RecordMatchSimulated(context.Background(), "league", matchSimulation.HomeGoals, matchSimulation.AwayGoals)

	return matchSimulation, nil
}

// updateSkillsAfterPlayedMatch drifts skills for every player who featured in a just-simulated
// match (see updatePlayerSkillsAfterMatch) - shared by league matches (here), cup ties (see
// engine/cups.go's simulateCupTie), and friendlies (see engine/friendlies.go's simulateFriendly) -
// a friendly still develops the players who feature in it, even though (unlike a league match or
// cup tie) it carries no financial stakes and doesn't trigger an unowned side's AI upkeep (see
// UpdateTeamFinancesAfterMatch). Looks up the current season itself, from the home side's own
// country (see common.GetCurrentSeasonForTeam - works the same whether match.Home is a club or
// a national team), rather than taking it as a parameter, since none of its three callers
// otherwise has it in scope.
func updateSkillsAfterPlayedMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	season, err := common.GetCurrentSeasonForTeam(dbCtx, match.Home)
	if err != nil {
		log.Errorf("error getting current season for match %d skill progression: %s", match.Id, err)
		return
	}
	updatePlayerSkillsAfterMatch(dbCtx, match, season)
}

// runMatchSimulation builds both teams' qualifications from the given lineups and runs the full
// 90-minute simulation pipeline (goals, cards, injuries). It does NOT persist anything - callers
// decide what to do with the result, since a league match updates classification differently
// than a cup tie does (see simulateMatch vs engine/cups.go's simulateCupTie).
//
// lineups and the scheduled substitutions are both resolved by the caller rather than looked up
// here, so each match kind can use its own resolution: simulateMatch uses common.GetLineupsForMatch/
// GetSubstitutionsForMatch (a per-match override, keyed to matches(match_id), falling back to the
// team's default lineup/substitutions), simulateCupTie uses the equivalent
// common.GetLineupsForCupTie/GetSubstitutionsForCupTie (its own cup_tie_lineup/
// cup_tie_lineup_positions/cup_tie_lineup_substitutions tables, keyed to cup_ties(tie_id) - a cup
// tie's id is drawn from a different, independent sequence than a real match's, so it could never
// safely share matches-keyed override rows), and simulateFriendly uses
// common.GetLineupsForFriendly/GetFriendlyLineupSubstitutions (friendly_lineup/
// friendly_lineup_positions/friendly_lineup_substitutions, keyed to friendly_matches(match_id), and
// never falling back to the team-wide default - a friendly's starting XI can itself be a freshly
// computed best XI rather than the default lineup, see common.GetLineupsForFriendly, so the two
// would otherwise reference different players) - three parallel, deliberately separate override
// mechanisms, one per competitive-match id space.
func runMatchSimulation(dbCtx *common.DatabaseContext, matchId int64, divisionId int64, homeTeamId int64, awayTeamId int64, lineups *map[int64]*common.Lineup, homeSubstitutions []common.LineupSubstitution, awaySubstitutions []common.LineupSubstitution, isFriendly bool) (*common.MatchSimulation, error) {
	homePlayers, err := resolveMatchTeamPlayers(dbCtx, homeTeamId, (*lineups)[homeTeamId])
	if err != nil {
		return nil, err
	}
	awayPlayers, err := resolveMatchTeamPlayers(dbCtx, awayTeamId, (*lineups)[awayTeamId])
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
		IsFriendly:            isFriendly,
	}
	events = nil
	matchInstants = nil

	// Suspended/injured starters must not play even if they're still in the saved lineup - and
	// rather than leaving their slot empty for the match, the next-best available squad player
	// steps in, same as a real substitute (see backfillEmptyPositions). The returned sets are
	// kept on the match so InjurySimulation can reuse them later, when picking a bench player to
	// bring on for someone injured mid-match, without re-querying the database.
	homeUnavailable, err := excludeUnavailablePlayers(dbCtx, homeTeamId, matchSimulation.HomePlayersByPosition, *homePlayers, isFriendly)
	if err != nil {
		return nil, err
	}
	awayUnavailable, err := excludeUnavailablePlayers(dbCtx, awayTeamId, matchSimulation.AwayPlayersByPosition, *awayPlayers, isFriendly)
	if err != nil {
		return nil, err
	}
	matchSimulation.HomeUnavailablePlayers = homeUnavailable
	matchSimulation.AwayUnavailablePlayers = awayUnavailable

	// A squad crisis can leave a side unable to even kick off with enough players (the backfill
	// above only ever restores what the squad actually has available) - checked before a single
	// minute is simulated, same threshold as the mid-match check below.
	if applyForfeitIfUnderstaffed(matchSimulation) {
		return matchSimulation, nil
	}

	simulations := make([]Simulation, 0)
	simulations = append(simulations, SubstitutionSimulation{}, StaminaSimulation{}, InitialSimulation{}, NewScorerSimulation(), NewDisciplineSimulation(), NewInjurySimulation())
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

		// A red card (never replaced) or an injury with no scheduled substitutions left (see
		// DisciplineSimulation/InjurySimulation) can thin a side below MinPlayersToContinueMatch
		// mid-match - stop right there rather than simulating the rest of the match with a side
		// that, in a real match, would have already been abandoned.
		if applyForfeitIfUnderstaffed(matchSimulation) {
			break
		}
	}

	return matchSimulation, nil
}

// countOnPitch totals every player currently occupying any position slot (goalkeeper included -
// playersByPosition's own map always carries a common.GOALKEEPER entry, see the playersByPosition
// helper below) for one side.
func countOnPitch(playersByPosition *map[int][]int64) int {
	count := 0
	for _, ids := range *playersByPosition {
		count += len(ids)
	}
	return count
}

// applyForfeitIfUnderstaffed checks whether either side has fallen to MinPlayersToContinueMatch
// players or fewer and, if so, ends the match immediately: the actual scoreline up to that point
// is discarded in favor of a flat ForfeitGoals-0 win for the side still standing (real football
// awards an abandoned match this way rather than freezing the score as played), and a
// common.MATCH_ABANDONED event records it. Returns whether the match was abandoned, so the caller
// knows to stop simulating further minutes. If both sides have collapsed at once (a degenerate
// case that should never actually happen) neither is awarded a forfeit win and the match plays on.
func applyForfeitIfUnderstaffed(match *common.MatchSimulation) bool {
	homeShort := countOnPitch(match.HomePlayersByPosition) <= MinPlayersToContinueMatch
	awayShort := countOnPitch(match.AwayPlayersByPosition) <= MinPlayersToContinueMatch
	if homeShort == awayShort {
		return false
	}

	shortTeam, otherTeam := match.Home, match.Away
	if homeShort {
		match.HomeGoals, match.AwayGoals = 0, ForfeitGoals
	} else {
		shortTeam, otherTeam = match.Away, match.Home
		match.HomeGoals, match.AwayGoals = ForfeitGoals, 0
	}

	*match.Events = append(*match.Events, common.Event{
		EventType: common.MATCH_ABANDONED,
		Minute:    match.Minute,
		TeamOwner: shortTeam,
		OtherTeam: otherTeam,
		HomeGoals: match.HomeGoals,
		AwayGoals: match.AwayGoals,
	})
	return true
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
func excludeUnavailablePlayers(dbCtx *common.DatabaseContext, teamId int64, playersByPosition *map[int][]int64, squad []*common.Player, isFriendly bool) (map[int64]bool, error) {
	var unavailable map[int64]bool
	var err error
	if isFriendly {
		unavailable, err = common.GetUnavailablePlayerIdsForFriendly(dbCtx, teamId)
	} else {
		unavailable, err = common.GetUnavailablePlayerIds(dbCtx, teamId)
	}
	if err != nil {
		return nil, err
	}

	originalCounts := make(map[int]int, len(common.ALL_OUTFIELD_POSITIONS))
	for _, position := range common.ALL_OUTFIELD_POSITIONS {
		originalCounts[position] = len((*playersByPosition)[position])
	}

	filterOutfieldPlayers(playersByPosition, unavailable)
	backfillEmptyPositions(playersByPosition, originalCounts, squad, unavailable)
	return unavailable, nil
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

// reportFlavorStore persists one match's report flavor in whichever table its competition keeps
// it in (see common.UpdateMatchFlavor / UpdateCupTieFlavor / UpdateFriendlyFlavor).
type reportFlavorStore func(weather string, attendance int, referee string, assistant1 string, assistant2 string) error

// updateMatchReportFlavor generates and persists a match's report flavor (weather, referee, two
// assistant referees) plus the attendance already computed for ticket revenue (see
// UpdateTeamFinancesAfterMatch) - purely cosmetic, for the Hattrick-style match report. Shared by
// league matches, cup ties and friendlies, which differ only in the store they write to; kind
// ("match", "cup tie", "friendly") is only for the log lines. Errors are logged, not propagated:
// the match itself is already fully simulated and persisted by this point, so a failure here
// should never be treated as the match having failed to process.
func updateMatchReportFlavor(dbCtx *common.DatabaseContext, kind string, id int64, attendance int, weather string, store reportFlavorStore) {
	referee, err := common.RandomPersonName(dbCtx)
	if err != nil {
		log.Errorf("error generating referee name for %s %d: %s", kind, id, err)
		return
	}
	assistant1, err := common.RandomPersonName(dbCtx)
	if err != nil {
		log.Errorf("error generating first assistant referee name for %s %d: %s", kind, id, err)
		return
	}
	assistant2, err := common.RandomPersonName(dbCtx)
	if err != nil {
		log.Errorf("error generating second assistant referee name for %s %d: %s", kind, id, err)
		return
	}
	if err := store(weather, attendance, referee, assistant1, assistant2); err != nil {
		log.Errorf("error updating report flavor for %s %d: %s", kind, id, err)
	}
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
