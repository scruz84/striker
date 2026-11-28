package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Club and league records (openspec/specs/records). Every figure is computed on demand from what
// the game already stores - fixtures, final tables, transfers and player stats - leaving out any
// fixture still inside its live-broadcast window, and cached for RecordsCacheTTL: records only
// move when a match ends, so a few minutes late is fine, never early.

// RecordsCacheTTL is how long a computed set of records is reused.
const RecordsCacheTTL = 10 * time.Minute

// RecordsTopPlayers is how many players the all-time scorer and appearance lists hold.
const RecordsTopPlayers = 5

// RecordMatch is one fixture that holds a record, seen from the record holder's side (a club) or
// as played (a league).
type RecordMatch struct {
	Kind         string    `json:"kind"`
	MatchId      int64     `json:"matchId"`
	MatchDate    time.Time `json:"matchDate"`
	Season       int       `json:"season"`
	HomeTeamId   int64     `json:"homeTeamId"`
	HomeTeamName string    `json:"homeTeamName"`
	AwayTeamId   int64     `json:"awayTeamId"`
	AwayTeamName string    `json:"awayTeamName"`
	HomeGoals    int       `json:"homeGoals"`
	AwayGoals    int       `json:"awayGoals"`
	Attendance   *int      `json:"attendance,omitempty"`
}

// RecordStreak is a run of consecutive league matches. Ongoing is true when it includes the club's
// latest league match.
type RecordStreak struct {
	Length  int       `json:"length"`
	From    time.Time `json:"from"`
	To      time.Time `json:"to"`
	Ongoing bool      `json:"ongoing"`
}

// RecordSeason is one of the club's finished seasons.
type RecordSeason struct {
	Season     int    `json:"season"`
	LeagueId   *int64 `json:"leagueId,omitempty"`
	LeagueName string `json:"leagueName"`
	LeagueTier int    `json:"leagueTier"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
}

// RecordTransfer is the club's biggest fee paid or received.
type RecordTransfer struct {
	PlayerId      int64     `json:"playerId"`
	PlayerName    string    `json:"playerName"`
	Fee           int64     `json:"fee"`
	OtherTeamId   *int64    `json:"otherTeamId,omitempty"`
	OtherTeamName string    `json:"otherTeamName"`
	OccurredAt    time.Time `json:"occurredAt"`
}

// RecordPlayer is a player of an all-time list: his goals and appearances.
type RecordPlayer struct {
	PlayerId     int64  `json:"playerId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	BirthCountry string `json:"birthCountry,omitempty"`
	Retired      bool   `json:"retired,omitempty"`
	Goals        int    `json:"goals"`
	Appearances  int    `json:"appearances"`
}

// ClubRecords is everything the club records card shows; a record nobody holds yet is nil.
type ClubRecords struct {
	BiggestWin            *RecordMatch    `json:"biggestWin,omitempty"`
	BiggestDefeat         *RecordMatch    `json:"biggestDefeat,omitempty"`
	HighestScoring        *RecordMatch    `json:"highestScoring,omitempty"`
	RecordAttendance      *RecordMatch    `json:"recordAttendance,omitempty"`
	LongestWinningStreak  *RecordStreak   `json:"longestWinningStreak,omitempty"`
	LongestUnbeatenStreak *RecordStreak   `json:"longestUnbeatenStreak,omitempty"`
	LongestLosingStreak   *RecordStreak   `json:"longestLosingStreak,omitempty"`
	BestFinish            *RecordSeason   `json:"bestFinish,omitempty"`
	MostPoints            *RecordSeason   `json:"mostPoints,omitempty"`
	RecordSigning         *RecordTransfer `json:"recordSigning,omitempty"`
	RecordSale            *RecordTransfer `json:"recordSale,omitempty"`
	TopScorers            []RecordPlayer  `json:"topScorers"`
	MostAppearances       []RecordPlayer  `json:"mostAppearances"`
}

// recordFixturesSql lists the official fixtures (league matches and cup ties - never friendlies)
// $1 played in, kicked off at or before $2, oldest first. A cup tie has no season of its own
// beyond its cup's.
const recordFixturesSql = "SELECT * FROM (" +
	"SELECT 'league' AS kind, m.match_id, m.match_date, m.season, m.home_team_id, COALESCE(h.name, ''), m.away_team_id, COALESCE(a.name, ''), " +
	"m.home_goals, m.away_goals, m.attendance " +
	"FROM matches m LEFT JOIN teams h ON h.team_id = m.home_team_id LEFT JOIN teams a ON a.team_id = m.away_team_id " +
	"WHERE m.played AND m.match_date <= $2 AND (m.home_team_id = $1 OR m.away_team_id = $1) " +
	"UNION ALL " +
	"SELECT 'cup', ct.tie_id, ct.match_date, c.season, ct.home_team_id, COALESCE(h.name, ''), ct.away_team_id, COALESCE(a.name, ''), " +
	"ct.home_goals, ct.away_goals, ct.attendance " +
	"FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id " +
	"LEFT JOIN teams h ON h.team_id = ct.home_team_id LEFT JOIN teams a ON a.team_id = ct.away_team_id " +
	"WHERE ct.played AND ct.match_date IS NOT NULL AND ct.match_date <= $2 AND (ct.home_team_id = $1 OR ct.away_team_id = $1)" +
	") f ORDER BY 3, 1, 2"

func scanRecordMatches(rows *sql.Rows) ([]RecordMatch, error) {
	defer rows.Close()
	out := make([]RecordMatch, 0)
	for rows.Next() {
		var m RecordMatch
		var homeId, awayId sql.NullInt64
		var attendance sql.NullInt64
		if err := rows.Scan(&m.Kind, &m.MatchId, &m.MatchDate, &m.Season, &homeId, &m.HomeTeamName, &awayId, &m.AwayTeamName,
			&m.HomeGoals, &m.AwayGoals, &attendance); err != nil {
			return nil, err
		}
		m.HomeTeamId, m.AwayTeamId = homeId.Int64, awayId.Int64
		if attendance.Valid {
			a := int(attendance.Int64)
			m.Attendance = &a
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// goalsOf returns the goals teamId scored and conceded in m.
func (m RecordMatch) goalsOf(teamId int64) (int, int) {
	if m.HomeTeamId == teamId {
		return m.HomeGoals, m.AwayGoals
	}
	return m.AwayGoals, m.HomeGoals
}

// ClubMatchRecords fills the match records of teamId from its official fixtures, oldest first
// (see recordFixturesSql). Pure so it is testable without a database:
//   - biggest win/defeat: the widest margin, then the most goals scored, then the earliest;
//   - highest scoring: the most goals in total, then the earliest;
//   - record attendance: the biggest crowd at one of its home fixtures, then the earliest;
//   - streaks count league matches only, in kickoff order: wins; matches without a defeat; defeats.
//     The first longest run wins a tie.
func ClubMatchRecords(teamId int64, fixtures []RecordMatch, records *ClubRecords) {
	pick := func(current *RecordMatch, candidate RecordMatch, better func(a, b RecordMatch) bool) *RecordMatch {
		if current == nil || better(candidate, *current) {
			c := candidate
			return &c
		}
		return current
	}
	margin := func(m RecordMatch) (int, int) {
		f, a := m.goalsOf(teamId)
		return f - a, f
	}
	for _, m := range fixtures {
		f, a := m.goalsOf(teamId)
		if f > a {
			records.BiggestWin = pick(records.BiggestWin, m, func(c, cur RecordMatch) bool {
				cm, cf := margin(c)
				um, uf := margin(cur)
				return cm > um || (cm == um && cf > uf)
			})
		}
		if f < a {
			records.BiggestDefeat = pick(records.BiggestDefeat, m, func(c, cur RecordMatch) bool {
				cm, _ := margin(c)
				um, _ := margin(cur)
				_, ca := c.goalsOf(teamId)
				_, ua := cur.goalsOf(teamId)
				return cm < um || (cm == um && ca > ua)
			})
		}
		if f+a > 0 {
			records.HighestScoring = pick(records.HighestScoring, m, func(c, cur RecordMatch) bool {
				return c.HomeGoals+c.AwayGoals > cur.HomeGoals+cur.AwayGoals
			})
		}
		if m.HomeTeamId == teamId && m.Attendance != nil && *m.Attendance > 0 {
			records.RecordAttendance = pick(records.RecordAttendance, m, func(c, cur RecordMatch) bool {
				return *c.Attendance > *cur.Attendance
			})
		}
	}

	league := make([]RecordMatch, 0, len(fixtures))
	for _, m := range fixtures {
		if m.Kind == SimulationKindLeague {
			league = append(league, m)
		}
	}
	streak := func(keeps func(f, a int) bool) *RecordStreak {
		var best *RecordStreak
		start := -1
		for i, m := range league {
			f, a := m.goalsOf(teamId)
			if keeps(f, a) {
				if start < 0 {
					start = i
				}
				// Only a strictly longer run replaces the best one, so the first longest run wins a
				// tie; the best run is ongoing when it reaches the latest league match.
				if length := i - start + 1; best == nil || length > best.Length {
					best = &RecordStreak{Length: length, From: league[start].MatchDate, To: m.MatchDate, Ongoing: i == len(league)-1}
				}
			} else {
				start = -1
			}
		}
		return best
	}
	records.LongestWinningStreak = streak(func(f, a int) bool { return f > a })
	records.LongestUnbeatenStreak = streak(func(f, a int) bool { return f >= a })
	records.LongestLosingStreak = streak(func(f, a int) bool { return f < a })
}

const clubSeasonsSql = "SELECT sh.season, sh.league_id, COALESCE(sh.league_name, l.name, ''), COALESCE(l.tier, 0), " +
	"sh.final_position, COALESCE(sh.points, 0), sh.points IS NOT NULL " +
	"FROM season_history sh LEFT JOIN leagues l ON l.league_id = sh.league_id " +
	"WHERE sh.team_id = $1 AND sh.final_position IS NOT NULL ORDER BY sh.season"

// clubSeasonRecords picks the best finish (highest tier, then best position, then earliest) and
// the most points in a season (then the earliest).
func clubSeasonRecords(dbCtx *DatabaseContext, teamId int64, records *ClubRecords) error {
	rows, err := dbCtx.Db.Query(clubSeasonsSql, teamId)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s RecordSeason
		var leagueId sql.NullInt64
		var hasPoints bool
		if err := rows.Scan(&s.Season, &leagueId, &s.LeagueName, &s.LeagueTier, &s.Position, &s.Points, &hasPoints); err != nil {
			return err
		}
		if leagueId.Valid {
			id := leagueId.Int64
			s.LeagueId = &id
		}
		if b := records.BestFinish; b == nil || (s.LeagueTier > 0 && (b.LeagueTier == 0 || s.LeagueTier < b.LeagueTier)) ||
			(s.LeagueTier == b.LeagueTier && s.Position < b.Position) {
			c := s
			records.BestFinish = &c
		}
		if hasPoints && (records.MostPoints == nil || s.Points > records.MostPoints.Points) {
			c := s
			records.MostPoints = &c
		}
	}
	return rows.Err()
}

// clubRecordTransferSql is the biggest fee of a paying transfer (see topTransferFeeKinds) to ($2 =
// 'to') or from ($2::text = 'from') club $1, with the other club and the player's name - retired or not.
var clubRecordTransferSql = "SELECT pt.player_id, " +
	"TRIM(COALESCE(p.first_name, rp.first_name, '') || ' ' || COALESCE(p.last_name, rp.last_name, '')), pt.fee, " +
	"CASE WHEN $2::text = 'to' THEN pt.from_team_id ELSE pt.to_team_id END, COALESCE(o.name, ''), pt.created_at " +
	"FROM player_transfers pt " +
	"LEFT JOIN players p ON p.player_id = pt.player_id LEFT JOIN retired_players rp ON rp.player_id = pt.player_id " +
	"LEFT JOIN teams o ON o.team_id = CASE WHEN $2::text = 'to' THEN pt.from_team_id ELSE pt.to_team_id END " +
	"WHERE pt.kind IN (" + topTransferFeeKinds + ") AND pt.fee > 0 " +
	"AND ((pt.to_team_id = $1 AND $2::text = 'to') OR (pt.from_team_id = $1 AND $2::text = 'from')) " +
	"ORDER BY pt.fee DESC, pt.created_at, pt.transfer_id LIMIT 1"

func clubRecordTransfer(dbCtx *DatabaseContext, teamId int64, direction string) (*RecordTransfer, error) {
	var t RecordTransfer
	var other sql.NullInt64
	err := dbCtx.Db.QueryRow(clubRecordTransferSql, teamId, direction).Scan(&t.PlayerId, &t.PlayerName, &t.Fee, &other, &t.OtherTeamName, &t.OccurredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if other.Valid {
		id := other.Int64
		t.OtherTeamId = &id
	}
	return &t, nil
}

// allTimePlayersSql totals the official player stats ($1 the WHERE condition's value, $2 the
// cutoff) of a club or league, ordered by goals or appearances.
func allTimePlayersSql(where string, order string) string {
	return "SELECT t.player_id, " + statsPlayerNameColumnsSql + ", t.goals, t.apps FROM (" +
		"SELECT s.player_id, COALESCE(SUM(s.goals), 0) AS goals, COUNT(*) FILTER (WHERE s.appeared) AS apps " +
		"FROM player_match_stats s WHERE " + where + " AND s.kickoff <= $2 GROUP BY s.player_id) t " +
		statsPlayerNameJoinsSql +
		"WHERE t." + order + " > 0 ORDER BY t." + order + " DESC, t.player_id LIMIT " + fmt.Sprint(RecordsTopPlayers)
}

// Official club competitions: league, domestic cup, playoff and regional cups.
var (
	clubTopScorersSql      = allTimePlayersSql("s.team_id = $1 AND s.competition IN (1, 2, 3, 4)", "goals")
	clubMostAppearancesSql = allTimePlayersSql("s.team_id = $1 AND s.competition IN (1, 2, 3, 4)", "apps")
	leagueTopScorersSql    = allTimePlayersSql("s.competition = 1 AND s.competition_id = $1", "goals")
)

func queryRecordPlayers(dbCtx *DatabaseContext, query string, id int64, cutoff time.Time) ([]RecordPlayer, error) {
	rows, err := dbCtx.Db.Query(query, id, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RecordPlayer, 0)
	for rows.Next() {
		var p RecordPlayer
		if err := rows.Scan(&p.PlayerId, &p.FirstName, &p.LastName, &p.BirthCountry, &p.Retired, &p.Goals, &p.Appearances); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// cachedRecords reads key from the object cache into dest, or computes it with compute and caches
// it. A missing or failing cache just computes.
func cachedRecords(dbCtx *DatabaseContext, key string, dest any, compute func() error) error {
	if dbCtx.ObjectCache != nil {
		if cached, err := dbCtx.ObjectCache.GetValue(key); err == nil && cached != "" {
			if err := json.Unmarshal([]byte(cached), dest); err == nil {
				return nil
			}
		}
	}
	if err := compute(); err != nil {
		return err
	}
	if dbCtx.ObjectCache != nil {
		if encoded, err := json.Marshal(dest); err == nil {
			dbCtx.ObjectCache.SetValueFor(key, string(encoded), RecordsCacheTTL)
		}
	}
	return nil
}

// GetClubRecords returns teamId's records (see ClubRecords).
func GetClubRecords(dbCtx *DatabaseContext, teamId int64) (*ClubRecords, error) {
	records := &ClubRecords{TopScorers: []RecordPlayer{}, MostAppearances: []RecordPlayer{}}
	err := cachedRecords(dbCtx, fmt.Sprintf("records:club:%d", teamId), records, func() error {
		cutoff := LiveBroadcastCutoff()
		rows, err := dbCtx.Db.Query(recordFixturesSql, teamId, cutoff)
		if err != nil {
			return err
		}
		fixtures, err := scanRecordMatches(rows)
		if err != nil {
			return err
		}
		ClubMatchRecords(teamId, fixtures, records)
		if err := clubSeasonRecords(dbCtx, teamId, records); err != nil {
			return err
		}
		if records.RecordSigning, err = clubRecordTransfer(dbCtx, teamId, "to"); err != nil {
			return err
		}
		if records.RecordSale, err = clubRecordTransfer(dbCtx, teamId, "from"); err != nil {
			return err
		}
		if records.TopScorers, err = queryRecordPlayers(dbCtx, clubTopScorersSql, teamId, cutoff); err != nil {
			return err
		}
		records.MostAppearances, err = queryRecordPlayers(dbCtx, clubMostAppearancesSql, teamId, cutoff)
		return err
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// LeagueChampion is one finished season of a league and its champion.
type LeagueChampion struct {
	Season   int    `json:"season"`
	TeamId   int64  `json:"teamId"`
	TeamName string `json:"teamName"`
	Points   *int   `json:"points,omitempty"`
}

// LeagueTitleCount is how many times a club won the league.
type LeagueTitleCount struct {
	TeamId   int64  `json:"teamId"`
	TeamName string `json:"teamName"`
	Titles   int    `json:"titles"`
	// LastSeason is the latest season it won.
	LastSeason int `json:"lastSeason"`
}

// LeagueGoldenBoot is one season's Golden Boot of the league.
type LeagueGoldenBoot struct {
	Season    int    `json:"season"`
	PlayerId  int64  `json:"playerId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Retired   bool   `json:"retired,omitempty"`
	Goals     int    `json:"goals"`
}

// LeagueRecordSeason is a club's season that holds a league record.
type LeagueRecordSeason struct {
	Season   int    `json:"season"`
	TeamId   int64  `json:"teamId"`
	TeamName string `json:"teamName"`
	Points   int    `json:"points"`
}

// LeagueRecords is everything the league history page shows.
type LeagueRecords struct {
	Champions      []LeagueChampion    `json:"champions"`
	MostTitles     []LeagueTitleCount  `json:"mostTitles"`
	GoldenBoots    []LeagueGoldenBoot  `json:"goldenBoots"`
	MostPoints     *LeagueRecordSeason `json:"mostPoints,omitempty"`
	BiggestWin     *RecordMatch        `json:"biggestWin,omitempty"`
	HighestScoring *RecordMatch        `json:"highestScoring,omitempty"`
	TopScorers     []RecordPlayer      `json:"topScorers"`
}

const leagueChampionsSql = "SELECT sh.season, sh.team_id, COALESCE(t.name, ''), sh.points " +
	"FROM season_history sh LEFT JOIN teams t ON t.team_id = sh.team_id " +
	"WHERE sh.league_id = $1 AND sh.final_position = 1 ORDER BY sh.season DESC"

const leagueMostPointsSql = "SELECT sh.season, sh.team_id, COALESCE(t.name, ''), sh.points " +
	"FROM season_history sh LEFT JOIN teams t ON t.team_id = sh.team_id " +
	"WHERE sh.league_id = $1 AND sh.points IS NOT NULL ORDER BY sh.points DESC, sh.season, sh.team_id LIMIT 1"

var leagueGoldenBootsSql = "SELECT a.season, a.player_id, COALESCE(p.first_name, rp.first_name, ''), COALESCE(p.last_name, rp.last_name, ''), " +
	"p.player_id IS NULL AND rp.player_id IS NOT NULL, a.value " +
	"FROM player_season_awards a LEFT JOIN players p ON p.player_id = a.player_id " +
	"LEFT JOIN retired_players rp ON rp.player_id = a.player_id " +
	fmt.Sprintf("WHERE a.league_id = $1 AND a.award = %d ORDER BY a.season DESC", AwardGoldenBoot)

// leagueRecordMatchSql is the league's first played match in order (a full ORDER BY list), ties
// to the earliest, among those kicked off at or before $2.
func leagueRecordMatchSql(order string) string {
	return "SELECT 'league', m.match_id, m.match_date, m.season, m.home_team_id, COALESCE(h.name, ''), m.away_team_id, COALESCE(a.name, ''), " +
		"m.home_goals, m.away_goals, m.attendance " +
		"FROM matches m LEFT JOIN teams h ON h.team_id = m.home_team_id LEFT JOIN teams a ON a.team_id = m.away_team_id " +
		"WHERE m.league_id = $1 AND m.played AND m.match_date <= $2 AND m.home_goals + m.away_goals > 0 " +
		"ORDER BY " + order + ", m.match_date, m.match_id LIMIT 1"
}

var (
	leagueBiggestWinSql     = leagueRecordMatchSql("abs(m.home_goals - m.away_goals) DESC, greatest(m.home_goals, m.away_goals) DESC")
	leagueHighestScoringSql = leagueRecordMatchSql("m.home_goals + m.away_goals DESC")
)

func queryLeagueRecordMatch(dbCtx *DatabaseContext, query string, leagueId int64, cutoff time.Time) (*RecordMatch, error) {
	rows, err := dbCtx.Db.Query(query, leagueId, cutoff)
	if err != nil {
		return nil, err
	}
	matches, err := scanRecordMatches(rows)
	if err != nil || len(matches) == 0 {
		return nil, err
	}
	if query == leagueBiggestWinSql && matches[0].HomeGoals == matches[0].AwayGoals {
		return nil, nil
	}
	return &matches[0], nil
}

// LeagueTitleCounts ranks the champions by titles, then by the latest title, then by team id.
// Pure so it is testable without a database.
func LeagueTitleCounts(champions []LeagueChampion) []LeagueTitleCount {
	byTeam := map[int64]*LeagueTitleCount{}
	for _, c := range champions {
		entry, ok := byTeam[c.TeamId]
		if !ok {
			entry = &LeagueTitleCount{TeamId: c.TeamId, TeamName: c.TeamName}
			byTeam[c.TeamId] = entry
		}
		entry.Titles++
		if c.Season > entry.LastSeason {
			entry.LastSeason = c.Season
		}
	}
	out := make([]LeagueTitleCount, 0, len(byTeam))
	for _, entry := range byTeam {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Titles != out[j].Titles {
			return out[i].Titles > out[j].Titles
		}
		if out[i].LastSeason != out[j].LastSeason {
			return out[i].LastSeason > out[j].LastSeason
		}
		return out[i].TeamId < out[j].TeamId
	})
	return out
}

// GetLeagueRecords returns leagueId's records (see LeagueRecords).
func GetLeagueRecords(dbCtx *DatabaseContext, leagueId int64) (*LeagueRecords, error) {
	records := &LeagueRecords{Champions: []LeagueChampion{}, MostTitles: []LeagueTitleCount{}, GoldenBoots: []LeagueGoldenBoot{}, TopScorers: []RecordPlayer{}}
	err := cachedRecords(dbCtx, fmt.Sprintf("records:league:%d", leagueId), records, func() error {
		cutoff := LiveBroadcastCutoff()
		rows, err := dbCtx.Db.Query(leagueChampionsSql, leagueId)
		if err != nil {
			return err
		}
		for rows.Next() {
			var c LeagueChampion
			var points sql.NullInt64
			if err := rows.Scan(&c.Season, &c.TeamId, &c.TeamName, &points); err != nil {
				rows.Close()
				return err
			}
			if points.Valid {
				p := int(points.Int64)
				c.Points = &p
			}
			records.Champions = append(records.Champions, c)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		records.MostTitles = LeagueTitleCounts(records.Champions)

		var most LeagueRecordSeason
		switch err := dbCtx.Db.QueryRow(leagueMostPointsSql, leagueId).Scan(&most.Season, &most.TeamId, &most.TeamName, &most.Points); {
		case err == nil:
			records.MostPoints = &most
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}

		boots, err := dbCtx.Db.Query(leagueGoldenBootsSql, leagueId)
		if err != nil {
			return err
		}
		for boots.Next() {
			var b LeagueGoldenBoot
			if err := boots.Scan(&b.Season, &b.PlayerId, &b.FirstName, &b.LastName, &b.Retired, &b.Goals); err != nil {
				boots.Close()
				return err
			}
			records.GoldenBoots = append(records.GoldenBoots, b)
		}
		boots.Close()
		if err := boots.Err(); err != nil {
			return err
		}

		if records.BiggestWin, err = queryLeagueRecordMatch(dbCtx, leagueBiggestWinSql, leagueId, cutoff); err != nil {
			return err
		}
		if records.HighestScoring, err = queryLeagueRecordMatch(dbCtx, leagueHighestScoringSql, leagueId, cutoff); err != nil {
			return err
		}
		records.TopScorers, err = queryRecordPlayers(dbCtx, leagueTopScorersSql, leagueId, cutoff)
		return err
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}
