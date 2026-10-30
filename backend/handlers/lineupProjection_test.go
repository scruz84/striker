package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// projectionTeam seeds a team with 14 players (a keeper and thirteen outfielders), its default
// lineup, and returns the lineup's first outfield starter.
func projectionTeam(t *testing.T) (*common.DatabaseContext, *sql.DB, int64, *common.Lineup, common.LineupPosition) {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	country := fmt.Sprintf("%04s", strconv.FormatInt(time.Now().UnixNano()%1679616, 36)) // countries.name is 4 characters
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	var leagueId, teamId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L', 1) RETURNING league_id`, country).Scan(&leagueId); err != nil {
		t.Fatalf("seeding league: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, country+" FC", country, leagueId).Scan(&teamId); err != nil {
		t.Fatalf("seeding team: %s", err)
	}
	for i := 0; i < 14; i++ {
		keeper, skill := 10, 60
		if i == 0 {
			keeper = 80
		}
		if i >= 11 {
			skill = 40 + i // the bench: 13th and 14th are the stronger reserves
		}
		if _, err := db.Exec(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield, dribbling, shot,
			header, quick, creative, technical, avatar, age, shirt_number, stamina, form)
			VALUES ($1, 'P', $2, 0, $3, $4, $4, $4, $4, $4, $4, false, false, false, false, '{}', 25, $5, 80, 50)`,
			teamId, fmt.Sprintf("%s%d", country, i), keeper, skill, i+1); err != nil {
			t.Fatalf("seeding player: %s", err)
		}
	}
	best, err := common.BestLineupForTeam(dbCtx, teamId)
	if err != nil {
		t.Fatalf("BestLineupForTeam: %s", err)
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, best.Positions); err != nil {
		t.Fatalf("SetDefaultLineup: %s", err)
	}
	lineup, err := common.GetDefaultLineup(dbCtx, teamId)
	if err != nil {
		t.Fatalf("GetDefaultLineup: %s", err)
	}
	for _, p := range lineup.Positions {
		if p.Position != common.GOALKEEPER {
			return dbCtx, db, teamId, lineup, p
		}
	}
	t.Fatal("no outfield starter in the seeded lineup")
	return nil, nil, 0, nil, common.LineupPosition{}
}

func TestWithProjectedReplacements_InjuredStarterShowsWhoReplacesHim(t *testing.T) {
	dbCtx, db, teamId, lineup, starter := projectionTeam(t)
	if _, err := db.Exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 3)`, starter.PlayerId, common.UNAVAILABLE_INJURY); err != nil {
		t.Fatalf("injuring the starter: %s", err)
	}
	saved := append([]common.LineupPosition(nil), lineup.Positions...)

	got := withProjectedReplacements(dbCtx, teamId, lineup, false)

	if len(got.ProjectedReplacements) != 1 || got.ProjectedReplacements[0].PlayerOutId != starter.PlayerId ||
		got.ProjectedReplacements[0].Position != starter.Position {
		t.Fatalf("expected one projected replacement for the injured starter, got %+v", got.ProjectedReplacements)
	}
	if replacement := got.ProjectedReplacements[0].PlayerInId; replacement == 0 || replacement == starter.PlayerId {
		t.Fatalf("expected a real replacement, got %+v", got.ProjectedReplacements)
	}
	for i, p := range saved {
		if got.Positions[i] != p {
			t.Fatalf("expected the saved lineup unchanged, got %v", got.Positions)
		}
	}
	stored, err := common.GetDefaultLineup(dbCtx, teamId)
	if err != nil {
		t.Fatalf("GetDefaultLineup: %s", err)
	}
	if len(stored.ProjectedReplacements) != 0 {
		t.Fatalf("expected nothing about the projection to be stored, got %+v", stored.ProjectedReplacements)
	}
}

func TestWithProjectedReplacements_RecoveredPlayerShowsNoReplacement(t *testing.T) {
	dbCtx, db, teamId, lineup, starter := projectionTeam(t)
	if _, err := db.Exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 0)`, starter.PlayerId, common.UNAVAILABLE_INJURY); err != nil {
		t.Fatalf("recording a finished injury: %s", err)
	}

	got := withProjectedReplacements(dbCtx, teamId, lineup, false)

	if len(got.ProjectedReplacements) != 0 {
		t.Fatalf("expected no replacement once the injury has run out, got %+v", got.ProjectedReplacements)
	}
}

func TestWithProjectedReplacements_SuspensionDoesNotBarAFriendly(t *testing.T) {
	dbCtx, db, teamId, lineup, starter := projectionTeam(t)
	if _, err := db.Exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 2)`, starter.PlayerId, common.UNAVAILABLE_SUSPENSION); err != nil {
		t.Fatalf("suspending the starter: %s", err)
	}

	league := withProjectedReplacements(dbCtx, teamId, lineup, false)
	if len(league.ProjectedReplacements) != 1 {
		t.Fatalf("expected a replacement for the suspended starter in a league match, got %+v", league.ProjectedReplacements)
	}
	league.ProjectedReplacements = nil

	friendly := withProjectedReplacements(dbCtx, teamId, lineup, true)
	if len(friendly.ProjectedReplacements) != 0 {
		t.Fatalf("expected no replacement in a friendly, got %+v", friendly.ProjectedReplacements)
	}
}
