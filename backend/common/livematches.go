package common

import (
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"strconv"
	"time"
)

// LiveMatchSummary is one match currently inside its live-broadcast window (see
// LiveBroadcastWindow), anywhere in the world - see GetLiveMatches. Enough to render a "N matches
// in play" figure and route straight to that match's live viewer (matchSummaryLink on the
// frontend), not the full match detail.
type LiveMatchSummary struct {
	Kind            string    `json:"kind"`
	MatchId         int64     `json:"matchId"`
	HomeTeamName    string    `json:"homeTeamName"`
	AwayTeamName    string    `json:"awayTeamName"`
	MatchDate       time.Time `json:"matchDate"`
	HomeGoals       int       `json:"homeGoals"`
	AwayGoals       int       `json:"awayGoals"`
	Country         string    `json:"country,omitempty"`
	CompetitionName string    `json:"competitionName,omitempty"`
	// Tier is the league's division tier (1 = the top flight) - only set for a league match
	// (Kind == MatchKindLeague). A cup tie or friendly has no division to rank, so it's left zero.
	Tier int `json:"tier,omitempty"`
	// Minute is only set on the one match GetLiveMatchesOverview suggests (see
	// enrichSuggestedLiveMatch) - populating it for every live match in the world on every poll
	// would mean one extra query per match for a number nothing else here uses.
	Minute int `json:"minute,omitempty"`
}

const getLiveLeagueMatchesStmtSql = "SELECT m.match_id, ht.name, at.name, m.match_date, m.home_goals, m.away_goals, l.country, l.name, l.tier " +
	"FROM matches m " +
	"JOIN teams ht ON ht.team_id = m.home_team_id " +
	"JOIN teams at ON at.team_id = m.away_team_id " +
	"JOIN leagues l ON l.league_id = m.league_id " +
	"WHERE m.played = true AND m.match_date <= now() AND m.match_date > $1"

func getLiveLeagueMatches(dbCtx *DatabaseContext, cutoff time.Time) ([]LiveMatchSummary, error) {
	rows, err := dbCtx.Db.Query(getLiveLeagueMatchesStmtSql, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]LiveMatchSummary, 0)
	for rows.Next() {
		var s LiveMatchSummary
		if err := rows.Scan(&s.MatchId, &s.HomeTeamName, &s.AwayTeamName, &s.MatchDate, &s.HomeGoals, &s.AwayGoals, &s.Country, &s.CompetitionName, &s.Tier); err != nil {
			return nil, err
		}
		s.Kind = MatchKindLeague
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// cup_ties.match_date is nullable (an unscheduled bye/future round) - the played = true guard
// below already excludes any row where it isn't set, so a plain scan into time.Time is safe here.
const getLiveCupTiesStmtSql = "SELECT ct.tie_id, ht.name, at.name, ct.match_date, ct.home_goals, ct.away_goals, c.country, c.name " +
	"FROM cup_ties ct " +
	"JOIN teams ht ON ht.team_id = ct.home_team_id " +
	"JOIN teams at ON at.team_id = ct.away_team_id " +
	"JOIN cups c ON c.cup_id = ct.cup_id " +
	"WHERE ct.played = true AND ct.match_date <= now() AND ct.match_date > $1"

func getLiveCupTies(dbCtx *DatabaseContext, cutoff time.Time) ([]LiveMatchSummary, error) {
	rows, err := dbCtx.Db.Query(getLiveCupTiesStmtSql, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]LiveMatchSummary, 0)
	for rows.Next() {
		var s LiveMatchSummary
		var country *string
		if err := rows.Scan(&s.MatchId, &s.HomeTeamName, &s.AwayTeamName, &s.MatchDate, &s.HomeGoals, &s.AwayGoals, &country, &s.CompetitionName); err != nil {
			return nil, err
		}
		s.Kind = MatchKindCup
		if country != nil {
			s.Country = *country
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

const getLiveFriendliesStmtSql = "SELECT fm.match_id, ht.name, at.name, fm.match_date, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm " +
	"JOIN teams ht ON ht.team_id = fm.home_team_id " +
	"JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE fm.played = true AND fm.match_date <= now() AND fm.match_date > $1"

func getLiveFriendlies(dbCtx *DatabaseContext, cutoff time.Time) ([]LiveMatchSummary, error) {
	rows, err := dbCtx.Db.Query(getLiveFriendliesStmtSql, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]LiveMatchSummary, 0)
	for rows.Next() {
		var s LiveMatchSummary
		if err := rows.Scan(&s.MatchId, &s.HomeTeamName, &s.AwayTeamName, &s.MatchDate, &s.HomeGoals, &s.AwayGoals); err != nil {
			return nil, err
		}
		s.Kind = MatchKindFriendly
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// LiveMatchesOverview is the dashboard's "matches in play right now" summary - how many are live
// across the whole world, plus one suggestion (prioritized toward the top divisions - see
// PickRandomLiveMatch) to jump straight into watching. Suggested is nil when Count is 0.
type LiveMatchesOverview struct {
	Count     int               `json:"count"`
	Suggested *LiveMatchSummary `json:"suggested,omitempty"`
}

// GetLiveMatchesOverview is GetLiveMatches plus PickRandomLiveMatch bundled into the one response
// shape the dashboard actually needs.
func GetLiveMatchesOverview(dbCtx *DatabaseContext) (LiveMatchesOverview, error) {
	matches, err := GetLiveMatches(dbCtx)
	if err != nil {
		return LiveMatchesOverview{}, err
	}

	overview := LiveMatchesOverview{Count: len(matches)}
	if suggested, ok := PickRandomLiveMatch(matches); ok {
		if err := enrichSuggestedLiveMatch(dbCtx, &suggested); err != nil {
			return LiveMatchesOverview{}, err
		}
		overview.Suggested = &suggested
	}
	return overview, nil
}

const getMatchScoreAsOfMinuteStmtSql = "SELECT home_goals, away_goals FROM match_events " +
	"WHERE match_id = $1 AND minute <= $2 ORDER BY minute DESC, event_id DESC LIMIT 1"
const getCupTieScoreAsOfMinuteStmtSql = "SELECT home_goals, away_goals FROM cup_tie_events " +
	"WHERE tie_id = $1 AND minute <= $2 ORDER BY minute DESC, event_id DESC LIMIT 1"
const getFriendlyScoreAsOfMinuteStmtSql = "SELECT home_goals, away_goals FROM friendly_events " +
	"WHERE match_id = $1 AND minute <= $2 ORDER BY minute DESC, event_id DESC LIMIT 1"

// liveMatchChannel returns the LiveMatchHub channel match's own broadcast actually publishes to
// (see LeagueMatchChannel/CupTieChannel/FriendlyMatchChannel) - the same one its own live-viewer
// page (handlers.serveLiveMatch) subscribes to, so enrichSuggestedLiveMatch can read that exact
// broadcast's own last message rather than a different estimate of it.
func liveMatchChannel(match LiveMatchSummary) string {
	switch match.Kind {
	case MatchKindCup:
		return CupTieChannel(match.MatchId)
	case MatchKindFriendly:
		return FriendlyMatchChannel(match.MatchId)
	default:
		return LeagueMatchChannel(match.MatchId)
	}
}

// minuteFromLiveMessage extracts the match-minute msg actually represents, if any - "lineups"
// (published once, before the broadcast's first real minute) has none, so callers fall back to
// an estimate for that one brief case; every other type maps onto a concrete minute the same way
// engine.BroadcastMatchLive paces them out.
func minuteFromLiveMessage(msg LiveMatchMessage) (int, bool) {
	switch msg.Type {
	case "instant":
		if msg.Instant != nil {
			return msg.Instant.Minute, true
		}
	case "event":
		if msg.Event != nil {
			return msg.Event.Minute, true
		}
	case "halftime":
		return liveMatchFirstHalfMinutes, true
	case "complete":
		return liveMatchFirstHalfMinutes + liveMatchSecondHalfMinutes, true
	}
	return 0, false
}

// enrichSuggestedLiveMatch fills in Minute and overwrites HomeGoals/AwayGoals (freshly queried
// off matches/cup_ties/friendly_matches, they're already the final result, which would spoil a
// match still supposedly in progress) with the match's actual current state.
//
// Reads the real broadcast's own last published message (see LiveMatchHub.LastMessage) rather
// than estimating from elapsed time, so this dashboard-wide suggestion always agrees with
// whatever that match's own live-viewer page (handlers.serveLiveMatch) would show right now - a
// time-based estimate (CurrentMatchMinute) drifts from the real broadcast whenever it stalls,
// lags behind a scheduler backlog, or has already gone stale and been closed out (see
// engine.RecoverStaleLiveBroadcasts), all of which the broadcast's own messages already reflect
// without needing to be reasoned about separately here. Falls back to that same estimate, plus
// the score-as-of-that-minute lookup this always used, only for the brief window before the
// broadcast's first message has actually been published yet (match_date just passed) or (far
// less likely) if Redis itself is unreachable.
func enrichSuggestedLiveMatch(dbCtx *DatabaseContext, match *LiveMatchSummary) error {
	minute, homeGoals, awayGoals, _, err := liveScoreAsOfNow(dbCtx, match.Kind, match.MatchId, match.MatchDate)
	if err != nil {
		return err
	}
	match.Minute, match.HomeGoals, match.AwayGoals = minute, homeGoals, awayGoals
	return nil
}

// Live broadcast statuses reported by liveStateAsOfNow (and by the division live scores).
const (
	LiveStatusPending  = "pending"
	LiveStatusLive     = "live"
	LiveStatusHalftime = "halftime"
	LiveStatusFinished = "finished"
)

// LiveState is what a viewer of one match's live broadcast is seeing right now.
type LiveState struct {
	Status    string
	Minute    int
	HomeGoals int
	AwayGoals int
}

// liveScoreAsOfNow is the minute and score a viewer of kind/id's live broadcast (kind is one of
// MatchKindLeague/MatchKindCup/MatchKindFriendly) is seeing right now, and whether that broadcast
// has already published its "complete" message - see enrichSuggestedLiveMatch for the sources it
// prefers. A match simulated ahead of its kickoff has not started broadcasting yet: minute 0, 0-0.
func liveScoreAsOfNow(dbCtx *DatabaseContext, kind string, id int64, kickoff time.Time) (minute, homeGoals, awayGoals int, complete bool, err error) {
	state, err := liveStateAsOfNow(dbCtx, kind, id, kickoff, true)
	if err != nil {
		return 0, 0, 0, false, err
	}
	return state.Minute, state.HomeGoals, state.AwayGoals, state.Status == LiveStatusFinished, nil
}

// liveStateAsOfNow is liveScoreAsOfNow plus an explicit status. A match that is not simulated yet
// (played false) or whose kickoff has not arrived is pending (minute 0, 0-0) without touching
// Redis; otherwise the broadcast's own last message decides (halftime, finished, or live), and
// only when it has none does the time-based estimate plus the score-as-of-that-minute apply.
func liveStateAsOfNow(dbCtx *DatabaseContext, kind string, id int64, kickoff time.Time, played bool) (LiveState, error) {
	if !played || time.Now().Before(kickoff) {
		return LiveState{Status: LiveStatusPending}, nil
	}
	if dbCtx.LiveHub != nil {
		if msg, ok := dbCtx.LiveHub.LastMessage(liveMatchChannel(LiveMatchSummary{Kind: kind, MatchId: id})); ok {
			if minute, ok := minuteFromLiveMessage(msg); ok {
				status := LiveStatusLive
				switch msg.Type {
				case "halftime":
					status = LiveStatusHalftime
				case "complete":
					status = LiveStatusFinished
				}
				return LiveState{Status: status, Minute: minute, HomeGoals: msg.HomeGoals, AwayGoals: msg.AwayGoals}, nil
			}
		}
	}

	state := LiveState{Status: LiveStatusLive, Minute: CurrentMatchMinute(kickoff)}

	var stmtSql string
	switch kind {
	case MatchKindLeague:
		stmtSql = getMatchScoreAsOfMinuteStmtSql
	case MatchKindCup:
		stmtSql = getCupTieScoreAsOfMinuteStmtSql
	case MatchKindFriendly:
		stmtSql = getFriendlyScoreAsOfMinuteStmtSql
	default:
		return state, nil
	}

	err := dbCtx.Db.QueryRow(stmtSql, id, state.Minute).Scan(&state.HomeGoals, &state.AwayGoals)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return LiveState{}, err
	}
	return state, nil
}

// liveEarlySimulationWindow is how far ahead of its kickoff a league match can already be
// simulated (the engine simulates up to 15 minutes early), mirroring the frontend's
// maxEarlySimulationMs. Such a match is inside the division live scores as "pending".
const liveEarlySimulationWindow = 15 * time.Minute

// leagueLiveScoresCacheTTL bounds how stale a division's live scores can be, whatever the number
// of viewers polling them.
const leagueLiveScoresCacheTTL = 5 * time.Second

// getLeagueLiveMatchesStmtSql deliberately selects no goal column: every goal in a division live
// score comes from the match's live broadcast (liveStateAsOfNow), never from the stored final
// result, so this query cannot spoil a match that is still being played out.
const getLeagueLiveMatchesStmtSql = "SELECT m.match_id, m.week_number, m.match_date, m.played, " +
	"m.home_team_id, ht.name, m.away_team_id, at.name " +
	"FROM matches m " +
	"JOIN teams ht ON ht.team_id = m.home_team_id " +
	"JOIN teams at ON at.team_id = m.away_team_id " +
	"WHERE m.league_id = $1 AND m.match_date > $2 AND m.match_date <= $3 " +
	"ORDER BY m.match_date, m.match_id"

// LeagueLiveScore is one league match inside its live window, as its broadcast shows it right now.
type LeagueLiveScore struct {
	MatchId    int64     `json:"matchId"`
	WeekNumber int       `json:"weekNumber"`
	MatchDate  time.Time `json:"matchDate"`
	Home       int64     `json:"home"`
	HomeName   string    `json:"homeName"`
	Away       int64     `json:"away"`
	AwayName   string    `json:"awayName"`
	Status     string    `json:"status"`
	Minute     int       `json:"minute"`
	HomeGoals  int       `json:"homeGoals"`
	AwayGoals  int       `json:"awayGoals"`
}

func leagueLiveScoresCacheKey(leagueId int64) string {
	return "league_live_scores:" + strconv.FormatInt(leagueId, 10)
}

// GetLeagueLiveScores lists the league matches of leagueId whose kickoff is between
// LiveBroadcastWindow ago and liveEarlySimulationWindow from now, ordered by kickoff then match id,
// each with the broadcast's own status, minute and score (see liveStateAsOfNow). Never the stored
// result. Always a non-nil slice. The response is cached for a few seconds per league, so its cost
// does not grow with the number of viewers; a cache error never fails the request.
func GetLeagueLiveScores(dbCtx *DatabaseContext, leagueId int64) ([]LeagueLiveScore, error) {
	cacheKey := leagueLiveScoresCacheKey(leagueId)
	if dbCtx.ObjectCache != nil {
		if cached, err := dbCtx.ObjectCache.GetValue(cacheKey); err == nil && cached != "" {
			var scores []LeagueLiveScore
			if json.Unmarshal([]byte(cached), &scores) == nil && scores != nil {
				return scores, nil
			}
		}
	}

	rows, err := dbCtx.Db.Query(getLeagueLiveMatchesStmtSql, leagueId, LiveBroadcastCutoff(), time.Now().Add(liveEarlySimulationWindow))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type liveMatchRow struct {
		score  LeagueLiveScore
		played bool
	}
	matches := make([]liveMatchRow, 0)
	for rows.Next() {
		var m liveMatchRow
		if err := rows.Scan(&m.score.MatchId, &m.score.WeekNumber, &m.score.MatchDate, &m.played,
			&m.score.Home, &m.score.HomeName, &m.score.Away, &m.score.AwayName); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	scores := make([]LeagueLiveScore, 0, len(matches))
	for _, m := range matches {
		state, err := liveStateAsOfNow(dbCtx, MatchKindLeague, m.score.MatchId, m.score.MatchDate, m.played)
		if err != nil {
			return nil, err
		}
		m.score.Status, m.score.Minute, m.score.HomeGoals, m.score.AwayGoals = state.Status, state.Minute, state.HomeGoals, state.AwayGoals
		scores = append(scores, m.score)
	}

	if dbCtx.ObjectCache != nil {
		if encoded, err := json.Marshal(scores); err == nil {
			dbCtx.ObjectCache.SetValueFor(cacheKey, string(encoded), leagueLiveScoresCacheTTL)
		}
	}
	return scores, nil
}

// GetLiveMatches lists every league match, cup tie, and friendly currently inside its
// live-broadcast window (see LiveBroadcastWindow), across every country and competition - the
// pool behind the dashboard's "N matches in play right now" figure and its random live-match
// suggestion (see PickRandomLiveMatch).
func GetLiveMatches(dbCtx *DatabaseContext) ([]LiveMatchSummary, error) {
	cutoff := LiveBroadcastCutoff()
	matches := make([]LiveMatchSummary, 0)

	leagueMatches, err := getLiveLeagueMatches(dbCtx, cutoff)
	if err != nil {
		return nil, err
	}
	matches = append(matches, leagueMatches...)

	cupTies, err := getLiveCupTies(dbCtx, cutoff)
	if err != nil {
		return nil, err
	}
	matches = append(matches, cupTies...)

	friendlies, err := getLiveFriendlies(dbCtx, cutoff)
	if err != nil {
		return nil, err
	}
	matches = append(matches, friendlies...)

	return matches, nil
}

// livePriority ranks a live match for PickRandomLiveMatch - lower is better. A league match ranks
// by its own division tier (1 = the top flight, the most prestigious), so the random pick below
// naturally favors the strongest available competition; a cup tie or friendly, which have no
// division to rank, always sorts behind every league match.
func livePriority(m LiveMatchSummary) int {
	if m.Kind == MatchKindLeague {
		return m.Tier
	}
	return math.MaxInt32
}

// PickRandomLiveMatch returns one uniformly random pick from whichever live matches share the
// best (numerically lowest) livePriority, so a live top-flight match anywhere in the world is
// always offered ahead of a lower division's, a cup tie, or a friendly - without ever starving
// those entirely on the (common) occasions no top-flight match happens to be live right now. The
// second return value is false when matches is empty.
func PickRandomLiveMatch(matches []LiveMatchSummary) (LiveMatchSummary, bool) {
	if len(matches) == 0 {
		return LiveMatchSummary{}, false
	}

	best := livePriority(matches[0])
	for _, m := range matches[1:] {
		if p := livePriority(m); p < best {
			best = p
		}
	}

	pool := make([]LiveMatchSummary, 0, len(matches))
	for _, m := range matches {
		if livePriority(m) == best {
			pool = append(pool, m)
		}
	}

	return pool[rand.Intn(len(pool))], true
}
