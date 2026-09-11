package engine

import "github.com/scruz84/striker/common"

// SubstitutionSimulation applies each team's manager-scheduled substitutions (see
// common.GetDefaultLineupSubstitutions/common.LineupSubstitution) the minute they're due -
// swapping the outgoing player out of whichever position they're still occupying for the
// incoming player, in that same slot. Runs first in the pipeline each minute (see
// runMatchSimulation) so every later stage that minute - qualification diffs, scoring,
// discipline, injury - already sees the new player on the pitch.
type SubstitutionSimulation struct{}

func (s SubstitutionSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	applyScheduledSubstitutions(match, match.HomeSubstitutions, match.HomePlayersByPosition,
		match.HomePlayers, match.Home, match.Away)
	applyScheduledSubstitutions(match, match.AwaySubstitutions, match.AwayPlayersByPosition,
		match.AwayPlayers, match.Away, match.Home)
}

// applyScheduledSubstitutions runs every one of a team's planned substitutions due this minute.
// A substitution whose outgoing player is no longer on the pitch (already sent off, injured, or
// backfilled away pre-kickoff for being suspended/injured - see excludeUnavailablePlayers) or
// whose incoming player is somehow already on the pitch (e.g. auto-backfilled into a different
// position for that same reason) is silently skipped rather than corrupting the lineup - the
// same "understaffed is fine, duplicated/invalid is not" tolerance the rest of the simulation
// already applies.
func applyScheduledSubstitutions(match *common.MatchSimulation, substitutions []common.LineupSubstitution,
	playersByPosition *map[int][]int64, players *map[int64]common.Player, teamId int64, otherTeam int64) {
	for _, sub := range substitutions {
		if sub.Minute != match.Minute {
			continue
		}

		position, found := findPlayerPosition(playersByPosition, sub.PlayerOutId)
		if !found || isPlayerOnPitch(playersByPosition, sub.PlayerInId) {
			continue
		}

		removePlayerFromPosition(playersByPosition, position, sub.PlayerOutId)
		(*playersByPosition)[position] = append((*playersByPosition)[position], sub.PlayerInId)

		outName := (*players)[sub.PlayerOutId].FirstName + " " + (*players)[sub.PlayerOutId].LastName
		inName := (*players)[sub.PlayerInId].FirstName + " " + (*players)[sub.PlayerInId].LastName
		*match.Events = append(*match.Events, common.Event{
			EventType:             common.SUBSTITUTION,
			Minute:                match.Minute,
			TeamOwner:             teamId,
			TeamOwnerPlayers:      []int64{sub.PlayerOutId, sub.PlayerInId},
			TeamOwnerPlayersNames: []string{outName, inName},
			OtherTeam:             otherTeam,
			HomeGoals:             match.HomeGoals,
			AwayGoals:             match.AwayGoals,
		})
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
