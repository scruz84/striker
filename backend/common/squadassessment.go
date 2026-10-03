package common

// CorePositions is every position SquadStrength/AssessPositionNeeds groups player depth by - the
// same four BestCorePosition ever returns, deliberately excluding every lateral position (a
// winger or full-back's value is folded into MIDFIELD/DEFENSE by GetLateralMidfieldValue/
// GetLateralDefenseValue, and STRIKER covers lateral-striker-capable players too - see
// PositionValue).
var CorePositions = []int{GOALKEEPER, DEFENSE, MIDFIELD, STRIKER}

// SquadStrength is an unmanaged team's overall quality - the sum of PositionValue for each slot
// in its best-XI (see SelectBestLineup), so squad depth beyond the starting XI doesn't inflate it
// and an understaffed team (fewer than 11 fillable slots) scores lower. Used by AI-managed teams
// (see RunAITeamManagement, added in a later phase) to gauge their own ambition - e.g. whether
// they can compete for a transfer target or should be looking to sell instead.
func SquadStrength(players []*Player) int {
	lineup := SelectBestLineup(players)
	byId := make(map[int64]*Player, len(players))
	for _, p := range players {
		byId[p.Id] = p
	}
	total := 0
	for position, ids := range lineup {
		for _, id := range ids {
			if p, ok := byId[id]; ok {
				total += PositionValue(position, *p)
			}
		}
	}
	return total
}

// MinPositionDepth is the fewest squad players an AI-managed team should have whose best core
// position (see BestCorePosition) is a given slot before AssessPositionNeeds flags it as thin -
// one full XI's worth of that slot plus a single backup: 2 for goalkeeper (FormationBalanced, the
// shape SquadStrength's SelectBestLineup always builds, only ever fills one), 3 for defense/
// midfield/striker (FormationBalanced fills two of each).
var MinPositionDepth = map[int]int{
	GOALKEEPER: 2,
	DEFENSE:    3,
	MIDFIELD:   3,
	STRIKER:    3,
}

// PositionNeed is how badly an AI-managed team could use reinforcement at one core position - see
// AssessPositionNeeds.
type PositionNeed struct {
	Position int
	// Depth is how many squad players have this as their BestCorePosition.
	Depth int
	// AvgValue is the average PositionValue of those players at this position, 0 if Depth is 0.
	AvgValue int
	// Thin is true when Depth is below MinPositionDepth for this position.
	Thin bool
}

// AssessPositionNeeds groups the squad by BestCorePosition and reports depth/quality for each of
// CorePositions - always four entries, one per core position, even when a squad has nobody at a
// given one (Depth 0, AvgValue 0, Thin true). Used by Level 2 AI-managed teams to decide whether
// to sign an available free agent, and Level 3 to pick a transfer-shopping target position - see
// WeakestPositionNeed and RunAITeamManagement (added in a later phase).
func AssessPositionNeeds(players []*Player) []PositionNeed {
	needs := make([]PositionNeed, 0, len(CorePositions))
	for _, position := range CorePositions {
		depth := 0
		totalValue := 0
		for _, p := range players {
			if BestCorePosition(*p) == position {
				depth++
				totalValue += PositionValue(position, *p)
			}
		}
		avgValue := 0
		if depth > 0 {
			avgValue = totalValue / depth
		}
		needs = append(needs, PositionNeed{
			Position: position,
			Depth:    depth,
			AvgValue: avgValue,
			Thin:     depth < MinPositionDepth[position],
		})
	}
	return needs
}

// WeakestPositionNeed picks the single neediest core position out of AssessPositionNeeds - the
// biggest depth deficit against MinPositionDepth wins, ties broken by lowest average quality -
// or nil if the squad has no players at all. Used by Level 3 AI-managed teams to choose which
// position to shop the transfer market for.
func WeakestPositionNeed(players []*Player) *PositionNeed {
	if len(players) == 0 {
		return nil
	}
	needs := AssessPositionNeeds(players)
	weakest := needs[0]
	for _, n := range needs[1:] {
		if positionNeedDeficit(n) != positionNeedDeficit(weakest) {
			if positionNeedDeficit(n) > positionNeedDeficit(weakest) {
				weakest = n
			}
			continue
		}
		if n.AvgValue < weakest.AvgValue {
			weakest = n
		}
	}
	return &weakest
}

func positionNeedDeficit(n PositionNeed) int {
	return MinPositionDepth[n.Position] - n.Depth
}
