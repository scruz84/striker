package engine

import (
	"math/rand"
	"sort"

	"github.com/scruz84/striker/common"
)

// regionalCupFirstRound draws a regional cup edition's first round (openspec/specs/international-cup
// "Regional cup draw"). With N entrants and P the largest power of two <= N, 2P - N of them get a
// bye when N != P: champions first, then runners-up, each by points (current league points, see
// classificationPointsForSeeding), ties in random order - unless that leaves a same-country tie
// that handing one bye to another team of the same position avoids. The others are paired champion against a
// runner-up of another country where possible, never two teams of one country while another
// pairing avoids it. Byes alternate with real ties in slot order, so - since the next round pairs
// slot 2i with 2i+1 (BuildNextRoundPairings) - two bye holders never meet in round 2 while there
// are real ties left to pair them with. A bye is [team, ByeTeamId]; in a real tie the champion (or
// the side drawn first) is at home. Pure, so the draw is unit-tested without a database.
func regionalCupFirstRound(entrants []common.RegionalCupEntrant, points map[int64]int, r *rand.Rand) [][2]int64 {
	if len(entrants) == 0 {
		return nil
	}
	order := append([]common.RegionalCupEntrant(nil), entrants...)
	r.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	sort.SliceStable(order, func(i, j int) bool {
		ci, cj := order[i].FinalPosition == 1, order[j].FinalPosition == 1
		if ci != cj {
			return ci
		}
		return points[order[i].TeamId] > points[order[j].TeamId]
	})

	n := len(order)
	byeCount := 0
	if p := largestPowerOfTwoAtMost(n); p != n {
		byeCount = 2*p - n
	}
	byes, rest := order[:byeCount], order[byeCount:]
	country := make(map[int64]string, n)
	for _, e := range order {
		country[e.TeamId] = e.Country
	}
	ties := pairRegionalCupTeams(rest, r)
	// A same-country tie the rest cannot avoid: hand a bye (lowest ranked first) to a team of the
	// same position that leaves a draw without one, when such a swap exists.
	for bi := len(byes) - 1; bi >= 0 && hasSameCountryTie(ties, country); bi-- {
		for ri := range rest {
			if rest[ri].FinalPosition != byes[bi].FinalPosition {
				continue
			}
			candidate := append([]common.RegionalCupEntrant(nil), rest...)
			candidate[ri] = byes[bi]
			if candidateTies := pairRegionalCupTeams(candidate, r); !hasSameCountryTie(candidateTies, country) {
				byes[bi], rest[ri] = rest[ri], byes[bi]
				ties = candidateTies
				break
			}
		}
	}

	pairs := make([][2]int64, 0, len(byes)+len(ties))
	for i := 0; i < len(byes) || i < len(ties); i++ {
		if i < len(byes) {
			pairs = append(pairs, [2]int64{byes[i].TeamId, ByeTeamId})
		}
		if i < len(ties) {
			pairs = append(pairs, ties[i])
		}
	}
	return pairs
}

// pairRegionalCupTeams pairs every team of rest (an even number, after the byes): each champion
// takes a runner-up of another country, leftover teams pair with each other avoiding one country,
// and a final pass swaps opponents between ties to undo any same-country tie a swap can undo. An odd
// leftover (only for a single entrant, which never starts an edition) gets a bye.
func pairRegionalCupTeams(rest []common.RegionalCupEntrant, r *rand.Rand) [][2]int64 {
	var champions, runnersUp []common.RegionalCupEntrant
	for _, e := range rest {
		if e.FinalPosition == 1 {
			champions = append(champions, e)
		} else {
			runnersUp = append(runnersUp, e)
		}
	}
	r.Shuffle(len(champions), func(i, j int) { champions[i], champions[j] = champions[j], champions[i] })
	r.Shuffle(len(runnersUp), func(i, j int) { runnersUp[i], runnersUp[j] = runnersUp[j], runnersUp[i] })

	ties := make([][2]common.RegionalCupEntrant, 0, len(rest)/2)
	used := make([]bool, len(runnersUp))
	leftover := make([]common.RegionalCupEntrant, 0)
	for _, champion := range champions {
		pick := -1
		for i, runnerUp := range runnersUp {
			if !used[i] && runnerUp.Country != champion.Country {
				pick = i
				break
			}
		}
		if pick < 0 {
			for i := range runnersUp {
				if !used[i] {
					pick = i
					break
				}
			}
		}
		if pick < 0 {
			leftover = append(leftover, champion)
			continue
		}
		used[pick] = true
		ties = append(ties, [2]common.RegionalCupEntrant{champion, runnersUp[pick]})
	}
	for i, runnerUp := range runnersUp {
		if !used[i] {
			leftover = append(leftover, runnerUp)
		}
	}

	var bye *common.RegionalCupEntrant
	for len(leftover) > 0 {
		first := leftover[0]
		leftover = leftover[1:]
		if len(leftover) == 0 {
			bye = &first
			break
		}
		pick := 0
		for i, other := range leftover {
			if other.Country != first.Country {
				pick = i
				break
			}
		}
		ties = append(ties, [2]common.RegionalCupEntrant{first, leftover[pick]})
		leftover = append(leftover[:pick], leftover[pick+1:]...)
	}

	separateSameCountryTies(ties)

	pairs := make([][2]int64, 0, len(ties)+1)
	for _, tie := range ties {
		pairs = append(pairs, [2]int64{tie[0].TeamId, tie[1].TeamId})
	}
	if bye != nil {
		pairs = append(pairs, [2]int64{bye.TeamId, ByeTeamId})
	}
	return pairs
}

// separateSameCountryTies swaps opponents between two ties while a tie has two teams of one country
// and a swap fixes it without creating another one. It first tries swapping the away sides (keeping
// every champion against a runner-up), then home for away.
func separateSameCountryTies(ties [][2]common.RegionalCupEntrant) {
	differ := func(a, b common.RegionalCupEntrant) bool { return a.Country != b.Country }
	for i := range ties {
		if differ(ties[i][0], ties[i][1]) {
			continue
		}
		for j := range ties {
			if j == i {
				continue
			}
			if differ(ties[i][0], ties[j][1]) && differ(ties[j][0], ties[i][1]) {
				ties[i][1], ties[j][1] = ties[j][1], ties[i][1]
				break
			}
			if differ(ties[i][0], ties[j][0]) && differ(ties[i][1], ties[j][1]) {
				ties[i][1], ties[j][0] = ties[j][0], ties[i][1]
				break
			}
		}
	}
}

// hasSameCountryTie reports whether any real tie of pairs has two teams of one country.
func hasSameCountryTie(pairs [][2]int64, country map[int64]string) bool {
	for _, pair := range pairs {
		if pair[1] != ByeTeamId && country[pair[0]] == country[pair[1]] {
			return true
		}
	}
	return false
}
