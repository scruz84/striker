package common

import "sort"

// planFriendlyRotation fields a CPU team's XI and scheduled substitutions for a friendly so its
// reserves get minutes, by AI level (see PlanAiMatchLineup). regulars is the XI the team would
// field in a league match that day, in formation. squad is the whole squad; players in
// matchUnavailable (injured - suspensions don't bar a friendly) are never named as a starter or
// substitute, except a Caretaker's regulars, which may name them for the kickoff backfill to
// replace.
//
//   - Caretaker: starts its regulars and brings on its 3 best bench players at minute 60.
//   - Manager: rests its 5 outfield regulars with the highest value at their slot (they neither
//     start nor come on), starts the best XI from the rest in its league formation, and makes 3
//     changes at minute 60 among the players who did not start.
//   - Director: starts a reserve XI - a backup goalkeeper too, if it has one - and keeps key
//     players out unless it needs them to field 11 (see fillWithRegulars). Its 3 changes at
//     minute 60 prefer players aged 21 or under, then the highest value.
//
// Each change takes the bench player with the highest value at his best open slot and replaces
// the starter there. Fewer than 3 are scheduled if there are not enough reserves.
func planFriendlyRotation(squad []*Player, regulars map[int][]int64, formation int, matchUnavailable map[int64]bool, aiLevel int) AiMatchPlan {
	fit := playersWithout(squad, matchUnavailable)

	var xi map[int][]int64
	var bench []*Player
	preferYoung := false
	switch {
	case aiLevel >= AiLevelDirector:
		xi = directorReserveXI(squad, fit, regulars, formation)
		bench = aiBench(fit, xi, nil)
		preferYoung = true
	case aiLevel >= AiLevelManager:
		xi, bench = managerRestedXI(squad, fit, regulars, formation)
	default:
		xi = regulars
		bench = aiBench(fit, xi, nil)
	}

	return AiMatchPlan{
		Formation:     formation,
		Positions:     orderedLineupPositions(xi),
		Substitutions: friendlySubstitutions(fit, xi, bench, preferYoung),
	}
}

// managerRestedXI rests the top 5 outfield regulars by value at their slot and picks the best XI
// from everyone else fit. If fewer than 11 others are fit, the weakest rested regulars are called
// back until it can field 11. Returns the XI and the bench - the fit players who neither start nor
// are still resting, so a rested regular never comes on.
func managerRestedXI(squad []*Player, fit []*Player, regulars map[int][]int64, formation int) (map[int][]int64, []*Player) {
	starters := aiStarters(squad, regulars, true)
	sort.SliceStable(starters, func(i, j int) bool {
		vi, vj := PositionValue(starters[i].position, *starters[i].player), PositionValue(starters[j].position, *starters[j].player)
		if vi != vj {
			return vi > vj
		}
		return starters[i].player.Id < starters[j].player.Id
	})
	rested := make(map[int64]bool, aiFriendlyRestedRegulars)
	restedOrder := make([]aiStarter, 0, aiFriendlyRestedRegulars)
	for i := 0; i < len(starters) && i < aiFriendlyRestedRegulars; i++ {
		rested[starters[i].player.Id] = true
		restedOrder = append(restedOrder, starters[i])
	}

	pool := playersWithout(fit, rested)
	// Weakest rested regular first, so the strongest stay out of the XI as long as possible.
	for i := len(restedOrder) - 1; i >= 0 && len(pool) < aiLineupSize; i-- {
		if p := findPlayer(fit, restedOrder[i].player.Id); p != nil {
			pool = append(pool, p)
			delete(rested, p.Id)
		}
	}

	xi := SelectBestLineupForFormation(pool, formation)
	return xi, aiBench(pool, xi, nil)
}

// directorReserveXI starts the best reserves, one per slot in formationSlotOrder: the best
// goalkeeper reserve in goal (only a reserve whose natural position is goalkeeper), the best
// remaining outfield reserve elsewhere. A slot no reserve can fill is left to fillWithRegulars.
func directorReserveXI(squad []*Player, fit []*Player, regulars map[int][]int64, formation int) map[int][]int64 {
	regular := make(map[int64]bool, aiLineupSize)
	for _, ids := range regulars {
		for _, id := range ids {
			regular[id] = true
		}
	}
	keepers := make([]*Player, 0, 2)
	outfield := make([]*Player, 0, len(fit))
	for _, p := range fit {
		if regular[p.Id] {
			continue
		}
		if BestCorePosition(*p) == GOALKEEPER {
			keepers = append(keepers, p)
		} else {
			outfield = append(outfield, p)
		}
	}

	xi := make(map[int][]int64)
	var gaps []int
	for _, position := range formationSlotOrder(formation) {
		pool := &outfield
		if position == GOALKEEPER {
			pool = &keepers
		}
		best := bestBenchPlayerFor(position, *pool)
		if best == nil {
			gaps = append(gaps, position)
			continue
		}
		xi[position] = append(xi[position], best.Id)
		*pool = withoutPlayer(*pool, best.Id)
	}

	fillWithRegulars(squad, fit, xi, regulars, gaps)
	return xi
}

// fillWithRegulars gives every gap in the XI a regular: the goalkeeper slot its own regular
// goalkeeper, any other the lowest-valued regular at that slot who is not a key player (see
// IsKeyPlayer). Key players are used only when no other regular is left.
func fillWithRegulars(squad []*Player, fit []*Player, xi map[int][]int64, regulars map[int][]int64, gaps []int) {
	used := make(map[int64]bool, aiLineupSize)
	for _, ids := range xi {
		for _, id := range ids {
			used[id] = true
		}
	}
	fitById := make(map[int64]*Player, len(fit))
	for _, p := range fit {
		fitById[p.Id] = p
	}
	var outfieldRegulars []*Player
	for _, starter := range aiStarters(fit, regulars, true) {
		outfieldRegulars = append(outfieldRegulars, starter.player)
	}

	keyById := make(map[int64]bool, len(outfieldRegulars))
	for _, p := range outfieldRegulars {
		keyById[p.Id] = IsKeyPlayer(squad, p.Id)
	}

	for _, position := range gaps {
		if position == GOALKEEPER {
			for _, id := range regulars[GOALKEEPER] {
				if p := fitById[id]; p != nil && !used[id] {
					xi[position] = append(xi[position], id)
					used[id] = true
					break
				}
			}
			continue
		}
		var pick *Player
		for _, p := range outfieldRegulars {
			if used[p.Id] {
				continue
			}
			if pick == nil || lowerPriority(position, p, pick, keyById) {
				pick = p
			}
		}
		if pick != nil {
			xi[position] = append(xi[position], pick.Id)
			used[pick.Id] = true
		}
	}
}

// lowerPriority reports whether candidate should start at position ahead of current: a non-key
// regular beats a key one, then the lower PositionValue wins (the lower id on a tie).
func lowerPriority(position int, candidate *Player, current *Player, keyById map[int64]bool) bool {
	if keyById[candidate.Id] != keyById[current.Id] {
		return !keyById[candidate.Id]
	}
	cv, uv := PositionValue(position, *candidate), PositionValue(position, *current)
	if cv != uv {
		return cv < uv
	}
	return candidate.Id < current.Id
}

// friendlySubstitutions schedules up to aiFriendlyMaxSubs changes at minute 60. Each round takes
// the bench player with the highest value at his best open outfield slot (preferring players aged
// 21 or under while any remain, when preferYoung) and replaces the starter there. A bench goalkeeper
// never comes on for an outfielder.
func friendlySubstitutions(squad []*Player, xi map[int][]int64, bench []*Player, preferYoung bool) []LineupSubstitution {
	subs := make([]LineupSubstitution, 0, aiFriendlyMaxSubs)
	open := aiStarters(squad, xi, true)
	candidates := make([]*Player, 0, len(bench))
	for _, p := range bench {
		if BestCorePosition(*p) != GOALKEEPER {
			candidates = append(candidates, p)
		}
	}

	for len(subs) < aiFriendlyMaxSubs && len(candidates) > 0 && len(open) > 0 {
		pool := candidates
		if preferYoung {
			young := make([]*Player, 0, len(candidates))
			for _, p := range candidates {
				if p.Age <= aiFriendlyYoungPlayerAge {
					young = append(young, p)
				}
			}
			if len(young) > 0 {
				pool = young
			}
		}

		var bestBench *Player
		bestSlot, bestValue := -1, -1
		for _, b := range pool {
			for i, s := range open {
				v := PositionValue(s.position, *b)
				if bestBench == nil || v > bestValue ||
					(v == bestValue && (b.Id < bestBench.Id || (b.Id == bestBench.Id && s.player.Id < open[bestSlot].player.Id))) {
					bestBench, bestSlot, bestValue = b, i, v
				}
			}
		}
		subs = append(subs, LineupSubstitution{PlayerOutId: open[bestSlot].player.Id, PlayerInId: bestBench.Id, Minute: aiFriendlySubMinute})
		open = append(open[:bestSlot:bestSlot], open[bestSlot+1:]...)
		candidates = withoutPlayer(candidates, bestBench.Id)
	}
	return subs
}

func findPlayer(players []*Player, id int64) *Player {
	for _, p := range players {
		if p.Id == id {
			return p
		}
	}
	return nil
}

func withoutPlayer(players []*Player, id int64) []*Player {
	kept := make([]*Player, 0, len(players))
	for _, p := range players {
		if p.Id != id {
			kept = append(kept, p)
		}
	}
	return kept
}
