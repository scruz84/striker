package engine

import (
	"sort"

	"github.com/scruz84/striker/common"
)

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
	ViaPlayoff  bool // the move came from a promotion playoff tie rather than a direct place
	// Lateral is a restructure's move to another group of the same tier (see applyRestructure):
	// neither a promotion nor a relegation.
	Lateral bool
}

// PlayoffPairing is one promotion playoff tie: an upper-tier team from a relegation-playoff place,
// at home, against a lower-tier runner-up (see openspec/specs/league-pyramid "Playoff pairings").
type PlayoffPairing struct {
	UpperTier    int
	HomeTeamId   int64
	HomeLeagueId int64
	AwayTeamId   int64
	AwayLeagueId int64
}

// PlayoffResult is a played playoff tie: its pairing plus the winner.
type PlayoffResult struct {
	PlayoffPairing
	WinnerTeamId int64
}

type tierCandidate struct {
	team     common.TeamStanding
	position int // 1-based final position in its own league
	leagueId int64
}

// adjacentTierPairs returns every (upper, lower) pair of tiers in leaguesByTier that are actually
// adjacent (e.g. 1 and 2, not 1 and 3), top down, plus the deepest tier.
func adjacentTierPairs(leaguesByTier map[int][]LeagueStandings) ([][2]int, int) {
	tiers := make([]int, 0, len(leaguesByTier))
	for tier := range leaguesByTier {
		tiers = append(tiers, tier)
	}
	sort.Ints(tiers)
	pairs := make([][2]int, 0)
	for i := 0; i < len(tiers)-1; i++ {
		if tiers[i+1] == tiers[i]+1 {
			pairs = append(pairs, [2]int{tiers[i], tiers[i+1]})
		}
	}
	deepest := 0
	if len(tiers) > 0 {
		deepest = tiers[len(tiers)-1]
	}
	return pairs, deepest
}

// bestFirst orders candidates by points, then goal difference, then goals scored, best first.
func bestFirst(candidates []tierCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i].team, candidates[j].team
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GoalDiff != b.GoalDiff {
			return a.GoalDiff > b.GoalDiff
		}
		if a.GoalsScored != b.GoalsScored {
			return a.GoalsScored > b.GoalsScored
		}
		return a.TeamId < b.TeamId
	})
}

// worstFirst orders candidates by final position (highest, i.e. worst, first), then fewest
// points, then goal difference and goals scored, worst first.
func worstFirst(candidates []tierCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.position != b.position {
			return a.position > b.position
		}
		sa, sb := a.team, b.team
		if sa.Points != sb.Points {
			return sa.Points < sb.Points
		}
		if sa.GoalDiff != sb.GoalDiff {
			return sa.GoalDiff < sb.GoalDiff
		}
		if sa.GoalsScored != sb.GoalsScored {
			return sa.GoalsScored < sb.GoalsScored
		}
		return sa.TeamId > sb.TeamId
	})
}

// zonePlaces collects, from every league of one tier, the teams whose 1-based final position falls
// in [from, to] counted from the top of the table (fromBottom=false) or from its bottom.
func zonePlaces(leagues []LeagueStandings, places func(n int) (from, to int)) []tierCandidate {
	out := make([]tierCandidate, 0)
	for _, league := range leagues {
		n := len(league.Standings)
		from, to := places(n)
		for pos := from; pos <= to; pos++ {
			if pos < 1 || pos > n {
				continue
			}
			out = append(out, tierCandidate{team: league.Standings[pos-1], position: pos, leagueId: league.LeagueId})
		}
	}
	return out
}

func swapMoves(promoted, relegated []tierCandidate, viaPlayoff bool) []LeagueMove {
	n := len(relegated)
	if len(promoted) < n {
		n = len(promoted)
	}
	moves := make([]LeagueMove, 0, n*2)
	for i := 0; i < n; i++ {
		up, down := promoted[i], relegated[i]
		// each pair trades places, so both leagues keep exactly the same number of teams.
		moves = append(moves, LeagueMove{TeamId: up.team.TeamId, NewLeagueId: down.leagueId, Promoted: true, ViaPlayoff: viaPlayoff})
		moves = append(moves, LeagueMove{TeamId: down.team.TeamId, NewLeagueId: up.leagueId, Promoted: false, ViaPlayoff: viaPlayoff})
	}
	return moves
}

// ComputeDirectMoves returns the direct promotion/relegation moves of a pyramid that follows the
// ratio rule (see openspec/specs/league-pyramid "Direct moves"): for every pair of adjacent tiers,
// the lower tier's group winners, best first, swap leagues one for one with the upper tier's
// direct-relegation teams, worst first. leaguesByTier must be keyed by tier number (1 = top) and
// hold every league's final standings, best to worst. Pure.
func ComputeDirectMoves(leaguesByTier map[int][]LeagueStandings) []LeagueMove {
	pairs, deepest := adjacentTierPairs(leaguesByTier)
	moves := make([]LeagueMove, 0)
	for _, pair := range pairs {
		upper, lower := pair[0], pair[1]
		relegated := zonePlaces(leaguesByTier[upper], func(n int) (int, int) {
			z := common.PyramidZones(upper, deepest, n, false)
			return n - z.DirectDown + 1, n
		})
		promoted := zonePlaces(leaguesByTier[lower], func(n int) (int, int) {
			z := common.PyramidZones(lower, deepest, n, false)
			return 1, z.DirectUp
		})
		worstFirst(relegated)
		bestFirst(promoted)
		moves = append(moves, swapMoves(promoted, relegated, false)...)
	}
	return moves
}

// ComputePlayoffPairings returns the promotion playoff ties of a pyramid that follows the ratio
// rule (see openspec/specs/league-pyramid "Playoff pairings"): for every pair of adjacent tiers, the
// upper tier's relegation-playoff teams, worst first, against the lower tier's runners-up, best
// first, the upper-tier team at home. Pure.
func ComputePlayoffPairings(leaguesByTier map[int][]LeagueStandings) []PlayoffPairing {
	pairs, deepest := adjacentTierPairs(leaguesByTier)
	out := make([]PlayoffPairing, 0)
	for _, pair := range pairs {
		upper, lower := pair[0], pair[1]
		upperTeams := zonePlaces(leaguesByTier[upper], func(n int) (int, int) {
			z := common.PyramidZones(upper, deepest, n, false)
			return n - z.DirectDown - z.PlayoffDown + 1, n - z.DirectDown
		})
		lowerTeams := zonePlaces(leaguesByTier[lower], func(n int) (int, int) {
			z := common.PyramidZones(lower, deepest, n, false)
			return z.DirectUp + 1, z.DirectUp + z.PlayoffUp
		})
		worstFirst(upperTeams)
		bestFirst(lowerTeams)
		n := len(upperTeams)
		if len(lowerTeams) < n {
			n = len(lowerTeams)
		}
		for i := 0; i < n; i++ {
			out = append(out, PlayoffPairing{
				UpperTier:    upper,
				HomeTeamId:   upperTeams[i].team.TeamId,
				HomeLeagueId: upperTeams[i].leagueId,
				AwayTeamId:   lowerTeams[i].team.TeamId,
				AwayLeagueId: lowerTeams[i].leagueId,
			})
		}
	}
	return out
}

// ApplyPlayoffOutcomes turns played playoff ties into moves (see openspec/specs/league-pyramid
// "Playoff outcome"): when the lower-tier (away) team won, the two teams swap leagues; when the
// upper-tier team won, both stay. Pure.
func ApplyPlayoffOutcomes(results []PlayoffResult) []LeagueMove {
	moves := make([]LeagueMove, 0)
	for _, r := range results {
		if r.WinnerTeamId != r.AwayTeamId {
			continue
		}
		moves = append(moves,
			LeagueMove{TeamId: r.AwayTeamId, NewLeagueId: r.HomeLeagueId, Promoted: true, ViaPlayoff: true},
			LeagueMove{TeamId: r.HomeTeamId, NewLeagueId: r.AwayLeagueId, Promoted: false, ViaPlayoff: true},
		)
	}
	return moves
}

// computeLegacyMoves is the one-for-one swap a pyramid that breaks the ratio rule still plays (see
// openspec/specs/league-pyramid "Legacy pyramid season"): for every pair of adjacent tiers, the
// bottom LegacyRelegationPerGroup of each upper league, worst first, swap with the best
// LegacyCandidatesPerGroup winners of every lower league, ranked across groups by points, goal
// difference and goals scored. With fewer candidates than relegation places, the worst teams are
// the ones that go down. Pure.
func computeLegacyMoves(leaguesByTier map[int][]LeagueStandings) []LeagueMove {
	pairs, _ := adjacentTierPairs(leaguesByTier)
	moves := make([]LeagueMove, 0)
	for _, pair := range pairs {
		upper, lower := pair[0], pair[1]
		relegated := zonePlaces(leaguesByTier[upper], func(n int) (int, int) {
			return n - common.LegacyRelegationPerGroup + 1, n
		})
		candidates := zonePlaces(leaguesByTier[lower], func(n int) (int, int) {
			return 1, common.LegacyCandidatesPerGroup
		})
		worstFirst(relegated)
		bestFirst(candidates)
		moves = append(moves, swapMoves(candidates, relegated, false)...)
	}
	return moves
}

// ComputeSeasonMoves is every promotion/relegation move of a country's season: the direct moves
// plus the playoff outcomes for a pyramid that follows the ratio rule, or the legacy swap for one
// that does not (which plays no playoff). Pure.
func ComputeSeasonMoves(leaguesByTier map[int][]LeagueStandings, legacy bool, playoff []PlayoffResult) []LeagueMove {
	if legacy {
		return computeLegacyMoves(leaguesByTier)
	}
	moves := ComputeDirectMoves(leaguesByTier)
	return append(moves, ApplyPlayoffOutcomes(playoff)...)
}
