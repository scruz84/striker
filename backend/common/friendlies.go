package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

var ErrCannotChallengeOwnTeam = errors.New("cannot challenge your own team to a friendly")
var ErrFriendlyChallengeAlreadyExists = errors.New("you already have a pending friendly challenge against this team")
var ErrFriendlyLimitReached = errors.New("weekly friendly match limit reached")
var ErrFriendlyChallengeNotFound = errors.New("friendly challenge not found")
var ErrNotChallengeSender = errors.New("you did not send this friendly challenge")
var ErrNotChallengeRecipient = errors.New("this friendly challenge is not for your team")
var ErrInvalidFriendlyLimit = errors.New("max friendlies per week must be zero or greater")
var ErrInvalidFriendlySchedule = errors.New("friendly schedule must select at least one weekday")
var ErrInvalidFriendlyKickoff = errors.New("kickoff must be far enough in the future and fall on a day the friendly schedule allows")
var ErrNotFriendlyParticipant = errors.New("your team is not part of this friendly")
var ErrFriendlyAlreadyPlayed = errors.New("this friendly has already been played")

// FriendlyChallenge is a pending invite from one team to another - see ChallengeToFriendly.
type FriendlyChallenge struct {
	ChallengeId        int64     `json:"challengeId"`
	ChallengerTeamId   int64     `json:"challengerTeamId"`
	ChallengerTeamName string    `json:"challengerTeamName"`
	OpponentTeamId     int64     `json:"opponentTeamId"`
	OpponentTeamName   string    `json:"opponentTeamName"`
	CreatedAt          time.Time `json:"createdAt"`
}

// FriendlyMatch is an accepted friendly, scheduled or already played - see
// AcceptFriendlyChallenge and engine.SimulateFriendlies.
type FriendlyMatch struct {
	MatchId      int64     `json:"matchId"`
	HomeTeamId   int64     `json:"homeTeamId"`
	HomeTeamName string    `json:"homeTeamName"`
	AwayTeamId   int64     `json:"awayTeamId"`
	AwayTeamName string    `json:"awayTeamName"`
	MatchDate    time.Time `json:"matchDate"`
	Played       bool      `json:"played"`
	HomeGoals    int       `json:"homeGoals"`
	AwayGoals    int       `json:"awayGoals"`
}

// StartOfWeek returns midnight UTC on the most recent Monday on or before t - the same
// real-calendar-week boundary a friendly-match weekly limit resets on as PayWeeklyWages' own
// Monday-midnight-UTC cadence (see engine.StartWeeklyWagesScheduler), so "once a week" means the
// same thing everywhere in this game's economy.
func StartOfWeek(t time.Time) time.Time {
	t = t.UTC()
	daysSinceMonday := (int(t.Weekday()) + 6) % 7 // Monday=0, Tuesday=1, ..., Sunday=6
	monday := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

const GetFriendlyScheduleWeekdaysStmtSql = "SELECT friendly_schedule_weekdays FROM season"
const SetFriendlyScheduleWeekdaysStmtSql = "UPDATE season SET friendly_schedule_weekdays = $1"

// GetFriendlyScheduleWeekdays returns the admin-configured bitmask of weekdays a friendly may
// kick off on (bit i = time.Weekday(i), same convention as MatchSchedule.WeekdaysMask), falling
// back to every day on a lookup error - same "never block on a settings-row hiccup" convention
// as GetMatchSchedule.
func GetFriendlyScheduleWeekdays(dbCtx *DatabaseContext) int {
	var mask int
	if err := dbCtx.Db.QueryRow(GetFriendlyScheduleWeekdaysStmtSql).Scan(&mask); err != nil {
		return AllWeekdaysMask
	}
	return mask
}

// SetFriendlyScheduleWeekdays updates which weekdays a friendly may be scheduled on - e.g.
// weekends only. It only affects challenges accepted after this call; already-scheduled
// friendlies keep their date.
func SetFriendlyScheduleWeekdays(dbCtx *DatabaseContext, weekdaysMask int) error {
	if weekdaysMask&AllWeekdaysMask == 0 {
		return ErrInvalidFriendlySchedule
	}
	_, err := dbCtx.Db.Exec(SetFriendlyScheduleWeekdaysStmtSql, weekdaysMask)
	return err
}

// nextAllowedFriendlyDate walks forward from candidate, one day at a time, to the first day
// whose weekday bit is set in weekdaysMask - preserving candidate's own time-of-day, since a
// friendly should still kick off as soon as the schedule allows rather than at some fixed hour.
// If candidate's own weekday is already allowed, candidate is returned unchanged. Bounded to 8
// iterations (one full week plus one) so it always terminates given a validated, non-zero mask.
func nextAllowedFriendlyDate(candidate time.Time, weekdaysMask int) time.Time {
	for offset := 0; offset < 8; offset++ {
		day := candidate.AddDate(0, 0, offset)
		bit := 1 << int(day.Weekday())
		if weekdaysMask&bit != 0 {
			return day
		}
	}
	return candidate // unreachable given a validated (non-zero) mask
}

// isAllowedFriendlyKickoff reports whether kickoff clears the simulation scheduler's polling
// window (see friendlyScheduleDelay) and falls on a weekday GetFriendlyScheduleWeekdays allows -
// the same two constraints the auto-accept path (an unowned opponent) already satisfies by
// construction via nextAllowedFriendlyDate, now enforced on a human-chosen kickoff too.
func isAllowedFriendlyKickoff(dbCtx *DatabaseContext, kickoff time.Time) bool {
	if kickoff.Before(time.Now().Add(friendlyScheduleDelay)) {
		return false
	}
	bit := 1 << int(kickoff.Weekday())
	return GetFriendlyScheduleWeekdays(dbCtx)&bit != 0
}

const GetMaxFriendliesPerWeekStmtSql = "SELECT max_friendlies_per_week FROM season"
const SetMaxFriendliesPerWeekStmtSql = "UPDATE season SET max_friendlies_per_week = $1"

// GetMaxFriendliesPerWeek returns the admin-configured cap on accepted friendlies a team may be
// part of per real calendar week (see StartOfWeek) - see handlers.AdminSetMaxFriendliesPerWeekHandler.
func GetMaxFriendliesPerWeek(dbCtx *DatabaseContext) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow(GetMaxFriendliesPerWeekStmtSql).Scan(&n)
	return n, err
}

// SetMaxFriendliesPerWeek updates the cap. Zero is allowed (turns friendlies off entirely -
// every challenge/accept call will report the limit already reached).
func SetMaxFriendliesPerWeek(dbCtx *DatabaseContext, n int) error {
	if n < 0 {
		return ErrInvalidFriendlyLimit
	}
	_, err := dbCtx.Db.Exec(SetMaxFriendliesPerWeekStmtSql, n)
	return err
}

const CountFriendliesThisWeekStmtSql = "SELECT count(*) FROM friendly_matches WHERE (home_team_id = $1 OR away_team_id = $1) AND match_date >= $2"

// CountFriendliesThisWeek counts teamId's accepted friendlies (scheduled or already played)
// since the start of the current real calendar week - a friendly counts against the weekly cap
// the moment it's accepted, not only once it's actually been played.
func CountFriendliesThisWeek(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(CountFriendliesThisWeekStmtSql, teamId, StartOfWeek(time.Now())).Scan(&count)
	return count, err
}

const InsertFriendlyChallengeStmtSql = "INSERT INTO friendly_challenges (challenger_team_id, opponent_team_id) VALUES ($1, $2) RETURNING challenge_id"
const GetFriendlyChallengeStmtSql = "SELECT challenger_team_id, opponent_team_id FROM friendly_challenges WHERE challenge_id = $1"
const GetFriendlyChallengeForUpdateStmtSql = GetFriendlyChallengeStmtSql + " " + ForUpdate
const DeleteFriendlyChallengeStmtSql = "DELETE FROM friendly_challenges WHERE challenge_id = $1"
const InsertFriendlyMatchStmtSql = "INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3)"

// friendlyScheduleDelay is how soon after acceptance a friendly kicks off - short enough to feel
// immediate, long enough to comfortably clear the simulation scheduler's per-minute polling (see
// engine.StartFriendlyScheduler).
const friendlyScheduleDelay = 5 * time.Minute

// ChallengeToFriendly proposes a friendly match to opponentTeamId, on behalf of challengerTeamId.
// If the opponent is unowned (an AI-managed team - see AiLevelCaretaker/Manager/Director, every
// tier of which accepts unconditionally, there being no downside to a free match for any of
// them), the challenge is accepted immediately rather than left pending forever with nobody to
// act on it; otherwise it waits for the opponent's owner to accept or reject it.
func ChallengeToFriendly(dbCtx *DatabaseContext, challengerTeamId int64, opponentTeamId int64) error {
	if challengerTeamId == opponentTeamId {
		return ErrCannotChallengeOwnTeam
	}

	challengerCount, err := CountFriendliesThisWeek(dbCtx, challengerTeamId)
	if err != nil {
		return err
	}
	maxPerWeek, err := GetMaxFriendliesPerWeek(dbCtx)
	if err != nil {
		return err
	}
	if challengerCount >= maxPerWeek {
		return ErrFriendlyLimitReached
	}

	var challengeId int64
	err = dbCtx.Db.QueryRow(InsertFriendlyChallengeStmtSql, challengerTeamId, opponentTeamId).Scan(&challengeId)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrFriendlyChallengeAlreadyExists
		}
		return err
	}

	opponentOwner, err := GetTeamOwner(dbCtx, opponentTeamId)
	if err != nil {
		return err
	}
	if opponentOwner > 0 {
		return nil // waits for a human to accept/reject from their Friendlies page
	}

	// Unowned - accept on the opponent's behalf right away, since nothing else will ever
	// revisit this challenge. If the AI opponent is itself already at its weekly cap, surface
	// that to the challenger now rather than leave an orphaned challenge nothing can resolve.
	// There's nobody to pick a kickoff, so it gets the same "as soon as allowed" default a
	// human accepting could also choose.
	defaultKickoff := nextAllowedFriendlyDate(time.Now().Add(friendlyScheduleDelay), GetFriendlyScheduleWeekdays(dbCtx))
	if err := AcceptFriendlyChallenge(dbCtx, challengeId, opponentTeamId, defaultKickoff); err != nil {
		_, _ = dbCtx.Db.Exec(DeleteFriendlyChallengeStmtSql, challengeId)
		return err
	}
	return nil
}

// AcceptFriendlyChallenge agrees to play a pending challenge: creates the actual fixture (see
// friendly_matches) at the given kickoff and removes the challenge, atomically so a
// double-accept can't create two fixtures from the same challenge. Re-checks the accepting
// team's weekly limit now, since time may have passed (and other friendlies accepted) since the
// challenge was sent - the same "re-validate at accept time" convention AcceptOffer uses for
// budget. kickoff must be far enough ahead to clear the simulation scheduler's polling window
// and fall on a weekday the friendly schedule allows (see isAllowedFriendlyKickoff) - callers
// with nobody to ask (see ChallengeToFriendly's auto-accept path) pass nextAllowedFriendlyDate's
// own output, which always satisfies both.
func AcceptFriendlyChallenge(dbCtx *DatabaseContext, challengeId int64, opponentTeamId int64, kickoff time.Time) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var challengerTeamId, actualOpponentTeamId int64
	err = tx.QueryRow(GetFriendlyChallengeForUpdateStmtSql, challengeId).Scan(&challengerTeamId, &actualOpponentTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFriendlyChallengeNotFound
	}
	if err != nil {
		return err
	}
	if actualOpponentTeamId != opponentTeamId {
		return ErrNotChallengeRecipient
	}

	opponentCount, err := CountFriendliesThisWeek(dbCtx, opponentTeamId)
	if err != nil {
		return err
	}
	maxPerWeek, err := GetMaxFriendliesPerWeek(dbCtx)
	if err != nil {
		return err
	}
	if opponentCount >= maxPerWeek {
		return ErrFriendlyLimitReached
	}
	if !isAllowedFriendlyKickoff(dbCtx, kickoff) {
		return ErrInvalidFriendlyKickoff
	}

	if _, err := tx.Exec(InsertFriendlyMatchStmtSql, challengerTeamId, opponentTeamId, kickoff); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteFriendlyChallengeStmtSql, challengeId); err != nil {
		return err
	}
	return tx.Commit()
}

// RejectFriendlyChallenge lets the challenged team decline, without spending a weekly slot.
func RejectFriendlyChallenge(dbCtx *DatabaseContext, challengeId int64, opponentTeamId int64) error {
	var challengerTeamId, actualOpponentTeamId int64
	err := dbCtx.Db.QueryRow(GetFriendlyChallengeStmtSql, challengeId).Scan(&challengerTeamId, &actualOpponentTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFriendlyChallengeNotFound
	}
	if err != nil {
		return err
	}
	if actualOpponentTeamId != opponentTeamId {
		return ErrNotChallengeRecipient
	}
	_, err = dbCtx.Db.Exec(DeleteFriendlyChallengeStmtSql, challengeId)
	return err
}

// WithdrawFriendlyChallenge lets the challenger cancel their own pending challenge.
func WithdrawFriendlyChallenge(dbCtx *DatabaseContext, challengeId int64, challengerTeamId int64) error {
	var actualChallengerTeamId, opponentTeamId int64
	err := dbCtx.Db.QueryRow(GetFriendlyChallengeStmtSql, challengeId).Scan(&actualChallengerTeamId, &opponentTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFriendlyChallengeNotFound
	}
	if err != nil {
		return err
	}
	if actualChallengerTeamId != challengerTeamId {
		return ErrNotChallengeSender
	}
	_, err = dbCtx.Db.Exec(DeleteFriendlyChallengeStmtSql, challengeId)
	return err
}

const GetFriendlyChallengesReceivedStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.opponent_team_id = $1 ORDER BY fc.created_at DESC"

const GetFriendlyChallengesSentStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.challenger_team_id = $1 ORDER BY fc.created_at DESC"

func scanFriendlyChallenges(rows *sql.Rows) ([]FriendlyChallenge, error) {
	defer rows.Close()
	challenges := make([]FriendlyChallenge, 0)
	for rows.Next() {
		var c FriendlyChallenge
		if err := rows.Scan(&c.ChallengeId, &c.ChallengerTeamId, &c.ChallengerTeamName,
			&c.OpponentTeamId, &c.OpponentTeamName, &c.CreatedAt); err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}
	return challenges, nil
}

// GetFriendlyChallengesReceived lists every pending challenge made to teamId.
func GetFriendlyChallengesReceived(dbCtx *DatabaseContext, teamId int64) ([]FriendlyChallenge, error) {
	rows, err := dbCtx.Db.Query(GetFriendlyChallengesReceivedStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanFriendlyChallenges(rows)
}

// GetFriendlyChallengesSent lists every pending challenge teamId has made.
func GetFriendlyChallengesSent(dbCtx *DatabaseContext, teamId int64) ([]FriendlyChallenge, error) {
	rows, err := dbCtx.Db.Query(GetFriendlyChallengesSentStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanFriendlyChallenges(rows)
}

const GetUpcomingFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE (fm.home_team_id = $1 OR fm.away_team_id = $1) AND fm.played = false ORDER BY fm.match_date ASC"

const GetRecentFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE (fm.home_team_id = $1 OR fm.away_team_id = $1) AND fm.played = true ORDER BY fm.match_date DESC LIMIT $2"

func scanFriendlyMatches(rows *sql.Rows) ([]FriendlyMatch, error) {
	defer rows.Close()
	matches := make([]FriendlyMatch, 0)
	for rows.Next() {
		var m FriendlyMatch
		if err := rows.Scan(&m.MatchId, &m.HomeTeamId, &m.HomeTeamName, &m.AwayTeamId, &m.AwayTeamName,
			&m.MatchDate, &m.Played, &m.HomeGoals, &m.AwayGoals); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// GetUpcomingFriendlies lists teamId's accepted-but-not-yet-played friendlies, soonest first.
func GetUpcomingFriendlies(dbCtx *DatabaseContext, teamId int64) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetUpcomingFriendliesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

// GetRecentFriendlies lists teamId's most recently played friendlies, most recent first.
func GetRecentFriendlies(dbCtx *DatabaseContext, teamId int64, limit int) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetRecentFriendliesStmtSql, teamId, limit)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

const GetFriendlyMatchByIdStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE fm.match_id = $1"

// GetFriendlyMatchById looks up one friendly by id - used by the lineup and live-view handlers,
// which need the match itself (and to check the caller owns one of its teams) rather than a
// per-team list.
func GetFriendlyMatchById(dbCtx *DatabaseContext, matchId int64) (*FriendlyMatch, error) {
	var m FriendlyMatch
	err := dbCtx.Db.QueryRow(GetFriendlyMatchByIdStmtSql, matchId).Scan(&m.MatchId, &m.HomeTeamId, &m.HomeTeamName,
		&m.AwayTeamId, &m.AwayTeamName, &m.MatchDate, &m.Played, &m.HomeGoals, &m.AwayGoals)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

const GetSimulableFriendliesStmtSql = "SELECT match_id, home_team_id, away_team_id, match_date FROM friendly_matches " +
	"WHERE match_date > now() AND match_date < now() + interval '15 minutes' AND played = false AND in_simulation = false " + ForUpdate

// GetSimulableFriendlies claims (marks in_simulation) and returns every friendly whose kickoff
// has arrived, all within one transaction - the same atomic claim-then-return pattern
// GetSimulableMatches uses (see database.go), which matters here for exactly the reason it
// mattered there: two goroutines racing an unguarded claim is a real, previously-shipped bug
// (see SetDefaultLineup's advisory lock fix), not a hypothetical one.
func GetSimulableFriendlies(dbCtx *DatabaseContext) ([]FriendlyMatch, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(GetSimulableFriendliesStmtSql)
	if err != nil {
		return nil, err
	}
	matches := make([]FriendlyMatch, 0)
	for rows.Next() {
		var m FriendlyMatch
		if err := rows.Scan(&m.MatchId, &m.HomeTeamId, &m.AwayTeamId, &m.MatchDate); err != nil {
			rows.Close()
			return nil, err
		}
		matches = append(matches, m)
	}
	rows.Close()

	if len(matches) > 0 {
		inClause := "("
		for i, m := range matches {
			if i > 0 {
				inClause += ", "
			}
			inClause += strconv.FormatInt(m.MatchId, 10)
		}
		inClause += ")"
		if _, err := tx.Exec("UPDATE friendly_matches SET in_simulation = true WHERE match_id IN " + inClause); err != nil {
			return nil, err
		}
	}

	return matches, tx.Commit()
}

const SetFriendlyPlayedStmtSql = "UPDATE friendly_matches SET played = true, in_simulation = false, home_goals = $1, away_goals = $2 WHERE match_id = $3"
const InsertFriendlyEventStmtSql = "INSERT INTO friendly_events (match_id, event_type, team_owner, other_team, minute, " +
	"team_players, team_player_names, home_goals, away_goals) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"

const InsertFriendlyLineupStmtSql = "INSERT INTO friendly_lineup (match_id, team_id, formation) VALUES ($1, $2, $3) " +
	"ON CONFLICT (match_id, team_id) DO UPDATE SET formation = EXCLUDED.formation RETURNING lineup_id"
const DeleteFriendlyLineupPositionsStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id = $1"
const InsertFriendlyLineupPositionStmtSql = "INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)"

// SetFriendlyLineup replaces teamId's lineup override for one friendly - upserting the
// (match_id, team_id) row (so re-saving doesn't need its own delete-then-insert) and always
// replacing its positions outright. Guarded by a non-blocking advisory lock keyed on
// (matchId, teamId), the same race-avoidance convention as SetDefaultLineup, scoped to just
// this match so editing one friendly never blocks editing another for the same team.
func SetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64, formation int, positions []LineupPosition) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1, $2)", matchId, teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	var lineupId int64
	if err := tx.QueryRow(InsertFriendlyLineupStmtSql, matchId, teamId, formation).Scan(&lineupId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteFriendlyLineupPositionsStmtSql, lineupId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertFriendlyLineupPositionStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	for _, p := range positions {
		if _, err := insertStmt.Exec(lineupId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}
	return tx.Commit()
}

const GetFriendlyLineupStmtSql = "SELECT lineup_id, formation FROM friendly_lineup WHERE match_id = $1 AND team_id = $2"
const GetFriendlyLineupPositionsStmtSql = "SELECT player_id, position FROM friendly_lineup_positions WHERE lineup_id = $1"

// GetFriendlyLineup returns teamId's saved override for one friendly. hasOverride is false (with
// a zero Lineup) when the team hasn't saved one for this match yet, so callers like the lineup
// editor can fall back to showing the team's current default lineup as a starting point.
func GetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64) (Lineup, bool, error) {
	lineup := Lineup{MatchId: matchId, TeamId: teamId, Positions: make([]LineupPosition, 0)}

	var lineupId int64
	err := dbCtx.Db.QueryRow(GetFriendlyLineupStmtSql, matchId, teamId).Scan(&lineupId, &lineup.Formation)
	if errors.Is(err, sql.ErrNoRows) {
		return lineup, false, nil
	}
	if err != nil {
		return Lineup{}, false, err
	}

	rows, err := dbCtx.Db.Query(GetFriendlyLineupPositionsStmtSql, lineupId)
	if err != nil {
		return Lineup{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var p LineupPosition
		if err := rows.Scan(&p.PlayerId, &p.Position); err != nil {
			return Lineup{}, false, err
		}
		lineup.Positions = append(lineup.Positions, p)
	}
	return lineup, true, nil
}

// GetLineupsForFriendly resolves both teams' lineups for one friendly: a team's saved override
// (see SetFriendlyLineup) if it has one, falling back to its standing default lineup otherwise -
// the friendly equivalent of GetLineupsForMatch, kept entirely separate (its own tables) so a
// friendly's override can never collide with a competitive match's id space - see the
// friendly_lineup table comment in resources/sql/init.sql for why that matters.
func GetLineupsForFriendly(dbCtx *DatabaseContext, matchId int64, teamIds []int64) (*map[int64]*Lineup, error) {
	lineups := make(map[int64]*Lineup, len(teamIds))
	for _, teamId := range teamIds {
		override, hasOverride, err := GetFriendlyLineup(dbCtx, matchId, teamId)
		if err != nil {
			return nil, err
		}
		if hasOverride && len(override.Positions) > 0 {
			lineups[teamId] = &override
			continue
		}
		defaultLineup, err := GetDefaultLineup(dbCtx, teamId)
		if err != nil {
			return nil, err
		}
		defaultLineup.MatchId = matchId
		lineups[teamId] = defaultLineup
	}
	return &lineups, nil
}

// SetFriendlyPlayed records a simulated friendly's result and its events - same shape as
// SetCupTiePlayed, minus the cup-specific penalty-shootout/winner bookkeeping a friendly has no
// use for.
func SetFriendlyPlayed(dbCtx *DatabaseContext, matchId int64, homeGoals int, awayGoals int, events []Event) error {
	if _, err := dbCtx.Db.Exec(SetFriendlyPlayedStmtSql, homeGoals, awayGoals, matchId); err != nil {
		return err
	}

	insertEvent, err := dbCtx.Db.Prepare(InsertFriendlyEventStmtSql)
	if err != nil {
		return err
	}
	defer insertEvent.Close()

	for _, e := range events {
		teamPlayersJson, _ := json.Marshal(e.TeamOwnerPlayers)
		teamPlayerNamesJson, _ := json.Marshal(e.TeamOwnerPlayersNames)
		if _, err := insertEvent.Exec(matchId, e.EventType, e.TeamOwner, e.OtherTeam, e.Minute,
			teamPlayersJson, teamPlayerNamesJson, e.HomeGoals, e.AwayGoals); err != nil {
			return err
		}
	}
	return nil
}
