package common

import (
	"database/sql"
	"errors"
	"sort"
)

// Team of the week and team of the season (openspec/specs/best-xi).

// Lines of a best XI.
const (
	BestXiLineGoalkeeper = 1
	BestXiLineDefence    = 2
	BestXiLineMidfield   = 3
	BestXiLineAttack     = 4
)

// BestXiFormation is how many players each line holds: a 4-4-2, indexed by line.
var BestXiFormation = map[int]int{BestXiLineGoalkeeper: 1, BestXiLineDefence: 4, BestXiLineMidfield: 4, BestXiLineAttack: 2}

// BestXiSeasonWeek is the week_number of a team of the season.
const BestXiSeasonWeek = 0

// BestXiLineOf maps a tactical position (a common position constant) to its line.
func BestXiLineOf(position int) int {
	switch position {
	case GOALKEEPER:
		return BestXiLineGoalkeeper
	case DEFENSE, LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT:
		return BestXiLineDefence
	case MIDFIELD, LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT:
		return BestXiLineMidfield
	default:
		return BestXiLineAttack
	}
}

// bestXiLineSql is BestXiLineOf in SQL, over player_match_stats.position.
const bestXiLineSql = "CASE WHEN s.position = 0 THEN 1 WHEN s.position IN (1, 2, 3) THEN 2 WHEN s.position IN (4, 5, 6) THEN 3 ELSE 4 END"

// BestXiCandidate is a player who can make a best XI: his line, rating (a matchday's, or the
// season's average) and what breaks a tie.
type BestXiCandidate struct {
	PlayerId int64
	TeamId   int64
	Line     int
	Rating   float64
	Minutes  int
	Goals    int
	Matches  int
}

// BestXiPick is a candidate placed in a slot: 1 the goalkeeper, 2-5 the defenders, 6-9 the
// midfielders, 10-11 the forwards.
type BestXiPick struct {
	Slot int
	BestXiCandidate
}

// SelectBestXi fills the 4-4-2 line by line with the best candidates: the higher rating, then
// more minutes, then more goals, then the lower player id. A line short of candidates leaves its
// last slots empty. Pure so it is testable without a database.
func SelectBestXi(candidates []BestXiCandidate) []BestXiPick {
	sorted := append([]BestXiCandidate(nil), candidates...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Rating != b.Rating {
			return a.Rating > b.Rating
		}
		if a.Minutes != b.Minutes {
			return a.Minutes > b.Minutes
		}
		if a.Goals != b.Goals {
			return a.Goals > b.Goals
		}
		return a.PlayerId < b.PlayerId
	})
	picks := make([]BestXiPick, 0, 11)
	slot := 1
	for line := BestXiLineGoalkeeper; line <= BestXiLineAttack; line++ {
		taken := 0
		for _, c := range sorted {
			if c.Line == line && taken < BestXiFormation[line] {
				picks = append(picks, BestXiPick{Slot: slot + taken, BestXiCandidate: c})
				taken++
			}
		}
		slot += BestXiFormation[line]
	}
	return picks
}

const getLeagueWeekOfMatchStmtSql = "SELECT league_id, season, week_number FROM matches WHERE match_id = $1"

// GetLeagueWeekOfMatch returns the league, season and matchday of a league match.
func GetLeagueWeekOfMatch(dbCtx *DatabaseContext, matchId int64) (leagueId int64, season int, week int, err error) {
	err = dbCtx.Db.QueryRow(getLeagueWeekOfMatchStmtSql, matchId).Scan(&leagueId, &season, &week)
	return
}

// isLeagueWeekCompleteStmtSql: the matchday has matches, all played, and none still waits for its
// post-match effects - which are applied when its live replay ends (or by the recovery job once
// its live window has passed).
const isLeagueWeekCompleteStmtSql = "SELECT EXISTS (SELECT 1 FROM matches WHERE league_id = $1 AND season = $2 AND week_number = $3) " +
	"AND NOT EXISTS (SELECT 1 FROM matches WHERE league_id = $1 AND season = $2 AND week_number = $3 AND NOT played) " +
	"AND NOT EXISTS (SELECT 1 FROM matches m JOIN pending_post_match_effects pe ON pe.kind = 'league' AND pe.match_id = m.match_id " +
	"WHERE m.league_id = $1 AND m.season = $2 AND m.week_number = $3)"

// IsLeagueWeekComplete reports whether every match of a league's matchday has finished its replay.
func IsLeagueWeekComplete(dbCtx *DatabaseContext, leagueId int64, season int, week int) (bool, error) {
	var complete bool
	err := dbCtx.Db.QueryRow(isLeagueWeekCompleteStmtSql, leagueId, season, week).Scan(&complete)
	return complete, err
}

const getWeekBestXiCandidatesStmtSql = "SELECT s.player_id, s.team_id, " + bestXiLineSql + ", s.rating, s.minutes, s.goals, 1 " +
	"FROM player_match_stats s JOIN matches m ON s.kind = 'league' AND m.match_id = s.match_id " +
	"WHERE m.league_id = $1 AND m.season = $2 AND m.week_number = $3 AND s.rated AND s.position IS NOT NULL"

// getSeasonBestXiCandidatesStmtSql: every player of the league season ($1, $2) with at least $3
// minutes in his rated league matches, on the line he played most often (the lower line on a
// tie), with his average rating and the team of his latest appearance.
const getSeasonBestXiCandidatesStmtSql = "SELECT s.player_id, (array_agg(s.team_id ORDER BY s.kickoff DESC, s.match_id DESC))[1], " +
	"mode() WITHIN GROUP (ORDER BY " + bestXiLineSql + "), AVG(s.rating), SUM(s.minutes), SUM(s.goals), COUNT(*) " +
	"FROM player_match_stats s " +
	"WHERE s.competition = 1 AND s.competition_id = $1 AND s.season = $2 AND s.rated AND s.position IS NOT NULL " +
	"GROUP BY s.player_id HAVING SUM(s.minutes) >= $3"

func queryBestXiCandidates(dbCtx *DatabaseContext, query string, args ...any) ([]BestXiCandidate, error) {
	rows, err := dbCtx.Db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BestXiCandidate, 0)
	for rows.Next() {
		var c BestXiCandidate
		if err := rows.Scan(&c.PlayerId, &c.TeamId, &c.Line, &c.Rating, &c.Minutes, &c.Goals, &c.Matches); err != nil {
			return nil, err
		}
		c.Rating = roundRating(c.Rating)
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetWeekBestXiCandidates lists the rated players of a league's matchday.
func GetWeekBestXiCandidates(dbCtx *DatabaseContext, leagueId int64, season int, week int) ([]BestXiCandidate, error) {
	return queryBestXiCandidates(dbCtx, getWeekBestXiCandidatesStmtSql, leagueId, season, week)
}

// GetSeasonBestXiCandidates lists the league season's players who played enough to make its team of
// the season: RatingLeaderMinutesPerMatch minutes per match of the team that played the most, the
// same bar as the rating leaders. Every match counts - the season is over.
func GetSeasonBestXiCandidates(dbCtx *DatabaseContext, leagueId int64, season int) ([]BestXiCandidate, error) {
	var mostMatches int
	if err := dbCtx.Db.QueryRow(leagueMostMatchesStmtSql, leagueId, season, nil).Scan(&mostMatches); err != nil {
		return nil, err
	}
	return queryBestXiCandidates(dbCtx, getSeasonBestXiCandidatesStmtSql, leagueId, season, RatingLeaderMinutesPerMatch*mostMatches)
}

const claimBestXiRoundStmtSql = "INSERT INTO best_xi_rounds (league_id, season, week_number) VALUES ($1, $2, $3) " +
	"ON CONFLICT (league_id, season, week_number) DO NOTHING RETURNING week_number"

const insertBestXiPlayerStmtSql = "INSERT INTO best_xi_players (league_id, season, week_number, slot, line, player_id, team_id, rating, goals, matches) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"

// SaveBestXi stores a league's best XI for a matchday (or BestXiSeasonWeek for the season) unless
// one is already stored, and reports whether this call stored it. The round row is the claim, in
// the same transaction as the players, so concurrent callers store it exactly once.
func SaveBestXi(dbCtx *DatabaseContext, leagueId int64, season int, week int, picks []BestXiPick) (bool, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var claimed int
	if err := tx.QueryRow(claimBestXiRoundStmtSql, leagueId, season, week).Scan(&claimed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	for _, p := range picks {
		if _, err := tx.Exec(insertBestXiPlayerStmtSql, leagueId, season, week, p.Slot, p.Line, p.PlayerId, p.TeamId,
			p.Rating, p.Goals, p.Matches); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}

// BestXiPlayer is one player of a stored best XI.
type BestXiPlayer struct {
	Slot         int     `json:"slot"`
	Line         int     `json:"line"`
	PlayerId     int64   `json:"playerId"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	BirthCountry string  `json:"birthCountry,omitempty"`
	Retired      bool    `json:"retired,omitempty"`
	TeamId       int64   `json:"teamId"`
	TeamName     string  `json:"teamName"`
	Rating       float64 `json:"rating"`
	Goals        int     `json:"goals"`
	Matches      int     `json:"matches"`
}

// BestXi is one stored best XI and the rounds of its season that have one. Week is
// BestXiSeasonWeek for the team of the season; Players is empty when none is stored yet.
type BestXi struct {
	Season  int            `json:"season"`
	Week    int            `json:"week"`
	Weeks   []int          `json:"weeks"`
	Players []BestXiPlayer `json:"players"`
}

const getBestXiWeeksStmtSql = "SELECT week_number FROM best_xi_rounds WHERE league_id = $1 AND season = $2 ORDER BY week_number"

const getBestXiPlayersStmtSql = "SELECT b.slot, b.line, b.player_id, " +
	"COALESCE(p.first_name, rp.first_name, ''), COALESCE(p.last_name, rp.last_name, ''), COALESCE(p.birth_country, rp.birth_country, ''), " +
	"p.player_id IS NULL AND rp.player_id IS NOT NULL, b.team_id, COALESCE(t.name, ''), b.rating, b.goals, b.matches " +
	"FROM best_xi_players b " +
	"LEFT JOIN players p ON p.player_id = b.player_id LEFT JOIN retired_players rp ON rp.player_id = b.player_id " +
	"LEFT JOIN teams t ON t.team_id = b.team_id " +
	"WHERE b.league_id = $1 AND b.season = $2 AND b.week_number = $3 ORDER BY b.slot"

// GetBestXi returns a league season's best XI for week, or - when week is negative - for its
// latest matchday that has one.
func GetBestXi(dbCtx *DatabaseContext, leagueId int64, season int, week int) (*BestXi, error) {
	xi := &BestXi{Season: season, Week: week, Weeks: []int{}, Players: []BestXiPlayer{}}
	rows, err := dbCtx.Db.Query(getBestXiWeeksStmtSql, leagueId, season)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var w int
		if err := rows.Scan(&w); err != nil {
			rows.Close()
			return nil, err
		}
		xi.Weeks = append(xi.Weeks, w)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if week < 0 {
		xi.Week = 0
		for _, w := range xi.Weeks {
			if w > xi.Week {
				xi.Week = w
			}
		}
		if xi.Week == 0 {
			return xi, nil
		}
	}
	players, err := dbCtx.Db.Query(getBestXiPlayersStmtSql, leagueId, season, xi.Week)
	if err != nil {
		return nil, err
	}
	defer players.Close()
	for players.Next() {
		var p BestXiPlayer
		if err := players.Scan(&p.Slot, &p.Line, &p.PlayerId, &p.FirstName, &p.LastName, &p.BirthCountry, &p.Retired,
			&p.TeamId, &p.TeamName, &p.Rating, &p.Goals, &p.Matches); err != nil {
			return nil, err
		}
		p.Rating = roundRating(p.Rating)
		xi.Players = append(xi.Players, p)
	}
	return xi, players.Err()
}

// PlayerBestXiCounts is how many teams of the week and of the season a player made.
type PlayerBestXiCounts struct {
	Weeks   int `json:"weeks"`
	Seasons int `json:"seasons"`
}

const countPlayerBestXiStmtSql = "SELECT COUNT(*) FILTER (WHERE week_number > 0), COUNT(*) FILTER (WHERE week_number = 0) " +
	"FROM best_xi_players WHERE player_id = $1"

// GetPlayerBestXiCounts counts a player's best XI selections.
func GetPlayerBestXiCounts(dbCtx *DatabaseContext, playerId int64) (PlayerBestXiCounts, error) {
	var c PlayerBestXiCounts
	err := dbCtx.Db.QueryRow(countPlayerBestXiStmtSql, playerId).Scan(&c.Weeks, &c.Seasons)
	return c, err
}
