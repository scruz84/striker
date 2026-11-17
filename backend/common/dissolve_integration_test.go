package common

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestDissolveTeam covers league-pyramid "Dissolved team": senior players become free agents with
// no last club, academy prospects retire, the trainer returns to the market, offers, challenges
// and unplayed friendlies end, a player loaned in goes home now, a player loaned out is released
// when his loan ends, the club leaves its league, and its played friendly stays. A second call, a
// user's club and a national team change nothing.
func TestDissolveTeam(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	country := fmt.Sprintf("d%03d", time.Now().UnixNano()%1000)
	seedTransfersCountry(t, db, country)
	club := seedTransfersTeam(t, db, "Dissolved", country)
	other := seedTransfersTeam(t, db, "Other", country)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	seniors := []int64{seedTransfersPlayer(t, db, club, country), seedTransfersPlayer(t, db, club, country)}
	academy := seedTransfersPlayer(t, db, club, country)
	exec(`UPDATE players SET in_academy = true WHERE player_id = $1`, academy)
	loanedIn := seedTransfersPlayer(t, db, club, country)   // belongs to other, on loan at club
	loanedOut := seedTransfersPlayer(t, db, other, country) // belongs to club, on loan at other
	exec(`INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season) VALUES ($1, $2, $3, 2)`, loanedIn, other, club)
	exec(`INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season) VALUES ($1, $2, $3, 2)`, loanedOut, club, other)
	trainer := scan(`INSERT INTO trainers (first_name, last_name, team_id, skill, wage) VALUES ('T', 'R', $1, 50, 100) RETURNING trainer_id`, club)
	otherPlayer := seedTransfersPlayer(t, db, other, country)
	exec(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount) VALUES ($1, $2, $3, 1000)`, otherPlayer, club, other)
	exec(`INSERT INTO friendly_challenges (challenger_team_id, opponent_team_id) VALUES ($1, $2)`, other, club)
	upcoming := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '2 days') RETURNING match_id`, other, club)
	exec(`INSERT INTO friendly_lineup (match_id, team_id) VALUES ($1, $2)`, upcoming, other)
	played := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played) VALUES ($1, $2, now() - interval '2 days', true) RETURNING match_id`, club, other)
	exec(`INSERT INTO classification (league_id, team_id, season) SELECT league_id, team_id, 1 FROM teams WHERE team_id = $1`, club)

	ok, returned, err := DissolveTeam(dbCtx, club, 3)
	if err != nil || !ok {
		t.Fatalf("DissolveTeam = %v, %v", ok, err)
	}
	if len(returned) != 1 || returned[0].PlayerId != loanedIn {
		t.Errorf("loans sent home = %+v, want the one of player %d", returned, loanedIn)
	}
	if n := scan(`SELECT count(*) FROM teams WHERE team_id = $1 AND dissolved_season = 3 AND league_id = -1 AND ai_level = 0`, club); n != 1 {
		t.Error("the club is not marked dissolved in season 3 out of its league")
	}
	for _, p := range seniors {
		if n := scan(`SELECT count(*) FROM players WHERE player_id = $1 AND team_id IS NULL AND last_team_id IS NULL AND free_agent_seasons = 0`, p); n != 1 {
			t.Errorf("senior player %d is not a free agent without a last club", p)
		}
	}
	checks := map[string]int64{
		"academy prospect retired":       scan(`SELECT count(*) FROM players WHERE player_id = $1`, academy),
		"trainer still at the club":      scan(`SELECT count(*) FROM trainers WHERE trainer_id = $1 AND team_id IS NOT NULL`, trainer),
		"transfer offer left":            scan(`SELECT count(*) FROM transfer_offers WHERE from_team_id = $1 OR to_team_id = $1`, club),
		"friendly challenge left":        scan(`SELECT count(*) FROM friendly_challenges WHERE opponent_team_id = $1 OR challenger_team_id = $1`, club),
		"unplayed friendly left":         scan(`SELECT count(*) FROM friendly_matches WHERE match_id = $1`, upcoming),
		"classification row left":        scan(`SELECT count(*) FROM classification WHERE team_id = $1`, club),
		"loaned-in player not home":      scan(`SELECT count(*) FROM players WHERE player_id = $1 AND team_id IS DISTINCT FROM $2`, loanedIn, other),
		"loaned-out player not released": 1 - scan(`SELECT count(*) FROM player_loans WHERE player_id = $1 AND release_on_return`, loanedOut),
	}
	for what, n := range checks {
		if n != 0 {
			t.Errorf("%s (%d)", what, n)
		}
	}
	if n := scan(`SELECT count(*) FROM friendly_matches WHERE match_id = $1`, played); n != 1 {
		t.Error("the played friendly is gone")
	}

	// Idempotent, and never a user's club or a national team.
	if ok, _, err := DissolveTeam(dbCtx, club, 4); ok || err != nil {
		t.Errorf("second DissolveTeam = %v, %v; want false, nil", ok, err)
	}
	if n := scan(`SELECT dissolved_season FROM teams WHERE team_id = $1`, club); n != 3 {
		t.Errorf("a second call changed dissolved_season to %d", n)
	}
	user := scan(`INSERT INTO users (name, provider, email) VALUES ('d', 'dev', $1) RETURNING user_id`, fmt.Sprintf("dissolve-%d@test", time.Now().UnixNano()))
	exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, user, other)
	national := scan(`INSERT INTO teams (name, country, is_national_team) VALUES ('NT', $1, true) RETURNING team_id`, country)
	for _, id := range []int64{other, national} {
		if ok, _, err := DissolveTeam(dbCtx, id, 3); ok || err != nil {
			t.Errorf("DissolveTeam(%d) = %v, %v; want false, nil", id, ok, err)
		}
	}
	if n := scan(`SELECT count(*) FROM teams WHERE team_id IN ($1, $2) AND dissolved_season IS NOT NULL`, other, national); n != 0 {
		t.Error("a user's club or a national team was dissolved")
	}
}

// TestDissolvedTeamStaysOut covers league-pyramid "Dissolved team": a dissolved club is never
// assigned to a user, is not among the clubs the engine's daily, AI, wage and rollover passes walk,
// is not counted as an active club, and cannot be challenged to a friendly.
func TestDissolvedTeamStaysOut(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	country := fmt.Sprintf("s%03d", time.Now().UnixNano()%1000)
	seedTransfersCountry(t, db, country)
	gone := seedTransfersTeam(t, db, "Gone", country)
	kept := seedTransfersTeam(t, db, "Kept", country)
	if ok, _, err := DissolveTeam(dbCtx, gone, 1); err != nil || !ok {
		t.Fatalf("DissolveTeam = %v, %v", ok, err)
	}
	contains := func(ids []int64, id int64) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	all, err := GetAllTeamIds(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if contains(all, gone) || !contains(all, kept) {
		t.Errorf("GetAllTeamIds: dissolved listed %v, kept listed %v", contains(all, gone), contains(all, kept))
	}
	inCountry, err := GetTeamIdsForCountry(dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	if len(inCountry) != 1 || inCountry[0] != kept {
		t.Errorf("GetTeamIdsForCountry = %v, want only %d", inCountry, kept)
	}
	// The free-team pick never offers it, even once the kept club is taken.
	user := int64(0)
	if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('s', 'dev', $1) RETURNING user_id`,
		fmt.Sprintf("stays-out-%d@test", time.Now().UnixNano())).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, user, kept); err != nil {
		t.Fatal(err)
	}
	var free int64
	if err := db.QueryRow(GetFreeTeamStmtSql, country).Scan(&free); err == nil {
		t.Errorf("the free-team pick offered team %d in a country whose only free club is dissolved", free)
	}
	counts, err := CountTeamsPerActiveCountry(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if counts[country].Teams != 1 || counts[country].UserTeams != 1 {
		t.Errorf("country counts %+v, want 1 team, 1 user team", counts[country])
	}
	if err := ChallengeToFriendly(dbCtx, kept, gone, nil, true); !errors.Is(err, ErrTeamDissolved) {
		t.Errorf("challenging a dissolved club: %v, want ErrTeamDissolved", err)
	}
	if dissolved, err := IsTeamDissolved(dbCtx, gone); err != nil || !dissolved {
		t.Errorf("IsTeamDissolved = %v, %v", dissolved, err)
	}
}
