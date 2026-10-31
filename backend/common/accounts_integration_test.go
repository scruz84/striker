package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset) - see
// TestGetAllTeamIds_ExcludesNationalTeams for the convention. Every fixture is created under its
// own ids and looked up by them, so reruns against the same database do not interfere.

const accTestCountry = "jm"

func accDb(t *testing.T) *DatabaseContext {
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
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, accTestCountry); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	return dbCtx
}

func accMust(t *testing.T, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", what, err)
	}
}

func accInt(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	accMust(t, db.QueryRow(query, args...).Scan(&n), "query "+query)
	return n
}

var accSeq int64

func accUnique() string {
	accSeq++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), accSeq)
}

func accSeedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	accMust(t, db.QueryRow(`INSERT INTO users (name, provider, email, country) VALUES ('Real Person', 'google', $1, $2) RETURNING user_id`,
		"person-"+accUnique()+"@example.com", accTestCountry).Scan(&id), "seeding user")
	return id
}

func accSeedLeague(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	accMust(t, db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'I', 1) RETURNING league_id`, accTestCountry).Scan(&id), "seeding league")
	return id
}

// accSeedTeam creates a team with a full senior squad and a couple of academy prospects.
func accSeedTeam(t *testing.T, dbCtx *DatabaseContext, leagueId int64, ownerId *int64, aiLevel int) int64 {
	t.Helper()
	var id int64
	accMust(t, dbCtx.Db.QueryRow(`INSERT INTO teams (name, country, league_id, owner_id, ai_level, stadium_name) VALUES ($1, $2, $3, $4, $5, 'Old Ground') RETURNING team_id`,
		"Club "+accUnique(), accTestCountry, leagueId, ownerId, aiLevel).Scan(&id), "seeding team")
	tx, err := dbCtx.Db.Begin()
	accMust(t, err, "begin")
	stmt, err := tx.Prepare(InsertNewPlayerStmtSql)
	accMust(t, err, "prepare")
	doCreatePlayers(dbCtx, id, stmt, tx, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, accTestCountry, false)
	doCreatePlayers(dbCtx, id, stmt, tx, 2, 20, 40, 15, 17, accTestCountry, true)
	stmt.Close()
	accMust(t, tx.Commit(), "commit squad")
	return id
}

func accPlayerIds(t *testing.T, db *sql.DB, teamId int64, academy bool) []int64 {
	t.Helper()
	rows, err := db.Query(`SELECT player_id FROM players WHERE team_id = $1 AND in_academy = $2 ORDER BY player_id`, teamId, academy)
	accMust(t, err, "listing players")
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		accMust(t, rows.Scan(&id), "scan")
		ids = append(ids, id)
	}
	return ids
}

func accRelease(t *testing.T, dbCtx *DatabaseContext, userId int64, reason string) int64 {
	t.Helper()
	tx, err := dbCtx.Db.Begin()
	accMust(t, err, "begin")
	defer tx.Rollback()
	teamId, err := ReleaseUserTeam(tx, userId, reason, true)
	accMust(t, err, "ReleaseUserTeam")
	accMust(t, tx.Commit(), "commit release")
	return teamId
}

func TestResetReleasedTeam_FullReset(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	owner := accSeedUser(t, db)
	team := accSeedTeam(t, dbCtx, league, &owner, 0)
	other := accSeedTeam(t, dbCtx, league, nil, 1)

	// State a previous owner could have built up.
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE teams SET budget = 12345, ticket_price = 99, stadium_capacity = 7777, fanbase = 91, socios = 9,
			friendly_formation = 3, formation = 2, name_changed = true, sponsor_deal = 4 WHERE team_id = $1`, team)
		return err
	}(), "customizing team")
	squad := accPlayerIds(t, db, team, false)
	academy := accPlayerIds(t, db, team, true)
	otherSquad := accPlayerIds(t, db, other, false)
	if len(squad) != SquadSize || len(academy) != 2 {
		t.Fatalf("fixture squad = %d senior / %d academy", len(squad), len(academy))
	}

	loanedIn, loanedOut := otherSquad[0], squad[0]
	accMust(t, func() error {
		// loanedIn plays for `team` on loan from `other`; loanedOut plays for `other` on loan from `team`.
		if _, err := db.Exec(`UPDATE players SET team_id = $1, shirt_number = NULL WHERE player_id = $2`, team, loanedIn); err != nil {
			return err
		}
		if _, err := db.Exec(`UPDATE players SET team_id = $1, shirt_number = NULL WHERE player_id = $2`, other, loanedOut); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season) VALUES ($1, $2, $3, 5), ($4, $3, $2, 5)`,
			loanedIn, other, team, loanedOut); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, 1000)`, squad[1], team); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons) VALUES ($1, $2, 1)`, squad[2], team); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount) VALUES ($1, $2, $3, 500), ($4, $3, $2, 500)`,
			squad[3], other, team, otherSquad[1]); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO friendly_challenges (challenger_team_id, opponent_team_id) VALUES ($1, $2)`, other, team); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO trainers (first_name, last_name, team_id, skill, wage) VALUES ('Pep', 'Coach', $1, 50, 100)`, team); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO default_lineup_positions (team_id, player_id, position) VALUES ($1, $2, 0)`, team, squad[4]); err != nil {
			return err
		}
		return nil
	}(), "building market and loan state")
	var scheduledFriendly int64
	accMust(t, db.QueryRow(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '3 days') RETURNING match_id`,
		team, other).Scan(&scheduledFriendly), "scheduling a friendly")
	ledgerBefore := accInt(t, db, `SELECT count(*) FROM team_transactions WHERE team_id = $1`, team)
	historyBefore := accInt(t, db, `SELECT count(*) FROM player_club_history WHERE team_id = $1`, team)

	// Pending clubs are not handed to new owners.
	accRelease(t, dbCtx, owner, ReleaseReasonInactivity)
	for i := 0; i < 40; i++ {
		var free int64
		accMust(t, db.QueryRow(GetFreeTeamStmtSql, accTestCountry).Scan(&free), "GetFreeTeam")
		if free == team {
			t.Fatalf("a club pending reset was offered to a new manager")
		}
	}
	if name := accString(t, db, `SELECT COALESCE(released_team_name, '') FROM users WHERE user_id = $1`, owner); name == "" {
		t.Errorf("the owner should be told which club was released")
	}

	reset, returned, err := ResetReleasedTeam(dbCtx, team)
	accMust(t, err, "ResetReleasedTeam")
	if !reset {
		t.Fatalf("the club should have been reset")
	}
	if len(returned) != 1 || returned[0].PlayerId != loanedIn {
		t.Errorf("returned loans = %+v, want the one loaned-in player", returned)
	}

	// Old senior squad: free agents remembering the club; loaned-in player home; academy gone.
	for _, p := range squad[1:] {
		var teamId sql.NullInt64
		var last sql.NullInt64
		var years int
		accMust(t, db.QueryRow(`SELECT team_id, last_team_id, contract_years_remaining FROM players WHERE player_id = $1`, p).Scan(&teamId, &last, &years), "old player")
		if loanedOutOnLoanAtOther(p, loanedOut) {
			continue
		}
		if teamId.Valid || !last.Valid || last.Int64 != team || years != 0 {
			t.Fatalf("old player %d: team %v last %v contract %d, want a free agent of club %d", p, teamId, last, years, team)
		}
	}
	if accInt(t, db, `SELECT team_id FROM players WHERE player_id = $1`, loanedIn) != other {
		t.Errorf("the loaned-in player should be back at his own club")
	}
	if accInt(t, db, `SELECT count(*) FROM player_loans WHERE to_team_id = $1`, team) != 0 {
		t.Errorf("loans to the reset club should be gone")
	}
	if got := accInt(t, db, `SELECT count(*) FROM players WHERE player_id = ANY($1)`, pqArray(academy)); got != 0 {
		t.Errorf("academy prospects should be deleted, %d remain", got)
	}
	// Loaned-out player stays at the borrower until the loan ends, then becomes a free agent.
	if !accBool(t, db, `SELECT release_on_return FROM player_loans WHERE player_id = $1`, loanedOut) {
		t.Errorf("the loan of the reset club's player should be flagged release_on_return")
	}
	if accInt(t, db, `SELECT team_id FROM players WHERE player_id = $1`, loanedOut) != other {
		t.Errorf("the loaned-out player stays with the borrower until the loan ends")
	}

	// Market and social state cleared; scheduled friendly kept.
	for _, q := range []string{
		`SELECT count(*) FROM transfer_listings WHERE selling_team_id = $1`,
		`SELECT count(*) FROM loan_listings WHERE lending_team_id = $1`,
		`SELECT count(*) FROM transfer_offers WHERE from_team_id = $1 OR to_team_id = $1`,
		`SELECT count(*) FROM friendly_challenges WHERE challenger_team_id = $1 OR opponent_team_id = $1`,
		`SELECT count(*) FROM trainers WHERE team_id = $1`,
	} {
		if n := accInt(t, db, q, team); n != 0 {
			t.Errorf("%s = %d, want 0", q, n)
		}
	}
	if accInt(t, db, `SELECT count(*) FROM friendly_matches WHERE match_id = $1`, scheduledFriendly) != 1 {
		t.Errorf("a scheduled friendly must be kept")
	}

	// New squad, identity, settings at defaults, budget through the ledger, CPU level drawn.
	if got := len(accPlayerIds(t, db, team, false)); got != SquadSize {
		t.Errorf("new senior squad = %d, want %d", got, SquadSize)
	}
	var budget int64
	var capacity, price, fanbase, socios, formation, friendlyFormation, sponsor, ai int
	var nameChanged bool
	var pending sql.NullTime
	var stadium string
	var owner2 sql.NullInt64
	accMust(t, db.QueryRow(`SELECT budget, stadium_capacity, ticket_price, fanbase, socios, formation, friendly_formation, sponsor_deal, ai_level,
		name_changed, reset_pending_since, stadium_name, owner_id FROM teams WHERE team_id = $1`, team).
		Scan(&budget, &capacity, &price, &fanbase, &socios, &formation, &friendlyFormation, &sponsor, &ai, &nameChanged, &pending, &stadium, &owner2), "reading reset team")
	if budget != DefaultTeamBudget || capacity != 1000 || price != 10 || fanbase != 50 || socios != 500 || formation != 0 || friendlyFormation != 0 || sponsor != 1 {
		t.Errorf("settings not at defaults: budget %d capacity %d price %d fanbase %d socios %d formation %d/%d sponsor %d",
			budget, capacity, price, fanbase, socios, formation, friendlyFormation, sponsor)
	}
	if ai < AiLevelCaretaker || ai > AiLevelDirector || nameChanged || pending.Valid || owner2.Valid || stadium == "Old Ground" {
		t.Errorf("identity/queue state wrong: ai %d nameChanged %v pending %v owner %v stadium %q", ai, nameChanged, pending, owner2, stadium)
	}
	var delta int64
	accMust(t, db.QueryRow(`SELECT amount FROM team_transactions WHERE team_id = $1 AND reason = $2`, team, TRANSACTION_TEAM_RESET).Scan(&delta), "reset ledger row")
	if delta != DefaultTeamBudget-12345 {
		t.Errorf("ledger delta = %d, want %d", delta, DefaultTeamBudget-12345)
	}
	if accInt(t, db, `SELECT count(*) FROM team_transactions WHERE team_id = $1`, team) != ledgerBefore+1 {
		t.Errorf("history must be kept and exactly one ledger row added")
	}
	if accInt(t, db, `SELECT count(*) FROM player_club_history WHERE team_id = $1`, team) < historyBefore {
		t.Errorf("player club history must not shrink")
	}
	if accInt(t, db, `SELECT count(*) FROM default_lineup_positions WHERE team_id = $1`, team) == 0 {
		t.Errorf("a default lineup should be seeded for the new squad")
	}

	// A second pass is a no-op.
	if again, _, err := ResetReleasedTeam(dbCtx, team); err != nil || again {
		t.Errorf("a club already reset must not be reset again (reset=%v err=%v)", again, err)
	}
}

func loanedOutOnLoanAtOther(p, loanedOut int64) bool { return p == loanedOut }

func TestResetReleasedTeam_WaitsForMatchesInGuardWindow(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	owner := accSeedUser(t, db)
	team := accSeedTeam(t, dbCtx, league, &owner, 0)
	other := accSeedTeam(t, dbCtx, league, nil, 1)
	var matchId int64
	accMust(t, db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() + interval '10 minutes', $2, $3) RETURNING match_id`, league, team, other).Scan(&matchId), "seeding match")
	accRelease(t, dbCtx, owner, ReleaseReasonSelfDeletion)

	if reset, _, err := ResetReleasedTeam(dbCtx, team); err != nil || reset {
		t.Fatalf("a club with a match 10 minutes away must wait (reset=%v err=%v)", reset, err)
	}
	if accInt(t, db, `SELECT count(*) FROM teams WHERE team_id = $1 AND reset_pending_since IS NOT NULL`, team) != 1 {
		t.Fatalf("the club must stay in the queue")
	}
	// Same inside the broadcast window after kickoff, and while in simulation.
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE matches SET match_date = now() - interval '90 minutes' WHERE match_id = $1`, matchId)
		return err
	}(), "moving match")
	if reset, _, _ := ResetReleasedTeam(dbCtx, team); reset {
		t.Fatalf("a match still inside the live window must hold the reset")
	}
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE matches SET match_date = now() - interval '3 hours', in_simulation = true WHERE match_id = $1`, matchId)
		return err
	}(), "marking in simulation")
	if reset, _, _ := ResetReleasedTeam(dbCtx, team); reset {
		t.Fatalf("a match in simulation must hold the reset")
	}
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE matches SET in_simulation = false, played = true WHERE match_id = $1`, matchId)
		return err
	}(), "finishing match")
	if reset, _, err := ResetReleasedTeam(dbCtx, team); err != nil || !reset {
		t.Fatalf("once the match is long over the reset runs (reset=%v err=%v)", reset, err)
	}
}

func TestReturnLoan_ReleasesPlayerOfResetLender(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	lender := accSeedTeam(t, dbCtx, league, nil, 1)
	borrower := accSeedTeam(t, dbCtx, league, nil, 1)
	player := accPlayerIds(t, db, lender, false)[0]
	accMust(t, func() error {
		if _, err := db.Exec(`UPDATE players SET team_id = $1, shirt_number = NULL WHERE player_id = $2`, borrower, player); err != nil {
			return err
		}
		_, err := db.Exec(`INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season, release_on_return) VALUES ($1, $2, $3, 2, true)`, player, lender, borrower)
		return err
	}(), "seeding flagged loan")

	loans, err := GetExpiredLoans(dbCtx, accTestCountry, 99)
	accMust(t, err, "GetExpiredLoans")
	var loan *PlayerLoan
	for i := range loans {
		if loans[i].PlayerId == player {
			loan = &loans[i]
		}
	}
	if loan == nil || !loan.ReleaseOnReturn {
		t.Fatalf("the expired loan should carry release_on_return, got %+v", loan)
	}
	accMust(t, ReturnLoan(dbCtx, *loan), "ReturnLoan")

	var teamId, last sql.NullInt64
	accMust(t, db.QueryRow(`SELECT team_id, last_team_id FROM players WHERE player_id = $1`, player).Scan(&teamId, &last), "reading player")
	if teamId.Valid || !last.Valid || last.Int64 != lender {
		t.Errorf("the player should be a free agent remembering his lender, got team %v last %v", teamId, last)
	}
	if accInt(t, db, `SELECT count(*) FROM player_loans WHERE player_id = $1`, player) != 0 {
		t.Errorf("the loan should be deleted")
	}
}

func TestDeleteAccount_ErasesPersonalDataAndReleasesClub(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	owner := accSeedUser(t, db)
	team := accSeedTeam(t, dbCtx, league, &owner, 0)
	early, late := accSeedUser(t, db), accSeedUser(t, db)
	email := accString(t, db, `SELECT email FROM users WHERE user_id = $1`, owner)

	var withMembers, alone, joined int64
	accMust(t, db.QueryRow(`INSERT INTO groups (name, owner_id, invite_code) VALUES ('Crew', $1, $2) RETURNING group_id`, owner, "c"+accUnique()[:18]).Scan(&withMembers), "group")
	accMust(t, db.QueryRow(`INSERT INTO groups (name, owner_id, invite_code) VALUES ('Solo', $1, $2) RETURNING group_id`, owner, "s"+accUnique()[:18]).Scan(&alone), "group")
	accMust(t, db.QueryRow(`INSERT INTO groups (name, owner_id, invite_code) VALUES ('Joined', $1, $2) RETURNING group_id`, late, "j"+accUnique()[:18]).Scan(&joined), "group")
	accMust(t, func() error {
		for _, q := range []string{
			fmt.Sprintf(`INSERT INTO group_members (group_id, user_id, joined_at) VALUES (%d, %d, now() - interval '3 days'), (%d, %d, now() - interval '2 days'), (%d, %d, now() - interval '1 day')`,
				withMembers, owner, withMembers, early, withMembers, late),
			fmt.Sprintf(`INSERT INTO group_members (group_id, user_id) VALUES (%d, %d), (%d, %d)`, alone, owner, joined, owner),
			fmt.Sprintf(`INSERT INTO notifications (user_id, notification_type, message) VALUES (%d, 0, 'hello')`, owner),
			fmt.Sprintf(`INSERT INTO notification_preferences (user_id, email_enabled, push_enabled) VALUES (%d, true, true)`, owner),
			fmt.Sprintf(`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES (%d, 'https://push.example/1', 'SECRET-P256DH', 'SECRET-AUTH')`, owner),
		} {
			if _, err := db.Exec(q); err != nil {
				return fmt.Errorf("%w (%s)", err, q)
			}
		}
		return nil
	}(), "seeding user data")

	// Export first: it carries the email and notification, never the push keys.
	export, err := ExportUserData(dbCtx, owner)
	accMust(t, err, "ExportUserData")
	raw, _ := json.Marshal(export)
	for _, want := range []string{email, `"hello"`, "https://push.example/1", `"Crew"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("export misses %s: %s", want, raw)
		}
	}
	for _, secret := range []string{"SECRET-P256DH", "SECRET-AUTH"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("export must not contain push key material %q", secret)
		}
	}
	if export.Club == nil || export.Club.TeamId != team {
		t.Errorf("export club = %+v, want team %d", export.Club, team)
	}

	releasedTeam, err := DeleteAccount(dbCtx, owner, ReleaseReasonSelfDeletion)
	accMust(t, err, "DeleteAccount")
	if releasedTeam != team {
		t.Errorf("released team = %d, want %d", releasedTeam, team)
	}
	for _, q := range []string{
		`SELECT count(*) FROM users WHERE user_id = $1`,
		`SELECT count(*) FROM notifications WHERE user_id = $1`,
		`SELECT count(*) FROM notification_preferences WHERE user_id = $1`,
		`SELECT count(*) FROM push_subscriptions WHERE user_id = $1`,
		`SELECT count(*) FROM group_members WHERE user_id = $1`,
	} {
		if n := accInt(t, db, q, owner); n != 0 {
			t.Errorf("%s = %d, want 0", q, n)
		}
	}
	var ownerId sql.NullInt64
	var ai int
	var pending sql.NullTime
	accMust(t, db.QueryRow(`SELECT owner_id, ai_level, reset_pending_since FROM teams WHERE team_id = $1`, team).Scan(&ownerId, &ai, &pending), "reading club")
	if ownerId.Valid || ai != 0 || !pending.Valid {
		t.Errorf("club must be ownerless, ai 0 and pending reset: owner %v ai %d pending %v", ownerId, ai, pending)
	}
	if got := accInt(t, db, `SELECT owner_id FROM groups WHERE group_id = $1`, withMembers); got != early {
		t.Errorf("group handed to %d, want the member who joined first (%d)", got, early)
	}
	if accInt(t, db, `SELECT count(*) FROM groups WHERE group_id = $1`, alone) != 0 {
		t.Errorf("a group nobody else is in should be deleted")
	}
	if accInt(t, db, `SELECT owner_id FROM groups WHERE group_id = $1`, joined) != late {
		t.Errorf("a group owned by someone else is untouched")
	}

	// Signing in again with the same email creates a brand-new account.
	var again int64
	accMust(t, db.QueryRow(RegisterPendingUserStmtSql, "Real Person", "google", email).Scan(&again), "re-registering")
	if again == owner {
		t.Errorf("a returning person must get a new user id")
	}

	if _, err := DeleteAccount(dbCtx, owner, ReleaseReasonSelfDeletion); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("deleting a deleted user = %v, want ErrUserNotFound", err)
	}
	if _, err := ExportUserData(dbCtx, owner); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("exporting a deleted user = %v, want ErrUserNotFound", err)
	}
}

func TestAssignTeam_RefusesDeletedUser(t *testing.T) {
	dbCtx := accDb(t)
	league := accSeedLeague(t, dbCtx.Db)
	accSeedTeam(t, dbCtx, league, nil, 1)
	owner := accSeedUser(t, dbCtx.Db)
	_, err := DeleteAccount(dbCtx, owner, ReleaseReasonSelfDeletion)
	accMust(t, err, "DeleteAccount")
	if _, err := AssignTeam(dbCtx, owner, accTestCountry); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("a stale session claiming a club = %v, want ErrUserNotFound", err)
	}
	if accInt(t, dbCtx.Db, `SELECT count(*) FROM teams WHERE owner_id = $1`, owner) != 0 {
		t.Fatalf("no club may be owned by a deleted user")
	}
}

func TestAssignTeam_ClearsReleasedNotice(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	owner := accSeedUser(t, db)
	team := accSeedTeam(t, dbCtx, league, &owner, 0)
	accRelease(t, dbCtx, owner, ReleaseReasonInactivity)
	details, err := GetUserDetails(owner, dbCtx)
	accMust(t, err, "GetUserDetails")
	if details.ReleasedTeamName == "" || !details.PendingCountrySelection {
		t.Fatalf("a released owner must see the notice and go through onboarding: %+v", details)
	}
	if _, _, err := ResetReleasedTeam(dbCtx, team); err != nil {
		t.Fatalf("reset: %s", err)
	}
	accSeedTeam(t, dbCtx, league, nil, 1) // something to claim
	if _, err := AssignTeam(dbCtx, owner, accTestCountry); err != nil {
		t.Fatalf("AssignTeam: %s", err)
	}
	details, err = GetUserDetails(owner, dbCtx)
	accMust(t, err, "GetUserDetails")
	if details.ReleasedTeamName != "" {
		t.Errorf("the notice must be consumed once a new club is claimed, got %q", details.ReleasedTeamName)
	}
}

func TestAcceptTerms_StoresVersionAndGatesOnboarding(t *testing.T) {
	dbCtx := accDb(t)
	owner := accSeedUser(t, dbCtx.Db)
	details, err := GetUserDetails(owner, dbCtx)
	accMust(t, err, "GetUserDetails")
	if details.TermsAccepted || details.TermsVersion != CurrentTermsVersion {
		t.Fatalf("a new user has not accepted: %+v", details)
	}
	accMust(t, AcceptTerms(dbCtx, owner, CurrentTermsVersion, true), "AcceptTerms")
	details, err = GetUserDetails(owner, dbCtx)
	accMust(t, err, "GetUserDetails")
	if !details.TermsAccepted {
		t.Fatalf("terms should now be accepted: %+v", details)
	}
	// A policy version bump asks everyone again.
	accMust(t, func() error {
		_, err := dbCtx.Db.Exec(`UPDATE users SET terms_version = $2 WHERE user_id = $1`, owner, CurrentTermsVersion-1)
		return err
	}(), "ageing the acceptance")
	if ok, err := HasAcceptedCurrentTerms(dbCtx, owner); err != nil || ok {
		t.Errorf("an older accepted version must not count (ok=%v err=%v)", ok, err)
	}
}

func TestInactivityClaims_AreSingleShotAndRecheckIdleness(t *testing.T) {
	dbCtx := accDb(t)
	db := dbCtx.Db
	league := accSeedLeague(t, db)
	owner := accSeedUser(t, db)
	team := accSeedTeam(t, dbCtx, league, &owner, 0)
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE users SET last_seen_at = now() - interval '50 days' WHERE user_id = $1`, owner)
		return err
	}(), "ageing user")
	cutoff := time.Now().AddDate(0, 0, -46)

	first, err := ClaimWarningStage(dbCtx, owner, WarningStageClubFirst, cutoff)
	accMust(t, err, "claim")
	second, err := ClaimWarningStage(dbCtx, owner, WarningStageClubFirst, cutoff)
	accMust(t, err, "claim again")
	if !first || second {
		t.Errorf("a stage is claimed exactly once (first=%v second=%v)", first, second)
	}

	// The user comes back after being listed as a release candidate: nothing happens.
	_, err = TouchUserActivity(dbCtx, owner)
	accMust(t, err, "TouchUserActivity")
	if stage := accInt(t, db, `SELECT inactivity_warning_stage FROM users WHERE user_id = $1`, owner); stage != 0 {
		t.Errorf("activity must clear the warning stage, got %d", stage)
	}
	if released, err := ReleaseInactiveOwner(dbCtx, owner, time.Now().AddDate(0, 0, -InactivityReleaseDays)); err != nil || released != 0 {
		t.Errorf("an active owner must not be released (team %d err %v)", released, err)
	}
	if accInt(t, db, `SELECT count(*) FROM teams WHERE team_id = $1 AND owner_id = $2`, team, owner) != 1 {
		t.Errorf("the club must still belong to the returning owner")
	}

	// Idle past the limit: released, and a second call finds nothing to release.
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE users SET last_seen_at = now() - interval '61 days' WHERE user_id = $1`, owner)
		return err
	}(), "ageing user further")
	cutoff = time.Now().AddDate(0, 0, -InactivityReleaseDays)
	if released, err := ReleaseInactiveOwner(dbCtx, owner, cutoff); err != nil || released != team {
		t.Fatalf("an idle owner's club is released (team %d err %v)", released, err)
	}
	if released, _ := ReleaseInactiveOwner(dbCtx, owner, cutoff); released != 0 {
		t.Errorf("an already released owner has no club to release")
	}
	// Idle accounts are erased only if still idle under the lock.
	if _, err := DeleteAccountIfIdle(dbCtx, owner, ReleaseReasonErasure, time.Now().AddDate(0, 0, -AccountErasureDays)); !errors.Is(err, ErrUserActive) {
		t.Errorf("an account idle 61 days is not erasable yet, got %v", err)
	}
}

func accString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var s string
	accMust(t, db.QueryRow(query, args...).Scan(&s), "query "+query)
	return s
}

func accBool(t *testing.T, db *sql.DB, query string, args ...any) bool {
	t.Helper()
	var b bool
	accMust(t, db.QueryRow(query, args...).Scan(&b), "query "+query)
	return b
}

// pqArray renders ids as a Postgres array literal for = ANY($1).
func pqArray(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// A user who signed in but never picked a country (or whose account has no club) has a NULL
// country; the commissioner's list must still load.
func TestGetAllUsersWithTeams_IncludesUsersWithoutCountryAndShowsLastSeen(t *testing.T) {
	dbCtx := accDb(t)
	var id int64
	accMust(t, dbCtx.Db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('No Country', 'dev', $1) RETURNING user_id`,
		"nocountry-"+accUnique()+"@example.com").Scan(&id), "seeding user without country")
	users, err := GetAllUsersWithTeams(dbCtx)
	accMust(t, err, "GetAllUsersWithTeams")
	for _, u := range users {
		if u.UserId == id {
			if u.Country != "" || u.LastSeenAt.IsZero() {
				t.Errorf("unexpected summary: %+v", u)
			}
			return
		}
	}
	t.Fatalf("user %d missing from the admin list", id)
}
