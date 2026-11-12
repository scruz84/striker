package engine

import "github.com/scruz84/striker/common"

// SubstitutionSimulation applies each team's manager-scheduled substitutions (see
// common.GetDefaultLineupSubstitutions/common.LineupSubstitution) the minute they're due -
// swapping the outgoing player out of whichever position they're still occupying for the
// incoming player, in that same slot. Runs first in the pipeline each minute (see
// runMatchSimulation) so every later stage that minute - qualification diffs, scoring,
// discipline, injury - already sees the new player on the pitch. Each swap that actually fires
// counts against the team's HomeSubstitutionsUsed/AwaySubstitutionsUsed, the same running total
// InjurySimulation checks before bringing on an emergency replacement for an injured player -
// the two draw from one shared pool of common.MaxScheduledSubstitutions per match.
type SubstitutionSimulation struct{}

func (s SubstitutionSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	applyScheduledSubstitutions(match, sideOf(match, true))
	applyScheduledSubstitutions(match, sideOf(match, false))
	// Conditional orders come after every scheduled substitution of the minute (see
	// openspec/specs/match-simulation "Order firing").
	applyConditionalOrders(match, sideOf(match, true))
	applyConditionalOrders(match, sideOf(match, false))
}

// teamSide bundles one team's mutable state in a match, so the substitution and injury stages
// can treat home and away the same way.
type teamSide struct {
	teamId            int64
	otherTeam         int64
	playersByPosition *map[int][]int64
	players           *map[int64]common.Player
	unavailable       map[int64]bool
	substitutedOff    map[int64]bool
	substitutionsUsed *int
	substitutions     []common.LineupSubstitution
	// orders are the side's conditional orders and ordersFired which of them are spent; mentality
	// points at the side's live mentality, goalsFor/goalsAgainst at its score.
	orders       []common.LineupOrder
	ordersFired  []bool
	mentality    *int
	goalsFor     int
	goalsAgainst int
}

// sideOf returns the home (isHome) or away team's state in match, creating its substituted-off
// set on first use.
func sideOf(match *common.MatchSimulation, isHome bool) teamSide {
	if isHome {
		if match.HomeSubstitutedOff == nil {
			match.HomeSubstitutedOff = make(map[int64]bool)
		}
		if len(match.HomeOrdersFired) != len(match.HomeOrders) {
			match.HomeOrdersFired = make([]bool, len(match.HomeOrders))
		}
		return teamSide{match.Home, match.Away, match.HomePlayersByPosition, match.HomePlayers,
			match.HomeUnavailablePlayers, match.HomeSubstitutedOff, &match.HomeSubstitutionsUsed, match.HomeSubstitutions,
			match.HomeOrders, match.HomeOrdersFired, &match.HomeMentality, match.HomeGoals, match.AwayGoals}
	}
	if match.AwaySubstitutedOff == nil {
		match.AwaySubstitutedOff = make(map[int64]bool)
	}
	if len(match.AwayOrdersFired) != len(match.AwayOrders) {
		match.AwayOrdersFired = make([]bool, len(match.AwayOrders))
	}
	return teamSide{match.Away, match.Home, match.AwayPlayersByPosition, match.AwayPlayers,
		match.AwayUnavailablePlayers, match.AwaySubstitutedOff, &match.AwaySubstitutionsUsed, match.AwaySubstitutions,
		match.AwayOrders, match.AwayOrdersFired, &match.AwayMentality, match.AwayGoals, match.HomeGoals}
}

// bestEligibleBenchPlayer finds the best squad player for position, by common.PositionValue, who
// is allowed to come on: not on the pitch (any position), not unavailable before kickoff
// (suspension or injury carried over), not sent off or injured earlier this match
// (match.SentOff, match.PendingUnavailabilities) and not already substituted off. Every way a
// player comes on mid-match - the injury replacement and the fallback for a scheduled
// substitution whose incoming player can't play - uses this one rule. alsoIneligible lists
// extra ids to skip (the just-injured player, whose pending unavailability may not be recorded
// yet).
func bestEligibleBenchPlayer(match *common.MatchSimulation, side teamSide, position int, alsoIneligible ...int64) (int64, bool) {
	bestId, bestValue := int64(-1), -1
	for id, p := range *side.players {
		if !isEligibleBenchPlayer(match, side, id, alsoIneligible) {
			continue
		}
		if v := common.PositionValue(position, p); v > bestValue || (v == bestValue && id < bestId) {
			bestId, bestValue = id, v
		}
	}
	return bestId, bestId != -1
}

// isEligibleBenchPlayer reports whether playerId may come on right now - see
// bestEligibleBenchPlayer for what rules a player out.
func isEligibleBenchPlayer(match *common.MatchSimulation, side teamSide, playerId int64, alsoIneligible []int64) bool {
	if _, known := (*side.players)[playerId]; !known {
		return false
	}
	for _, id := range alsoIneligible {
		if id == playerId {
			return false
		}
	}
	if side.unavailable[playerId] || match.SentOff[playerId] || side.substitutedOff[playerId] {
		return false
	}
	for _, pending := range match.PendingUnavailabilities {
		if pending.PlayerId == playerId {
			return false
		}
	}
	return !isPlayerOnPitch(side.playersByPosition, playerId)
}

// applyScheduledSubstitutions runs every one of a team's planned substitutions due this minute.
// A substitution fires only if the team has used fewer than common.MaxScheduledSubstitutions
// (scheduled swaps and injury replacements share that pool) and its outgoing player is still on
// the pitch - one already sent off, injured, or backfilled away pre-kickoff for being
// suspended/injured (see excludeUnavailablePlayers) is silently skipped, not counted. If the
// incoming player can't play (injured or suspended since the lineup was saved, sent off, already
// on the pitch, already substituted off) the best eligible bench player for that slot comes on
// instead; with nobody eligible the substitution is skipped, also uncounted. The outgoing player
// is then marked substituted off, so nobody re-enters after leaving.
//
// Scheduled substitutions only have minutes 1 to 90, so none fires in added time: one scheduled
// for minute 45 fires once, at 45 itself.
func applyScheduledSubstitutions(match *common.MatchSimulation, side teamSide) {
	if match.AddedMinute != 0 {
		return
	}
	for _, sub := range side.substitutions {
		if sub.Minute != match.Minute {
			continue
		}
		applySubstitution(match, side, sub.PlayerOutId, sub.PlayerInId)
	}
}

// applySubstitution takes playerOut off for playerIn with the scheduled-substitution rules: only
// while the side has used fewer than common.MaxScheduledSubstitutions and playerOut is on the
// pitch, with the best eligible bench player for his slot standing in for an incoming player who
// can't come on. It reports whether a substitution happened (and was counted and logged).
func applySubstitution(match *common.MatchSimulation, side teamSide, playerOut int64, playerIn int64) bool {
	if *side.substitutionsUsed >= common.MaxScheduledSubstitutions {
		return false
	}
	position, found := findPlayerPosition(side.playersByPosition, playerOut)
	if !found {
		return false
	}
	incoming := playerIn
	if !isEligibleBenchPlayer(match, side, incoming, nil) {
		replacement, ok := bestEligibleBenchPlayer(match, side, position, playerOut)
		if !ok {
			return false
		}
		incoming = replacement
	}

	removePlayerFromPosition(side.playersByPosition, position, playerOut)
	(*side.playersByPosition)[position] = append((*side.playersByPosition)[position], incoming)
	side.substitutedOff[playerOut] = true
	*side.substitutionsUsed++

	outName := (*side.players)[playerOut].FirstName + " " + (*side.players)[playerOut].LastName
	inName := (*side.players)[incoming].FirstName + " " + (*side.players)[incoming].LastName
	*match.Events = append(*match.Events, common.Event{
		EventType:             common.SUBSTITUTION,
		Minute:                match.Minute,
		TeamOwner:             side.teamId,
		TeamOwnerPlayers:      []int64{playerOut, incoming},
		TeamOwnerPlayersNames: []string{outName, inName},
		OtherTeam:             side.otherTeam,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
	return true
}

// applyConditionalOrders checks, in slot order, every unspent order of the side whose from-minute
// has come against the score at the start of this minute (no goal of this minute has been
// scored yet - the scorer stage runs later). An order whose condition holds fires and is spent
// for good: a mentality order switches the side's mentality (logging a MENTALITY_CHANGE event,
// unless the side already plays that way), a substitution order is applied with the scheduled
// substitution rules (see applySubstitution) and is spent even when it can't be. No random draw
// is involved, so a seeded simulation stays reproducible.
func applyConditionalOrders(match *common.MatchSimulation, side teamSide) {
	for i, order := range side.orders {
		if side.ordersFired[i] || order.FromMinute > match.Minute {
			continue
		}
		if !common.OrderConditionHolds(order.Condition, side.goalsFor, side.goalsAgainst) {
			continue
		}
		side.ordersFired[i] = true
		switch order.Action {
		case common.ORDER_ACTION_MENTALITY:
			if *side.mentality == order.Mentality {
				continue
			}
			*side.mentality = order.Mentality
			mentality := order.Mentality
			*match.Events = append(*match.Events, common.Event{
				EventType: common.MENTALITY_CHANGE,
				Minute:    match.Minute,
				TeamOwner: side.teamId,
				OtherTeam: side.otherTeam,
				HomeGoals: match.HomeGoals,
				AwayGoals: match.AwayGoals,
				Mentality: &mentality,
			})
		case common.ORDER_ACTION_SUBSTITUTION:
			applySubstitution(match, side, order.PlayerOutId, order.PlayerInId)
		}
	}
}

func findPlayerPosition(playersByPosition *map[int][]int64, playerId int64) (int, bool) {
	for position, ids := range *playersByPosition {
		for _, id := range ids {
			if id == playerId {
				return position, true
			}
		}
	}
	return 0, false
}

func isPlayerOnPitch(playersByPosition *map[int][]int64, playerId int64) bool {
	_, found := findPlayerPosition(playersByPosition, playerId)
	return found
}
