package engine

import "math/rand"

// ByeTeamId marks a bracket slot with no real opponent - it advances automatically without a
// tie being played.
const ByeTeamId int64 = 0

// BuildFirstRoundPairings draws a single-elimination bracket's first round from a list of
// entrants. When the field is already an exact power of two, every entrant gets a real opponent
// (pairSequentially never needs to hand out a bye for an even count). Otherwise this round
// doubles as a play-in/preliminary round: just enough real matches are drawn to trim the field
// down to the next power of two below it, so every round from here on is itself an exact power of
// two and never needs a bye again - see largestPowerOfTwoAtMost's own doc comment for why that
// needs more than one bye here, not just the single leftover-entrant bye a plain odd/even check
// would give. seedPoints (e.g. each entrant's current league classification points - see
// engine.classificationPointsForSeeding) is used to decide WHICH entrants draw the play-in's byes
// when there's more than one to hand out: the top seedPoints-ranked entrants get the free pass to
// the next round, same as a real tournament's higher seeds, while the rest are shuffled and play
// each other for the remaining spot(s). A nil/empty seedPoints (no ranking available for any
// entrant - the normal case for a World Cup knockout bracket's national teams, which have no
// league classification at all) falls back to handing the bye(s) to whichever entrants the random
// shuffle happens to leave over, the same as before seeding existed.
func BuildFirstRoundPairings(teamIds []int64, r *rand.Rand, seedPoints map[int64]int) [][2]int64 {
	shuffled := append([]int64(nil), teamIds...)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	n := len(shuffled)
	p := largestPowerOfTwoAtMost(n)
	if p == n {
		// Already an exact power of two (or 0/1 entrants) - every entrant gets a real opponent,
		// no play-in needed.
		return pairSequentially(shuffled, seedPoints)
	}

	byeCount := 2*p - n
	byeTeams, matchTeams := splitByeCandidates(shuffled, byeCount, seedPoints)

	pairs := make([][2]int64, 0, len(byeTeams)+len(matchTeams)/2)
	for _, team := range byeTeams {
		pairs = append(pairs, [2]int64{team, ByeTeamId})
	}
	for i := 0; i+1 < len(matchTeams); i += 2 {
		pairs = append(pairs, [2]int64{matchTeams[i], matchTeams[i+1]})
	}
	return pairs
}

// largestPowerOfTwoAtMost returns the largest power of two <= n (0 for n <= 0), by repeated
// doubling rather than a bit trick - the field sizes this ever runs against (a handful of teams up
// to, at most, a few hundred) make the extra clarity worth far more than the handful of saved
// iterations.
func largestPowerOfTwoAtMost(n int) int {
	if n <= 0 {
		return 0
	}
	p := 1
	for p*2 <= n {
		p *= 2
	}
	return p
}

// splitByeCandidates picks byeCount entrants out of shuffled to receive a play-in bye, preferring
// (when seedPoints has any data) whichever entrants rank highest by seedPoints - ties, and any
// entrant seedPoints has no data for, keep shuffled's existing random order, so a seedless or
// partially-seeded field still degrades gracefully to "whoever the shuffle left over" rather than
// erroring or panicking. Returns the chosen byeCount entrants first, then the rest (still in their
// relative shuffled order) to pair off against each other.
func splitByeCandidates(shuffled []int64, byeCount int, seedPoints map[int64]int) (byeTeams []int64, matchTeams []int64) {
	if byeCount <= 0 {
		return nil, shuffled
	}
	ordered := shuffled
	if len(seedPoints) > 0 {
		ordered = append([]int64(nil), shuffled...)
		// A stable sort keeps the shuffle's own order as the tiebreak for any two entrants with
		// equal (or equally absent) seed points, rather than introducing a second, hidden source
		// of randomness on top of the shuffle that already ran.
		stableSortDescByPoints(ordered, seedPoints)
	}
	return ordered[:byeCount], ordered[byeCount:]
}

// stableSortDescByPoints sorts ids in place, highest seedPoints first - a missing entry counts as
// 0, the lowest possible rank, same as a team with no classification row at all (see
// classificationPointsForSeeding's own doc comment).
func stableSortDescByPoints(ids []int64, seedPoints map[int64]int) {
	// Insertion sort: ids is always small (a cup's entrant count), and being stable without
	// pulling in sort.SliceStable's reflection-based overhead isn't worth it at this size.
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && seedPoints[ids[j]] > seedPoints[ids[j-1]]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
}

// BuildNextRoundPairings pairs up a completed round's winners for the next round, in slot order
// (slot 2i plays slot 2i+1) - see pairSequentially. Once a bracket's first round has trimmed the
// field to a power of two (see BuildFirstRoundPairings), every later round's winner count is
// itself already a power of two and this never needs to hand out a bye - the odd-count branch
// below only still matters for a cup whose first round was built before that trimming existed
// (an already-in-progress cup at the time this shipped).
func BuildNextRoundPairings(winnersBySlot []int64, seedPoints map[int64]int) [][2]int64 {
	return pairSequentially(winnersBySlot, seedPoints)
}

// pairSequentially pairs up entrants two at a time in list order, giving the leftover entrant a
// bye if the count is odd - rather than padding the whole field up to the next power of two the
// way a traditional fixed-size bracket (round of 16, round of 8, ...) would. That padding was the
// root cause of issue #20 ("no cup matches"): with the default 10 tier-1 teams per country,
// rounding up to a 16-slot bracket meant 6 of the 10 teams got an automatic, unplayed bye and
// only 2 real matches happened in the entire opening round. Pairing sequentially instead needs
// exactly one bye per round at most (only when the field is odd), so with 10 teams every single
// team plays a real match immediately. When seedPoints has data for the leftover candidates, the
// single bye goes to whichever entrant ranks highest by it rather than whoever the incoming order
// happened to leave last.
func pairSequentially(teamIds []int64, seedPoints map[int64]int) [][2]int64 {
	ids := teamIds
	if len(ids)%2 == 1 && len(seedPoints) > 0 {
		ids = append([]int64(nil), teamIds...)
		byeIdx := len(ids) - 1
		for i := 0; i < len(ids)-1; i++ {
			if seedPoints[ids[i]] > seedPoints[ids[byeIdx]] {
				byeIdx = i
			}
		}
		ids[byeIdx], ids[len(ids)-1] = ids[len(ids)-1], ids[byeIdx]
	}

	pairs := make([][2]int64, 0, (len(ids)+1)/2)
	i := 0
	for ; i+1 < len(ids); i += 2 {
		pairs = append(pairs, [2]int64{ids[i], ids[i+1]})
	}
	if i < len(ids) {
		pairs = append(pairs, [2]int64{ids[i], ByeTeamId})
	}
	return pairs
}
