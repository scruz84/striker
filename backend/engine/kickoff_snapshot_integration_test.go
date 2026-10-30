package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// injureStarter marks one of the team's default-lineup starters (an outfielder, so the backfill
// applies) as injured and returns his id and the position he starts in.
func (e *simEnv) injureStarter(teamId int64) (playerId int64, position int) {
	e.t.Helper()
	err := e.db.QueryRow(`SELECT player_id, position FROM default_lineup_positions WHERE team_id = $1 AND position = $2 LIMIT 1`,
		teamId, common.MIDFIELD).Scan(&playerId, &position)
	if err != nil {
		e.t.Fatalf("reading a starter: %s", err)
	}
	e.exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 3)`, playerId, common.UNAVAILABLE_INJURY)
	return playerId, position
}

func lineupHas(lineup *common.Lineup, playerId int64) bool {
	for _, p := range lineup.Positions {
		if p.PlayerId == playerId {
			return true
		}
	}
	return false
}

func positionCount(lineup *common.Lineup, position int) int {
	count := 0
	for _, p := range lineup.Positions {
		if p.Position == position {
			count++
		}
	}
	return count
}

// TestSimulateMatch_StoredLineupIsTheKickoffXI pins that a starter injured before kickoff is not in
// the match's stored lineup, and the player who took his slot is.
func TestSimulateMatch_StoredLineupIsTheKickoffXI(t *testing.T) {
	e := newSimEnv(t, "ks1")
	injured, position := e.injureStarter(e.home)
	before, err := common.GetLineupsForMatch(e.dbCtx, 0, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForMatch: %s", err)
	}
	wantInMidfield := positionCount((*before)[e.home], position)
	matchId := e.leagueMatch(0)

	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForMatch: %s", err)
	}
	stored := (*lineups)[e.home]
	if lineupHas(stored, injured) {
		t.Fatalf("expected the injured starter %d to be absent from the stored lineup, got %v", injured, stored.Positions)
	}
	if got := positionCount(stored, position); got != wantInMidfield {
		t.Fatalf("expected the injured starter's slot to be refilled (%d at position %d), got %d", wantInMidfield, position, got)
	}
	if len(stored.Positions) != 11 {
		t.Fatalf("expected a full XI of 11, got %d: %v", len(stored.Positions), stored.Positions)
	}

	// The unavailable starter never played, so he has no rating at all - not even an "s.c." one.
	played, err := common.WeekMatchById(e.dbCtx, matchId)
	if err != nil || played == nil {
		t.Fatalf("WeekMatchById: %v, %v", played, err)
	}
	for _, r := range played.PlayerRatings {
		if r.PlayerId == injured {
			t.Fatalf("expected no rating for the unavailable starter %d, got %+v", injured, r)
		}
	}
}

// TestSimulateCupTie_StoredLineupIsTheKickoffXI is the same pin for a cup tie.
func TestSimulateCupTie_StoredLineupIsTheKickoffXI(t *testing.T) {
	e := newSimEnv(t, "ks2")
	injured, position := e.injureStarter(e.home)
	_, tieId := e.cupTie(0)
	tie, err := common.GetCupTieById(e.dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieById: %s", err)
	}

	if err := simulateCupTie(e.dbCtx, tie); err != nil {
		t.Fatalf("simulateCupTie: %s", err)
	}

	lineups, err := common.GetLineupsForCupTie(e.dbCtx, tieId, []int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForCupTie: %s", err)
	}
	stored := (*lineups)[e.home]
	if lineupHas(stored, injured) {
		t.Fatalf("expected the injured starter %d to be absent from the stored lineup, got %v", injured, stored.Positions)
	}
	if len(stored.Positions) != 11 || positionCount(stored, position) == 0 {
		t.Fatalf("expected a full XI with the slot refilled, got %v", stored.Positions)
	}
}

// makeAiTiredStarters turns teamId into an AI-managed team of the given level whose regulars
// (shirts 1-11) are stronger than its reserves (shirts 12-16), with ten outfield starters worn
// out (Stamina 5) so a substitution has something to gain.
func (e *simEnv) makeAiTiredStarters(teamId int64, aiLevel int) {
	e.t.Helper()
	e.exec(`UPDATE teams SET ai_level = $2 WHERE team_id = $1`, teamId, aiLevel)
	e.exec(`UPDATE players SET defense = 70, "lateral" = 70, pass = 70, midfield = 70, dribbling = 70, shot = 70, stamina = 5
		WHERE team_id = $1 AND shirt_number BETWEEN 2 AND 11`, teamId)
}

func startersOfLineup(lineup *common.Lineup) map[int64]bool {
	ids := map[int64]bool{}
	for _, p := range lineup.Positions {
		ids[p.PlayerId] = true
	}
	return ids
}

// TestSimulateMatch_DirectorSnapshotCarriesPlannedSubstitutions pins that a Director decides its
// scheduled substitutions at kickoff and the stored snapshot records them.
func TestSimulateMatch_DirectorSnapshotCarriesPlannedSubstitutions(t *testing.T) {
	e := newSimEnv(t, "ks3")
	e.makeAiTiredStarters(e.home, common.AiLevelDirector)
	matchId := e.leagueMatch(0)

	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	subs, err := common.GetSubstitutionsForMatch(e.dbCtx, matchId, e.home)
	if err != nil {
		t.Fatalf("GetSubstitutionsForMatch: %s", err)
	}
	if len(subs) != common.MaxScheduledSubstitutions {
		t.Fatalf("expected the Director's %d planned substitutions in the snapshot, got %v", common.MaxScheduledSubstitutions, subs)
	}
	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForMatch: %s", err)
	}
	started := startersOfLineup((*lineups)[e.home])
	for _, s := range subs {
		if !started[s.PlayerOutId] || started[s.PlayerInId] || s.Minute < 55 || s.Minute > 85 {
			t.Fatalf("expected each substitution to take off a starter for a bench player between minutes 55 and 85, got %v", subs)
		}
	}
	// The human-managed away side is untouched: no substitutions were invented for it.
	awaySubs, _ := common.GetSubstitutionsForMatch(e.dbCtx, matchId, e.away)
	if len(awaySubs) != 0 {
		t.Fatalf("expected no substitutions for the unplanned side, got %v", awaySubs)
	}
}

// TestSimulateFriendly_ManagerSnapshotStartsReserves pins the friendly rotation: a Manager rests
// its five best outfield regulars and the snapshot starts reserves.
func TestSimulateFriendly_ManagerSnapshotStartsReserves(t *testing.T) {
	e := newSimEnv(t, "ks4")
	e.exec(`UPDATE teams SET ai_level = $2 WHERE team_id = $1`, e.home, common.AiLevelManager)
	// shirts 2-6 are clearly the five best outfield regulars
	e.exec(`UPDATE players SET defense = 90, "lateral" = 90, pass = 90, midfield = 90, dribbling = 90, shot = 90
		WHERE team_id = $1 AND shirt_number BETWEEN 2 AND 6`, e.home)
	var rested []int64
	rows, err := e.db.Query(`SELECT player_id FROM players WHERE team_id = $1 AND shirt_number BETWEEN 2 AND 6`, e.home)
	if err != nil {
		t.Fatalf("reading rested players: %s", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scanning: %s", err)
		}
		rested = append(rested, id)
	}
	rows.Close()
	friendlyId := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() - interval '3 hours') RETURNING match_id`, e.home, e.away)
	friendly, err := common.GetFriendlyMatchById(e.dbCtx, friendlyId)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById: %s", err)
	}

	if err := simulateFriendly(e.dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}

	lineups, err := common.GetLineupsForFriendly(e.dbCtx, friendlyId, []int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForFriendly: %s", err)
	}
	started := startersOfLineup((*lineups)[e.home])
	for _, id := range rested {
		if started[id] {
			t.Fatalf("expected the rested regular %d not to start, got %v", id, (*lineups)[e.home].Positions)
		}
	}
	subs, err := common.GetFriendlyLineupSubstitutions(e.dbCtx, friendlyId, e.home)
	if err != nil {
		t.Fatalf("GetFriendlyLineupSubstitutions: %s", err)
	}
	// The 16-man test squad has no reserve left once the five rested regulars sit out, so there may
	// be no substitutes; any scheduled are at minute 60 and none is a rested regular.
	for _, s := range subs {
		for _, id := range rested {
			if s.PlayerInId == id {
				t.Fatalf("expected a rested regular not to come on, got %v", subs)
			}
		}
		if s.Minute != 60 {
			t.Fatalf("expected friendly changes at minute 60, got %v", subs)
		}
	}
}

// TestPlanAiKickoff_SameMatchTwiceYieldsTheSameLineup pins stuck-match recovery determinism: a
// second plan for the same match and team (no snapshot stored in between) is identical.
func TestPlanAiKickoff_SameMatchTwiceYieldsTheSameLineup(t *testing.T) {
	e := newSimEnv(t, "ks5")
	e.makeAiTiredStarters(e.home, common.AiLevelCaretaker)
	matchId := e.leagueMatch(0)
	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForMatch: %s", err)
	}

	first, firstSubs := planAiKickoff(e.dbCtx, kickoffLeague, matchId, e.home, (*lineups)[e.home], nil)
	for i := 0; i < 10; i++ {
		again, againSubs := planAiKickoff(e.dbCtx, kickoffLeague, matchId, e.home, (*lineups)[e.home], nil)
		if again.Formation != first.Formation || len(again.Positions) != len(first.Positions) || len(againSubs) != len(firstSubs) {
			t.Fatalf("expected the same plan, got %+v / %v vs %+v / %v", again, againSubs, first, firstSubs)
		}
		for j := range first.Positions {
			if again.Positions[j] != first.Positions[j] {
				t.Fatalf("expected the same positions, got %v vs %v", again.Positions, first.Positions)
			}
		}
		for j := range firstSubs {
			if againSubs[j] != firstSubs[j] {
				t.Fatalf("expected the same substitutions, got %v vs %v", againSubs, firstSubs)
			}
		}
	}
}

// TestPlanAiKickoff_KeepsAHumanTeamsLineup pins that only AI-managed sides are replanned.
func TestPlanAiKickoff_KeepsAHumanTeamsLineup(t *testing.T) {
	e := newSimEnv(t, "ks6")
	matchId := e.leagueMatch(0)
	lineups, _ := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	saved := []common.LineupSubstitution{{PlayerOutId: 1, PlayerInId: 2, Minute: 70}}

	lineup, subs := planAiKickoff(e.dbCtx, kickoffLeague, matchId, e.home, (*lineups)[e.home], saved)

	if lineup != (*lineups)[e.home] || len(subs) != 1 || subs[0] != saved[0] {
		t.Fatalf("expected an unmanaged team's lineup and substitutions untouched, got %+v / %v", lineup, subs)
	}
}
