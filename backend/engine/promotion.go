package engine

import (
	"github.com/scruz84/striker/common"
	"sort"
)

// RelegationSlotsPerGroup is how many of a league's bottom teams are candidates to drop to
// the tier below at the end of a season.
const RelegationSlotsPerGroup = 2

// PromotionCandidatesPerGroup is how many of a league's top teams are candidates to move up
// to the tier above. When the tier below has more groups than the tier above has vacancies
// (as in this game's seed data: tier 2 has 4 groups feeding a single tier-1 group), only the
// best-ranked candidates across all of that tier's groups actually get promoted - there is no
// play-off model, so the rest simply stay put for another season.
const PromotionCandidatesPerGroup = 1

// LeagueStandings is one league's teams, already ranked best to worst.
type LeagueStandings struct {
	LeagueId  int64
	Standings []common.TeamStanding
}

// LeagueMove is a team's reassignment to a different league.
type LeagueMove struct {
	TeamId      int64
	NewLeagueId int64
	Promoted    bool // true if this move is a promotion, false if a relegation
}

type tierCandidate struct {
	team     common.TeamStanding
	leagueId int64
}

// ComputePromotionRelegation walks a country's tiers from the top down and, for every pair of
// adjacent tiers, swaps the bottom RelegationSlotsPerGroup of each upper-tier league with the
// best-ranked promotion candidates from the tier below - matched one-for-one so no league's
// team count ever changes. leaguesByTier must be keyed by tier number (1 = top division);
// only tiers that are actually adjacent (e.g. 1 and 2, not 1 and 3) form a pair.
func ComputePromotionRelegation(leaguesByTier map[int][]LeagueStandings) []LeagueMove {
	tiers := make([]int, 0, len(leaguesByTier))
	for tier := range leaguesByTier {
		tiers = append(tiers, tier)
	}
	sort.Ints(tiers)

	moves := make([]LeagueMove, 0)
	for i := 0; i < len(tiers)-1; i++ {
		upperTier, lowerTier := tiers[i], tiers[i+1]
		if lowerTier != upperTier+1 {
			continue
		}
		moves = append(moves, swapAdjacentTiers(leaguesByTier[upperTier], leaguesByTier[lowerTier])...)
	}
	return moves
}

func swapAdjacentTiers(upperLeagues []LeagueStandings, lowerLeagues []LeagueStandings) []LeagueMove {
	relegationSlots := make([]tierCandidate, 0)
	for _, league := range upperLeagues {
		n := RelegationSlotsPerGroup
		if n > len(league.Standings) {
			n = len(league.Standings)
		}
		// worst-first, so that if there aren't enough promotion candidates to fill every
		// relegation slot (see numMoves below), it's the genuinely worst teams that go down.
		bottom := league.Standings[len(league.Standings)-n:]
		for i := len(bottom) - 1; i >= 0; i-- {
			relegationSlots = append(relegationSlots, tierCandidate{team: bottom[i], leagueId: league.LeagueId})
		}
	}

	promotionCandidates := make([]tierCandidate, 0)
	for _, league := range lowerLeagues {
		n := PromotionCandidatesPerGroup
		if n > len(league.Standings) {
			n = len(league.Standings)
		}
		for _, team := range league.Standings[:n] {
			promotionCandidates = append(promotionCandidates, tierCandidate{team: team, leagueId: league.LeagueId})
		}
	}
	sort.SliceStable(promotionCandidates, func(i, j int) bool {
		a, b := promotionCandidates[i].team, promotionCandidates[j].team
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GoalDiff != b.GoalDiff {
			return a.GoalDiff > b.GoalDiff
		}
		return a.GoalsScored > b.GoalsScored
	})

	numMoves := len(relegationSlots)
	if len(promotionCandidates) < numMoves {
		numMoves = len(promotionCandidates)
	}

	moves := make([]LeagueMove, 0, numMoves*2)
	for i := 0; i < numMoves; i++ {
		promoted, relegated := promotionCandidates[i], relegationSlots[i]
		// each pair trades places: the promoted team takes over the relegated team's league,
		// and vice versa, so both leagues keep exactly the same number of teams.
		moves = append(moves, LeagueMove{TeamId: promoted.team.TeamId, NewLeagueId: relegated.leagueId, Promoted: true})
		moves = append(moves, LeagueMove{TeamId: relegated.team.TeamId, NewLeagueId: promoted.leagueId, Promoted: false})
	}
	return moves
}
