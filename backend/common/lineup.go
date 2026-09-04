package common

// bestLineupSlot is one position to fill when auto-picking a team's best available XI - see
// SelectBestLineup. Two players can't share a slot, so a doubled-up position (central defense,
// central midfield, striker) appears as two separate entries.
type bestLineupSlot struct {
	position int
	value    func(Player) int
}

var bestLineupSlots = []bestLineupSlot{
	{GOALKEEPER, func(p Player) int { return p.GoalKeeper }},
	{LATERAL_DEFENSE_LEFT, GetLateralDefenseValue},
	{DEFENSE, func(p Player) int { return p.Defense }},
	{DEFENSE, func(p Player) int { return p.Defense }},
	{LATERAL_DEFENSE_RIGHT, GetLateralDefenseValue},
	{LATERAL_MIDFIELD_LEFT, GetLateralMidfieldValue},
	{MIDFIELD, GetMidfieldValue},
	{MIDFIELD, GetMidfieldValue},
	{LATERAL_MIDFIELD_RIGHT, GetLateralMidfieldValue},
	{STRIKER, GetStrikerValue},
	{STRIKER, GetStrikerValue},
}

// SelectBestLineup greedily fills each position slot (see bestLineupSlots) with the
// highest-value remaining player, so nobody is picked twice - a simple stand-in for a manager's
// judgement. Used both to seed a brand new team's default lineup (see AssignTeam and the
// initializer's createTeam) and to keep an unmanaged team (no owner) fielding its best available
// players as its squad changes over time - see engine.ProgressPlayers.
//
// If players has fewer than len(bestLineupSlots) entries, the remaining slots are simply left
// unfilled rather than erroring - the match simulation already tolerates an understaffed
// position group (see excludeUnavailablePlayers and its callers).
func SelectBestLineup(players []*Player) map[int][]int64 {
	lineup := make(map[int][]int64)
	pool := append([]*Player(nil), players...)

	for _, slot := range bestLineupSlots {
		if len(pool) == 0 {
			continue
		}
		bestIdx := 0
		bestValue := slot.value(*pool[0])
		for i := 1; i < len(pool); i++ {
			if v := slot.value(*pool[i]); v > bestValue {
				bestIdx = i
				bestValue = v
			}
		}
		lineup[slot.position] = append(lineup[slot.position], pool[bestIdx].Id)
		pool[bestIdx] = pool[len(pool)-1]
		pool = pool[:len(pool)-1]
	}

	return lineup
}

// FlattenLineup converts a position->players map (as returned by SelectBestLineup) into the
// flat slice SetDefaultLineup expects.
func FlattenLineup(lineup map[int][]int64) []LineupPosition {
	positions := make([]LineupPosition, 0, len(bestLineupSlots))
	for position, playerIds := range lineup {
		for _, playerId := range playerIds {
			positions = append(positions, LineupPosition{PlayerId: playerId, Position: position})
		}
	}
	return positions
}
