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
	ChallengeId        int64      `json:"challengeId"`
	ChallengerTeamId   int64      `json:"challengerTeamId"`
	ChallengerTeamName string     `json:"challengerTeamName"`
	OpponentTeamId     int64      `json:"opponentTeamId"`
	OpponentTeamName   string     `json:"opponentTeamName"`
	CreatedAt          time.Time  `json:"createdAt"`
	// ProposedKickoff is the kickoff the challenger optionally suggested when sending the
	// challenge (see ChallengeToFriendly) - nil if they left it up to the opponent. Purely a
	// suggested default for the opponent's own accept-time kickoff picker; never enforced.
	ProposedKickoff *time.Time `json:"proposedKickoff,omitempty"`
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
	// Events/MatchInstants/HomeLineup/AwayLineup/HomePlayers/AwayPlayers/Ratings are only
	// populated for a played friendly (see EnrichPlayedFriendly) - the same match-detail
	// breakdown a league match's WeekMatch carries (events, zone-by-zone qualifications, the
	// lineups actually used, and per-player ratings), so the friendly detail page can show the
	// same report a league match gets.
	Events        *[]Event          `json:"events,omitempty"`
	MatchInstants *[]MatchInstant   `json:"matchInstants,omitempty"`
	HomeLineup    *Lineup           `json:"homeLineup,omitempty"`
	AwayLineup    *Lineup           `json:"awayLineup,omitempty"`
	HomePlayers   *map[int64]Player `json:"homePlayers,omitempty"`
	AwayPlayers   *map[int64]Player `json:"awayPlayers,omitempty"`
	Ratings       *map[int64]float64 `json:"ratings,omitempty"`
}

// StartOfWeek returns midnight UTC on the most recent Monday on or before t - the real-calendar-
// week boundary a friendly-match weekly limit resets on.
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
// window (see friendlyScheduleDelay) and falls on a weekday GetFriendlyScheduleWeekdays allows.
// now is the reference instant the "far enough ahead" check is measured from - a human-chosen
// kickoff is checked against the actual current time, while the auto-accept path (see
// ChallengeToFriendly) passes the same now it used to derive kickoff via nextAllowedFriendlyDate,
// so a kickoff that was correct by construction can't be judged "too soon" purely because some
// processing time (a DB round trip or two) has elapsed since it was computed - using a fresh
// time.Now() here for that path made this fail on every call, since real time only ever advances
// between the two.
func isAllowedFriendlyKickoff(dbCtx *DatabaseContext, kickoff time.Time, now time.Time) bool {
	if kickoff.Before(now.Add(friendlyScheduleDelay)) {
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

const InsertFriendlyChallengeStmtSql = "INSERT INTO friendly_challenges (challenger_team_id, opponent_team_id, proposed_kickoff) VALUES ($1, $2, $3) RETURNING challenge_id"
const GetFriendlyChallengeStmtSql = "SELECT challenger_team_id, opponent_team_id FROM friendly_challenges WHERE challenge_id = $1"
const GetFriendlyChallengeForUpdateStmtSql = GetFriendlyChallengeStmtSql + " " + ForUpdate
const DeleteFriendlyChallengeStmtSql = "DELETE FROM friendly_challenges WHERE challenge_id = $1"
const InsertFriendlyMatchStmtSql = "INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3)"

// friendlyScheduleDelay is how soon after acceptance a friendly kicks off - short enough to feel
// immediate, long enough to comfortably clear the simulation scheduler's per-minute polling (see
// engine.StartFriendlyScheduler).
const friendlyScheduleDelay = 5 * time.Minute

// ChallengeToFriendly proposes a friendly match to opponentTeamId, on behalf of challengerTeamId,
// optionally suggesting a kickoff (proposedKickoff may be nil to leave it entirely up to the
// opponent). A non-nil proposedKickoff must itself be a valid kickoff (see
// isAllowedFriendlyKickoff) - the challenger doesn't get to suggest something the opponent
// couldn't have picked themselves anyway. If the opponent is unowned (an AI-managed team - see
// AiLevelCaretaker/Manager/Director, every tier of which accepts unconditionally, there being no
// downside to a free match for any of them), the challenge is accepted immediately rather than
// left pending forever with nobody to act on it, using proposedKickoff if one was given and is
// still valid by then; otherwise it waits for the opponent's owner to accept or reject it, with
// proposedKickoff surfaced to them as a suggested default (see FriendlyChallenge.ProposedKickoff).
func ChallengeToFriendly(dbCtx *DatabaseContext, challengerTeamId int64, opponentTeamId int64, proposedKickoff *time.Time) error {
	if challengerTeamId == opponentTeamId {
		return ErrCannotChallengeOwnTeam
	}
	if proposedKickoff != nil && !isAllowedFriendlyKickoff(dbCtx, *proposedKickoff, time.Now()) {
		return ErrInvalidFriendlyKickoff
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
	err = dbCtx.Db.QueryRow(InsertFriendlyChallengeStmtSql, challengerTeamId, opponentTeamId, proposedKickoff).Scan(&challengeId)
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
	// now is captured once and threaded through to acceptFriendlyChallenge's validation below -
	// see its own comment on why a second, independently-captured time.Now() there made this
	// fail on every call.
	now := time.Now()
	defaultKickoff := nextAllowedFriendlyDate(now.Add(friendlyScheduleDelay), GetFriendlyScheduleWeekdays(dbCtx))
	kickoff := defaultKickoff
	if proposedKickoff != nil && isAllowedFriendlyKickoff(dbCtx, *proposedKickoff, now) {
		kickoff = *proposedKickoff
	}
	if err := acceptFriendlyChallenge(dbCtx, challengeId, opponentTeamId, kickoff, now); err != nil {
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
// budget. kickoff must be far enough ahead of the current moment to clear the simulation
// scheduler's polling window and fall on a weekday the friendly schedule allows (see
// isAllowedFriendlyKickoff) - this is the human-accept entry point, so "the current moment" is
// the real wall clock time right now.
func AcceptFriendlyChallenge(dbCtx *DatabaseContext, challengeId int64, opponentTeamId int64, kickoff time.Time) error {
	return acceptFriendlyChallenge(dbCtx, challengeId, opponentTeamId, kickoff, time.Now())
}

// acceptFriendlyChallenge is AcceptFriendlyChallenge's implementation, parameterized on now so
// ChallengeToFriendly's auto-accept path (an unowned opponent, nobody to ask) can pass the exact
// instant it used to derive kickoff via nextAllowedFriendlyDate, rather than this function
// capturing its own later time.Now() - kickoff is only ever "far enough ahead of now" at the
// moment it was computed; using this function's own subsequently-later now to judge it would
// reject an at-construction-valid kickoff purely because some processing time (a couple of DB
// round trips) elapsed in between, which is guaranteed to happen on every real call.
func acceptFriendlyChallenge(dbCtx *DatabaseContext, challengeId int64, opponentTeamId int64, kickoff time.Time, now time.Time) error {
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
	if !isAllowedFriendlyKickoff(dbCtx, kickoff, now) {
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

const GetFriendlyChallengesReceivedStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at, fc.proposed_kickoff " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.opponent_team_id = $1 ORDER BY fc.created_at DESC"

const GetFriendlyChallengesSentStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at, fc.proposed_kickoff " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.challenger_team_id = $1 ORDER BY fc.created_at DESC"

func scanFriendlyChallenges(rows *sql.Rows) ([]FriendlyChallenge, error) {
	defer rows.Close()
	challenges := make([]FriendlyChallenge, 0)
	for rows.Next() {
		var c FriendlyChallenge
		if err := rows.Scan(&c.ChallengeId, &c.ChallengerTeamId, &c.ChallengerTeamName,
			&c.OpponentTeamId, &c.OpponentTeamName, &c.CreatedAt, &c.ProposedKickoff); err != nil {
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

// fm.match_date > now() (rather than fm.played = false) keeps a friendly "upcoming" up to its own
// kickoff even though it's read into simulation up to 15 minutes early and `played` flips true
// before then (see GetSimulableFriendliesStmtSql) - otherwise it would vanish from "upcoming" and
// show up in "recent" (below) with its final score, before the viewer ever saw it kick off.
const GetUpcomingFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE (fm.home_team_id = $1 OR fm.away_team_id = $1) AND fm.match_date > now() ORDER BY fm.match_date ASC"

const GetRecentFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE (fm.home_team_id = $1 OR fm.away_team_id = $1) AND fm.played = true AND fm.match_date <= now() ORDER BY fm.match_date DESC LIMIT $2"

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

// GetUpcomingFriendlies lists teamId's accepted friendlies whose kickoff hasn't arrived yet,
// soonest first - see GetUpcomingFriendliesStmtSql on why that's match_date rather than played.
func GetUpcomingFriendlies(dbCtx *DatabaseContext, teamId int64) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetUpcomingFriendliesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

// GetRecentFriendlies lists teamId's most recently kicked-off, played friendlies, most recent
// first.
func GetRecentFriendlies(dbCtx *DatabaseContext, teamId int64, limit int) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetRecentFriendliesStmtSql, teamId, limit)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

const GetTeamFriendlyMatchesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE fm.home_team_id = $1 OR fm.away_team_id = $1 ORDER BY fm.match_date"

// GetTeamFriendlyMatches lists every one of teamId's friendlies, played or not, oldest first -
// unlike GetUpcomingFriendlies/GetRecentFriendlies (which serve the friendlies page's separate
// "upcoming"/"recent" sections), this is for GetTeamMatches, which needs one combined
// past-and-future list to merge alongside a team's league matches and cup ties.
func GetTeamFriendlyMatches(dbCtx *DatabaseContext, teamId int64) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetTeamFriendlyMatchesStmtSql, teamId)
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
const InsertFriendlyInstantStmtSql = "INSERT INTO friendly_instants(match_id, minute, home_goals, away_goals, home_qua, away_qua, " +
	"midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"

const InsertFriendlyLineupStmtSql = "INSERT INTO friendly_lineup (match_id, team_id, formation) VALUES ($1, $2, $3) " +
	"ON CONFLICT (match_id, team_id) DO UPDATE SET formation = EXCLUDED.formation RETURNING lineup_id"
const DeleteFriendlyLineupPositionsStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id = $1"
const InsertFriendlyLineupPositionStmtSql = "INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)"

// SetFriendlyLineup replaces teamId's lineup override for one friendly, along with any
// substitutions planned specifically for it (see friendly_lineup_substitutions) - upserting the
// (match_id, team_id) row (so re-saving doesn't need its own delete-then-insert) and always
// replacing its positions and substitutions outright. Guarded by a non-blocking advisory lock
// keyed on (matchId, teamId), the same race-avoidance convention as SetDefaultLineup, scoped to
// just this match so editing one friendly never blocks editing another for the same team.
func SetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64, formation int, positions []LineupPosition, substitutions []LineupSubstitution) error {
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

	if _, err := tx.Exec(DeleteFriendlyLineupSubstitutionsStmtSql, matchId, teamId); err != nil {
		return err
	}
	insertSubStmt, err := tx.Prepare(InsertFriendlyLineupSubstitutionStmtSql)
	if err != nil {
		return err
	}
	defer insertSubStmt.Close()
	for i, s := range substitutions {
		if _, err := insertSubStmt.Exec(matchId, teamId, i+1, s.PlayerOutId, s.PlayerInId, s.Minute); err != nil {
			return err
		}
	}

	return tx.Commit()
}

const GetFriendlyLineupStmtSql = "SELECT lineup_id, formation FROM friendly_lineup WHERE match_id = $1 AND team_id = $2"
const GetFriendlyLineupPositionsStmtSql = "SELECT player_id, position FROM friendly_lineup_positions WHERE lineup_id = $1"

// GetFriendlyLineup returns teamId's saved override for one friendly, including any
// substitutions planned specifically for it (see GetFriendlyLineupSubstitutions). hasOverride is
// false (with a zero Lineup) when the team hasn't saved one for this match yet, so callers like
// the lineup editor can fall back to showing the team's current best available XI as a starting
// point (see BestLineupForTeam).
func GetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64) (Lineup, bool, error) {
	lineup := Lineup{MatchId: matchId, TeamId: teamId, Positions: make([]LineupPosition, 0), Substitutions: make([]LineupSubstitution, 0)}

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

	substitutions, err := GetFriendlyLineupSubstitutions(dbCtx, matchId, teamId)
	if err != nil {
		return Lineup{}, false, err
	}
	lineup.Substitutions = substitutions

	return lineup, true, nil
}

// GetLineupsForFriendly resolves both teams' lineups for one friendly, in priority order: a
// team's saved per-match override (see SetFriendlyLineup) if it has one; else its standing
// friendly-only default lineup (see GetDefaultFriendlyLineup) if the manager has set one up; else
// a freshly computed best available XI (see BestLineupForTeam) - so a friendly is always
// fulfilled by default with the strongest squad currently available rather than a stale or
// never-configured default. Deliberately never falls back to the league/cup default (see
// GetDefaultLineup): that's a separate, competitive-only concept a manager may not want touched by
// friendlies at all. Kept entirely separate from GetLineupsForMatch (its own tables) so a
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

		friendlyDefault, err := GetDefaultFriendlyLineup(dbCtx, teamId)
		if err != nil {
			return nil, err
		}
		if len(friendlyDefault.Positions) > 0 {
			friendlyDefault.MatchId = matchId
			lineups[teamId] = friendlyDefault
			continue
		}

		bestLineup, err := BestLineupForTeam(dbCtx, teamId)
		if err != nil {
			return nil, err
		}
		bestLineup.MatchId = matchId
		lineups[teamId] = bestLineup
	}
	return &lineups, nil
}

// SetFriendlyPlayed records a simulated friendly's result, its events, and its per-minute zone
// instants (see friendly_instants) - the friendly equivalent of engine.UpdatePlayedMatch, minus
// the cup-specific penalty-shootout/winner bookkeeping a friendly has no use for.
func SetFriendlyPlayed(dbCtx *DatabaseContext, matchId int64, homeGoals int, awayGoals int, events []Event, instants []MatchInstant) error {
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

	insertInstant, err := dbCtx.Db.Prepare(InsertFriendlyInstantStmtSql)
	if err != nil {
		return err
	}
	defer insertInstant.Close()

	for _, i := range instants {
		homeQuaJson, _ := json.Marshal(i.HomeQua)
		awayQuaJson, _ := json.Marshal(i.AwayQua)
		if _, err := insertInstant.Exec(matchId, i.Minute, i.HomeGoals, i.AwayGoals, homeQuaJson, awayQuaJson, i.MidfieldDiff,
			i.HomeAttackRightDiff, i.HomeAttackLeftDiff, i.HomeAttackCenterDiff,
			i.AwayAttackRightDiff, i.AwayAttackLeftDiff, i.AwayAttackCenterDiff); err != nil {
			return err
		}
	}
	return nil
}

const GetFriendlyEventsStmtSql = "SELECT event_type, team_owner, other_team, minute, team_players, team_player_names, home_goals, away_goals " +
	"FROM friendly_events WHERE match_id = $1 order by minute asc"

// GetFriendlyEvents reads back a played friendly's events (see SetFriendlyPlayed) - the friendly
// equivalent of GetMatchEvents. friendly_events has no other_team_players/other_team_player_names
// columns (nothing in this codebase reads them even for league matches - see Event's own doc
// comment), so those two fields are always left nil here.
func GetFriendlyEvents(dbCtx *DatabaseContext, matchId int64) *[]Event {
	rows, err := dbCtx.Db.Query(GetFriendlyEventsStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var eventType int
		var teamOwner int64
		var otherTeam int64
		var minute int
		var teamPlayers string
		var teamPlayerNames string
		var homeGoals int
		var awayGoals int
		if err := rows.Scan(&eventType, &teamOwner, &otherTeam, &minute, &teamPlayers, &teamPlayerNames,
			&homeGoals, &awayGoals); err != nil {
			continue
		}

		event := Event{
			EventType: eventType,
			Minute:    minute,
			TeamOwner: teamOwner,
			OtherTeam: otherTeam,
			HomeGoals: homeGoals,
			AwayGoals: awayGoals,
		}
		if teamPlayers != "" {
			players := make([]int64, 0)
			_ = json.Unmarshal([]byte(teamPlayers), &players)
			event.TeamOwnerPlayers = players
		}
		if teamPlayerNames != "" {
			names := make([]string, 0)
			_ = json.Unmarshal([]byte(teamPlayerNames), &names)
			event.TeamOwnerPlayersNames = names
		}
		events = append(events, event)
	}
	return &events
}

const GetFriendlyInstantsStmtSql = "SELECT match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, " +
	"homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff " +
	"FROM friendly_instants WHERE match_id = $1 order by minute asc"

// GetFriendlyInstants reads back a played friendly's per-minute zone instants (see
// SetFriendlyPlayed) - the friendly equivalent of GetMatchInstants.
func GetFriendlyInstants(dbCtx *DatabaseContext, matchId int64) *[]MatchInstant {
	rows, err := dbCtx.Db.Query(GetFriendlyInstantsStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()

	instants := make([]MatchInstant, 0)
	for rows.Next() {
		var instantMatchId int64
		var minute int
		var homeGoals int
		var awayGoals int
		var homeQua string
		var awayQua string
		var midFieldDiff float32
		var homeAttackRightDiff float32
		var homeAttackLeftDiff float32
		var homeAttackCenterDiff float32
		var awayAttackRightDiff float32
		var awayAttackLeftDiff float32
		var awayAttackCenterDiff float32
		if err := rows.Scan(&instantMatchId, &minute, &homeGoals, &awayGoals, &homeQua, &awayQua, &midFieldDiff,
			&homeAttackRightDiff, &homeAttackLeftDiff, &homeAttackCenterDiff,
			&awayAttackRightDiff, &awayAttackLeftDiff, &awayAttackCenterDiff); err != nil {
			continue
		}

		homeTeamQua := TeamQualification{}
		awayTeamQua := TeamQualification{}
		if homeQua != "" {
			_ = json.Unmarshal([]byte(homeQua), &homeTeamQua)
		}
		if awayQua != "" {
			_ = json.Unmarshal([]byte(awayQua), &awayTeamQua)
		}
		instants = append(instants, MatchInstant{
			MatchId:              instantMatchId,
			Minute:               minute,
			HomeGoals:            homeGoals,
			AwayGoals:            awayGoals,
			HomeQua:              homeTeamQua,
			AwayQua:              awayTeamQua,
			MidfieldDiff:         midFieldDiff,
			HomeAttackRightDiff:  homeAttackRightDiff,
			HomeAttackLeftDiff:   homeAttackLeftDiff,
			HomeAttackCenterDiff: homeAttackCenterDiff,
			AwayAttackRightDiff:  awayAttackRightDiff,
			AwayAttackLeftDiff:   awayAttackLeftDiff,
			AwayAttackCenterDiff: awayAttackCenterDiff,
		})
	}
	return &instants
}

// EnrichPlayedFriendly populates a played friendly's match-detail breakdown - events, per-minute
// zone instants, the lineups actually used, both squads' players, and per-player ratings - the
// same shape a league match's WeekMatch carries (see WeekMatchById), so the friendly detail page
// can show the same report tabs a league match gets. A no-op (returns nil) for a match that
// hasn't been played yet, since none of this data exists until then.
func EnrichPlayedFriendly(dbCtx *DatabaseContext, match *FriendlyMatch) error {
	if !match.Played {
		return nil
	}

	match.Events = GetFriendlyEvents(dbCtx, match.MatchId)
	match.MatchInstants = GetFriendlyInstants(dbCtx, match.MatchId)

	lineups, err := GetLineupsForFriendly(dbCtx, match.MatchId, []int64{match.HomeTeamId, match.AwayTeamId})
	if err != nil {
		return err
	}
	match.HomeLineup = (*lineups)[match.HomeTeamId]
	match.AwayLineup = (*lineups)[match.AwayTeamId]

	homePlayers, err := GetPlayersFromTeamId(dbCtx, match.HomeTeamId)
	if err != nil {
		return err
	}
	homeMap := playerListToMap(homePlayers)
	match.HomePlayers = &homeMap

	awayPlayers, err := GetPlayersFromTeamId(dbCtx, match.AwayTeamId)
	if err != nil {
		return err
	}
	awayMap := playerListToMap(awayPlayers)
	match.AwayPlayers = &awayMap

	if match.HomeLineup != nil && match.AwayLineup != nil && match.Events != nil {
		match.Ratings = ComputeMatchRatings(&WeekMatch{
			MatchId:    match.MatchId,
			HomeGoals:  match.HomeGoals,
			AwayGoals:  match.AwayGoals,
			Events:     match.Events,
			HomeLineup: match.HomeLineup,
			AwayLineup: match.AwayLineup,
		})
	}

	return nil
}
