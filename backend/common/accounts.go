package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Account lifecycle (OpenSpec user-account-lifecycle): activity tracking, releasing the club of an
// inactive owner, resetting a released club into a fresh CPU club, account deletion, data export
// and terms acceptance. The engine drives the timers (engine.RunAccountLifecycle); the API
// drives self-service and admin deletion.

// InactivityWarningDays are the days of inactivity at which an owner is warned that their club
// will be released (stage 1 and 2 of users.inactivity_warning_stage).
var InactivityWarningDays = []int{46, 53}

const (
	// InactivityReleaseDays is how long an owner may be inactive before losing their club.
	InactivityReleaseDays = 60
	// AccountErasureWarningDays / AccountErasureDays: an account with no activity at all is
	// warned at 351 days (stage 3) and erased at 365.
	AccountErasureWarningDays = 351
	AccountErasureDays        = 365

	// ActivityWriteInterval throttles users.last_seen_at writes to one per user per replica.
	ActivityWriteInterval = time.Hour

	// ResetMatchGuardBefore / ResetMatchGuardAfter: a released club is not reset while one of its
	// matches is in simulation or inside [now - Before, now + After] - the reset swaps the whole
	// squad, which must not happen under a match being played or broadcast.
	ResetMatchGuardBefore = LiveBroadcastWindow
	ResetMatchGuardAfter  = 30 * time.Minute

	// DefaultTeamBudget matches the teams.budget column default (init.sql); a reset club's budget
	// is written back to it through the ledger.
	DefaultTeamBudget int64 = 400000

	// CurrentTermsVersion is the version of the Privacy Policy and Terms the frontend shows.
	// Bumping it asks every user to accept again. The texts are versioned with the code.
	CurrentTermsVersion = 1

	// MinimumAge is the age a user confirms to be at least when accepting the terms.
	MinimumAge = 16
)

// Reasons a club is released, used as the `reason` attribute of striker.teams.reset.
const (
	ReleaseReasonInactivity    = "inactivity"
	ReleaseReasonSelfDeletion  = "self_deletion"
	ReleaseReasonAdminDeletion = "admin_deletion"
	ReleaseReasonErasure       = "erasure"
)

var ErrUserNotFound = errors.New("user not found")
var ErrTermsNotAccepted = errors.New("terms not accepted")
var ErrInvalidTermsVersion = errors.New("terms version is not current")
var ErrAgeNotConfirmed = errors.New("age not confirmed")

// --- activity ------------------------------------------------------------------------------

const TouchUserActivityStmtSql = "UPDATE users SET last_seen_at = now(), inactivity_warning_stage = 0 WHERE user_id = $1"

// TouchUserActivity records that userId was just active and clears any inactivity warning.
// Reports false when the user no longer exists (a session outliving its account).
func TouchUserActivity(dbCtx *DatabaseContext, userId int64) (bool, error) {
	res, err := dbCtx.Db.Exec(TouchUserActivityStmtSql, userId)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ActivityTracker throttles TouchUserActivity per replica: the first request of a user, and then
// at most one per ActivityWriteInterval, hits the database. It also remembers users known to be
// deleted so their still-valid tokens are refused without a query.
type ActivityTracker struct {
	lastWrite sync.Map // userId -> time.Time
	deleted   sync.Map // userId -> struct{}
	interval  time.Duration
	now       func() time.Time
}

// NewActivityTracker returns a tracker writing at most once per interval per user.
func NewActivityTracker(interval time.Duration) *ActivityTracker {
	return &ActivityTracker{interval: interval, now: time.Now}
}

// Seen is called on every authenticated request. It returns false when the user no longer exists
// (the caller answers 401). touch performs the database write and reports whether the user exists.
func (t *ActivityTracker) Seen(userId int64, touch func(int64) (bool, error)) (bool, error) {
	if _, gone := t.deleted.Load(userId); gone {
		return false, nil
	}
	now := t.now()
	if last, ok := t.lastWrite.Load(userId); ok && now.Sub(last.(time.Time)) < t.interval {
		return true, nil
	}
	exists, err := touch(userId)
	if err != nil {
		// Fail open: an outage must not log everybody out. The next request tries again.
		return true, err
	}
	if !exists {
		t.MarkDeleted(userId)
		return false, nil
	}
	t.lastWrite.Store(userId, now)
	return true, nil
}

// MarkDeleted makes Seen refuse userId from now on (this replica only).
func (t *ActivityTracker) MarkDeleted(userId int64) {
	t.deleted.Store(userId, struct{}{})
	t.lastWrite.Delete(userId)
}

// --- releasing a club ----------------------------------------------------------------------

// ReleaseTeamStmtSql takes the club away from its owner without touching anything else:
// ai_level 0 keeps every CPU routine away and reset_pending_since queues the heavy reset (D3).
const ReleaseTeamStmtSql = "UPDATE teams SET owner_id = NULL, ai_level = 0, reset_pending_since = now(), reset_reason = $2 " +
	"WHERE team_id = $1"

// ReleaseTeam releases one club inside the caller's transaction. Cheap by design: the squad swap
// happens later in ResetReleasedTeam.
func ReleaseTeam(tx *sql.Tx, teamId int64, reason string) error {
	_, err := tx.Exec(ReleaseTeamStmtSql, teamId, reason)
	return err
}

const GetOwnedTeamForUpdateStmtSql = "SELECT team_id, name FROM teams WHERE owner_id = $1 FOR UPDATE"
const SetReleasedTeamNameStmtSql = "UPDATE users SET released_team_name = $2 WHERE user_id = $1"

// ReleaseUserTeam releases the club userId owns, if any, and (when rememberName) stores its name
// in users.released_team_name so the user is told on their next sign-in. Returns the team id (0
// when the user had no club).
func ReleaseUserTeam(tx *sql.Tx, userId int64, reason string, rememberName bool) (int64, error) {
	var teamId int64
	var name string
	err := tx.QueryRow(GetOwnedTeamForUpdateStmtSql, userId).Scan(&teamId, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if err := ReleaseTeam(tx, teamId, reason); err != nil {
		return 0, err
	}
	if rememberName {
		if _, err := tx.Exec(SetReleasedTeamNameStmtSql, userId, name); err != nil {
			return 0, err
		}
	}
	return teamId, nil
}

// --- resetting a released club -------------------------------------------------------------

const (
	GetPendingResetTeamsStmtSql = "SELECT team_id FROM teams WHERE reset_pending_since IS NOT NULL ORDER BY reset_pending_since"

	// Locks the club only if it is still waiting for its reset and nobody owns it: no row means
	// another replica already reset it (or someone claimed it).
	LockPendingResetTeamStmtSql = "SELECT t.country, l.tier, COALESCE(t.reset_reason, ''), t.budget " +
		"FROM teams t JOIN leagues l ON l.league_id = t.league_id " +
		"WHERE t.team_id = $1 AND t.reset_pending_since IS NOT NULL AND t.owner_id IS NULL FOR UPDATE OF t"

	// Any league match, cup tie or friendly of the club in simulation or inside the guard window.
	TeamHasMatchInGuardWindowStmtSql = "SELECT EXISTS (" +
		"SELECT 1 FROM matches WHERE (home_team_id = $1 OR away_team_id = $1) " +
		"AND (in_simulation OR match_date BETWEEN now() - make_interval(secs => $2) AND now() + make_interval(secs => $3)) " +
		"UNION ALL SELECT 1 FROM cup_ties WHERE (home_team_id = $1 OR away_team_id = $1) " +
		"AND (in_simulation OR match_date BETWEEN now() - make_interval(secs => $2) AND now() + make_interval(secs => $3)) " +
		"UNION ALL SELECT 1 FROM friendly_matches WHERE (home_team_id = $1 OR away_team_id = $1) " +
		"AND (in_simulation OR match_date BETWEEN now() - make_interval(secs => $2) AND now() + make_interval(secs => $3)))"

	GetLoansToTeamForResetStmtSql      = "SELECT loan_id, player_id, from_team_id, to_team_id, release_on_return FROM player_loans WHERE to_team_id = $1"
	MarkLoansOutReleaseOnReturnStmtSql = "UPDATE player_loans SET release_on_return = true WHERE from_team_id = $1"
	DeleteAcademyPlayersStmtSql        = "DELETE FROM players WHERE team_id = $1 AND in_academy = true"

	// Pending market and social state of the club, cleared before its squad is replaced.
	DeleteTeamTransferListingsStmtSql   = "DELETE FROM transfer_listings WHERE selling_team_id = $1"
	DeleteTeamLoanListingsStmtSql       = "DELETE FROM loan_listings WHERE lending_team_id = $1"
	DeleteTeamTransferOffersStmtSql     = "DELETE FROM transfer_offers WHERE from_team_id = $1 OR to_team_id = $1"
	DeleteTeamFriendlyChallengesStmtSql = "DELETE FROM friendly_challenges WHERE challenger_team_id = $1 OR opponent_team_id = $1"
	ReleaseTeamTrainerStmtSql           = "UPDATE trainers SET team_id = NULL WHERE team_id = $1"

	// Lineups: the club's standing lineups go, and so do its overrides for matches not played
	// yet. Played-match lineups stay - match pages read them.
	DeleteTeamDefaultLineupStmtSql             = "DELETE FROM default_lineup_positions WHERE team_id = $1"
	DeleteTeamDefaultSubstitutionsStmtSql      = "DELETE FROM default_lineup_substitutions WHERE team_id = $1"
	DeleteTeamDefaultFriendlyLineupStmtSql     = "DELETE FROM default_friendly_lineup_positions WHERE team_id = $1"
	DeleteUnplayedLeagueLineupPositionsStmtSql = "DELETE FROM lineup_positions WHERE lineup_id IN " +
		"(SELECT l.lineup_id FROM lineup l JOIN matches m ON m.match_id = l.match_id WHERE l.team_id = $1 AND NOT m.played)"
	DeleteUnplayedLeagueLineupsStmtSql           = "DELETE FROM lineup WHERE team_id = $1 AND match_id IN (SELECT match_id FROM matches WHERE NOT played)"
	DeleteUnplayedLeagueSubstitutionsStmtSql     = "DELETE FROM lineup_substitutions WHERE team_id = $1 AND match_id IN (SELECT match_id FROM matches WHERE NOT played)"
	DeleteUnplayedFriendlyLineupPositionsStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id IN " +
		"(SELECT l.lineup_id FROM friendly_lineup l JOIN friendly_matches m ON m.match_id = l.match_id WHERE l.team_id = $1 AND NOT m.played)"
	DeleteUnplayedFriendlyLineupsStmtSql    = "DELETE FROM friendly_lineup WHERE team_id = $1 AND match_id IN (SELECT match_id FROM friendly_matches WHERE NOT played)"
	DeleteUnplayedCupLineupPositionsStmtSql = "DELETE FROM cup_tie_lineup_positions WHERE lineup_id IN " +
		"(SELECT l.lineup_id FROM cup_tie_lineup l JOIN cup_ties t ON t.tie_id = l.tie_id WHERE l.team_id = $1 AND NOT t.played)"
	DeleteUnplayedCupLineupsStmtSql       = "DELETE FROM cup_tie_lineup WHERE team_id = $1 AND tie_id IN (SELECT tie_id FROM cup_ties WHERE NOT played)"
	DeleteUnplayedCupSubstitutionsStmtSql = "DELETE FROM cup_tie_lineup_substitutions WHERE team_id = $1 AND tie_id IN (SELECT tie_id FROM cup_ties WHERE NOT played)"

	// The senior squad becomes free agents. player_club_history already has their stint.
	ReleaseSeniorSquadStmtSql = "UPDATE players SET last_team_id = team_id, team_id = NULL, contract_years_remaining = 0, " +
		"shirt_number = NULL, free_agent_seasons = 0 WHERE team_id = $1 AND in_academy = false"

	SetTeamBudgetStmtSql = "UPDATE teams SET budget = $2 WHERE team_id = $1"

	// ResetTeamSettingsStmtSql puts everything a previous owner could have changed back to the
	// column default (DEFAULT keeps this in sync with init.sql) and takes the club off the
	// pending-reset queue.
	ResetTeamSettingsStmtSql = "UPDATE teams SET name = $1, stadium_name = $2, name_changed = false, " +
		"stadium_capacity = DEFAULT, ticket_price = DEFAULT, formation = DEFAULT, friendly_formation = DEFAULT, " +
		"sponsor_deal = DEFAULT, sponsor_deal_season = DEFAULT, fanbase = DEFAULT, socios = DEFAULT, " +
		"stadium_upgrade_target = NULL, stadium_upgrade_completes_at = NULL, last_academy_promotion_at = NULL, " +
		"ai_level = $3, reset_pending_since = NULL, reset_reason = NULL WHERE team_id = $4"

	// Frees a loaned player of a reset club once the loan ends (release_on_return).
	ReleaseLoanedPlayerToFreeAgencyStmtSql = "UPDATE players SET last_team_id = $2, team_id = NULL, contract_years_remaining = 0, " +
		"shirt_number = NULL, free_agent_seasons = 0 WHERE player_id = $1"
)

// GetPendingResetTeams lists the clubs waiting for ResetReleasedTeam, oldest first.
func GetPendingResetTeams(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetPendingResetTeamsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// returnLoanInTx ends a loan inside tx: the player goes back to his parent club, or - when the
// parent club was reset while he was out (release_on_return) - becomes a free agent. His slot in
// the borrower's default lineup goes either way, and the loan record is deleted.
func returnLoanInTx(tx *sql.Tx, loan PlayerLoan) error {
	if loan.ReleaseOnReturn {
		if _, err := tx.Exec(ReleaseLoanedPlayerToFreeAgencyStmtSql, loan.PlayerId, loan.FromTeamId); err != nil {
			return err
		}
	} else if err := movePlayerToTeam(tx, loan.FromTeamId, loan.PlayerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, loan.PlayerId); err != nil {
		return err
	}
	_, err := tx.Exec(DeletePlayerLoanStmtSql, loan.LoanId)
	return err
}

// ResetReleasedTeam turns a released club into a fresh CPU club, in one transaction: loans are
// settled, offers/listings/challenges/trainer/lineups cleared, the senior squad freed, a new squad
// and identity generated, the budget written back through the ledger and an AI level drawn by
// tier. History (results, trophies, ledger, career histories) is untouched. Returns false when
// there was nothing to do (another replica got there first, or the club was claimed) or the club
// has a match inside the guard window (retried on the next pass). The loans of players sent home
// are returned so the caller can notify the owners involved after the commit.
func ResetReleasedTeam(dbCtx *DatabaseContext, teamId int64) (bool, []PlayerLoan, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return false, nil, err
	}
	defer tx.Rollback()

	var country, reason string
	var tier int
	var budget int64
	err = tx.QueryRow(LockPendingResetTeamStmtSql, teamId).Scan(&country, &tier, &reason, &budget)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	var busy bool
	if err := tx.QueryRow(TeamHasMatchInGuardWindowStmtSql, teamId,
		ResetMatchGuardBefore.Seconds(), ResetMatchGuardAfter.Seconds()).Scan(&busy); err != nil {
		return false, nil, err
	}
	if busy {
		return false, nil, nil
	}

	// Players on loan TO this club go home (or to free agency if their own club was reset too).
	rows, err := tx.Query(GetLoansToTeamForResetStmtSql, teamId)
	if err != nil {
		return false, nil, err
	}
	returned := make([]PlayerLoan, 0)
	for rows.Next() {
		var loan PlayerLoan
		if err := rows.Scan(&loan.LoanId, &loan.PlayerId, &loan.FromTeamId, &loan.ToTeamId, &loan.ReleaseOnReturn); err != nil {
			rows.Close()
			return false, nil, err
		}
		returned = append(returned, loan)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, nil, err
	}
	for _, loan := range returned {
		if err := returnLoanInTx(tx, loan); err != nil {
			return false, nil, err
		}
	}
	// Players this club has out on loan are released when their loan ends.
	if _, err := tx.Exec(MarkLoansOutReleaseOnReturnStmtSql, teamId); err != nil {
		return false, nil, err
	}

	for _, stmt := range []string{
		DeleteAcademyPlayersStmtSql,
		DeleteTeamTransferListingsStmtSql, DeleteTeamLoanListingsStmtSql, DeleteTeamTransferOffersStmtSql,
		DeleteTeamFriendlyChallengesStmtSql, ReleaseTeamTrainerStmtSql,
		DeleteTeamDefaultLineupStmtSql, DeleteTeamDefaultSubstitutionsStmtSql, DeleteTeamDefaultFriendlyLineupStmtSql,
		DeleteUnplayedLeagueLineupPositionsStmtSql, DeleteUnplayedLeagueLineupsStmtSql, DeleteUnplayedLeagueSubstitutionsStmtSql,
		DeleteUnplayedFriendlyLineupPositionsStmtSql, DeleteUnplayedFriendlyLineupsStmtSql,
		DeleteUnplayedCupLineupPositionsStmtSql, DeleteUnplayedCupLineupsStmtSql, DeleteUnplayedCupSubstitutionsStmtSql,
		ReleaseSeniorSquadStmtSql,
	} {
		if _, err := tx.Exec(stmt, teamId); err != nil {
			return false, nil, fmt.Errorf("resetting team %d: %w (%.60s)", teamId, err, stmt)
		}
	}

	insertStmt, err := tx.Prepare(InsertNewPlayerStmtSql)
	if err != nil {
		return false, nil, err
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, tx, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, country, false)

	var flows LedgerFlows
	if delta := DefaultTeamBudget - budget; delta != 0 {
		if _, err := tx.Exec(SetTeamBudgetStmtSql, teamId, DefaultTeamBudget); err != nil {
			return false, nil, err
		}
		if err := insertLedgerRow(tx, &flows, teamId, delta, TRANSACTION_TEAM_RESET, "Club reset after owner left", nil); err != nil {
			return false, nil, err
		}
	}

	if _, err := tx.Exec(ResetTeamSettingsStmtSql, RandomTeamName(country), RandomStadiumName(country),
		RandomAiLevel(tier), teamId); err != nil {
		return false, nil, err
	}
	if err := tx.Commit(); err != nil {
		return false, nil, err
	}
	flows.Commit(context.Background())

	// Best effort from here: the club is already reset and consistent.
	if players, err := GetPlayersFromTeamId(dbCtx, teamId); err != nil {
		log.Errorf("error fetching new squad for reset team %d to seed its lineup: %s", teamId, err)
	} else if err := SetDefaultLineup(dbCtx, teamId, FlattenLineup(SelectBestLineup(*players))); err != nil {
		log.Errorf("error seeding default lineup for reset team %d: %s", teamId, err)
	}
	if reason == "" {
		reason = ReleaseReasonInactivity
	}
	TeamsResetCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("reason", reason)))
	return true, returned, nil
}

// --- inactivity timers ---------------------------------------------------------------------

const (
	// Owners idle since cutoff whose warning stage is below `stage` (still to be warned).
	GetOwnersDueWarningStmtSql = "SELECT u.user_id, t.name, u.last_seen_at FROM users u JOIN teams t ON t.owner_id = u.user_id " +
		"WHERE u.last_seen_at <= $1 AND u.inactivity_warning_stage < $2 ORDER BY u.user_id"
	GetAccountsDueErasureWarningStmtSql = "SELECT user_id, last_seen_at FROM users WHERE last_seen_at <= $1 AND inactivity_warning_stage < $2 ORDER BY user_id"
	// Single-statement claim: only the caller that moves the stage forward gets a row back, so a
	// re-run or a second replica never sends the same warning twice.
	ClaimWarningStageStmtSql     = "UPDATE users SET inactivity_warning_stage = $2 WHERE user_id = $1 AND inactivity_warning_stage < $2 AND last_seen_at <= $3 RETURNING user_id"
	GetOwnersDueReleaseStmtSql   = "SELECT u.user_id FROM users u JOIN teams t ON t.owner_id = u.user_id WHERE u.last_seen_at <= $1 ORDER BY u.user_id"
	GetAccountsDueErasureStmtSql = "SELECT user_id FROM users WHERE last_seen_at <= $1 ORDER BY user_id"
	WarningStageClubFirst        = 1
	WarningStageClubSecond       = 2
	WarningStageErasure          = 3
)

// InactiveOwner is an owner due a warning, with the club that would be released.
type InactiveOwner struct {
	UserId     int64
	TeamName   string
	LastSeenAt time.Time
}

// GetOwnersDueWarning lists owners idle since cutoff who have not yet reached warning stage `stage`.
func GetOwnersDueWarning(dbCtx *DatabaseContext, cutoff time.Time, stage int) ([]InactiveOwner, error) {
	rows, err := dbCtx.Db.Query(GetOwnersDueWarningStmtSql, cutoff, stage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InactiveOwner, 0)
	for rows.Next() {
		var o InactiveOwner
		if err := rows.Scan(&o.UserId, &o.TeamName, &o.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetAccountsDueErasureWarning lists accounts idle since cutoff not yet warned about erasure.
func GetAccountsDueErasureWarning(dbCtx *DatabaseContext, cutoff time.Time) ([]InactiveOwner, error) {
	rows, err := dbCtx.Db.Query(GetAccountsDueErasureWarningStmtSql, cutoff, WarningStageErasure)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InactiveOwner, 0)
	for rows.Next() {
		var o InactiveOwner
		if err := rows.Scan(&o.UserId, &o.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ClaimWarningStage moves userId's warning stage forward to `stage` if they are still idle since
// cutoff and not already at or past it. True means this caller owns sending that warning.
func ClaimWarningStage(dbCtx *DatabaseContext, userId int64, stage int, cutoff time.Time) (bool, error) {
	var id int64
	err := dbCtx.Db.QueryRow(ClaimWarningStageStmtSql, userId, stage, cutoff).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// getUserIds runs a query returning a single user_id column.
func getUserIds(dbCtx *DatabaseContext, query string, args ...any) ([]int64, error) {
	rows, err := dbCtx.Db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetOwnersDueRelease lists owners idle since cutoff (they will lose their club).
func GetOwnersDueRelease(dbCtx *DatabaseContext, cutoff time.Time) ([]int64, error) {
	return getUserIds(dbCtx, GetOwnersDueReleaseStmtSql, cutoff)
}

// GetAccountsDueErasure lists accounts idle since cutoff (they will be erased).
func GetAccountsDueErasure(dbCtx *DatabaseContext, cutoff time.Time) ([]int64, error) {
	return getUserIds(dbCtx, GetAccountsDueErasureStmtSql, cutoff)
}

// ReleaseInactiveOwner releases userId's club for inactivity, remembering its name for the
// "your club was released" notice - but only if, under the user's row lock, they are still idle
// since cutoff (they may have come back since the candidates were listed). Returns the released
// team id (0 if nothing happened).
func ReleaseInactiveOwner(dbCtx *DatabaseContext, userId int64, cutoff time.Time) (int64, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var lastSeen time.Time
	if err := tx.QueryRow(LockUserLastSeenStmtSql, userId).Scan(&lastSeen); errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	if lastSeen.After(cutoff) {
		return 0, nil
	}
	teamId, err := ReleaseUserTeam(tx, userId, ReleaseReasonInactivity, true)
	if err != nil {
		return 0, err
	}
	return teamId, tx.Commit()
}
