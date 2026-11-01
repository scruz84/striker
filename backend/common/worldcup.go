package common

import (
	"database/sql"
	"errors"
	"math/rand"
	"sort"
	"time"
)

// WorldCupGroupSize is a group's target size - a real group has exactly this many teams
// whenever the entrant count divides evenly; see AssignWorldCupGroups for how a remainder is
// spread out instead of left as an undersized group.
const WorldCupGroupSize = 4

// MinWorldCupCountries is the fewest active countries needed to start a World Cup - two full
// groups of four, the smallest group stage that still means anything (see
// StartWorldCupForNewSeason, which checks this before creating anything).
const MinWorldCupCountries = 2 * WorldCupGroupSize

// WorldCup is one edition's top-level record - see init72_world_cup.sql. Stage moves from
// "groups" to "knockout" exactly once, when every group's fixtures are played (see
// engine.AdvanceWorldCups); KnockoutCupId is nil until then, after which the knockout bracket is
// an ordinary Cup (region "world") and this struct stops needing to know anything about ties -
// the existing cup-detail endpoints already serve that.
type WorldCup struct {
	WorldCupId       int64  `json:"worldCupId"`
	Season           int    `json:"season"`
	Name             string `json:"name"`
	Stage            string `json:"stage"`
	Completed        bool   `json:"completed"`
	ChampionTeamId   *int64 `json:"championTeamId"`
	ChampionTeamName string `json:"championTeamName"`
	KnockoutCupId    *int64 `json:"knockoutCupId"`
}

// WorldCupGroupStanding is one team's running record in a group - see
// world_cup_group_teams/UpdateWorldCupStandingsAfterMatch.
type WorldCupGroupStanding struct {
	TeamId       int64  `json:"teamId"`
	TeamName     string `json:"teamName"`
	Played       int    `json:"played"`
	Won          int    `json:"won"`
	Drawn        int    `json:"drawn"`
	Lost         int    `json:"lost"`
	GoalsFor     int    `json:"goalsFor"`
	GoalsAgainst int    `json:"goalsAgainst"`
}

func (s WorldCupGroupStanding) Points() int         { return s.Won*3 + s.Drawn }
func (s WorldCupGroupStanding) GoalDifference() int { return s.GoalsFor - s.GoalsAgainst }

// WorldCupGroupMatch is one group fixture, enriched from its underlying friendly_matches row -
// see GetWorldCupGroupMatches. Broadcast-safe like NationalMatch: Status is "scheduled" (no
// score), "live" (Minute and the score as of that minute) or "finished" (the final score).
type WorldCupGroupMatch struct {
	MatchId      int64     `json:"matchId"`
	Round        int       `json:"round"`
	HomeTeamId   int64     `json:"homeTeamId"`
	HomeTeamName string    `json:"homeTeamName"`
	AwayTeamId   int64     `json:"awayTeamId"`
	AwayTeamName string    `json:"awayTeamName"`
	MatchDate    time.Time `json:"matchDate"`
	Status       string    `json:"status"`
	Minute       *int      `json:"minute,omitempty"`
	HomeGoals    *int      `json:"homeGoals,omitempty"`
	AwayGoals    *int      `json:"awayGoals,omitempty"`
}

// worldCupGroupFixture is a group match as stored, final score included - never sent to a
// client as is (see GetWorldCupGroups).
type worldCupGroupFixture struct {
	match     WorldCupGroupMatch
	played    bool
	homeGoals int
	awayGoals int
}

// WorldCupGroup is one group's name, ranked standings, and fixtures - see GetWorldCupGroups.
type WorldCupGroup struct {
	GroupId   int64                   `json:"groupId"`
	GroupName string                  `json:"groupName"`
	Standings []WorldCupGroupStanding `json:"standings"`
	Matches   []WorldCupGroupMatch    `json:"matches"`
}

// AssignWorldCupGroups splits countries into balanced groups - every group's size differs from
// every other's by at most one, since countries are simply dealt round-robin across numGroups
// slots after a shuffle. numGroups is countries' count divided by WorldCupGroupSize, floored but
// never below 2 (the caller only calls this once MinWorldCupCountries is already met, so that
// floor is always safe - it only ever raises group size above 4, never creates an undersized
// group). Deterministic given r's seed, for testing.
func AssignWorldCupGroups(countries []string, r *rand.Rand) [][]string {
	numGroups := len(countries) / WorldCupGroupSize
	if numGroups < 2 {
		numGroups = 2
	}

	shuffled := append([]string(nil), countries...)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	groups := make([][]string, numGroups)
	for i, country := range shuffled {
		groups[i%numGroups] = append(groups[i%numGroups], country)
	}
	return groups
}

// BuildRoundRobinRounds returns a single round-robin schedule for n participants (indices
// 0..n-1), using the standard "circle method": every round pairs up all of them (one sits out
// per round, as a bye, if n is odd), and every pair of participants plays each other exactly
// once across the whole schedule. Used to schedule a World Cup group's fixtures - see
// StartWorldCupForNewSeason. Indices are the group's teams in whatever order the caller passed
// them in; the ordering itself carries no weight, since AssignWorldCupGroups already randomized
// group membership.
func BuildRoundRobinRounds(n int) [][][2]int {
	if n < 2 {
		return nil
	}
	const bye = -1
	participants := make([]int, n)
	for i := range participants {
		participants[i] = i
	}
	if len(participants)%2 != 0 {
		participants = append(participants, bye)
	}
	count := len(participants)

	rounds := make([][][2]int, 0, count-1)
	for round := 0; round < count-1; round++ {
		pairs := make([][2]int, 0, count/2)
		for i := 0; i < count/2; i++ {
			a, b := participants[i], participants[count-1-i]
			if a != bye && b != bye {
				pairs = append(pairs, [2]int{a, b})
			}
		}
		rounds = append(rounds, pairs)

		next := make([]int, count)
		next[0] = participants[0]
		next[1] = participants[count-1]
		copy(next[2:], participants[1:count-1])
		participants = next
	}
	return rounds
}

// RankGroupStandings sorts a group's standings the usual way: points, then goal difference, then
// goals scored. A real tournament settles a further tie with head-to-head results or a draw;
// this game settles it alphabetically by team name instead, favoring a fully deterministic order
// (and therefore a deterministic knockout seeding, see GetWorldCupGroupQualifiers) over a
// perfectly realistic one.
func RankGroupStandings(standings []WorldCupGroupStanding) []WorldCupGroupStanding {
	ranked := append([]WorldCupGroupStanding(nil), standings...)
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Points() != b.Points() {
			return a.Points() > b.Points()
		}
		if a.GoalDifference() != b.GoalDifference() {
			return a.GoalDifference() > b.GoalDifference()
		}
		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}
		return a.TeamName < b.TeamName
	})
	return ranked
}

const insertWorldCupStmtSql = "INSERT INTO world_cups (season, name) VALUES ($1, $2) RETURNING world_cup_id"
const insertWorldCupGroupStmtSql = "INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, $2) RETURNING group_id"
const insertWorldCupGroupTeamStmtSql = "INSERT INTO world_cup_group_teams (group_id, team_id) VALUES ($1, $2)"
const insertWorldCupGroupMatchStmtSql = "INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, $2, $3)"

// CreateWorldCup starts a new edition's top-level record, in its initial "groups" stage.
func CreateWorldCup(dbCtx *DatabaseContext, season int, name string) (int64, error) {
	var worldCupId int64
	err := dbCtx.Db.QueryRow(insertWorldCupStmtSql, season, name).Scan(&worldCupId)
	return worldCupId, err
}

// CreateWorldCupGroup adds one named group to worldCupId.
func CreateWorldCupGroup(dbCtx *DatabaseContext, worldCupId int64, groupName string) (int64, error) {
	var groupId int64
	err := dbCtx.Db.QueryRow(insertWorldCupGroupStmtSql, worldCupId, groupName).Scan(&groupId)
	return groupId, err
}

// AddWorldCupGroupTeam enters teamId into groupId with a zeroed record.
func AddWorldCupGroupTeam(dbCtx *DatabaseContext, groupId int64, teamId int64) error {
	_, err := dbCtx.Db.Exec(insertWorldCupGroupTeamStmtSql, groupId, teamId)
	return err
}

// LinkWorldCupGroupMatch records that matchId (a friendly_matches row - see
// common.ScheduleNationalTeamFriendlies' sibling, StartWorldCupForNewSeason) is groupId's fixture
// for round.
func LinkWorldCupGroupMatch(dbCtx *DatabaseContext, groupId int64, round int, matchId int64) error {
	_, err := dbCtx.Db.Exec(insertWorldCupGroupMatchStmtSql, groupId, round, matchId)
	return err
}

const getActiveWorldCupStmtSql = "SELECT world_cup_id, season, name, stage, completed, champion_team_id, knockout_cup_id " +
	"FROM world_cups WHERE completed = false ORDER BY world_cup_id DESC LIMIT 1"

const getLatestWorldCupStmtSql = "SELECT wc.world_cup_id, wc.season, wc.name, wc.stage, wc.completed, wc.champion_team_id, wc.knockout_cup_id, " +
	"COALESCE(t.name, '') " +
	"FROM world_cups wc LEFT JOIN teams t ON t.team_id = wc.champion_team_id " +
	"ORDER BY wc.world_cup_id DESC LIMIT 1"

func scanWorldCup(row rowScanner) (*WorldCup, error) {
	var wc WorldCup
	err := row.Scan(&wc.WorldCupId, &wc.Season, &wc.Name, &wc.Stage, &wc.Completed, &wc.ChampionTeamId, &wc.KnockoutCupId, &wc.ChampionTeamName)
	if err != nil {
		return nil, err
	}
	return &wc, nil
}

// GetLatestWorldCup returns the most recently created World Cup, completed or not, or
// sql.ErrNoRows if none has ever been started.
func GetLatestWorldCup(dbCtx *DatabaseContext) (*WorldCup, error) {
	return scanWorldCup(dbCtx.Db.QueryRow(getLatestWorldCupStmtSql))
}

// GetActiveWorldCupIds lists every not-yet-completed World Cup - in practice at most one, since
// StartWorldCupForNewSeason refuses to start a new one while another is still active, but
// AdvanceWorldCups iterates it the same defensive way AdvanceCups iterates GetActiveCups.
const getActiveWorldCupIdsStmtSql = "SELECT world_cup_id FROM world_cups WHERE completed = false"

func GetActiveWorldCupIds(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(getActiveWorldCupIdsStmtSql)
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

// HasActiveWorldCup reports whether a World Cup is currently running (either stage) - used both
// to stop StartWorldCupForNewSeason double-booking one, and to pause the weekly national-team
// friendly cadence while one is (see ScheduleNationalTeamFriendlies) rather than double-booking a
// country's national team in the same week.
func HasActiveWorldCup(dbCtx *DatabaseContext) (bool, error) {
	ids, err := GetActiveWorldCupIds(dbCtx)
	if err != nil {
		return false, err
	}
	return len(ids) > 0, nil
}

const getWorldCupByIdStmtSql = "SELECT wc.world_cup_id, wc.season, wc.name, wc.stage, wc.completed, wc.champion_team_id, wc.knockout_cup_id, " +
	"COALESCE(t.name, '') " +
	"FROM world_cups wc LEFT JOIN teams t ON t.team_id = wc.champion_team_id " +
	"WHERE wc.world_cup_id = $1"

func GetWorldCupById(dbCtx *DatabaseContext, worldCupId int64) (*WorldCup, error) {
	return scanWorldCup(dbCtx.Db.QueryRow(getWorldCupByIdStmtSql, worldCupId))
}

const getWorldCupGroupsStmtSql = "SELECT group_id, group_name FROM world_cup_groups WHERE world_cup_id = $1 ORDER BY group_name"

const getWorldCupGroupStandingsStmtSql = "SELECT wcgt.team_id, t.name, wcgt.played, wcgt.won, wcgt.drawn, wcgt.lost, wcgt.goals_for, wcgt.goals_against " +
	"FROM world_cup_group_teams wcgt JOIN teams t ON t.team_id = wcgt.team_id WHERE wcgt.group_id = $1"

const getWorldCupGroupMatchesStmtSql = "SELECT wcgm.round, fm.match_id, fm.home_team_id, th.name, fm.away_team_id, ta.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM world_cup_group_matches wcgm " +
	"JOIN friendly_matches fm ON fm.match_id = wcgm.match_id " +
	"JOIN teams th ON th.team_id = fm.home_team_id JOIN teams ta ON ta.team_id = fm.away_team_id " +
	"WHERE wcgm.group_id = $1 ORDER BY wcgm.round, fm.match_date, fm.match_id"

// GetWorldCupGroups returns every group of worldCupId, each with its standings (ranked, see
// RankGroupStandings) and fixtures - the full payload a World Cup page's group-stage view needs.
// Both are broadcast-safe: standings count only matches whose broadcast has finished (see
// deriveWorldCupGroupStandings) and a match still being broadcast shows its live score (see
// applyNationalMatchStatus). The stored running totals (world_cup_group_teams) are updated as
// soon as a match is simulated, up to two hours before its broadcast ends, so they are only read
// for the knockout draw (GetWorldCupGroupQualifiers), which happens after every match is played.
func GetWorldCupGroups(dbCtx *DatabaseContext, worldCupId int64) ([]WorldCupGroup, error) {
	rows, err := dbCtx.Db.Query(getWorldCupGroupsStmtSql, worldCupId)
	if err != nil {
		return nil, err
	}
	type groupRow struct {
		id   int64
		name string
	}
	groupRows := make([]groupRow, 0)
	for rows.Next() {
		var gr groupRow
		if err := rows.Scan(&gr.id, &gr.name); err != nil {
			rows.Close()
			return nil, err
		}
		groupRows = append(groupRows, gr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	cutoff := LiveBroadcastCutoff()
	lookup := func(kind string, id int64, kickoff time.Time) (int, int, int, bool, error) {
		return liveScoreAsOfNow(dbCtx, kind, id, kickoff)
	}
	groups := make([]WorldCupGroup, 0, len(groupRows))
	for _, gr := range groupRows {
		stored, err := getWorldCupGroupStandings(dbCtx, gr.id)
		if err != nil {
			return nil, err
		}
		fixtures, err := getWorldCupGroupFixtures(dbCtx, gr.id)
		if err != nil {
			return nil, err
		}
		matches := make([]WorldCupGroupMatch, 0, len(fixtures))
		for _, f := range fixtures {
			nm := NationalMatch{Kind: NationalMatchKindWorldCupGroup, Id: f.match.MatchId, Kickoff: f.match.MatchDate}
			if err := applyNationalMatchStatus(&nm, f.played, f.homeGoals, f.awayGoals, cutoff, lookup); err != nil {
				return nil, err
			}
			m := f.match
			m.Status, m.Minute, m.HomeGoals, m.AwayGoals = nm.Status, nm.Minute, nm.HomeGoals, nm.AwayGoals
			matches = append(matches, m)
		}
		groups = append(groups, WorldCupGroup{
			GroupId:   gr.id,
			GroupName: gr.name,
			Standings: deriveWorldCupGroupStandings(stored, fixtures, cutoff),
			Matches:   matches,
		})
	}
	return groups, nil
}

// deriveWorldCupGroupStandings rebuilds a group's table from its fixtures, counting only those
// played with a kickoff at or before cutoff (broadcast finished), then ranks it. teams supplies
// the group's members and names; their stored totals are ignored.
func deriveWorldCupGroupStandings(teams []WorldCupGroupStanding, fixtures []worldCupGroupFixture, cutoff time.Time) []WorldCupGroupStanding {
	byTeam := make(map[int64]*WorldCupGroupStanding, len(teams))
	standings := make([]WorldCupGroupStanding, len(teams))
	for i, team := range teams {
		standings[i] = WorldCupGroupStanding{TeamId: team.TeamId, TeamName: team.TeamName}
		byTeam[team.TeamId] = &standings[i]
	}
	record := func(s *WorldCupGroupStanding, goalsFor, goalsAgainst int) {
		if s == nil {
			return
		}
		s.Played++
		s.GoalsFor += goalsFor
		s.GoalsAgainst += goalsAgainst
		switch {
		case goalsFor > goalsAgainst:
			s.Won++
		case goalsFor == goalsAgainst:
			s.Drawn++
		default:
			s.Lost++
		}
	}
	for _, f := range fixtures {
		if !f.played || f.match.MatchDate.After(cutoff) {
			continue
		}
		record(byTeam[f.match.HomeTeamId], f.homeGoals, f.awayGoals)
		record(byTeam[f.match.AwayTeamId], f.awayGoals, f.homeGoals)
	}
	return RankGroupStandings(standings)
}

func getWorldCupGroupStandings(dbCtx *DatabaseContext, groupId int64) ([]WorldCupGroupStanding, error) {
	rows, err := dbCtx.Db.Query(getWorldCupGroupStandingsStmtSql, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	standings := make([]WorldCupGroupStanding, 0)
	for rows.Next() {
		var s WorldCupGroupStanding
		if err := rows.Scan(&s.TeamId, &s.TeamName, &s.Played, &s.Won, &s.Drawn, &s.Lost, &s.GoalsFor, &s.GoalsAgainst); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}
	return RankGroupStandings(standings), rows.Err()
}

// getWorldCupGroupFixtures lists groupId's fixtures as stored, earliest matchday first.
func getWorldCupGroupFixtures(dbCtx *DatabaseContext, groupId int64) ([]worldCupGroupFixture, error) {
	rows, err := dbCtx.Db.Query(getWorldCupGroupMatchesStmtSql, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fixtures := make([]worldCupGroupFixture, 0)
	for rows.Next() {
		var f worldCupGroupFixture
		m := &f.match
		if err := rows.Scan(&m.Round, &m.MatchId, &m.HomeTeamId, &m.HomeTeamName, &m.AwayTeamId, &m.AwayTeamName, &m.MatchDate, &f.played, &f.homeGoals, &f.awayGoals); err != nil {
			return nil, err
		}
		fixtures = append(fixtures, f)
	}
	return fixtures, rows.Err()
}

const getWorldCupGroupIdForMatchStmtSql = "SELECT group_id FROM world_cup_group_matches WHERE match_id = $1"

const updateWorldCupGroupTeamStmtSql = "UPDATE world_cup_group_teams SET played = played + 1, " +
	"won = won + $1, drawn = drawn + $2, lost = lost + $3, goals_for = goals_for + $4, goals_against = goals_against + $5 " +
	"WHERE group_id = $6 AND team_id = $7"

// UpdateWorldCupStandingsAfterMatchTx updates both sides' running group record after matchId (a
// friendly_matches row) is simulated, inside tx - a no-op for the (overwhelming) majority of
// friendlies, which aren't part of any World Cup group at all. It runs in the same transaction as
// the friendly's result (see SaveFriendlyResultTx), so a result and its group standings are stored
// together or not at all. StartWorldCupForNewSeason's own engine.AdvanceWorldCups decides
// tournament-wide completion by checking every group itself, not from anything this returns.
func UpdateWorldCupStandingsAfterMatchTx(tx *sql.Tx, matchId int64, homeTeamId int64, awayTeamId int64, homeGoals int, awayGoals int) error {
	var groupId int64
	err := tx.QueryRow(getWorldCupGroupIdForMatchStmtSql, matchId).Scan(&groupId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	homeWon, homeDrawn, homeLost := 0, 0, 0
	awayWon, awayDrawn, awayLost := 0, 0, 0
	switch {
	case homeGoals > awayGoals:
		homeWon, awayLost = 1, 1
	case homeGoals < awayGoals:
		awayWon, homeLost = 1, 1
	default:
		homeDrawn, awayDrawn = 1, 1
	}

	if _, err := tx.Exec(updateWorldCupGroupTeamStmtSql, homeWon, homeDrawn, homeLost, homeGoals, awayGoals, groupId, homeTeamId); err != nil {
		return err
	}
	_, err = tx.Exec(updateWorldCupGroupTeamStmtSql, awayWon, awayDrawn, awayLost, awayGoals, homeGoals, groupId, awayTeamId)
	return err
}

const getWorldCupGroupMatchCountsStmtSql = "SELECT count(*), count(*) FILTER (WHERE fm.played), MAX(fm.match_date) " +
	"FROM world_cup_group_matches wcgm JOIN friendly_matches fm ON fm.match_id = wcgm.match_id " +
	"JOIN world_cup_groups wcg ON wcg.group_id = wcgm.group_id WHERE wcg.world_cup_id = $1"

// IsWorldCupGroupStageComplete reports whether every group match of worldCupId has been played
// and the last one's live broadcast has finished (its kickoff at or before LiveBroadcastCutoff).
// A match is simulated up to 15 minutes before kickoff and broadcast for ~105 minutes, so
// "every match played" alone would let engine.advanceWorldCup draw the knockout bracket - and
// notify its call-ups - while the deciding group matches are still on air, revealing who
// qualified. Cup rounds wait the same way (see GetUnresolvedCupTieCountStmtSql).
func IsWorldCupGroupStageComplete(dbCtx *DatabaseContext, worldCupId int64) (bool, error) {
	var total, played int
	var latestKickoff sql.NullTime
	err := dbCtx.Db.QueryRow(getWorldCupGroupMatchCountsStmtSql, worldCupId).Scan(&total, &played, &latestKickoff)
	if err != nil {
		return false, err
	}
	if total == 0 || total != played || !latestKickoff.Valid {
		return false, nil
	}
	return !latestKickoff.Time.After(LiveBroadcastCutoff()), nil
}

const getWorldCupGroupIdsStmtSql = "SELECT group_id FROM world_cup_groups WHERE world_cup_id = $1 ORDER BY group_name"

// GetWorldCupGroupQualifiers returns, for every group of worldCupId in group-name order, its top
// two teams by RankGroupStandings - the knockout stage's entrants (see
// engine.startWorldCupKnockout). Entrants come back flattened in group order (group A's winner
// and runner-up, then group B's, ...), ready for engine.BuildFirstRoundPairings to shuffle - this
// deliberately doesn't try to keep a group's two qualifiers apart in the first knockout round the
// way a real World Cup's bracket does, since with a shuffled bracket already a real tournament's
// bracket isn't just "fair", it's seeded by strength, which this game has no ranking to base on
// anyway.
func GetWorldCupGroupQualifiers(dbCtx *DatabaseContext, worldCupId int64) ([]int64, error) {
	rows, err := dbCtx.Db.Query(getWorldCupGroupIdsStmtSql, worldCupId)
	if err != nil {
		return nil, err
	}
	groupIds := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		groupIds = append(groupIds, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	qualifiers := make([]int64, 0, len(groupIds)*2)
	for _, groupId := range groupIds {
		standings, err := getWorldCupGroupStandings(dbCtx, groupId)
		if err != nil {
			return nil, err
		}
		for i := 0; i < 2 && i < len(standings); i++ {
			qualifiers = append(qualifiers, standings[i].TeamId)
		}
	}
	return qualifiers, nil
}

const setWorldCupKnockoutStmtSql = "UPDATE world_cups SET stage = 'knockout', knockout_cup_id = $1 WHERE world_cup_id = $2"

// SetWorldCupKnockoutStarted records that worldCupId's group stage is done and its knockout
// bracket is cupId - see engine.startWorldCupKnockout.
func SetWorldCupKnockoutStarted(dbCtx *DatabaseContext, worldCupId int64, cupId int64) error {
	_, err := dbCtx.Db.Exec(setWorldCupKnockoutStmtSql, cupId, worldCupId)
	return err
}

const setWorldCupCompletedStmtSql = "UPDATE world_cups SET completed = true, champion_team_id = $1 WHERE world_cup_id = $2"

// SetWorldCupCompleted records worldCupId's champion once its underlying knockout cup is itself
// complete - see engine.AdvanceWorldCups.
func SetWorldCupCompleted(dbCtx *DatabaseContext, worldCupId int64, championTeamId int64) error {
	_, err := dbCtx.Db.Exec(setWorldCupCompletedStmtSql, championTeamId, worldCupId)
	return err
}
