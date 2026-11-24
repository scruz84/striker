package common

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// archiveWorld seeds fixtures for the instant archive tests in its own countries and region, so
// assertions only ever look at its own rows in the shared test database.
type archiveWorld struct {
	t      *testing.T
	db     *sql.DB
	dbCtx  *DatabaseContext
	region string
}

func newArchiveWorld(t *testing.T) *archiveWorld {
	t.Helper()
	db, dbCtx := openTransfersTestDB(t)
	return &archiveWorld{t: t, db: db, dbCtx: dbCtx, region: fmt.Sprintf("arch-%d", time.Now().UnixNano()%1e12)}
}

func (w *archiveWorld) scan(query string, args ...any) int64 {
	w.t.Helper()
	var v int64
	if err := w.db.QueryRow(query, args...).Scan(&v); err != nil {
		w.t.Fatalf("%s: %s", query, err)
	}
	return v
}

func (w *archiveWorld) exec(query string, args ...any) {
	w.t.Helper()
	if _, err := w.db.Exec(query, args...); err != nil {
		w.t.Fatalf("%s: %s", query, err)
	}
}

// country creates a country in currentSeason with one league and two clubs, and returns the
// country code, the league and the clubs.
func (w *archiveWorld) country(currentSeason int) (string, int64, int64, int64) {
	w.t.Helper()
	const letters = "abcdefghijklmnopqrstuvwxyz"
	code := "z"
	for i := 0; i < 3; i++ {
		code += string(letters[rand.Intn(len(letters))])
	}
	w.exec(`INSERT INTO countries (name, current_season) VALUES ($1, $2)`, code, currentSeason)
	league := w.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'I', 1) RETURNING league_id`, code)
	home := w.scan(`INSERT INTO teams (name, country, league_id) VALUES ('Home', $1, $2) RETURNING team_id`, code, league)
	away := w.scan(`INSERT INTO teams (name, country, league_id) VALUES ('Away', $1, $2) RETURNING team_id`, code, league)
	return code, league, home, away
}

const archiveTestMinutes = 3

// match inserts a league match of season with instants, an event, a rating and a lineup.
func (w *archiveWorld) match(league, home, away int64, season int, played bool, kickoff time.Time) int64 {
	w.t.Helper()
	id := w.scan(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played) VALUES ($1, $2, $3, $4, $5, $6) RETURNING match_id`,
		league, season, kickoff, home, away, played)
	for m := 1; m <= archiveTestMinutes; m++ {
		w.exec(`INSERT INTO match_instants (match_id, minute, player_contributions) VALUES ($1, $2, '{"h":{},"a":{}}')`, id, m)
	}
	w.exec(`INSERT INTO match_events (match_id, event_type, minute) VALUES ($1, 1, 10)`, id)
	w.exec(`INSERT INTO match_player_ratings (kind, match_id, player_id, team_id, rated, rating, minutes, position, zone_delta, goals_delta, cards_delta, injury_delta, result_delta)
		VALUES ('league', $1, 1, $2, true, 6.5, 90, 1, 0, 0, 0, 0, 0)`, id, home)
	w.exec(`INSERT INTO lineup (match_id, team_id) VALUES ($1, $2)`, id, home)
	return id
}

// cupTie inserts a played tie with instants and an event in cup.
func (w *archiveWorld) cupTie(cup, home, away int64) int64 {
	w.t.Helper()
	id := w.scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played) VALUES ($1, 1, $2, $3, $4, now() - interval '1 day', true) RETURNING tie_id`,
		cup, rand.Intn(1_000_000), home, away)
	for m := 1; m <= archiveTestMinutes; m++ {
		w.exec(`INSERT INTO cup_tie_instants (tie_id, minute) VALUES ($1, $2)`, id, m)
	}
	w.exec(`INSERT INTO cup_tie_events (tie_id, event_type, minute) VALUES ($1, 1, 10)`, id)
	return id
}

func (w *archiveWorld) friendly(home, away int64, kickoff time.Time) int64 {
	w.t.Helper()
	id := w.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played) VALUES ($1, $2, $3, true) RETURNING match_id`, home, away, kickoff)
	for m := 1; m <= archiveTestMinutes; m++ {
		w.exec(`INSERT INTO friendly_instants (match_id, minute) VALUES ($1, $2)`, id, m)
	}
	w.exec(`INSERT INTO friendly_events (match_id, event_type, minute) VALUES ($1, 1, 10)`, id)
	return id
}

// expectMatch checks one league match: archived (no instants, flagged) or kept (all instants, not
// flagged), and its event, rating and lineup kept either way.
func (w *archiveWorld) expectMatch(id int64, season int, archived bool) {
	w.t.Helper()
	instants := w.scan(`SELECT count(*) FROM match_instants WHERE match_id = $1`, id)
	flagged := w.scan(`SELECT count(*) FROM matches WHERE match_id = $1 AND instants_archived`, id) == 1
	if archived && (instants != 0 || !flagged) {
		w.t.Errorf("season %d match: %d instants, flagged %v; want archived", season, instants, flagged)
	}
	if !archived && (instants != archiveTestMinutes || flagged) {
		w.t.Errorf("season %d match: %d instants, flagged %v; want kept", season, instants, flagged)
	}
	kept := w.scan(`SELECT (SELECT count(*) FROM match_events WHERE match_id = $1) + (SELECT count(*) FROM match_player_ratings WHERE match_id = $1) +
		(SELECT count(*) FROM lineup WHERE match_id = $1)`, id)
	if kept != 3 {
		w.t.Errorf("season %d match lost its event, rating or lineup (%d of 3 left)", season, kept)
	}
}

func (w *archiveWorld) archive(keep int) InstantArchiveResult {
	w.t.Helper()
	res, err := ArchiveOldInstants(w.dbCtx, keep)
	if err != nil {
		w.t.Fatalf("ArchiveOldInstants(%d): %s", keep, err)
	}
	return res
}

// TestInstantArchiveSettings covers "Instant archive setting" and the save validation: off and 1
// by default, saved as given, and 11 rejected without a change.
func TestInstantArchiveSettings(t *testing.T) {
	w := newArchiveWorld(t)
	// A brand-new test database has no season row until --initialize seeds one.
	w.exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
	before, err := GetInstantArchiveSettings(w.dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SetInstantArchiveSettings(w.dbCtx, before) })

	var enabledDefault bool
	var keepDefault int
	if err := w.db.QueryRow(`SELECT (SELECT column_default FROM information_schema.columns WHERE table_name = 'season' AND column_name = 'archive_instants_enabled') = 'false',
		(SELECT column_default FROM information_schema.columns WHERE table_name = 'season' AND column_name = 'archive_keep_seasons')::int`).Scan(&enabledDefault, &keepDefault); err != nil {
		t.Fatal(err)
	}
	if !enabledDefault || keepDefault != 1 {
		t.Errorf("defaults: off=%v keep=%d, want off and 1", enabledDefault, keepDefault)
	}

	if err := SetInstantArchiveSettings(w.dbCtx, InstantArchiveSettings{Enabled: true, KeepSeasons: 2}); err != nil {
		t.Fatal(err)
	}
	if got, _ := GetInstantArchiveSettings(w.dbCtx); got != (InstantArchiveSettings{Enabled: true, KeepSeasons: 2}) {
		t.Errorf("saved %+v, want enabled with 2 kept", got)
	}
	for _, bad := range []int{11, -1} {
		if err := SetInstantArchiveSettings(w.dbCtx, InstantArchiveSettings{Enabled: false, KeepSeasons: bad}); !errors.Is(err, ErrInvalidInstantArchiveKeepSeasons) {
			t.Errorf("keep %d: err %v, want ErrInvalidInstantArchiveKeepSeasons", bad, err)
		}
	}
	if got, _ := GetInstantArchiveSettings(w.dbCtx); got != (InstantArchiveSettings{Enabled: true, KeepSeasons: 2}) {
		t.Errorf("a rejected save changed the setting to %+v", got)
	}
}

// TestArchiveOldInstants_LeagueSeasons covers "Keeping one season", "Keeping none" and "Countries
// advance on their own", plus an unplayed match never being touched.
func TestArchiveOldInstants_LeagueSeasons(t *testing.T) {
	w := newArchiveWorld(t)
	_, spain, sHome, sAway := w.country(5)
	_, france, fHome, fAway := w.country(4)
	past := time.Now().Add(-24 * time.Hour)
	spainMatches := map[int]int64{}
	for season := 1; season <= 5; season++ {
		spainMatches[season] = w.match(spain, sHome, sAway, season, true, past)
	}
	franceS3 := w.match(france, fHome, fAway, 3, true, past)
	unplayed := w.match(spain, sHome, sAway, 2, false, past)

	w.archive(1)
	for season, id := range spainMatches {
		w.expectMatch(id, season, season <= 3)
	}
	w.expectMatch(franceS3, 3, false)
	if got := w.scan(`SELECT count(*) FROM match_instants WHERE match_id = $1`, unplayed); got != archiveTestMinutes {
		t.Errorf("an unplayed match lost its instants (%d left)", got)
	}

	// Running again changes none of ours; keeping none then archives everything but season 5.
	w.archive(1)
	for season, id := range spainMatches {
		w.expectMatch(id, season, season <= 3)
	}
	w.archive(0)
	for season, id := range spainMatches {
		w.expectMatch(id, season, season <= 4)
	}
	w.expectMatch(franceS3, 3, true)
}

// TestArchiveOldInstants_Cups covers domestic cups by their country's season, a regional edition
// against its region's newest, and an incomplete cup left alone.
func TestArchiveOldInstants_Cups(t *testing.T) {
	w := newArchiveWorld(t)
	code, _, home, away := w.country(5)
	domestic := func(season int, completed bool) int64 {
		cup := w.scan(`INSERT INTO cups (country, season, name, completed) VALUES ($1, $2, 'Cup', $3) RETURNING cup_id`, code, season, completed)
		return w.cupTie(cup, home, away)
	}
	regional := func(edition int, completed bool) int64 {
		cup := w.scan(`INSERT INTO cups (region, season, name, completed) VALUES ($1, $2, 'Regional', $3) RETURNING cup_id`, w.region, edition, completed)
		return w.cupTie(cup, home, away)
	}
	d3, d4 := domestic(3, true), domestic(4, true)
	dOpen := domestic(2, false)
	r1, r2, r3, r4 := regional(1, true), regional(2, true), regional(3, true), regional(4, false)

	w.archive(1)
	check := func(name string, tie int64, archived bool) {
		t.Helper()
		instants := w.scan(`SELECT count(*) FROM cup_tie_instants WHERE tie_id = $1`, tie)
		flagged := w.scan(`SELECT count(*) FROM cup_ties WHERE tie_id = $1 AND instants_archived`, tie) == 1
		events := w.scan(`SELECT count(*) FROM cup_tie_events WHERE tie_id = $1`, tie)
		if archived != (instants == 0 && flagged) || (!archived && instants != archiveTestMinutes) {
			t.Errorf("%s: %d instants, flagged %v; want archived=%v", name, instants, flagged, archived)
		}
		if events != 1 {
			t.Errorf("%s lost its event", name)
		}
	}
	check("domestic season 3", d3, true)
	check("domestic season 4", d4, false)
	check("incomplete domestic season 2", dOpen, false)
	check("regional edition 1", r1, true)
	check("regional edition 2", r2, true)
	check("regional edition 3", r3, false)
	check("regional edition 4", r4, false)
}

// TestArchiveOldInstants_Friendlies covers "Friendly from an archived season" and "Recent
// friendly": the cutoff is the first league kickoff of season current - kept in the home team's
// country, and a country without a match in that season archives none of its friendlies.
func TestArchiveOldInstants_Friendlies(t *testing.T) {
	w := newArchiveWorld(t)
	_, league, home, away := w.country(5)
	start4 := time.Now().Add(-10 * 24 * time.Hour)
	w.match(league, home, away, 4, true, start4)
	w.match(league, home, away, 4, true, start4.Add(48*time.Hour))
	before := w.friendly(home, away, start4.Add(-time.Hour))
	after := w.friendly(home, away, start4.Add(time.Hour))
	_, _, lonelyHome, lonelyAway := w.country(5) // no season-4 league match at all
	lonely := w.friendly(lonelyHome, lonelyAway, start4.Add(-30*24*time.Hour))

	w.archive(1)
	count := func(id int64) int64 { return w.scan(`SELECT count(*) FROM friendly_instants WHERE match_id = $1`, id) }
	if got := count(before); got != 0 {
		t.Errorf("friendly before season 4: %d instants, want archived", got)
	}
	if w.scan(`SELECT count(*) FROM friendly_matches WHERE match_id = $1 AND instants_archived`, before) != 1 {
		t.Error("friendly before season 4 is not flagged archived")
	}
	if got := count(after); got != archiveTestMinutes {
		t.Errorf("friendly after season 4 started: %d instants, want kept", got)
	}
	if got := count(lonely); got != archiveTestMinutes {
		t.Errorf("friendly of a country without a cutoff: %d instants, want kept", got)
	}
	if got := w.scan(`SELECT count(*) FROM friendly_events WHERE match_id = $1`, before); got != 1 {
		t.Errorf("archived friendly lost its event")
	}
}

// TestInstantKeepDaysSetting covers the instant retention save: 7 days by default, saved
// as given, and a value outside 0-365 rejected without a change.
func TestInstantKeepDaysSetting(t *testing.T) {
	w := newArchiveWorld(t)
	w.exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
	before, err := GetInstantKeepDays(w.dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SetInstantKeepDays(w.dbCtx, before) })

	if got := w.scan(`SELECT column_default::int FROM information_schema.columns WHERE table_name = 'season' AND column_name = 'instants_keep_days'`); got != 7 {
		t.Errorf("default %d days, want 7", got)
	}
	if err := SetInstantKeepDays(w.dbCtx, 30); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []int{366, -1} {
		if err := SetInstantKeepDays(w.dbCtx, bad); !errors.Is(err, ErrInvalidInstantKeepDays) {
			t.Errorf("%d days: err %v, want ErrInvalidInstantKeepDays", bad, err)
		}
	}
	if got, _ := GetInstantKeepDays(w.dbCtx); got != 30 {
		t.Errorf("saved %d days, want 30", got)
	}
}

// TestArchiveExpiredInstants covers "Old match", "Recent match within the days kept", "World Cup
// group match" and "Zero days keeps instants": a played league match, cup tie or friendly older than
// the days kept loses its instants and keeps its event, rating and lineup; newer and unplayed ones
// keep them. The old fixtures sit more than a year back and the kept ones within it, so other tests'
// passes, which keep at most 365 days, never reach the ones expected to stay.
func TestArchiveExpiredInstants(t *testing.T) {
	w := newArchiveWorld(t)
	_, league, home, away := w.country(5)
	now := time.Now()
	oldAge, recentAge := 500*24*time.Hour, 300*24*time.Hour
	oldMatch := w.match(league, home, away, 5, true, now.Add(-oldAge))
	recentMatch := w.match(league, home, away, 5, true, now.Add(-recentAge))
	unplayedMatch := w.match(league, home, away, 5, false, now.Add(-oldAge))
	cup := w.scan(`INSERT INTO cups (country, season, name) SELECT l.country, 5, 'Open cup' FROM leagues l WHERE l.league_id = $1 RETURNING cup_id`, league)
	tie := func(age time.Duration) int64 {
		id := w.cupTie(cup, home, away)
		w.exec(`UPDATE cup_ties SET match_date = $1 WHERE tie_id = $2`, now.Add(-age), id)
		return id
	}
	oldTie, recentTie := tie(oldAge), tie(recentAge)
	oldFriendly := w.friendly(home, away, now.Add(-oldAge))
	recentFriendly := w.friendly(home, away, now.Add(-recentAge))
	unplayedFriendly := w.friendly(home, away, now.Add(-oldAge))
	w.exec(`UPDATE friendly_matches SET played = false WHERE match_id = $1`, unplayedFriendly)
	worldCupGroup := w.friendly(home, away, now.Add(-oldAge))
	worldCup := w.scan(`INSERT INTO world_cups (season, name) VALUES (1, 'Archive test World Cup') RETURNING world_cup_id`)
	group := w.scan(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCup)
	w.exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, group, worldCupGroup)

	tieInstants := func(id int64) int64 { return w.scan(`SELECT count(*) FROM cup_tie_instants WHERE tie_id = $1`, id) }
	friendlyInstants := func(id int64) int64 {
		return w.scan(`SELECT count(*) FROM friendly_instants WHERE match_id = $1`, id)
	}
	friendlyFlagged := func(id int64) bool {
		return w.scan(`SELECT count(*) FROM friendly_matches WHERE match_id = $1 AND instants_archived`, id) == 1
	}

	if res, err := ArchiveExpiredInstants(w.dbCtx, 0); err != nil || res != (InstantArchiveResult{}) {
		t.Fatalf("0 days: %+v, %v; want nothing archived", res, err)
	}
	w.expectMatch(oldMatch, 5, false)
	if _, err := ArchiveExpiredInstants(w.dbCtx, 366); !errors.Is(err, ErrInvalidInstantKeepDays) {
		t.Errorf("366 days: err %v, want ErrInvalidInstantKeepDays", err)
	}

	res, err := ArchiveExpiredInstants(w.dbCtx, 365)
	if err != nil {
		t.Fatal(err)
	}
	if res.LeagueMatches < 1 || res.CupTies < 1 || res.Friendlies < 2 || res.Instants < 4*archiveTestMinutes {
		t.Errorf("result %+v, want at least our old match, tie, friendly and World Cup group match", res)
	}

	w.expectMatch(oldMatch, 5, true)
	w.expectMatch(recentMatch, 5, false)
	if got := w.scan(`SELECT count(*) FROM match_instants WHERE match_id = $1`, unplayedMatch); got != archiveTestMinutes {
		t.Errorf("unplayed league match: %d instants, want kept", got)
	}
	if got := tieInstants(oldTie); got != 0 || w.scan(`SELECT count(*) FROM cup_ties WHERE tie_id = $1 AND instants_archived`, oldTie) != 1 {
		t.Errorf("old cup tie: %d instants; want archived and flagged", got)
	}
	if got := w.scan(`SELECT count(*) FROM cup_tie_events WHERE tie_id = $1`, oldTie); got != 1 {
		t.Error("archived cup tie lost its event")
	}
	if got := tieInstants(recentTie); got != archiveTestMinutes {
		t.Errorf("recent cup tie: %d instants, want kept", got)
	}
	for name, id := range map[string]int64{"old friendly": oldFriendly, "old World Cup group match": worldCupGroup} {
		if got := friendlyInstants(id); got != 0 || !friendlyFlagged(id) {
			t.Errorf("%s: %d instants, flagged %v; want archived", name, got, friendlyFlagged(id))
		}
	}
	if got := w.scan(`SELECT count(*) FROM friendly_events WHERE match_id = $1`, oldFriendly); got != 1 {
		t.Error("archived friendly lost its event")
	}
	for name, id := range map[string]int64{"recent friendly": recentFriendly, "unplayed friendly": unplayedFriendly} {
		if got := friendlyInstants(id); got != archiveTestMinutes || friendlyFlagged(id) {
			t.Errorf("%s: %d instants, flagged %v; want kept", name, got, friendlyFlagged(id))
		}
	}
}

// TestArchiveOldInstants_Concurrent runs two passes at once over the same fixtures: every eligible
// fixture ends archived once, and the two passes together flag each of ours exactly once.
func TestArchiveOldInstants_Concurrent(t *testing.T) {
	w := newArchiveWorld(t)
	_, league, home, away := w.country(5)
	past := time.Now().Add(-24 * time.Hour)
	var old []int64
	for i := 0; i < 2*InstantArchiveBatch+7; i++ {
		old = append(old, w.match(league, home, away, 1+i%3, true, past))
	}
	recent := w.match(league, home, away, 4, true, past)

	var wg sync.WaitGroup
	results := make([]InstantArchiveResult, 2)
	errs := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = ArchiveOldInstants(w.dbCtx, 1)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("concurrent ArchiveOldInstants: %s", err)
		}
	}
	if total := results[0].LeagueMatches + results[1].LeagueMatches; total < int64(len(old)) {
		t.Errorf("the two passes flagged %d league matches, want at least our %d", total, len(old))
	}
	for _, id := range old {
		if n := w.scan(`SELECT count(*) FROM match_instants WHERE match_id = $1`, id); n != 0 {
			t.Fatalf("match %d kept %d instants", id, n)
		}
	}
	w.expectMatch(recent, 4, false)
}
