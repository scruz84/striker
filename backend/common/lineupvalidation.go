package common

import "fmt"

// The human lineup rules live in common, not handlers, so the engine can run the exact same
// checks on a CPU team's kickoff pick (see PlanAiMatchLineup) that a manager's saved lineup goes
// through.

// ValidateLineup ensures the submitted lineup matches the chosen formation's required headcount
// at every tactical position (see FormationPositionCounts - always 11 players total,
// goalkeeper included) with distinct players that actually belong to the team - without this, a
// manager could otherwise field an empty/understaffed position, duplicate a player, or field an
// opponent's player, since the DB foreign key alone doesn't enforce team ownership.
//
// enforceAvailability additionally rejects any player currently suspended or injured (see
// GetUnavailablePlayerIds) - the simulation itself would exclude them anyway (see
// engine.excludeUnavailablePlayers), but rejecting it here gives the manager a clear reason
// instead of a lineup that quietly plays short. This only applies to a lineup being saved for a
// specific match (handlers.SetMatchLineupHandler/SetCupTieLineupHandler/SetFriendlyLineupHandler): a
// team's standing default lineup (handlers.SetLineupHandler/SetFriendlyDefaultLineupHandler) isn't tied to
// any one matchday's squad, so a manager can save it as-is with a currently unavailable player
// still in it - they'll simply sit out whichever matches find them still unavailable.
func ValidateLineup(formation int, positions []LineupPosition, teamPlayers *[]*Player, unavailable map[int64]bool, enforceAvailability bool) error {
	if !IsValidFormation(formation) {
		return fmt.Errorf("invalid formation %d", formation)
	}
	wantCounts := FormationPositionCounts(formation)

	wantTotal := 0
	for _, count := range wantCounts {
		wantTotal += count
	}
	if len(positions) != wantTotal {
		return fmt.Errorf("lineup must have exactly %d positions for this formation, got %d", wantTotal, len(positions))
	}

	rosterIds := make(map[int64]bool, len(*teamPlayers))
	for _, p := range *teamPlayers {
		rosterIds[p.Id] = true
	}

	gotCounts := make(map[int]int, len(wantCounts))
	seenPlayers := make(map[int64]bool, len(positions))
	for _, pos := range positions {
		if pos.Position < GOALKEEPER || pos.Position > LATERAL_STRIKER_RIGHT {
			return fmt.Errorf("invalid position %d", pos.Position)
		}
		gotCounts[pos.Position]++

		if seenPlayers[pos.PlayerId] {
			return fmt.Errorf("player %d assigned to more than one position", pos.PlayerId)
		}
		seenPlayers[pos.PlayerId] = true

		if !rosterIds[pos.PlayerId] {
			return fmt.Errorf("player %d does not belong to this team", pos.PlayerId)
		}

		if enforceAvailability && unavailable[pos.PlayerId] {
			return fmt.Errorf("player %d is suspended or injured and unavailable", pos.PlayerId)
		}
	}

	for position, want := range wantCounts {
		if gotCounts[position] != want {
			return fmt.Errorf("formation requires %d players at position %d, got %d", want, position, gotCounts[position])
		}
	}

	return nil
}

// ValidateSubstitutions ensures a manager's planned in-match substitutions (see
// LineupSubstitution) are all sound before they're saved: at most
// MaxScheduledSubstitutions of them, each swapping out a player who's actually in the
// submitted starting lineup for a fit bench player who isn't, at a minute between 1 and 90 -
// with no player subbed out twice or brought on twice. positions is the lineup being saved in
// the same request, not the team's previous lineup, since a substitution only makes sense
// against the XI it's paired with.
func ValidateSubstitutions(positions []LineupPosition, subs []LineupSubstitution, teamPlayers *[]*Player, unavailable map[int64]bool) error {
	if len(subs) > MaxScheduledSubstitutions {
		return fmt.Errorf("at most %d substitutions may be scheduled, got %d", MaxScheduledSubstitutions, len(subs))
	}

	starters := make(map[int64]bool, len(positions))
	for _, pos := range positions {
		starters[pos.PlayerId] = true
	}

	rosterIds := make(map[int64]bool, len(*teamPlayers))
	for _, p := range *teamPlayers {
		rosterIds[p.Id] = true
	}

	usedOut := make(map[int64]bool, len(subs))
	usedIn := make(map[int64]bool, len(subs))
	for _, sub := range subs {
		if sub.Minute < 1 || sub.Minute > 90 {
			return fmt.Errorf("substitution minute must be between 1 and 90, got %d", sub.Minute)
		}
		if !starters[sub.PlayerOutId] {
			return fmt.Errorf("player %d being substituted out is not in the starting lineup", sub.PlayerOutId)
		}
		if usedOut[sub.PlayerOutId] {
			return fmt.Errorf("player %d is scheduled to be substituted out more than once", sub.PlayerOutId)
		}
		usedOut[sub.PlayerOutId] = true

		if !rosterIds[sub.PlayerInId] {
			return fmt.Errorf("player %d does not belong to this team", sub.PlayerInId)
		}
		if starters[sub.PlayerInId] {
			return fmt.Errorf("player %d being substituted in is already in the starting lineup", sub.PlayerInId)
		}
		if unavailable[sub.PlayerInId] {
			return fmt.Errorf("player %d is suspended or injured and unavailable", sub.PlayerInId)
		}
		if usedIn[sub.PlayerInId] {
			return fmt.Errorf("player %d is scheduled to be substituted in more than once", sub.PlayerInId)
		}
		usedIn[sub.PlayerInId] = true
	}

	return nil
}
