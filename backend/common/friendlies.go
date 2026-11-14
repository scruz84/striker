package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
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
var ErrInvalidFriendlyKickoff = errors.New("kickoff must be far enough in the future, on the hour, quarter past, half past or quarter to, and fall on a day the friendly schedule allows")
var ErrNotFriendlyParticipant = errors.New("your team is not part of this friendly")
var ErrFriendlyAlreadyPlayed = errors.New("this friendly has already been played")
var ErrFriendlyKickoffTaken = errors.New("one of the teams already has a friendly at that kickoff")

// FriendlyChallenge is a pending invite from one team to another - see ChallengeToFriendly.
type FriendlyChallenge struct {
	ChallengeId        int64     `json:"challengeId"`
	ChallengerTeamId   int64     `json:"challengerTeamId"`
	ChallengerTeamName string    `json:"challengerTeamName"`
	OpponentTeamId     int64     `json:"opponentTeamId"`
	OpponentTeamName   string    `json:"opponentTeamName"`
	CreatedAt          time.Time `json:"createdAt"`
	// ProposedKickoff is the kickoff the challenger optionally suggested when sending the
	// challenge (see ChallengeToFriendly) - nil if they left it up to the opponent. Purely a
	// suggested default for the opponent's own accept-time kickoff picker; never enforced.
	ProposedKickoff *time.Time `json:"proposedKickoff,omitempty"`
	// ChallengerIsHome is which side of the fixture the challenger chose to take when sending
	// the challenge (see ChallengeToFriendly) - true puts the challenger at home once accepted,
	// false puts the opponent at home instead.
	ChallengerIsHome bool `json:"challengerIsHome"`
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
	Events        *[]Event           `json:"events,omitempty"`
	MatchInstants *[]MatchInstant    `json:"matchInstants,omitempty"`
	HomeLineup    *Lineup            `json:"homeLineup,omitempty"`
	AwayLineup    *Lineup            `json:"awayLineup,omitempty"`
	HomePlayers   *map[int64]Player  `json:"homePlayers,omitempty"`
	AwayPlayers   *map[int64]Player  `json:"awayPlayers,omitempty"`
	Ratings       *map[int64]float64 `json:"ratings,omitempty"`
	// PlayerRatings is Ratings with every featured player's breakdown (see LoadPlayerRatings).
	PlayerRatings []PlayerRating `json:"playerRatings,omitempty"`
	// HomeStadiumName is the home team's stadium at fetch time (empty for a national team);
	// Weather/Attendance/RefereeName/Assistant1Name/Assistant2Name are the stored report flavor of a
	// played friendly (see UpdateFriendlyFlavor), empty for one played before
	// resources/sql/init83_match_viewer_parity.sql - the friendly equivalent of WeekMatch's fields.
	HomeStadiumName string `json:"homeStadiumName,omitempty"`
	Weather         string `json:"weather,omitempty"`
	Attendance      int    `json:"attendance,omitempty"`
	RefereeName     string `json:"refereeName,omitempty"`
	Assistant1Name  string `json:"assistant1Name,omitempty"`
	Assistant2Name  string `json:"assistant2Name,omitempty"`
	// WorldCupGroupName/WorldCupRound are set only when this friendly is a World Cup group fixture
	// (see world_cup_group_matches), so the viewer can label it "Mundial · Grupo B · Jornada 2".
	WorldCupGroupName string `json:"worldCupGroupName,omitempty"`
	WorldCupRound     int    `json:"worldCupRound,omitempty"`
}

// StartOfWeek returns midnight UTC on the most recent Monday on or before t - the real-calendar-
// week boundary a friendly-match weekly limit resets on.
func StartOfWeek(t time.Time) time.Time {
	t = t.UTC()
	daysSinceMonday := (int(t.Weekday()) + 6) % 7 // Monday=0, Tuesday=1, ..., Sunday=6
	monday := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// StartOfDay returns midnight UTC on t's own calendar day - the boundary
// GetTeamsWithFriendlyToday uses to cap an AI-managed team at one new friendly per day (see its
// own doc comment).
func StartOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
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

// friendlyKickoffStep is the grid friendlies kick off on: only on the hour, at quarter past, half
// past or quarter to. Every timezone offset in use is a multiple of 15 minutes, so a kickoff on
// this grid in UTC is on it in every viewer's local time too.
const friendlyKickoffStep = 15 * time.Minute

// IsFriendlyKickoffSlot reports whether kickoff falls exactly on a friendlyKickoffStep slot
// (:00, :15, :30 or :45, zero seconds).
func IsFriendlyKickoffSlot(kickoff time.Time) bool {
	return kickoff.Equal(kickoff.Truncate(friendlyKickoffStep))
}

// nextFriendlyKickoffSlot rounds t up to the next friendlyKickoffStep slot, t itself if it is one.
func nextFriendlyKickoffSlot(t time.Time) time.Time {
	slot := t.Truncate(friendlyKickoffStep)
	if slot.Before(t) {
		slot = slot.Add(friendlyKickoffStep)
	}
	return slot
}

// isAllowedFriendlyKickoff reports whether kickoff clears the simulation scheduler's polling
// window (see friendlyScheduleDelay), falls on a quarter-hour slot (see IsFriendlyKickoffSlot) and
// on a weekday GetFriendlyScheduleWeekdays allows.
// now is the reference instant the "far enough ahead" check is measured from - a human-chosen
// kickoff is checked against the actual current time, while the auto-accept path (see
// ChallengeToFriendly) passes the same now it used to derive kickoff via nextAllowedFriendlyDate,
// so a kickoff that was correct by construction can't be judged "too soon" purely because some
// processing time (a DB round trip or two) has elapsed since it was computed - using a fresh
// time.Now() here for that path made this fail on every call, since real time only ever advances
// between the two.
func isAllowedFriendlyKickoff(dbCtx *DatabaseContext, kickoff time.Time, now time.Time) bool {
	if kickoff.Before(now.Add(friendlyScheduleDelay)) || !IsFriendlyKickoffSlot(kickoff) {
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

const CountFriendliesInWeekStmtSql = "SELECT count(*) FROM friendly_matches WHERE (home_team_id = $1 OR away_team_id = $1) AND match_date >= $2 AND match_date < $3"

// CountFriendliesInWeekOf counts teamId's accepted friendlies (scheduled or already played)
// kicking off in the real calendar week (Monday to Monday, UTC - see StartOfWeek) that contains
// t - a friendly counts against its week's cap the moment it's accepted, not only once it's
// actually been played, and only against its own week: one scheduled for next week does not use
// up this week's.
func CountFriendliesInWeekOf(q sqlQuerier, teamId int64, t time.Time) (int, error) {
	start := StartOfWeek(t)
	var count int
	err := q.QueryRow(CountFriendliesInWeekStmtSql, teamId, start, start.AddDate(0, 0, 7)).Scan(&count)
	return count, err
}

// CountFriendliesAtStmtSql counts the friendlies teamId already has at exactly one kickoff.
const CountFriendliesAtStmtSql = "SELECT count(*) FROM friendly_matches WHERE (home_team_id = $1 OR away_team_id = $1) AND match_date = $2"

// lockTeamFriendliesTx serializes, for the rest of tx, every friendly acceptance involving one
// of teamIds: the weekly cap and the clash check read and then insert, so two acceptances
// running at once must not both pass. Locks are taken in ascending id order, so two acceptances
// sharing both teams can never deadlock.
func lockTeamFriendliesTx(tx *sql.Tx, teamIds ...int64) error {
	sorted := append([]int64(nil), teamIds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for _, id := range sorted {
		if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", NamespacedLockKey(LockNamespaceTeamFriendlies, id)); err != nil {
			return err
		}
	}
	return nil
}

const GetTeamsWithFriendlyTodayStmtSql = "SELECT home_team_id, away_team_id FROM friendly_matches WHERE match_date >= $1 AND match_date < $2"

// GetTeamsWithFriendlyToday returns the set of team ids with any accepted friendly (scheduled or
// already played) kicking off today (the UTC calendar day - see StartOfDay), on either side of the
// fixture. engine.RunAiTeamDailyActivity uses this to cap an AI-managed team at one new friendly
// challenge per day - both as the would-be challenger and as a candidate opponent - so a week's
// worth of challenges actually spread across distinct days instead of a lucky streak exhausting
// the whole weekly cap (see GetMaxFriendliesPerWeek) in a single day while other days go unused.
func GetTeamsWithFriendlyToday(dbCtx *DatabaseContext) (map[int64]bool, error) {
	start := StartOfDay(time.Now())
	rows, err := dbCtx.Db.Query(GetTeamsWithFriendlyTodayStmtSql, start, start.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := make(map[int64]bool)
	for rows.Next() {
		var home, away int64
		if err := rows.Scan(&home, &away); err != nil {
			return nil, err
		}
		teams[home] = true
		teams[away] = true
	}
	return teams, rows.Err()
}

const InsertFriendlyChallengeStmtSql = "INSERT INTO friendly_challenges (challenger_team_id, opponent_team_id, proposed_kickoff, challenger_is_home) VALUES ($1, $2, $3, $4) RETURNING challenge_id"
const GetFriendlyChallengeStmtSql = "SELECT challenger_team_id, opponent_team_id, challenger_is_home FROM friendly_challenges WHERE challenge_id = $1"
const GetFriendlyChallengeForUpdateStmtSql = GetFriendlyChallengeStmtSql + " " + ForUpdate
const DeleteFriendlyChallengeStmtSql = "DELETE FROM friendly_challenges WHERE challenge_id = $1"
const InsertFriendlyMatchStmtSql = "INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3)"

// friendlyScheduleDelay is the least notice a friendly gets: an auto-accepted one kicks off at the
// first quarter-hour slot (see nextFriendlyKickoffSlot) at least this long after acceptance -
// short enough to feel immediate, long enough to comfortably clear the simulation scheduler's
// per-minute polling (see engine.StartFriendlyScheduler).
const friendlyScheduleDelay = 5 * time.Minute

// ChallengeToFriendly proposes a friendly match to opponentTeamId, on behalf of challengerTeamId,
// optionally suggesting a kickoff (proposedKickoff may be nil to leave it entirely up to the
// opponent). challengerIsHome picks which side of the fixture the challenger takes once the
// challenge is accepted - true keeps the challenger at home, false makes the opponent the home
// team instead. A non-nil proposedKickoff must itself be a valid kickoff (see
// isAllowedFriendlyKickoff) - the challenger doesn't get to suggest something the opponent
// couldn't have picked themselves anyway. If the opponent is unowned (an AI-managed team - see
// AiLevelCaretaker/Manager/Director, every tier of which accepts unconditionally, there being no
// downside to a free match for any of them), the challenge is accepted immediately rather than
// left pending forever with nobody to act on it, using proposedKickoff if one was given and is
// still valid by then; otherwise it waits for the opponent's owner to accept or reject it, with
// proposedKickoff surfaced to them as a suggested default (see FriendlyChallenge.ProposedKickoff).
func ChallengeToFriendly(dbCtx *DatabaseContext, challengerTeamId int64, opponentTeamId int64, proposedKickoff *time.Time, challengerIsHome bool) error {
	if challengerTeamId == opponentTeamId {
		return ErrCannotChallengeOwnTeam
	}
	if proposedKickoff != nil && !isAllowedFriendlyKickoff(dbCtx, *proposedKickoff, time.Now()) {
		return ErrInvalidFriendlyKickoff
	}

	// Judged for the week the friendly would be in when the challenger names a kickoff, else for
	// this week; acceptance re-checks both sides for the week of the actual kickoff.
	week := time.Now()
	if proposedKickoff != nil {
		week = *proposedKickoff
	}
	challengerCount, err := CountFriendliesInWeekOf(dbCtx.Db, challengerTeamId, week)
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
	err = dbCtx.Db.QueryRow(InsertFriendlyChallengeStmtSql, challengerTeamId, opponentTeamId, proposedKickoff, challengerIsHome).Scan(&challengeId)
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
	defaultKickoff := nextAllowedFriendlyDate(nextFriendlyKickoffSlot(now.Add(friendlyScheduleDelay)), GetFriendlyScheduleWeekdays(dbCtx))
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
// double-accept can't create two fixtures from the same challenge. Re-checks both teams' weekly
// limit for the kickoff's week, and that neither already has a friendly at that kickoff, now,
// since time may have passed (and other friendlies accepted) since the challenge was sent - the same "re-validate at accept time" convention AcceptOffer uses for
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
	var challengerIsHome bool
	err = tx.QueryRow(GetFriendlyChallengeForUpdateStmtSql, challengeId).Scan(&challengerTeamId, &actualOpponentTeamId, &challengerIsHome)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFriendlyChallengeNotFound
	}
	if err != nil {
		return err
	}
	if actualOpponentTeamId != opponentTeamId {
		return ErrNotChallengeRecipient
	}

	if !isAllowedFriendlyKickoff(dbCtx, kickoff, now) {
		return ErrInvalidFriendlyKickoff
	}
	// Both sides' cap for the kickoff's week and their clashes are checked under a per-team lock,
	// so challenges sent before either side reached its cap can't all be accepted past it, and
	// two acceptances at the same moment can't both pass.
	if err := lockTeamFriendliesTx(tx, challengerTeamId, opponentTeamId); err != nil {
		return err
	}
	maxPerWeek, err := GetMaxFriendliesPerWeek(dbCtx)
	if err != nil {
		return err
	}
	for _, teamId := range []int64{opponentTeamId, challengerTeamId} {
		count, err := CountFriendliesInWeekOf(tx, teamId, kickoff)
		if err != nil {
			return err
		}
		if count >= maxPerWeek {
			return ErrFriendlyLimitReached
		}
		var clashes int
		if err := tx.QueryRow(CountFriendliesAtStmtSql, teamId, kickoff).Scan(&clashes); err != nil {
			return err
		}
		if clashes > 0 {
			return ErrFriendlyKickoffTaken
		}
	}

	homeTeamId, awayTeamId := challengerTeamId, opponentTeamId
	if !challengerIsHome {
		homeTeamId, awayTeamId = opponentTeamId, challengerTeamId
	}
	if _, err := tx.Exec(InsertFriendlyMatchStmtSql, homeTeamId, awayTeamId, kickoff); err != nil {
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
	var challengerIsHome bool
	err := dbCtx.Db.QueryRow(GetFriendlyChallengeStmtSql, challengeId).Scan(&challengerTeamId, &actualOpponentTeamId, &challengerIsHome)
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
	var challengerIsHome bool
	err := dbCtx.Db.QueryRow(GetFriendlyChallengeStmtSql, challengeId).Scan(&actualChallengerTeamId, &opponentTeamId, &challengerIsHome)
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

const GetFriendlyChallengesReceivedStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at, fc.proposed_kickoff, fc.challenger_is_home " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.opponent_team_id = $1 ORDER BY fc.created_at DESC"

const GetFriendlyChallengesSentStmtSql = "SELECT fc.challenge_id, fc.challenger_team_id, ct.name, fc.opponent_team_id, ot.name, fc.created_at, fc.proposed_kickoff, fc.challenger_is_home " +
	"FROM friendly_challenges fc JOIN teams ct ON ct.team_id = fc.challenger_team_id JOIN teams ot ON ot.team_id = fc.opponent_team_id " +
	"WHERE fc.challenger_team_id = $1 ORDER BY fc.created_at DESC"

func scanFriendlyChallenges(rows *sql.Rows) ([]FriendlyChallenge, error) {
	defer rows.Close()
	challenges := make([]FriendlyChallenge, 0)
	for rows.Next() {
		var c FriendlyChallenge
		if err := rows.Scan(&c.ChallengeId, &c.ChallengerTeamId, &c.ChallengerTeamName,
			&c.OpponentTeamId, &c.OpponentTeamName, &c.CreatedAt, &c.ProposedKickoff, &c.ChallengerIsHome); err != nil {
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

const GetFriendlyMatchByIdStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals, " +
	"ht.stadium_name, fm.weather, fm.attendance, fm.referee_name, fm.assistant1_name, fm.assistant2_name, wcg.group_name, wcgm.round " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"LEFT JOIN world_cup_group_matches wcgm ON wcgm.match_id = fm.match_id LEFT JOIN world_cup_groups wcg ON wcg.group_id = wcgm.group_id " +
	"WHERE fm.match_id = $1"

// GetFriendlyMatchById looks up one friendly by id - used by the lineup and live-view handlers,
// which need the match itself (and to check the caller owns one of its teams) rather than a
// per-team list.
func GetFriendlyMatchById(dbCtx *DatabaseContext, matchId int64) (*FriendlyMatch, error) {
	var m FriendlyMatch
	var stadiumName, weather, refereeName, assistant1, assistant2, groupName sql.NullString
	var attendance, round sql.NullInt64
	err := dbCtx.Db.QueryRow(GetFriendlyMatchByIdStmtSql, matchId).Scan(&m.MatchId, &m.HomeTeamId, &m.HomeTeamName,
		&m.AwayTeamId, &m.AwayTeamName, &m.MatchDate, &m.Played, &m.HomeGoals, &m.AwayGoals,
		&stadiumName, &weather, &attendance, &refereeName, &assistant1, &assistant2, &groupName, &round)
	if err != nil {
		return nil, err
	}
	m.HomeStadiumName = stadiumName.String
	m.WorldCupGroupName = groupName.String
	m.WorldCupRound = int(round.Int64)
	// The stored flavor only reads back once the friendly is played, like its events and instants
	// (see EnrichPlayedFriendly).
	if m.Played {
		m.Weather = weather.String
		m.Attendance = int(attendance.Int64)
		m.RefereeName = refereeName.String
		m.Assistant1Name = assistant1.String
		m.Assistant2Name = assistant2.String
	}
	return &m, nil
}

// GetSimulableFriendliesStmtSql deliberately has no lower bound on match_date (unlike the upper
// bound, which keeps this from ever grabbing something scheduled further out than the next 15
// minutes) - a friendly whose 15-minute window was missed entirely (a brief restart or deploy
// spanning its exact kickoff) must still be picked up on the very next tick once match_date is
// back in the past, not left stranded at played = false forever. A permanently stuck friendly is
// not just that one match unresolved: for a national team specifically, it also permanently blocks
// GetCountriesWithPendingNationalTeamFriendly from ever pairing that country again (the "pending"
// check has no notion of "this is hopelessly overdue, give up on it"), which is the most likely
// explanation for a national team that's never played at all despite being active.
const GetSimulableFriendliesStmtSql = "SELECT match_id, home_team_id, away_team_id, match_date FROM friendly_matches " +
	"WHERE match_date < now() + interval '15 minutes' AND played = false AND in_simulation = false " +
	"AND " + simulationEligibleSql + " " + ForUpdate

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
		if _, err := tx.Exec("UPDATE friendly_matches SET in_simulation = true, in_simulation_since = now(), " + simulationClaimSetSql + " WHERE match_id IN " + inClause); err != nil {
			return nil, err
		}
	}

	return matches, tx.Commit()
}

// GetStuckFriendliesStmtSql is GetStuckMatchesStmtSql's friendly counterpart - same age threshold
// ($1 seconds) and same atomic re-claim.
const GetStuckFriendliesStmtSql = "UPDATE friendly_matches SET in_simulation_since = now(), " + simulationClaimSetSql + " " +
	"WHERE played = false AND in_simulation = true AND " + simulationAttemptsLeftSql + " " +
	"AND (in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second')) " +
	"RETURNING match_id, home_team_id, away_team_id, match_date"

// GetStuckFriendlies re-claims and returns every friendly a simulation pass claimed via
// GetSimulableFriendlies (setting in_simulation = true) at least StuckSimulationAge ago but never
// finished simulating - most likely because the process crashed mid-run. The caller is expected
// to re-simulate them right away (see engine.FinishStuckFriendlies), since
// GetSimulableFriendliesStmtSql's own "and in_simulation = false" filter means the normal
// scheduler will never pick these back up on its own. See GetStuckMatches for why the age
// threshold and the one-statement re-claim matter with several --engine replicas.
func GetStuckFriendlies(dbCtx *DatabaseContext) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(GetStuckFriendliesStmtSql, StuckSimulationAge.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]FriendlyMatch, 0)
	for rows.Next() {
		var m FriendlyMatch
		if err := rows.Scan(&m.MatchId, &m.HomeTeamId, &m.AwayTeamId, &m.MatchDate); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

const InsertFriendlyEventStmtSql = "INSERT INTO friendly_events (match_id, event_type, team_owner, other_team, minute, " +
	"team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals, set_piece, mentality, added_minute, added_time) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"
const InsertFriendlyInstantStmtSql = "INSERT INTO friendly_instants(match_id, minute, home_goals, away_goals, home_qua, away_qua, " +
	"midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff, player_contributions, added_minute) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"

const InsertFriendlyLineupStmtSql = "INSERT INTO friendly_lineup (match_id, team_id, formation, mentality, captain_id, penalty_taker_id, free_kick_taker_id) VALUES ($1, $2, $3, $4, $5, $6, $7) " +
	"ON CONFLICT (match_id, team_id) DO UPDATE SET formation = EXCLUDED.formation, mentality = EXCLUDED.mentality, " +
	"captain_id = EXCLUDED.captain_id, penalty_taker_id = EXCLUDED.penalty_taker_id, free_kick_taker_id = EXCLUDED.free_kick_taker_id RETURNING lineup_id"
const DeleteFriendlyLineupPositionsStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id = $1"
const InsertFriendlyLineupPositionStmtSql = "INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)"

// SetFriendlyLineup replaces teamId's lineup override for one friendly, along with any
// substitutions planned specifically for it (see friendly_lineup_substitutions) - upserting the
// (match_id, team_id) row (so re-saving doesn't need its own delete-then-insert) and always
// replacing its positions and substitutions outright. Guarded by a non-blocking advisory lock
// keyed on (matchId, teamId), the same race-avoidance convention as SetDefaultLineup, scoped to
// just this match so editing one friendly never blocks editing another for the same team.
func SetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64, formation int, mentality int, roles LineupRoles, positions []LineupPosition, substitutions []LineupSubstitution, orders []LineupOrder) error {
	if err := ValidateMentality(mentality); err != nil {
		return err
	}
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
	if err := tx.QueryRow(InsertFriendlyLineupStmtSql, append([]any{matchId, teamId, formation, mentality}, roleValues(roles)...)...).Scan(&lineupId); err != nil {
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

	if err := replaceOrders(tx, DeleteFriendlyLineupOrdersStmtSql, InsertFriendlyLineupOrderStmtSql, []any{matchId, teamId}, orders); err != nil {
		return err
	}

	return tx.Commit()
}

var GetFriendlyLineupStmtSql = "SELECT a.lineup_id, a.formation, a.mentality, " + rolesSelectSql("a", friendlyPlayedSql("a")) + " FROM friendly_lineup a WHERE a.match_id = $1 AND a.team_id = $2"

const GetFriendlyLineupPositionsStmtSql = "SELECT player_id, position FROM friendly_lineup_positions WHERE lineup_id = $1"

// GetFriendlyLineup returns teamId's saved override for one friendly, including any
// substitutions planned specifically for it (see GetFriendlyLineupSubstitutions). hasOverride is
// false (with a zero Lineup) when the team hasn't saved one for this match yet, so callers like
// the lineup editor can fall back to showing the team's current best available XI as a starting
// point (see BestLineupForTeam).
func GetFriendlyLineup(dbCtx *DatabaseContext, matchId int64, teamId int64) (Lineup, bool, error) {
	lineup := Lineup{MatchId: matchId, TeamId: teamId, Positions: make([]LineupPosition, 0), Substitutions: make([]LineupSubstitution, 0)}

	var lineupId int64
	var roles roleScan
	err := dbCtx.Db.QueryRow(GetFriendlyLineupStmtSql, matchId, teamId).Scan(append([]any{&lineupId, &lineup.Formation, &lineup.Mentality}, roles.targets()...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return lineup, false, nil
	}
	if err != nil {
		return Lineup{}, false, err
	}
	lineup.LineupRoles = roles.roles()

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

const GetFriendlyEventsStmtSql = "SELECT event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, " +
	"home_goals, away_goals, set_piece, mentality, " +
	"added_minute, added_time FROM friendly_events WHERE match_id = $1 order by minute asc, added_minute asc, event_id asc"

// GetFriendlyEvents reads back a played friendly's events (see SaveFriendlyResultTx) - the friendly
// equivalent of GetMatchEvents, the other side's players (the defending keeper on a chance)
// included. A row stored before init108 has NULL there and reads as no keeper.
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
		var otherTeamPlayers, otherTeamPlayerNames sql.NullString
		var homeGoals int
		var awayGoals int
		var setPiece int
		var mentality sql.NullInt64
		var addedMinute int
		var addedTime sql.NullInt64
		if err := rows.Scan(&eventType, &teamOwner, &otherTeam, &minute, &teamPlayers, &teamPlayerNames,
			&otherTeamPlayers, &otherTeamPlayerNames, &homeGoals, &awayGoals, &setPiece, &mentality, &addedMinute, &addedTime); err != nil {
			continue
		}

		event := Event{
			EventType:   eventType,
			Minute:      minute,
			TeamOwner:   teamOwner,
			OtherTeam:   otherTeam,
			HomeGoals:   homeGoals,
			AwayGoals:   awayGoals,
			SetPiece:    setPiece,
			Mentality:   nullableInt(mentality),
			AddedMinute: addedMinute,
			AddedTime:   nullableInt(addedTime),
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
		if otherTeamPlayers.Valid && otherTeamPlayers.String != "" {
			_ = json.Unmarshal([]byte(otherTeamPlayers.String), &event.OtherTeamPlayers)
		}
		if otherTeamPlayerNames.Valid && otherTeamPlayerNames.String != "" {
			_ = json.Unmarshal([]byte(otherTeamPlayerNames.String), &event.OtherTeamPlayersNames)
		}
		events = append(events, event)
	}
	return &events
}

const GetFriendlyInstantsStmtSql = "SELECT match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, " +
	"homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff, player_contributions, " +
	"added_minute FROM friendly_instants WHERE match_id = $1 order by minute asc, added_minute asc"

// GetFriendlyInstants reads back a played friendly's per-minute zone instants (see
// SaveFriendlyResultTx) - the friendly equivalent of GetMatchInstants.
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
		var contributions sql.NullString
		var addedMinute int
		if err := rows.Scan(&instantMatchId, &minute, &homeGoals, &awayGoals, &homeQua, &awayQua, &midFieldDiff,
			&homeAttackRightDiff, &homeAttackLeftDiff, &homeAttackCenterDiff,
			&awayAttackRightDiff, &awayAttackLeftDiff, &awayAttackCenterDiff, &contributions, &addedMinute); err != nil {
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
			AddedMinute:          addedMinute,
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
			PlayerContributions:  decodePlayerContributions(contributions),
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
	// The set-piece takers are the owning manager's business; only the captain's armband is public.
	RedactLineupTakers(match.HomeLineup)
	RedactLineupTakers(match.AwayLineup)

	homePlayers, err := GetPlayersForMatchTeam(dbCtx, match.HomeTeamId, match.HomeLineup)
	if err != nil {
		return err
	}
	homeMap := playerListToMap(homePlayers)
	match.HomePlayers = &homeMap

	awayPlayers, err := GetPlayersForMatchTeam(dbCtx, match.AwayTeamId, match.AwayLineup)
	if err != nil {
		return err
	}
	awayMap := playerListToMap(awayPlayers)
	match.AwayPlayers = &awayMap

	if match.HomeLineup != nil && match.AwayLineup != nil && match.Events != nil {
		match.PlayerRatings = LoadPlayerRatings(dbCtx, SimulationKindFriendly, match.MatchId, match.HomeTeamId, match.AwayTeamId,
			match.HomeLineup, match.AwayLineup, match.MatchInstants, match.Events)
		match.Ratings = RatingsById(match.PlayerRatings)
		addRatedPlayers(dbCtx, match.PlayerRatings, match.HomeTeamId, *match.HomePlayers, *match.AwayPlayers)
	}

	return nil
}
