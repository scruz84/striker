package common

// ProjectedReplacement tells a team owner who the pre-kickoff refill would put in an unavailable
// starter's slot if the match kicked off now (see ProjectKickoffReplacements). It is shown next to
// the unavailable player in the lineup editor; the saved lineup itself is never changed.
type ProjectedReplacement struct {
	PlayerOutId int64 `json:"playerOutId"`
	PlayerInId  int64 `json:"playerInId"`
	Position    int   `json:"position"`
}

// ExcludeUnavailableAndBackfill is the pre-kickoff step of a match: it removes suspended/injured
// players from a team's active position groups, then backfills each thinned position back up to
// its original headcount from the rest of the squad (see BackfillEmptyPositions) - the closest
// thing this simulation has to a substitute coming off the bench. Goalkeepers are exempt from the
// exclusion itself even though nothing in this codebase currently can flag one unavailable (see
// engine.DisciplineSimulation/InjurySimulation) - if it ever did, an empty GOALKEEPER slice would
// panic elsewhere (e.g. ScorerSimulation's goalkeeper value lookup), so this is cheap, deliberate
// insurance; the backfill pass, unlike the exclusion, does cover every position - including the
// goalkeeper, when the saved lineup has none at all. The lineup editor's projection (see
// ProjectKickoffReplacements) runs this same function, so what it shows is what the match does.
//
// A player in the lineup who is no longer in squad (sold, loaned out or released after the lineup
// was saved - see removePlayerFromTeamLineupsTx) is taken out the same way, goalkeeper included:
// he no longer plays for this club at all.
func ExcludeUnavailableAndBackfill(playersByPosition *map[int][]int64, squad []*Player, unavailable map[int64]bool) {
	ExcludeUnavailableAndBackfillToFormation(playersByPosition, squad, unavailable, -1)
}

// ExcludeUnavailableAndBackfillToFormation is ExcludeUnavailableAndBackfill for a match kickoff:
// when formation is valid, a lineup that has lost players is also topped back up to the
// formation's full XI from the rest of the squad. A player who leaves the club (a loan that ends,
// a sale, a contract that runs out, a retirement) has his row deleted from the club's saved
// lineups (see removePlayerFromTeamLineupsTx), so his slot is no longer there to be refilled;
// without this, a club whose saved lineup had been thinned that way kicked off with whoever was
// left and forfeited at minute 0 (MinPlayersToContinueMatch) despite a full squad. An empty
// lineup is topped up the same way. Positions are filled in positionFillOrder and the XI never
// grows past the formation's size.
func ExcludeUnavailableAndBackfillToFormation(playersByPosition *map[int][]int64, squad []*Player, unavailable map[int64]bool, formation int) {
	originalCounts := make(map[int]int, len(ALL_OUTFIELD_POSITIONS)+1)
	for _, position := range ALL_OUTFIELD_POSITIONS {
		originalCounts[position] = len((*playersByPosition)[position])
	}
	// A saved lineup can be left without its goalkeeper - selling, loaning out or retiring the
	// only starting keeper deletes his default_lineup_positions row (see
	// removePlayerFromTeamLineupsTx) and nothing re-picks one - and every goalkeeper lookup
	// would then panic on an empty slice, so the match could never be simulated. The squad's
	// best available keeper steps in; a lineup that already has one is left alone.
	originalCounts[GOALKEEPER] = 1
	topUpCountsToFormation(originalCounts, formation)

	departed := departedLineupPlayers(*playersByPosition, squad)
	if len(departed) > 0 {
		keepers := (*playersByPosition)[GOALKEEPER]
		kept := keepers[:0]
		for _, id := range keepers {
			if !departed[id] {
				kept = append(kept, id)
			}
		}
		(*playersByPosition)[GOALKEEPER] = kept
		for id := range unavailable {
			departed[id] = true
		}
		unavailable = departed
	}

	FilterOutfieldPlayers(playersByPosition, unavailable)
	BackfillEmptyPositions(playersByPosition, originalCounts, squad, unavailable)
}

// topUpCountsToFormation raises counts towards formation's headcount at each position, in
// positionFillOrder, until the total reaches the formation's size; a position already at or above
// its count is left as it is. A no-op for an invalid formation.
func topUpCountsToFormation(counts map[int]int, formation int) {
	if !IsValidFormation(formation) {
		return
	}
	want := FormationPositionCounts(formation)
	total, size := 0, 0
	for _, c := range counts {
		total += c
	}
	for _, c := range want {
		size += c
	}
	for _, position := range positionFillOrder {
		for counts[position] < want[position] && total < size {
			counts[position]++
			total++
		}
	}
}

// departedLineupPlayers returns the players of playersByPosition who are not in squad, nil when
// there are none. An empty squad (nothing to check against, e.g. a national team, whose players
// belong to their clubs) departs nobody.
func departedLineupPlayers(playersByPosition map[int][]int64, squad []*Player) map[int64]bool {
	if len(squad) == 0 {
		return nil
	}
	inSquad := make(map[int64]bool, len(squad))
	for _, p := range squad {
		inSquad[p.Id] = true
	}
	var departed map[int64]bool
	for _, ids := range playersByPosition {
		for _, id := range ids {
			if !inSquad[id] {
				if departed == nil {
					departed = make(map[int64]bool)
				}
				departed[id] = true
			}
		}
	}
	return departed
}

// FilterOutfieldPlayers removes every player in `exclude` from a team's non-goalkeeper
// position groups, in place.
func FilterOutfieldPlayers(playersByPosition *map[int][]int64, exclude map[int64]bool) {
	if len(exclude) == 0 {
		return
	}
	for _, position := range ALL_OUTFIELD_POSITIONS {
		ids := (*playersByPosition)[position]
		kept := ids[:0]
		for _, id := range ids {
			if !exclude[id] {
				kept = append(kept, id)
			}
		}
		(*playersByPosition)[position] = kept
	}
}

// ProjectKickoffReplacements lists, for each unavailable outfield starter of lineup (and each
// starter, keeper included, who is no longer in squad), the player
// the pre-kickoff refill (ExcludeUnavailableAndBackfill) would put in his slot if the match
// kicked off now. The i-th starter removed from a position is paired with the i-th player the
// refill adds there; a starter nobody can replace (the squad ran out) has no entry. lineup is
// not modified.
func ProjectKickoffReplacements(lineup *Lineup, squad []*Player, unavailable map[int64]bool) []ProjectedReplacement {
	if lineup == nil {
		return nil
	}
	playersByPosition := make(map[int][]int64)
	for _, p := range lineup.Positions {
		playersByPosition[p.Position] = append(playersByPosition[p.Position], p.PlayerId)
	}
	departed := departedLineupPlayers(playersByPosition, squad)
	if len(unavailable) == 0 && len(departed) == 0 {
		return nil
	}

	removed := make(map[int][]int64)
	kept := make(map[int][]int64)
	for _, position := range positionFillOrder {
		for _, id := range playersByPosition[position] {
			// The refill only takes a goalkeeper out when he has left the club.
			if departed[id] || (position != GOALKEEPER && unavailable[id]) {
				removed[position] = append(removed[position], id)
			} else {
				kept[position] = append(kept[position], id)
			}
		}
	}
	if len(removed) == 0 {
		return nil
	}

	ExcludeUnavailableAndBackfill(&playersByPosition, squad, unavailable)

	var replacements []ProjectedReplacement
	for _, position := range positionFillOrder {
		added := playersByPosition[position][len(kept[position]):]
		for i, out := range removed[position] {
			if i >= len(added) {
				break
			}
			replacements = append(replacements, ProjectedReplacement{PlayerOutId: out, PlayerInId: added[i], Position: position})
		}
	}
	return replacements
}

// ApplyProjectedReplacements returns a copy of lineup with each projected replacement swapped into
// the slot of the unavailable starter it replaces - the XI the pre-kickoff refill would field if the
// match kicked off now (see ProjectKickoffReplacements). lineup itself is not modified.
func ApplyProjectedReplacements(lineup *Lineup, replacements []ProjectedReplacement) *Lineup {
	projected := *lineup
	projected.Positions = make([]LineupPosition, len(lineup.Positions))
	copy(projected.Positions, lineup.Positions)
	projected.ProjectedReplacements = nil
	for _, r := range replacements {
		for i, p := range projected.Positions {
			if p.PlayerId == r.PlayerOutId && p.Position == r.Position {
				projected.Positions[i].PlayerId = r.PlayerInId
				break
			}
		}
	}
	return &projected
}
