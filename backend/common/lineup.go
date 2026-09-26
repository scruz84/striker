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

// PositionValue scores how good a player is for one tactical position. Used both to build a
// team's best-XI lineup (see bestLineupSlots/SelectBestLineup, which only ever fills GOALKEEPER,
// DEFENSE, MIDFIELD, STRIKER, and the four LATERAL_* positions except the lateral strikers) and,
// unlike that fixed slot list, covers every position constant including LATERAL_STRIKER_LEFT/
// RIGHT - a human-edited lineup (see handlers.validateLineup) can use those, so the match-day
// backfill that finds a replacement for a suspended/injured starter (see engine's
// excludeUnavailablePlayers/backfillEmptyPositions) needs a value function for them too.
func PositionValue(position int, p Player) int {
	switch position {
	case GOALKEEPER:
		return p.GoalKeeper
	case DEFENSE:
		return p.Defense
	case LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT:
		return GetLateralDefenseValue(p)
	case MIDFIELD:
		return GetMidfieldValue(p)
	case LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT:
		return GetLateralMidfieldValue(p)
	case STRIKER:
		return GetStrikerValue(p)
	case LATERAL_STRIKER_LEFT, LATERAL_STRIKER_RIGHT:
		return GetLateralStrikerValue(p)
	default:
		return 0
	}
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

// BestLineupForTeam computes teamId's current strongest available XI fresh from its full squad
// (see SelectBestLineup), rather than reading back whatever was last saved as the team's standing
// default lineup - used to give a friendly a sensible starting XI by default (see
// GetLineupsForFriendly/handlers.GetFriendlyLineupHandler) without requiring the manager to have
// set up, or kept up to date, a default lineup first. Substitutions are deliberately left empty:
// the caller decides whether to attach any (see GetFriendlyLineupSubstitutions).
func BestLineupForTeam(dbCtx *DatabaseContext, teamId int64) (*Lineup, error) {
	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	return &Lineup{
		TeamId:        teamId,
		Formation:     FormationBalanced,
		Positions:     FlattenLineup(SelectBestLineup(*players)),
		Substitutions: make([]LineupSubstitution, 0),
	}, nil
}
