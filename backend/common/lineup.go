package common

import (
	"math"
	"math/rand"
	"sort"
)

// formationScoreTolerance is how close two formations' scores must be to count as tied (see
// SelectBestFormationAndLineup).
const formationScoreTolerance = 1e-9

// formationSlotOrder returns formation's position slots in a fixed, deterministic fill order -
// goalkeeper first, then the back line (left flank, then every central slot together, then the
// right flank - not interleaved, so a doubled-up center is filled before either flank), then
// midfield in the same left/center/right shape, then attack. Built from FormationPositionCounts
// so the two can never drift out of sync; the specific order only matters for tie-breaking when
// several remaining slots could fit the same unassigned player equally well. Reproduces
// bestLineupSlots' old fixed order exactly when fed FormationBalanced's own counts.
func formationSlotOrder(formation int) []int {
	counts := FormationPositionCounts(formation)
	order := make([]int, 0, 11)
	for _, position := range positionFillOrder {
		for i := 0; i < counts[position]; i++ {
			order = append(order, position)
		}
	}
	return order
}

// positionFillOrder is the fixed order positions are filled in, both when picking a best XI (see
// formationSlotOrder) and when the match-day backfill restores a thinned lineup (see
// BackfillEmptyPositions): goalkeeper, the back line, midfield, then attack - each left flank,
// centre, right flank.
var positionFillOrder = []int{
	GOALKEEPER,
	LATERAL_DEFENSE_LEFT, DEFENSE, LATERAL_DEFENSE_RIGHT,
	LATERAL_MIDFIELD_LEFT, MIDFIELD, LATERAL_MIDFIELD_RIGHT,
	LATERAL_STRIKER_LEFT, STRIKER, LATERAL_STRIKER_RIGHT,
}

// BackfillEmptyPositions restores each position in wantCounts back to its original headcount,
// using the next-best available (not already fielded elsewhere, not unavailable) squad player
// for that specific position each time - see PositionValue. Positions are visited in
// positionFillOrder (goalkeeper first, then defence, midfield, attack), so when two positions
// would both pick the same reserve the earlier one wins, the same way every time - the result
// never depends on map iteration order. Works regardless of how many players a position started
// with: an auto-generated lineup (SelectBestLineup) can double up DEFENSE/MIDFIELD/STRIKER, while
// a human-edited one (common.ValidateLineup) always has exactly one player per position - this
// only ever restores what a position already had, never invents extra slots. If the squad
// genuinely runs out of eligible players, a position is left short rather than fielding someone
// twice or someone unavailable - the simulation already tolerates an understaffed position group.
func BackfillEmptyPositions(playersByPosition *map[int][]int64, wantCounts map[int]int, squad []*Player, unavailable map[int64]bool) {
	used := make(map[int64]bool, len(squad))
	for _, ids := range *playersByPosition {
		for _, id := range ids {
			used[id] = true
		}
	}

	for _, position := range positionFillOrder {
		wantCount := wantCounts[position]
		for len((*playersByPosition)[position]) < wantCount {
			bestId, bestValue := int64(-1), -1
			for _, p := range squad {
				if used[p.Id] || unavailable[p.Id] {
					continue
				}
				if v := PositionValue(position, *p); v > bestValue {
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

// SelectBestLineupForFormation picks formation's best XI: the assignment of players to its slots
// (see formationSlotOrder) with the highest total PositionValue, so nobody is picked twice and a
// player whose best slot comes late in the fill order is not taken by an earlier one (openspec
// cpu-team-management "Best XI by optimal assignment"); ties field the lowest-id players. If
// players has fewer entries than the formation needs, slots are instead filled greedily in fill
// order and the remaining ones are simply left unfilled rather than erroring - the match
// simulation already tolerates an understaffed position group (see excludeUnavailablePlayers and
// its callers).
func SelectBestLineupForFormation(players []*Player, formation int) map[int][]int64 {
	slots := formationSlotOrder(formation)
	if len(players) >= len(slots) {
		return bestAssignmentLineup(players, slots)
	}

	lineup := make(map[int][]int64)
	pool := append([]*Player(nil), players...)
	for _, position := range slots {
		if len(pool) == 0 {
			continue
		}
		bestIdx := 0
		bestValue := PositionValue(position, *pool[0])
		for i := 1; i < len(pool); i++ {
			if v := PositionValue(position, *pool[i]); v > bestValue {
				bestIdx = i
				bestValue = v
			}
		}
		lineup[position] = append(lineup[position], pool[bestIdx].Id)
		pool[bestIdx] = pool[len(pool)-1]
		pool = pool[:len(pool)-1]
	}

	return lineup
}

// SelectBestLineup is SelectBestLineupForFormation pinned to FormationBalanced (4-4-2) - used
// wherever a sensible default lineup is needed without any formation decision attached: seeding a
// brand new team's default lineup (see AssignTeam and the initializer's createTeam), a friendly's
// fallback starting XI (see BestLineupForTeam), and SquadStrength's ambition heuristic. Unlike
// SelectBestFormationAndLineup below, this never varies by AI level - see that function for the
// formation-aware, AI-level-aware version an unowned team's own default lineup actually uses.
func SelectBestLineup(players []*Player) map[int][]int64 {
	return SelectBestLineupForFormation(players, FormationBalanced)
}

// lineupStrength totals PositionValue for every player SelectBestLineupForFormation picked, for
// the slot each was picked for - used only to compare a squad's best-XI strength across different
// formations (see SelectBestFormationAndLineup), not as any single meaningful in-game number on
// its own (see SquadStrength for that, pinned to FormationBalanced for its own, unrelated use).
func lineupStrength(lineup map[int][]int64, playersById map[int64]*Player) int {
	total := 0
	for position, ids := range lineup {
		for _, id := range ids {
			if p, ok := playersById[id]; ok {
				total += PositionValue(position, *p)
			}
		}
	}
	return total
}

// formationRankForAiLevel decides which rank (0 = strongest) among AllFormations' candidates an
// AI-managed team of this level settles for. A Manager and a Director take the best-ranked shape
// (a Director then re-picks against its opponent at kickoff, by expected goals - see
// engine.chooseFormationAgainst); a Caretaker settles for the third or fourth best, decided by a
// coin flip each time it's recomputed (see common.AiLevelCaretaker/Manager/Director). numCandidates guards against a shorter list than the
// chosen rank expects (there are only ever len(AllFormations), but this keeps the function safe
// for any slice length). rng drives the Caretaker's coin flip, so a caller that needs the same pick
// twice (the CPU kickoff planner, see PlanAiMatchLineup) can pass a seeded one; nil falls back to
// the global source.
func formationRankForAiLevel(aiLevel int, numCandidates int, rng *rand.Rand) int {
	rank := 0
	switch aiLevel {
	case AiLevelDirector, AiLevelManager:
		// A Manager settles for the best-ranked shape; a Director goes on to pick
		// its formation against the opponent at kickoff (see engine.chooseFormationAgainst).
		rank = 0
	default: // AiLevelCaretaker, and any unassigned/legacy level below it
		if rng != nil {
			rank = 2 + rng.Intn(2)
		} else {
			rank = 2 + rand.Intn(2)
		}
	}
	if rank >= numCandidates {
		rank = numCandidates - 1
	}
	return rank
}

// FormationScorer rates how well a squad's best XI in one formation would do against the same
// squad's best 4-4-2 XI (reference), higher being better. The engine registers one built on its
// expected-goals projection (engine.ProjectExpectedGoals), which common cannot import.
type FormationScorer func(squad []*Player, lineup *Lineup, reference *Lineup) float64

// formationScorer is the registered FormationScorer, nil until SetFormationScorer is called.
var formationScorer FormationScorer

// SetFormationScorer registers the score SelectBestFormationAndLineup ranks formations by (see
// openspec cpu-team-management "Lineup and formation"). Called once, from the engine package's
// init, before any CPU lineup is picked.
func SetFormationScorer(scorer FormationScorer) {
	formationScorer = scorer
}

// SelectBestFormationAndLineup picks both a formation and that formation's best-XI for an
// AI-managed team, factoring in how sharp its AI level is (see formationRankForAiLevel) - this is
// what makes an unowned team actually vary its formation by squad composition instead of always
// defaulting to FormationBalanced's 4-4-2. Builds every formation's own best-XI (see
// SelectBestLineupForFormation) and ranks them by the registered FormationScorer against the
// squad's own best 4-4-2 XI - ties going to the stronger XI by summed slot value (see
// lineupStrength), then to the earlier formation in AllFormations - before picking the rank the
// team's AI level settles for. With no scorer registered (common's own unit tests) the ranking is
// by summed slot value alone. A formation whose XI is empty (no players at all) is left out. rng
// is passed straight to formationRankForAiLevel.
func SelectBestFormationAndLineup(players []*Player, aiLevel int, rng *rand.Rand) (int, map[int][]int64) {
	type candidate struct {
		formation int
		lineup    map[int][]int64
		score     float64
		strength  int
	}

	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}

	reference := &Lineup{TeamId: -1, Formation: FormationBalanced, Positions: FlattenLineup(SelectBestLineupForFormation(players, FormationBalanced))}
	candidates := make([]candidate, 0, len(AllFormations))
	for _, formation := range AllFormations {
		lineup := SelectBestLineupForFormation(players, formation)
		c := candidate{formation: formation, lineup: lineup, strength: lineupStrength(lineup, playersById)}
		if formationScorer != nil && len(lineup) > 0 && len(reference.Positions) > 0 {
			c.score = formationScorer(players, &Lineup{TeamId: -1, Formation: formation, Positions: FlattenLineup(lineup)}, reference)
		}
		candidates = append(candidates, c)
	}
	// Stable, so equal candidates keep AllFormations order.
	sort.SliceStable(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].score-candidates[j].score) > formationScoreTolerance {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].strength > candidates[j].strength
	})

	chosen := candidates[formationRankForAiLevel(aiLevel, len(candidates), rng)]
	return chosen.formation, chosen.lineup
}

// PositionValue scores how good a player is for one tactical position - used to build a best-XI
// lineup for any formation (see formationSlotOrder/SelectBestLineupForFormation) and, since it
// covers every position constant including LATERAL_STRIKER_LEFT/RIGHT (the only two
// SelectBestLineupForFormation itself only reaches for a formation that actually uses them), by
// the match-day backfill that finds a replacement for a suspended/injured starter too (see
// engine's excludeUnavailablePlayers/common.BackfillEmptyPositions) - a human-edited lineup (see
// common.ValidateLineup) can use those positions regardless of which formation generated it.
func PositionValue(position int, p Player) int {
	weights, ok := SlotWeights[position]
	if !ok {
		return 0
	}
	return weights.Value(p)
}

// FlattenLineup converts a position->players map (as returned by SelectBestLineup/
// SelectBestLineupForFormation) into the flat slice SetDefaultLineup expects.
func FlattenLineup(lineup map[int][]int64) []LineupPosition {
	positions := make([]LineupPosition, 0, 11)
	for position, playerIds := range lineup {
		for _, playerId := range playerIds {
			positions = append(positions, LineupPosition{PlayerId: playerId, Position: position})
		}
	}
	return positions
}

// LineupPlayerIds returns every player id referenced in lineup's starting positions - nil-safe,
// and empty rather than erroring when lineup itself is nil. Used to resolve a national team's
// called-up players by id (see GetPlayersForMatchTeam), since they're never on the national
// team's own roster the way GetPlayersFromTeamId expects.
func LineupPlayerIds(lineup *Lineup) []int64 {
	if lineup == nil {
		return nil
	}
	ids := make([]int64, len(lineup.Positions))
	for i, p := range lineup.Positions {
		ids[i] = p.PlayerId
	}
	return ids
}

// BestLineupForTeam computes teamId's current strongest available XI fresh from its full squad
// (see SelectBestLineup), rather than reading back whatever was last saved as the team's standing
// default lineup - used to give a friendly a sensible starting XI by default (see
// GetLineupsForFriendly/handlers.GetFriendlyLineupHandler) without requiring the manager to have
// set up, or kept up to date, a default lineup first. Substitutions are deliberately left empty:
// the caller decides whether to attach any (see GetFriendlyLineupSubstitutions).
func BestLineupForTeam(dbCtx *DatabaseContext, teamId int64) (*Lineup, error) {
	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	return &Lineup{
		TeamId:        teamId,
		Formation:     FormationBalanced,
		Positions:     FlattenLineup(SelectBestLineup(*players)),
		Substitutions: make([]LineupSubstitution, 0),
	}, nil
}
