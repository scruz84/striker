package engine

import (
	"sort"

	"github.com/scruz84/striker/common"
)

// Seed groups of a new season's starting order: teams relegated into a league open it, then the
// teams that stayed, then the teams promoted into it (see openspec league-page "Starting order of a
// new season").
const (
	seedGroupRelegated = iota
	seedGroupStayed
	seedGroupPromoted
)

type seedEntry struct {
	group    int
	position int // final position last season, in the league the team finished in
	team     common.TeamStanding
}

// ComputeSeasonSeeds builds every league's starting order for the new season from the final
// ranked standings of the season just finished (best to worst, as GetRankedClassification returns
// them) and the promotion/relegation moves applied to them. It returns, per league the teams play
// in next season, each team's seed 1..N: teams relegated into the league first, then the teams that
// stayed, then the teams promoted into it. Each group is ordered by final position, then points,
// goal difference and goals scored (the promotion-candidate ordering of swapAdjacentTiers), so of
// two group winners promoted into one league the better one starts second to last. A team that is
// in none of the standings gets no seed and sorts after all three groups. Pure: no database access.
func ComputeSeasonSeeds(leagues []LeagueStandings, moves []LeagueMove) map[int64]map[int64]int {
	moveByTeam := make(map[int64]LeagueMove, len(moves))
	for _, move := range moves {
		moveByTeam[move.TeamId] = move
	}

	entriesByLeague := make(map[int64][]seedEntry)
	for _, league := range leagues {
		for i, team := range league.Standings {
			entry := seedEntry{group: seedGroupStayed, position: i + 1, team: team}
			newLeague := league.LeagueId
			if move, ok := moveByTeam[team.TeamId]; ok {
				newLeague = move.NewLeagueId
				entry.group = seedGroupRelegated
				if move.Promoted {
					entry.group = seedGroupPromoted
				} else if move.Lateral {
					entry.group = seedGroupStayed
				}
			}
			entriesByLeague[newLeague] = append(entriesByLeague[newLeague], entry)
		}
	}

	seeds := make(map[int64]map[int64]int, len(entriesByLeague))
	for leagueId, entries := range entriesByLeague {
		sort.SliceStable(entries, func(i, j int) bool {
			a, b := entries[i], entries[j]
			if a.group != b.group {
				return a.group < b.group
			}
			if a.position != b.position {
				return a.position < b.position
			}
			sa, sb := a.team, b.team
			if sa.Points != sb.Points {
				return sa.Points > sb.Points
			}
			if sa.GoalDiff != sb.GoalDiff {
				return sa.GoalDiff > sb.GoalDiff
			}
			if sa.GoalsScored != sb.GoalsScored {
				return sa.GoalsScored > sb.GoalsScored
			}
			return sa.TeamId < sb.TeamId
		})
		leagueSeeds := make(map[int64]int, len(entries))
		for i, entry := range entries {
			leagueSeeds[entry.team.TeamId] = i + 1
		}
		seeds[leagueId] = leagueSeeds
	}
	return seeds
}
