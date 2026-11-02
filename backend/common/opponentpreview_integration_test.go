package common

import (
	"testing"
	"time"
)

// TestOpponentPreview_PublicHistory runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when
// unset). It checks the four opponent-preview scenarios that depend on stored data: an established
// opponent (form, likely XI, in-form players), an opponent with no history, an opponent that saved
// a different XI for a future match, and an opponent whose latest match is still being replayed.
func TestOpponentPreview_PublicHistory(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	const country = "zopv"
	seedTransfersCountry(t, db, country)
	opponent := seedTransfersTeam(t, db, "Preview Opponent", country)
	other := seedTransfersTeam(t, db, "Preview Other", country)
	fresh := seedTransfersTeam(t, db, "Preview Fresh", country)

	var leagueId int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, opponent).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	players := make([]int64, 0, 12)
	for i := 0; i < 12; i++ {
		players = append(players, seedTransfersPlayer(t, db, opponent, country))
	}
	exec(`UPDATE players SET first_name = 'Star', last_name = 'Striker' WHERE player_id = $1`, players[0])
	xi := func(ids []int64) []LineupPosition {
		positions := make([]LineupPosition, 0, 11)
		for i, id := range ids[:11] {
			positions = append(positions, LineupPosition{PlayerId: id, Position: GOALKEEPER + i})
		}
		return positions
	}

	now := time.Now()
	insertMatch := func(daysAgo float64, played bool) int64 {
		t.Helper()
		var id int64
		date := now.Add(-time.Duration(daysAgo * float64(24*time.Hour)))
		if err := db.QueryRow(`INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id, home_goals, away_goals)
			VALUES ($1, 1, $2, $3, $4, $5, 2, 1) RETURNING match_id`, leagueId, played, date, opponent, other).Scan(&id); err != nil {
			t.Fatalf("inserting match: %s", err)
		}
		return id
	}
	rate := func(matchId int64, playerId int64, rating float64) {
		exec(`INSERT INTO match_player_ratings (kind, match_id, player_id, team_id, rated, rating, minutes, position, zone_delta, goals_delta, cards_delta, injury_delta, result_delta)
			VALUES ('league', $1, $2, $3, true, $4, 90, 5, 0, 0, 0, 0, 0)`, matchId, playerId, opponent, rating)
	}

	// Fresh opponent: "No history".
	freshPreview, err := GetOpponentPreview(dbCtx, fresh)
	if err != nil {
		t.Fatalf("GetOpponentPreview(fresh): %s", err)
	}
	if len(freshPreview.Opponent.Form) != 0 || freshPreview.Likely != nil || len(freshPreview.InForm) != 0 {
		t.Fatalf("no history preview = %+v", freshPreview)
	}

	// Established opponent: 8 finished league matches, the oldest 3 with another XI.
	var matchIds []int64
	for i := 0; i < 8; i++ {
		id := insertMatch(float64(10-i), true)
		matchIds = append(matchIds, id)
		lineupIds := players[:11]
		if i < 3 {
			lineupIds = players[1:12]
		}
		if err := SetLineupForMatch(dbCtx, id, opponent, 2, MENTALITY_BALANCED, xi(lineupIds), nil); err != nil {
			t.Fatal(err)
		}
		rate(id, players[0], 8.0)
		rate(id, players[1], 7.0)
		rate(id, players[2], 6.0)
	}
	latest := matchIds[7]

	preview, err := GetOpponentPreview(dbCtx, opponent)
	if err != nil {
		t.Fatalf("GetOpponentPreview: %s", err)
	}
	if len(preview.Opponent.Form) != 5 || preview.Opponent.Form[0].MatchId != latest || preview.Opponent.Form[4].MatchId != matchIds[3] {
		t.Fatalf("form = %+v", preview.Opponent.Form)
	}
	if preview.Likely == nil || preview.Likely.MatchId != latest || preview.Likely.Formation != 2 || len(preview.Likely.Players) != 11 {
		t.Fatalf("likely = %+v", preview.Likely)
	}
	if len(preview.InForm) != 3 || preview.InForm[0].PlayerId != players[0] || preview.InForm[0].Name != "Star Striker" ||
		preview.InForm[0].AverageRating != 8.0 || preview.InForm[0].Appearances != 5 {
		t.Fatalf("inForm = %+v", preview.InForm)
	}

	// Opponent changed its lineup: a completely new XI saved for an upcoming match is not used.
	upcoming := insertMatch(-2, false)
	if err := SetLineupForMatch(dbCtx, upcoming, opponent, 5, MENTALITY_BALANCED, xi(players[1:12]), nil); err != nil {
		t.Fatal(err)
	}
	preview, err = GetOpponentPreview(dbCtx, opponent)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Likely == nil || preview.Likely.MatchId != latest || preview.Likely.Formation != 2 || len(preview.Opponent.Form) != 5 {
		t.Fatalf("after saving a future lineup, likely = %+v", preview.Likely)
	}

	// Latest match still being replayed: played flag set, kickoff 30 minutes ago.
	replaying := insertMatch(0.5/24, true)
	if err := SetLineupForMatch(dbCtx, replaying, opponent, 7, MENTALITY_BALANCED, xi(players[1:12]), nil); err != nil {
		t.Fatal(err)
	}
	rate(replaying, players[2], 10.0)
	preview, err = GetOpponentPreview(dbCtx, opponent)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Likely == nil || preview.Likely.MatchId != latest || preview.Opponent.Form[0].MatchId != latest {
		t.Fatalf("with a match being replayed, form[0]/likely = %+v / %+v", preview.Opponent.Form[0], preview.Likely)
	}
	for _, p := range preview.InForm {
		if p.PlayerId == players[2] && p.AverageRating != 6.0 {
			t.Fatalf("the replaying match's rating leaked: %+v", p)
		}
	}
}
