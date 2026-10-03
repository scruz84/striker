package common

import (
	"math/rand"
	"sort"
)

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
	appendN := func(position int) {
		for i := 0; i < counts[position]; i++ {
			order = append(order, position)
		}
	}
	appendN(GOALKEEPER)
	appendN(LATERAL_DEFENSE_LEFT)
	appendN(DEFENSE)
	appendN(LATERAL_DEFENSE_RIGHT)
	appendN(LATERAL_MIDFIELD_LEFT)
	appendN(MIDFIELD)
	appendN(LATERAL_MIDFIELD_RIGHT)
	appendN(LATERAL_STRIKER_LEFT)
	appendN(STRIKER)
	appendN(LATERAL_STRIKER_RIGHT)
	return order
}

// SelectBestLineupForFormation greedily fills formation's position slots (see
// formationSlotOrder), each with the highest-value remaining player for that slot (see
// PositionValue), so nobody is picked twice - a simple stand-in for a manager's judgement. If
// players has fewer entries than the formation needs, the remaining slots are simply left
// unfilled rather than erroring - the match simulation already tolerates an understaffed position
// group (see excludeUnavailablePlayers and its callers).
func SelectBestLineupForFormation(players []*Player, formation int) map[int][]int64 {
	lineup := make(map[int][]int64)
	pool := append([]*Player(nil), players...)

	for _, position := range formationSlotOrder(formation) {
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
// AI-managed team of this level settles for. Only Director reliably finds the genuinely best
// shape for its squad; anyone less sharp settles for "almost the best" rather than the best
// outright, and how far from the top grows as the level drops - Manager's "almost the best" is
// tighter (always the second-best) than Caretaker's (the third or fourth best, decided by a coin
// flip each time it's recomputed) matching the tiers' own documented gap in sophistication (see
// common.AiLevelCaretaker/Manager/Director). numCandidates guards against a shorter list than the
// chosen rank expects (there are only ever len(AllFormations), but this keeps the function safe
// for any slice length).
func formationRankForAiLevel(aiLevel int, numCandidates int) int {
	rank := 0
	switch aiLevel {
	case AiLevelDirector:
		rank = 0
	case AiLevelManager:
		rank = 1
	default: // AiLevelCaretaker, and any unassigned/legacy level below it
		rank = 2 + rand.Intn(2)
	}
	if rank >= numCandidates {
		rank = numCandidates - 1
	}
	return rank
}

// SelectBestFormationAndLineup picks both a formation and that formation's best-XI for an
// AI-managed team, factoring in how sharp its AI level is (see formationRankForAiLevel) - this is
// what makes an unowned team actually vary its formation by squad composition instead of always
// defaulting to FormationBalanced's 4-4-2 (see refreshAiTeamLineup, the only caller). Builds every
// formation's own best-XI (see SelectBestLineupForFormation) and ranks them by total strength (see
// lineupStrength) before picking the rank the team's AI level settles for.
func SelectBestFormationAndLineup(players []*Player, aiLevel int) (int, map[int][]int64) {
	type candidate struct {
		formation int
		lineup    map[int][]int64
		strength  int
	}

	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}

	candidates := make([]candidate, 0, len(AllFormations))
	for _, formation := range AllFormations {
		lineup := SelectBestLineupForFormation(players, formation)
		candidates = append(candidates, candidate{formation, lineup, lineupStrength(lineup, playersById)})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].strength > candidates[j].strength })

	chosen := candidates[formationRankForAiLevel(aiLevel, len(candidates))]
	return chosen.formation, chosen.lineup
}

// PositionValue scores how good a player is for one tactical position - used to build a best-XI
// lineup for any formation (see formationSlotOrder/SelectBestLineupForFormation) and, since it
// covers every position constant including LATERAL_STRIKER_LEFT/RIGHT (the only two
// SelectBestLineupForFormation itself only reaches for a formation that actually uses them), by
// the match-day backfill that finds a replacement for a suspended/injured starter too (see
// engine's excludeUnavailablePlayers/backfillEmptyPositions) - a human-edited lineup (see
// handlers.validateLineup) can use those positions regardless of which formation generated it.
func PositionValue(position int, p Player) int {
	switch position {
	case GOALKEEPER:
		return p.GoalKeeper
	case DEFENSE:
		return p.Defense
	case LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT:
		return GetLateralDefenseValue(p)
	case MIDFIELD:
		return GetMidfieldValue(p)
	case LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT:
		return GetLateralMidfieldValue(p)
	case STRIKER:
		return GetStrikerValue(p)
	case LATERAL_STRIKER_LEFT, LATERAL_STRIKER_RIGHT:
		return GetLateralStrikerValue(p)
	default:
		return 0
	}
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
