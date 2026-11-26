package common

import "sort"

// optimalAssignment assigns each row (a lineup slot) to a distinct column (a player) so the total
// cost is lowest, for len(cost) <= len(cost[0]) - the Hungarian algorithm with potentials,
// O(rows² x columns). It returns, for each row, the index of its column.
func optimalAssignment(cost [][]int64) []int {
	rows := len(cost)
	if rows == 0 {
		return nil
	}
	cols := len(cost[0])
	const inf = int64(1) << 62
	// 1-based potentials and matching, column 0 is the virtual start column.
	u := make([]int64, rows+1)
	v := make([]int64, cols+1)
	match := make([]int, cols+1) // match[j] = row assigned to column j (1-based), 0 when free
	way := make([]int, cols+1)
	for i := 1; i <= rows; i++ {
		match[0] = i
		j0 := 0
		minv := make([]int64, cols+1)
		used := make([]bool, cols+1)
		for j := range minv {
			minv[j] = inf
		}
		for {
			used[j0] = true
			i0, delta, j1 := match[j0], inf, 0
			for j := 1; j <= cols; j++ {
				if used[j] {
					continue
				}
				if cur := cost[i0-1][j-1] - u[i0] - v[j]; cur < minv[j] {
					minv[j], way[j] = cur, j0
				}
				if minv[j] < delta {
					delta, j1 = minv[j], j
				}
			}
			for j := 0; j <= cols; j++ {
				if used[j] {
					u[match[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if match[j0] == 0 {
				break
			}
		}
		for j0 != 0 {
			j1 := way[j0]
			match[j0] = match[j1]
			j0 = j1
		}
	}
	assignment := make([]int, rows)
	for j := 1; j <= cols; j++ {
		if match[j] != 0 {
			assignment[match[j]-1] = j - 1
		}
	}
	return assignment
}

// bestAssignmentLineup is the optimal best XI for slots (openspec cpu-team-management
// "Best XI by optimal assignment"): the players-to-slots assignment with the highest total
// PositionValue. Among equally valued assignments it fields the lowest-id players, so the result
// is deterministic whatever order players come in. Requires len(players) >= len(slots).
func bestAssignmentLineup(players []*Player, slots []int) map[int][]int64 {
	ordered := append([]*Player(nil), players...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Id < ordered[j].Id })
	// Maximize value first, then prefer earlier (lower-id) players: one value point outweighs any
	// combination of index penalties, which add up to at most len(slots) x len(ordered).
	scale := int64(len(slots)*len(ordered) + 1)
	cost := make([][]int64, len(slots))
	for s, position := range slots {
		cost[s] = make([]int64, len(ordered))
		for i, p := range ordered {
			cost[s][i] = -int64(PositionValue(position, *p))*scale + int64(i)
		}
	}
	lineup := make(map[int][]int64)
	for s, i := range optimalAssignment(cost) {
		lineup[slots[s]] = append(lineup[slots[s]], ordered[i].Id)
	}
	// Players sharing a position (two centre-backs) are listed best first, then by id.
	byId := make(map[int64]*Player, len(ordered))
	for _, p := range ordered {
		byId[p.Id] = p
	}
	for position, ids := range lineup {
		sort.Slice(ids, func(a, b int) bool {
			va, vb := PositionValue(position, *byId[ids[a]]), PositionValue(position, *byId[ids[b]])
			if va != vb {
				return va > vb
			}
			return ids[a] < ids[b]
		})
	}
	return lineup
}
