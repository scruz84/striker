package engine

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestAiTransferMarket_Simulation runs several days of CPU management over a small world against a
// real Postgres (STRIKER_TEST_DB_DSN; skipped when unset) - six CPU teams (two per AI level), two
// human managers who turn down everything offered to them, and a few good free agents - and
// checks the invariants the transfer plan promises rather than exact outcomes (the CPU's choices
// are randomized): moves happen when upgrades exist, a Caretaker never shops, only a Director
// bids for unlisted players, no CPU squad drops under the minimum size or a position under its
// minimum depth, the caps on open offers hold, a rejected team doesn't bid again straight away,
// nobody is signed and sold on at once, and no international is put on the transfer list.
func TestAiTransferMarket_Simulation(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &common.DatabaseContext{Db: db}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	common.ResetNationalRadarCache() // pools are rebuilt over this world's players
	exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
	exec(`UPDATE season SET transfer_window_open = true, offer_negotiation_window_minutes = 0`)
	exec(`INSERT INTO countries (name, current_season) VALUES ('zx', 1) ON CONFLICT (name) DO NOTHING`)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('zx', 1, 'L1', 1) RETURNING league_id`)

	gen := rand.New(rand.NewSource(42))
	newPlayer := func(teamId any, position int, skill int, age int) int64 {
		var gk, def, lat, pass, mid, dri, shot int
		switch position {
		case common.GOALKEEPER:
			gk = skill
		case common.DEFENSE:
			def = skill
		case common.MIDFIELD:
			mid, pass = skill, skill
		default:
			shot, pass, dri = skill, skill, skill
		}
		wage := common.PlayerWageFromSkills(gk, def, lat, pass, mid, dri, shot)
		shirt := any(nil)
		if teamId != nil {
			shirt = scan(`SELECT COALESCE(MAX(shirt_number), 0) + 1 FROM players WHERE team_id = $1`, teamId)
		}
		return scan(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
			dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number,
			goal_keeper_max, defense_max, lateral_max, pass_max, midfield_max, dribbling_max, shot_max)
			VALUES ($1, 'Sim', 'Player', 0, $2, $3, $4, $5, $6, $7, $8, false, false, false, false, '', $9, $10, 3, 'zx', $11,
			$12, $13, $14, $15, $16, $17, $18) RETURNING player_id`,
			teamId, gk, def, lat, pass, mid, dri, shot, wage, age, shirt,
			min(99, gk+10), min(99, def+10), 10, min(99, pass+10), min(99, mid+10), min(99, dri+10), min(99, shot+10))
	}
	// A balanced 18: 2 GK, 5 D, 6 M, 5 S, skills 45-75, ages 21-33.
	seedSquad := func(teamId int64) {
		for _, n := range []struct{ position, count int }{
			{common.GOALKEEPER, 2}, {common.DEFENSE, 5}, {common.MIDFIELD, 6}, {common.STRIKER, 5},
		} {
			for i := 0; i < n.count; i++ {
				newPlayer(teamId, n.position, 45+gen.Intn(31), 21+gen.Intn(13))
			}
		}
	}
	newTeam := func(name string, level int, owner any) int64 {
		return scan(`INSERT INTO teams (name, country, league_id, budget, ai_level, owner_id) VALUES ($1, 'zx', $2, 5000000, $3, $4) RETURNING team_id`,
			name, league, level, owner)
	}

	type cpuTeam struct {
		id    int64
		level int
	}
	cpus := make([]cpuTeam, 0)
	for _, level := range []int{common.AiLevelCaretaker, common.AiLevelCaretaker, common.AiLevelManager, common.AiLevelManager,
		common.AiLevelDirector, common.AiLevelDirector} {
		id := newTeam(fmt.Sprintf("Sim CPU %d", len(cpus)), level, nil)
		seedSquad(id)
		cpus = append(cpus, cpuTeam{id, level})
	}
	humans := []int64{newTeam("Sim Human 0", 0, 111111), newTeam("Sim Human 1", 0, 222222)}
	humanStars := make([]int64, 0)
	for _, h := range humans {
		seedSquad(h)
		humanStars = append(humanStars, newPlayer(h, common.STRIKER, 97, 26))
	}
	for i := 0; i < 6; i++ {
		newPlayer(nil, []int{common.GOALKEEPER, common.DEFENSE, common.MIDFIELD, common.STRIKER}[i%4], 85+gen.Intn(10), 25)
	}

	startedAt := time.Now().Add(-time.Second)
	rejectedAt := map[[2]int64]time.Time{} // (player, team) -> when the human turned the team down

	checkInvariants := func(day int) {
		t.Helper()
		for _, c := range cpus {
			players, err := common.GetPlayersFromTeamId(dbCtx, c.id)
			if err != nil {
				t.Fatal(err)
			}
			if len(*players) < common.MinSquadSizeAfterSale {
				t.Errorf("day %d: team %d (level %d) has %d players, under the minimum %d", day, c.id, c.level, len(*players), common.MinSquadSizeAfterSale)
			}
			for _, position := range common.CorePositions {
				if depth := positionDepth(*players, position); depth < common.MinPositionDepth[position] {
					t.Errorf("day %d: team %d (level %d) has %d players at position %d, under the minimum %d",
						day, c.id, c.level, depth, position, common.MinPositionDepth[position])
				}
			}
			var open int
			if err := db.QueryRow(`SELECT count(*) FROM transfer_offers WHERE from_team_id = $1`, c.id).Scan(&open); err != nil {
				t.Fatal(err)
			}
			if open > common.AiMaxOpenOffersPerTeam {
				t.Errorf("day %d: team %d has %d offers open, over the cap %d", day, c.id, open, common.AiMaxOpenOffersPerTeam)
			}
		}
		for _, star := range humanStars {
			n, err := common.CountOpenAiOffersOnPlayer(dbCtx, star)
			if err != nil {
				t.Fatal(err)
			}
			if n > common.AiMaxOpenOffersPerHumanPlayer {
				t.Errorf("day %d: human player %d has %d CPU offers open, over the cap %d", day, star, n, common.AiMaxOpenOffersPerHumanPlayer)
			}
		}
		for key := range rejectedAt {
			var again int
			if err := db.QueryRow(`SELECT count(*) FROM transfer_offers WHERE player_id = $1 AND from_team_id = $2`, key[0], key[1]).Scan(&again); err != nil {
				t.Fatal(err)
			}
			if again != 0 {
				t.Errorf("day %d: team %d bid again for player %d right after being turned down", day, key[1], key[0])
			}
		}
	}
	checkInvariants(0)

	const days = 12
	for day := 1; day <= days; day++ {
		for _, c := range cpus {
			RunAITeamManagement(dbCtx, c.id)
		}
		RespondToAllPendingOffers(dbCtx)
		checkInvariants(day)

		// The humans decline every CPU offer, remembering who they turned down.
		for _, h := range humans {
			offers, err := common.GetOffersReceived(dbCtx, h)
			if err != nil {
				t.Fatal(err)
			}
			for _, o := range offers {
				if err := common.RejectOffer(dbCtx, o.OfferId, h); err != nil {
					t.Fatal(err)
				}
				rejectedAt[[2]int64{o.Player.Id, o.FromTeamId}] = time.Now()
			}
		}
		checkInvariants(day)
	}

	isLevel := func(level int) []int64 {
		ids := make([]int64, 0)
		for _, c := range cpus {
			if c.level == level {
				ids = append(ids, c.id)
			}
		}
		return ids
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	for _, caretaker := range isLevel(common.AiLevelCaretaker) {
		if n := count(`SELECT count(*) FROM player_transfers WHERE to_team_id = $1 AND created_at > $2`, caretaker, startedAt); n != 0 {
			t.Errorf("caretaker %d signed %d players, want 0", caretaker, n)
		}
		if n := count(`SELECT count(*) FROM player_offer_rejections WHERE from_team_id = $1`, caretaker); n != 0 {
			t.Errorf("caretaker %d made offers", caretaker)
		}
	}
	for _, manager := range isLevel(common.AiLevelManager) {
		if n := count(`SELECT count(*) FROM player_offer_rejections WHERE from_team_id = $1`, manager); n != 0 {
			t.Errorf("manager %d bid for unlisted players, want only Directors to", manager)
		}
	}

	// A signing isn't turned around and sold on at once: nobody moves more than once or twice in
	// twelve days (a free agent signed, then a buyer for him later at most).
	rows, err := db.Query(`SELECT player_id, count(*) FROM player_transfers WHERE created_at > $1 GROUP BY player_id HAVING count(*) > 2`, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var playerId int64
		var moves int
		if err := rows.Scan(&playerId, &moves); err != nil {
			t.Fatal(err)
		}
		t.Errorf("player %d changed club %d times in %d days", playerId, moves, days)
	}
	rows.Close()

	// Internationals are protected: no CPU team listed a player of its country's national-team
	// pool as surplus (nor to fund a signing).
	radar, err := common.GetNationalRadar(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cpus {
		listed, err := common.GetTeamMarketCommitments(dbCtx, c.id)
		if err != nil {
			t.Fatal(err)
		}
		for playerId := range listed.TransferListed {
			if _, international := radar.Lookup(playerId); international {
				t.Errorf("team %d (level %d) listed international %d", c.id, c.level, playerId)
			}
		}
	}

	signings := 0
	for _, c := range cpus {
		signings += count(`SELECT count(*) FROM player_transfers WHERE to_team_id = $1 AND created_at > $2`, c.id, startedAt)
	}
	offers := 0
	for _, d := range isLevel(common.AiLevelDirector) {
		offers += count(`SELECT count(*) FROM player_offer_rejections WHERE from_team_id = $1`, d)
	}
	if signings == 0 {
		t.Error("no CPU team signed anyone although strong free agents were available")
	}
	if offers == 0 {
		t.Errorf("no Director bid for the humans' 97-rated strikers in %d days", days)
	}
	t.Logf("over %d days: %d signings by CPU teams, %d offers by Directors (all turned down)", days, signings, offers)
}
