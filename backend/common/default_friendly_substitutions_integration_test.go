package common

import (
	"reflect"
	"testing"
)

// The friendly default owns its substitutions: independent of the league/cup set, starting empty,
// replaced in save order, cleared by nil (needs STRIKER_TEST_DB_DSN).
func TestDefaultFriendlyLineupSubstitutions_AreIndependentOfLeagueCup(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zf")
	team := seedTransfersTeam(t, db, "Friendly Subs FC", "zf")
	var p []int64
	for i := 0; i < 6; i++ {
		p = append(p, seedTransfersPlayer(t, db, team, "zf"))
	}

	// "After the migration": league/cup substitutions exist, the friendly set reads empty.
	leagueSubs := []LineupSubstitution{
		{PlayerOutId: p[0], PlayerInId: p[3], Minute: 55},
		{PlayerOutId: p[1], PlayerInId: p[4], Minute: 65},
	}
	if err := SetDefaultLineupSubstitutions(dbCtx, team, leagueSubs); err != nil {
		t.Fatalf("SetDefaultLineupSubstitutions: %s", err)
	}
	got, err := GetDefaultFriendlyLineupSubstitutions(dbCtx, team)
	if err != nil || len(got) != 0 {
		t.Fatalf("friendly set before any save = %v (err %v), want empty", got, err)
	}
	friendlyLineup, err := GetDefaultFriendlyLineup(dbCtx, team)
	if err != nil || len(friendlyLineup.Substitutions) != 0 {
		t.Fatalf("GetDefaultFriendlyLineup substitutions = %v (err %v), want empty", friendlyLineup.Substitutions, err)
	}

	// Two friendly substitutions come back in save order and leave the league/cup set untouched.
	friendlySubs := []LineupSubstitution{
		{PlayerOutId: p[2], PlayerInId: p[5], Minute: 70},
		{PlayerOutId: p[0], PlayerInId: p[3], Minute: 60},
	}
	if err := SetDefaultFriendlyLineupSubstitutions(dbCtx, team, friendlySubs); err != nil {
		t.Fatalf("SetDefaultFriendlyLineupSubstitutions: %s", err)
	}
	got, _ = GetDefaultFriendlyLineupSubstitutions(dbCtx, team)
	if !reflect.DeepEqual(got, friendlySubs) {
		t.Fatalf("friendly set = %v, want %v", got, friendlySubs)
	}
	league, _ := GetDefaultLineupSubstitutions(dbCtx, team)
	if !reflect.DeepEqual(league, leagueSubs) {
		t.Fatalf("league/cup set = %v, want it untouched %v", league, leagueSubs)
	}
	friendlyLineup, _ = GetDefaultFriendlyLineup(dbCtx, team)
	if !reflect.DeepEqual(friendlyLineup.Substitutions, friendlySubs) {
		t.Fatalf("GetDefaultFriendlyLineup substitutions = %v, want %v", friendlyLineup.Substitutions, friendlySubs)
	}
	defaultLineup, _ := GetDefaultLineup(dbCtx, team)
	if !reflect.DeepEqual(defaultLineup.Substitutions, leagueSubs) {
		t.Fatalf("GetDefaultLineup substitutions = %v, want %v", defaultLineup.Substitutions, leagueSubs)
	}

	// Saving the league/cup set leaves the friendly one alone.
	if err := SetDefaultLineupSubstitutions(dbCtx, team, leagueSubs[:1]); err != nil {
		t.Fatalf("SetDefaultLineupSubstitutions: %s", err)
	}
	got, _ = GetDefaultFriendlyLineupSubstitutions(dbCtx, team)
	if !reflect.DeepEqual(got, friendlySubs) {
		t.Fatalf("friendly set after a league/cup save = %v, want %v", got, friendlySubs)
	}

	// nil clears only the friendly set.
	if err := SetDefaultFriendlyLineupSubstitutions(dbCtx, team, nil); err != nil {
		t.Fatalf("clearing: %s", err)
	}
	got, _ = GetDefaultFriendlyLineupSubstitutions(dbCtx, team)
	if len(got) != 0 {
		t.Fatalf("friendly set after nil = %v, want empty", got)
	}
	league, _ = GetDefaultLineupSubstitutions(dbCtx, team)
	if len(league) != 1 {
		t.Fatalf("league/cup set after clearing the friendly one = %v, want 1", league)
	}
}
