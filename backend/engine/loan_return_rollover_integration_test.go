package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// loanRolloverWorld builds a loanWorld whose dbCtx also has the Redis cache the rollover needs.
func loanRolloverWorld(t *testing.T, country string) *loanWorld {
	w := newLoanWorld(t, country)
	w.dbCtx.ObjectCache = common.NewRedisCache()
	return w
}

// completeSeason marks the season country is on as fully played (one finished match, well outside
// the live-broadcast window), so advanceCountrySeasonIfComplete rolls it over.
func (w *loanWorld) completeSeason(league int64, home, away int64) {
	season, err := common.GetCurrentSeasonForCountry(w.dbCtx, w.countryOfLeague(league))
	if err != nil {
		w.t.Fatal(err)
	}
	w.exec(`UPDATE matches SET played = true, match_date = now() - interval '3 hours' WHERE league_id = $1`, league)
	w.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, $2, 1, now() - interval '3 hours', $3, $4, true)`, league, season, home, away)
}

func (w *loanWorld) countryOfLeague(league int64) string {
	var country string
	if err := w.db.QueryRow(`SELECT country FROM leagues WHERE league_id = $1`, league).Scan(&country); err != nil {
		w.t.Fatal(err)
	}
	return country
}

func (w *loanWorld) acceptLoan(playerId, lender, borrower int64, seasons int) {
	w.t.Helper()
	listing := w.scan(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, $3, 0, 0, 0) RETURNING listing_id`, playerId, lender, seasons)
	if err := common.AcceptLoan(w.dbCtx, listing, borrower); err != nil {
		w.t.Fatalf("AcceptLoan: %s", err)
	}
}

// A one-season loan between two clubs of the same country goes home at that country's next rollover.
func TestLoanReturn_SameCountry_ReturnsAtNextRollover(t *testing.T) {
	country := uniqueCountryCode()
	w := loanRolloverWorld(t, country)
	lender := w.team(country, "Lender", 0, nil, 1_000_000)
	borrower := w.team(country, "Borrower", 0, nil, 1_000_000)
	w.squad(country, lender)
	w.squad(country, borrower)
	star := w.player(country, lender, common.STRIKER, 60, 24)

	w.acceptLoan(star, lender, borrower, 1)
	if got := w.playerTeam(star); got != borrower {
		t.Fatalf("after accept the player is at %d, want borrower %d", got, borrower)
	}

	w.completeSeason(w.league, lender, borrower)
	advanceCountrySeasonIfComplete(w.dbCtx, country)

	if got := w.playerTeam(star); got != lender {
		t.Fatalf("after the rollover the player is at %d, want lender %d", got, lender)
	}
	if n := w.count(`SELECT count(*) FROM player_loans WHERE player_id = $1`, star); n != 0 {
		t.Fatalf("loan row still present after the rollover")
	}
}

// A loan to a club abroad runs on the borrower's country's clock: the lender's own rollover does
// not bring the player home, the borrower's does.
func TestLoanReturn_Abroad_ReturnsAtBorrowerCountryRollover(t *testing.T) {
	home := uniqueCountryCode()
	w := loanRolloverWorld(t, home)
	away := uniqueCountryCode()
	for away == home {
		away = uniqueCountryCode()
	}
	w.exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, away)
	awayLeague := w.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'A1', 1) RETURNING league_id`, away)

	lender := w.team(home, "Lender", 0, nil, 1_000_000)
	homeRival := w.team(home, "Home rival", 0, nil, 1_000_000)
	borrower := w.scan(`INSERT INTO teams (name, country, league_id, budget, ai_level) VALUES ('Borrower', $1, $2, 1000000, 0) RETURNING team_id`, away, awayLeague)
	awayRival := w.scan(`INSERT INTO teams (name, country, league_id, budget, ai_level) VALUES ('Away rival', $1, $2, 1000000, 0) RETURNING team_id`, away, awayLeague)
	w.squad(home, lender)
	w.squad(away, borrower)
	star := w.player(home, lender, common.STRIKER, 60, 24)
	w.acceptLoan(star, lender, borrower, 1)

	// The lender's country finishes its season; the borrower's is still being played.
	w.completeSeason(w.league, lender, homeRival)
	w.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 1, 2, now() + interval '1 day', $2, $3, false)`, awayLeague, borrower, awayRival)
	advanceCountrySeasonIfComplete(w.dbCtx, home)
	advanceCountrySeasonIfComplete(w.dbCtx, away)
	if got := w.playerTeam(star); got != borrower {
		t.Fatalf("after only the lender's rollover the player is at %d, want still at borrower %d", got, borrower)
	}

	w.completeSeason(awayLeague, borrower, awayRival)
	advanceCountrySeasonIfComplete(w.dbCtx, away)
	if got := w.playerTeam(star); got != lender {
		t.Fatalf("after the borrower's rollover the player is at %d, want lender %d", got, lender)
	}
}

// A CPU borrower whose XI includes a loanee who goes home at the rollover starts the new season
// with a full saved lineup: the rollover re-picks it once the loans have returned.
func TestLoanReturn_CpuBorrowerLineupIsFullAfterTheRollover(t *testing.T) {
	country := uniqueCountryCode()
	w := loanRolloverWorld(t, country)
	lender := w.team(country, "Lender", common.AiLevelManager, nil, 1_000_000)
	borrower := w.team(country, "Cpu Borrower", common.AiLevelDirector, nil, 1_000_000)
	w.squad(country, lender)
	w.squad(country, borrower)
	star := w.player(country, lender, common.STRIKER, 90, 24)
	w.acceptLoan(star, lender, borrower, 1)
	refreshAiTeamLineup(w.dbCtx, borrower, common.AiLevelDirector)
	if n := w.count(`SELECT count(*) FROM default_lineup_positions WHERE team_id = $1 AND player_id = $2`, borrower, star); n != 1 {
		t.Fatalf("setup: expected the loanee in the borrower's XI")
	}

	w.completeSeason(w.league, lender, borrower)
	advanceCountrySeasonIfComplete(w.dbCtx, country)

	if got := w.playerTeam(star); got != lender {
		t.Fatalf("after the rollover the player is at %d, want lender %d", got, lender)
	}
	if n := w.count(`SELECT count(*) FROM default_lineup_positions WHERE team_id = $1`, borrower); n != 11 {
		t.Fatalf("expected the borrower to start the season with a full saved XI, got %d", n)
	}
}
