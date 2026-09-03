package engine

import "math/rand"

// ByeTeamId marks a bracket slot with no real opponent - it advances automatically without a
// tie being played.
const ByeTeamId int64 = 0

// NextPowerOfTwo returns the smallest power of two >= n (minimum 1).
func NextPowerOfTwo(n int) int {
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

// BuildFirstRoundPairings randomly draws a single-elimination bracket's first round from a
// list of entrants, padding with byes up to the next power of two. Byes are spread one per
// pair before any pair gets two real teams, so two byes never land in the same pair as long as
// there are at least as many pairs as byes (always true here, since byes < half the bracket).
func BuildFirstRoundPairings(teamIds []int64, r *rand.Rand) [][2]int64 {
	shuffled := append([]int64(nil), teamIds...)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	bracketSize := NextPowerOfTwo(len(shuffled))
	numPairs := bracketSize / 2
	numByes := bracketSize - len(shuffled)

	pairs := make([][2]int64, 0, numPairs)
	idx := 0
	for i := 0; i < numPairs; i++ {
		if i < numByes {
			pairs = append(pairs, [2]int64{shuffled[idx], ByeTeamId})
			idx++
		} else {
			pairs = append(pairs, [2]int64{shuffled[idx], shuffled[idx+1]})
			idx += 2
		}
	}
	return pairs
}

// BuildNextRoundPairings pairs up a completed round's winners, in slot order (slot 2i plays
// slot 2i+1), into the next round. len(winners) must be even - true whenever they came from a
// full round built by this same bracket logic.
func BuildNextRoundPairings(winnersBySlot []int64) [][2]int64 {
	pairs := make([][2]int64, 0, len(winnersBySlot)/2)
	for i := 0; i+1 < len(winnersBySlot); i += 2 {
		pairs = append(pairs, [2]int64{winnersBySlot[i], winnersBySlot[i+1]})
	}
	return pairs
}
