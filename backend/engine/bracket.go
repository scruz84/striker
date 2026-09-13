package engine

import "math/rand"

// ByeTeamId marks a bracket slot with no real opponent - it advances automatically without a
// tie being played.
const ByeTeamId int64 = 0

// BuildFirstRoundPairings randomly draws a single-elimination bracket's first round from a list
// of entrants, pairing as many real matches as the field allows and giving a bye to one leftover
// team if the count is odd - see pairSequentially for why this doesn't pad the whole field up to
// a fixed bracket size (e.g. 16) the way a traditional bracket poster would.
func BuildFirstRoundPairings(teamIds []int64, r *rand.Rand) [][2]int64 {
	shuffled := append([]int64(nil), teamIds...)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return pairSequentially(shuffled)
}

// BuildNextRoundPairings pairs up a completed round's winners for the next round, in slot order
// (slot 2i plays slot 2i+1) - see pairSequentially.
func BuildNextRoundPairings(winnersBySlot []int64) [][2]int64 {
	return pairSequentially(winnersBySlot)
}

// pairSequentially pairs up entrants two at a time in list order, giving the leftover entrant a
// bye if the count is odd - rather than padding the whole field up to the next power of two the
// way a traditional fixed-size bracket (round of 16, round of 8, ...) would. That padding was the
// root cause of issue #20 ("no cup matches"): with the default 10 tier-1 teams per country,
// rounding up to a 16-slot bracket meant 6 of the 10 teams got an automatic, unplayed bye and
// only 2 real matches happened in the entire opening round. Pairing sequentially instead needs
// exactly one bye per round at most (only when the field is odd), so with 10 teams every single
// team plays a real match immediately; the very next round (5 winners, odd) needs just one bye,
// and so on until a single champion remains - still exactly N-1 matches in total either way, the
// same as any single-elimination format, just with far fewer teams sitting out early rounds.
func pairSequentially(teamIds []int64) [][2]int64 {
	pairs := make([][2]int64, 0, (len(teamIds)+1)/2)
	i := 0
	for ; i+1 < len(teamIds); i += 2 {
		pairs = append(pairs, [2]int64{teamIds[i], teamIds[i+1]})
	}
	if i < len(teamIds) {
		pairs = append(pairs, [2]int64{teamIds[i], ByeTeamId})
	}
	return pairs
}
