package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

func TestParseSeason(t *testing.T) {
	for param, wantErr := range map[string]bool{"0": true, "-1": true, "abc": true, "": true, "1": false, "12": false} {
		if _, err := parseSeason(param); (err != nil) != wantErr {
			t.Errorf("parseSeason(%q) error = %v, wantErr %v", param, err, wantErr)
		}
	}
}

// The league season endpoints (openspec league-page "League seasons endpoint", "Past season table
// endpoint", "Matchday matches endpoint"). A 10-team league whose country is in season 3, with
// history for seasons 1 and 2. Needs STRIKER_TEST_DB_DSN.
func TestLeagueSeasonHandlers(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: &loansTestCache{values: map[string]string{}}}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	country := string([]byte{'x', byte('a' + r.Intn(26)), byte('a' + r.Intn(26)), byte('a' + r.Intn(26))})
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 3)`, country)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'S1', 1) RETURNING league_id`, country)
	teams := make([]int64, 10)
	for i := range teams {
		teams[i] = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, "Club "+strconv.Itoa(i+1), country, league)
		for season := 1; season <= 2; season++ {
			// Season 2: club i finishes (10 - i)th, so the table is not in team id order.
			position := i + 1
			if season == 2 {
				position = 10 - i
			}
			exec(`INSERT INTO season_history (team_id, season, league_id, league_name, final_position, promoted, relegated,
				matches, points, wins, draws, losses, goals_scored, goals_received)
				VALUES ($1, $2, $3, 'S1', $4, false, $5, 18, $6, 0, 0, 0, 0, 0)`, teams[i], season, league, position, position >= 9, 40-position)
		}
	}
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals, played)
		VALUES ($1, 2, 3, now() - interval '100 days', $2, $3, 2, 1, true), ($1, 3, 3, now() + interval '10 days', $3, $2, 0, 0, false)`,
		league, teams[0], teams[1])

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/division/:"+ID+"/seasons", LeagueSeasonsHandler(dbCtx))
	router.GET("/division/:"+ID+"/seasons/:season", LeagueSeasonTableHandler(dbCtx))
	router.GET("/division/:"+ID+"/weeks/:week", LeagueWeekMatchesHandler(dbCtx))
	base := "/division/" + strconv.FormatInt(league, 10)

	// Third season.
	rec := serve(router, http.MethodGet, base+"/seasons")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"current":3,"past":[2,1]}` {
		t.Errorf("GET seasons = %d %s, want 200 {\"current\":3,\"past\":[2,1]}", rec.Code, rec.Body.String())
	}
	if rec := serve(router, http.MethodGet, "/division/abc/seasons"); rec.Code != http.StatusBadRequest {
		t.Errorf("GET seasons of a non-numeric league = %d, want 400", rec.Code)
	}

	// Season 2 table, twice: the second read is served from the cache.
	for _, from := range []string{"database", "cache"} {
		rec = serve(router, http.MethodGet, base+"/seasons/2")
		var table common.LeagueSeasonTable
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &table) != nil {
			t.Fatalf("GET seasons/2 (%s) = %d: %s", from, rec.Code, rec.Body.String())
		}
		if len(table.Positions) != 10 || table.TotalWeeks != 18 || table.Name != "S1" || table.Season != 2 {
			t.Fatalf("season 2 table (%s) = %d positions, %d weeks, name %q; want 10, 18, S1", from, len(table.Positions), table.TotalWeeks, table.Name)
		}
		for i, p := range table.Positions {
			if p.Position != i+1 {
				t.Errorf("season 2 table (%s) row %d has position %d", from, i, p.Position)
			}
		}
		if first := table.Positions[0]; first.TeamId != teams[9] || first.Name != "Club 10" || first.Points != 39 || first.Matches != 18 {
			t.Errorf("season 2 champion (%s) = %+v, want Club 10 with 39 points", from, first)
		}
		if last := table.Positions[9]; !last.Relegated || last.TeamId != teams[0] {
			t.Errorf("season 2 last (%s) = %+v, want Club 1 relegated", from, last)
		}
	}

	if v, _ := dbCtx.ObjectCache.GetValue("leagueSeasonTable_key_" + strconv.FormatInt(league, 10) + "_2"); v == "" {
		t.Error("season 2 table was not cached")
	}

	// Current season, a season never played, and an invalid season.
	for path, want := range map[string]int{base + "/seasons/3": 404, base + "/seasons/9": 404, base + "/seasons/0": 400, base + "/seasons/abc": 400} {
		if rec := serve(router, http.MethodGet, path); rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}

	// Past season week, and the current season when no season is given.
	for path, wantSeason := range map[string]int{base + "/weeks/3?season=2": 2, base + "/weeks/3": 3} {
		rec := serve(router, http.MethodGet, path)
		var matches []common.WeekMatch
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &matches) != nil {
			t.Fatalf("GET %s = %d: %s", path, rec.Code, rec.Body.String())
		}
		if len(matches) != 1 || matches[0].Season != wantSeason {
			t.Errorf("GET %s = %+v, want the one match of season %d", path, matches, wantSeason)
		}
	}

	// Invalid week.
	for _, path := range []string{base + "/weeks/0", base + "/weeks/abc", base + "/weeks/3?season=0", base + "/weeks/3?season=x"} {
		if rec := serve(router, http.MethodGet, path); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", path, rec.Code)
		}
	}
}
