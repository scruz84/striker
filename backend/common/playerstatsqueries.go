package common

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

// Every reader here totals player_match_stats (see replacePlayerMatchStatsTx). A non-nil cutoff
// leaves out the rows of fixtures that kicked off after it - the public views pass
// LiveBroadcastCutoff(), so no stat shows before its match's replay ends; the season awards pass
// nil, as the season is over.

// statsPlayerNameJoinsSql resolves a player id to a name whether he is still in the game or has
// retired (openspec/specs/retired-players). It expects the player id as t.player_id.
const statsPlayerNameJoinsSql = "LEFT JOIN players p ON p.player_id = t.player_id " +
	"LEFT JOIN retired_players rp ON rp.player_id = t.player_id "

const statsPlayerNameColumnsSql = "COALESCE(p.first_name, rp.first_name, ''), COALESCE(p.last_name, rp.last_name, ''), " +
	"COALESCE(p.birth_country, rp.birth_country, ''), p.player_id IS NULL AND rp.player_id IS NOT NULL"

// roundRating keeps two decimals of an average rating.
func roundRating(v float64) float64 {
	return math.Round(v*100) / 100
}

// PlayerCareerStats are the headline totals of a player's profile.
type PlayerCareerStats struct {
	// Appearances, Minutes, Goals and the cards count every competition but club friendlies
	// (openspec/specs/player-stats "Career totals").
	Appearances int `json:"appearances"`
	Minutes     int `json:"minutes"`
	Goals       int `json:"goals"`
	YellowCards int `json:"yellowCards"`
	RedCards    int `json:"redCards"`
	// GoalsForTeam is the part of Goals scored for whichever team currently employs this player (0
	// for a free agent, who plays for none).
	GoalsForTeam int `json:"goalsForTeam"`
	// Caps is how many played national-team friendlies this player was selected to start (see
	// common.SetNationalTeamLineup/ScheduleNationalTeamFriendlies) - counted from
	// friendly_lineup_positions rather than from his match stats, since a cap is a selection.
	Caps int `json:"caps"`
}

const getPlayerCareerTotalsStmtSql = "SELECT COUNT(*) FILTER (WHERE appeared), COALESCE(SUM(minutes), 0), COALESCE(SUM(goals), 0), " +
	"COALESCE(SUM(yellow_cards), 0), COALESCE(SUM(red_cards), 0), COALESCE(SUM(goals) FILTER (WHERE team_id = $2), 0) " +
	"FROM player_match_stats WHERE player_id = $1 AND competition <> 6 AND kickoff <= $3"

// GetPlayerCareerStats totals a player's career (see PlayerCareerStats), GoalsForTeam scoped to
// teamId (his current team - see PlayerDetails.TeamId).
func GetPlayerCareerStats(dbCtx *DatabaseContext, playerId int64, teamId int64) (PlayerCareerStats, error) {
	var stats PlayerCareerStats
	if err := dbCtx.Db.QueryRow(getPlayerCareerTotalsStmtSql, playerId, teamId, LiveBroadcastCutoff()).Scan(
		&stats.Appearances, &stats.Minutes, &stats.Goals, &stats.YellowCards, &stats.RedCards, &stats.GoalsForTeam); err != nil {
		return stats, err
	}
	if err := dbCtx.Db.QueryRow(countPlayerCapsStmtSql, playerId).Scan(&stats.Caps); err != nil {
		return stats, err
	}
	return stats, nil
}

// StatLine is the totals of a set of player_match_stats rows.
type StatLine struct {
	Appearances  int `json:"appearances"`
	Starts       int `json:"starts"`
	Minutes      int `json:"minutes"`
	Goals        int `json:"goals"`
	PenaltyGoals int `json:"penaltyGoals"`
	YellowCards  int `json:"yellowCards"`
	RedCards     int `json:"redCards"`
	// AverageRating is the mean of the rated appearances (10 minutes or more, see
	// openspec/specs/match-ratings), two decimals; nil without one.
	AverageRating *float64 `json:"averageRating,omitempty"`
	RatedMatches  int      `json:"ratedMatches"`
}

const statLineColumnsSql = "COUNT(*) FILTER (WHERE s.appeared), COUNT(*) FILTER (WHERE s.started), COALESCE(SUM(s.minutes), 0), " +
	"COALESCE(SUM(s.goals), 0), COALESCE(SUM(s.penalty_goals), 0), COALESCE(SUM(s.yellow_cards), 0), COALESCE(SUM(s.red_cards), 0), " +
	"AVG(s.rating) FILTER (WHERE s.rated), COUNT(*) FILTER (WHERE s.rated)"

func scanStatLineDest(l *StatLine, avg *sql.NullFloat64) []any {
	return []any{&l.Appearances, &l.Starts, &l.Minutes, &l.Goals, &l.PenaltyGoals, &l.YellowCards, &l.RedCards, avg, &l.RatedMatches}
}

func finishStatLine(l *StatLine, avg sql.NullFloat64) {
	if avg.Valid {
		v := roundRating(avg.Float64)
		l.AverageRating = &v
	}
}

// PlayerSeasonStats is a player's line for one season, competition and team.
type PlayerSeasonStats struct {
	Season        int    `json:"season"`
	Competition   int    `json:"competition"`
	CompetitionId *int64 `json:"competitionId,omitempty"`
	// CompetitionName is the league's, cup's or World Cup's stored name; empty for a friendly.
	CompetitionName string `json:"competitionName"`
	// CompetitionRegion is a regional cup's region (see cups.region), so it can be named in the
	// viewer's language; empty otherwise.
	CompetitionRegion string `json:"competitionRegion,omitempty"`
	TeamId            int64  `json:"teamId"`
	TeamName          string `json:"teamName"`
	StatLine
}

const getPlayerSeasonStatsStmtSql = "SELECT s.season, s.competition, s.competition_id, COALESCE(l.name, cu.name, wc.name, ''), " +
	"COALESCE(cu.region, ''), s.team_id, COALESCE(t.name, ''), " + statLineColumnsSql + " " +
	"FROM player_match_stats s " +
	"LEFT JOIN teams t ON t.team_id = s.team_id " +
	"LEFT JOIN leagues l ON s.competition = 1 AND l.league_id = s.competition_id " +
	"LEFT JOIN cups cu ON s.competition IN (2, 3, 4) AND cu.cup_id = s.competition_id " +
	"LEFT JOIN world_cups wc ON s.competition = 5 AND wc.world_cup_id = s.competition_id " +
	"WHERE s.player_id = $1 AND s.kickoff <= $2 " +
	"GROUP BY s.season, s.competition, s.competition_id, l.name, cu.name, wc.name, cu.region, s.team_id, t.name " +
	"ORDER BY s.season DESC, s.competition, s.team_id, s.competition_id"

// GetPlayerSeasonStats lists a player's lines, newest season first and, within a season, by
// competition (openspec/specs/player-stats "Season lines on the profile").
func GetPlayerSeasonStats(dbCtx *DatabaseContext, playerId int64) ([]PlayerSeasonStats, error) {
	rows, err := dbCtx.Db.Query(getPlayerSeasonStatsStmtSql, playerId, LiveBroadcastCutoff())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PlayerSeasonStats, 0)
	for rows.Next() {
		var r PlayerSeasonStats
		var competitionId sql.NullInt64
		var avg sql.NullFloat64
		dest := append([]any{&r.Season, &r.Competition, &competitionId, &r.CompetitionName, &r.CompetitionRegion, &r.TeamId, &r.TeamName},
			scanStatLineDest(&r.StatLine, &avg)...)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if competitionId.Valid {
			id := competitionId.Int64
			r.CompetitionId = &id
		}
		finishStatLine(&r.StatLine, avg)
		out = append(out, r)
	}
	return out, rows.Err()
}

// League leader categories (openspec/specs/player-stats "League leaders").
const (
	LeadersGoals   = "goals"
	LeadersRating  = "rating"
	LeadersMinutes = "minutes"
	LeadersCards   = "cards"
)

// RatingLeaderMinutesPerMatch is the share of a match a player must have played, on average over
// the most matches any team of the league has played, to enter the rating leaders: 40% of 90.
const RatingLeaderMinutesPerMatch = 36

// LeagueLeader is one player of a league leaders list, with his whole line for that league
// season. TeamId is the team of his latest appearance in it.
type LeagueLeader struct {
	PlayerId     int64  `json:"playerId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	BirthCountry string `json:"birthCountry,omitempty"`
	Retired      bool   `json:"retired,omitempty"`
	TeamId       int64  `json:"teamId"`
	TeamName     string `json:"teamName"`
	StatLine
}

// statTotalsColumnsSql names the columns of a statLineColumnsSql subquery grouped by player.
const statTotalsColumnsSql = "player_id, apps, starts, minutes, goals, pens, yellow, red, avg_rating, rated"

const statTotalsSelectSql = "t.apps, t.starts, t.minutes, t.goals, t.pens, t.yellow, t.red, t.avg_rating, t.rated"

// leaderOrderSql is each category's filter and order; ties fall to the lower player id. Every
// category reads $4, the minimum minutes (0 but for the rating leaders), so the statement always
// binds the same parameters.
var leaderOrderSql = map[string]string{
	LeadersGoals:   "WHERE t.goals > 0 AND t.minutes >= $4 ORDER BY t.goals DESC, t.player_id",
	LeadersRating:  "WHERE t.rated > 0 AND t.minutes >= $4 ORDER BY t.avg_rating DESC, t.minutes DESC, t.player_id",
	LeadersMinutes: "WHERE t.minutes > 0 AND t.minutes >= $4 ORDER BY t.minutes DESC, t.player_id",
	LeadersCards:   "WHERE t.yellow + t.red > 0 AND t.minutes >= $4 ORDER BY t.yellow + 3 * t.red DESC, t.red DESC, t.player_id",
}

// leagueTotalsSql is every player's line of one league season ($1 league, $2 season, $3 cutoff),
// with the team of his latest appearance in it.
const leagueTotalsSql = "(SELECT s.player_id, " + statLineColumnsSql + ", " +
	"(array_agg(s.team_id ORDER BY s.kickoff DESC, s.match_id DESC))[1] " +
	"FROM player_match_stats s " +
	"WHERE s.competition = 1 AND s.competition_id = $1 AND s.season = $2 AND ($3::timestamptz IS NULL OR s.kickoff <= $3) " +
	"GROUP BY s.player_id) AS t(" + statTotalsColumnsSql + ", team_id) "

func leagueLeadersStmtSql(category string) string {
	return "SELECT t.player_id, " + statsPlayerNameColumnsSql + ", t.team_id, COALESCE(tm.name, ''), " + statTotalsSelectSql + " " +
		"FROM " + leagueTotalsSql + statsPlayerNameJoinsSql + "LEFT JOIN teams tm ON tm.team_id = t.team_id " +
		leaderOrderSql[category] + " LIMIT CASE WHEN $5::int > 0 THEN $5::int END"
}

const leagueMostMatchesStmtSql = "SELECT COALESCE(MAX(n), 0) FROM (SELECT team_id, COUNT(DISTINCT match_id) AS n " +
	"FROM player_match_stats WHERE competition = 1 AND competition_id = $1 AND season = $2 " +
	"AND ($3::timestamptz IS NULL OR kickoff <= $3) GROUP BY team_id) x"

// IsLeadersCategory reports whether category is one GetLeagueLeaders knows.
func IsLeadersCategory(category string) bool {
	_, ok := leaderOrderSql[category]
	return ok
}

// GetLeagueLeaders returns a league season's top `limit` players in category (see the Leaders*
// constants), counting only league matches of that league and season. A limit of 0 or less
// returns them all.
func GetLeagueLeaders(dbCtx *DatabaseContext, leagueId int64, season int, category string, limit int, cutoff *time.Time) ([]LeagueLeader, error) {
	if !IsLeadersCategory(category) {
		return nil, fmt.Errorf("unknown leaders category %q", category)
	}
	minMinutes := 0
	if category == LeadersRating {
		var mostMatches int
		if err := dbCtx.Db.QueryRow(leagueMostMatchesStmtSql, leagueId, season, cutoff).Scan(&mostMatches); err != nil {
			return nil, err
		}
		minMinutes = RatingLeaderMinutesPerMatch * mostMatches
	}
	rows, err := dbCtx.Db.Query(leagueLeadersStmtSql(category), leagueId, season, cutoff, minMinutes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeagueLeader, 0)
	for rows.Next() {
		var l LeagueLeader
		var avg sql.NullFloat64
		dest := append([]any{&l.PlayerId, &l.FirstName, &l.LastName, &l.BirthCountry, &l.Retired, &l.TeamId, &l.TeamName},
			scanStatLineDest(&l.StatLine, &avg)...)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		finishStatLine(&l.StatLine, avg)
		out = append(out, l)
	}
	return out, rows.Err()
}

// TeamPlayerStats is one player's line for a team over one season (openspec/specs/player-stats
// "Squad stats"), whether he is still at the club, has left it or has retired.
type TeamPlayerStats struct {
	PlayerId     int64  `json:"playerId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	BirthCountry string `json:"birthCountry,omitempty"`
	Retired      bool   `json:"retired,omitempty"`
	// InSquad is true while he still belongs to the team.
	InSquad bool `json:"inSquad"`
	StatLine
}

// TeamSeasonStats is a team's squad stats for one season, and the seasons it has any for.
type TeamSeasonStats struct {
	Season  int               `json:"season"`
	Seasons []int             `json:"seasons"`
	Players []TeamPlayerStats `json:"players"`
}

const getTeamStatsSeasonsStmtSql = "SELECT DISTINCT season FROM player_match_stats WHERE team_id = $1 AND competition <> 6 " +
	"AND kickoff <= $2 ORDER BY season DESC"

const getTeamSeasonStatsStmtSql = "SELECT t.player_id, " + statsPlayerNameColumnsSql + ", COALESCE(p.team_id = $1, false), " +
	statTotalsSelectSql + " " +
	"FROM (SELECT s.player_id, " + statLineColumnsSql + " FROM player_match_stats s " +
	"WHERE s.team_id = $1 AND s.season = $2 AND s.competition <> 6 AND s.kickoff <= $3 GROUP BY s.player_id) " +
	"AS t(" + statTotalsColumnsSql + ") " + statsPlayerNameJoinsSql +
	"ORDER BY t.minutes DESC, t.player_id"

// GetTeamSeasonStats returns teamId's squad stats for season, or for its newest season with any
// when season is 0. Club friendlies are left out.
func GetTeamSeasonStats(dbCtx *DatabaseContext, teamId int64, season int) (TeamSeasonStats, error) {
	result := TeamSeasonStats{Seasons: []int{}, Players: []TeamPlayerStats{}}
	cutoff := LiveBroadcastCutoff()
	rows, err := dbCtx.Db.Query(getTeamStatsSeasonsStmtSql, teamId, cutoff)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var s int
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return result, err
		}
		result.Seasons = append(result.Seasons, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}
	if season == 0 {
		if len(result.Seasons) == 0 {
			return result, nil
		}
		season = result.Seasons[0]
	}
	result.Season = season

	statRows, err := dbCtx.Db.Query(getTeamSeasonStatsStmtSql, teamId, season, cutoff)
	if err != nil {
		return result, err
	}
	defer statRows.Close()
	for statRows.Next() {
		var p TeamPlayerStats
		var avg sql.NullFloat64
		dest := append([]any{&p.PlayerId, &p.FirstName, &p.LastName, &p.BirthCountry, &p.Retired, &p.InSquad},
			scanStatLineDest(&p.StatLine, &avg)...)
		if err := statRows.Scan(dest...); err != nil {
			return result, err
		}
		finishStatLine(&p.StatLine, avg)
		result.Players = append(result.Players, p)
	}
	return result, statRows.Err()
}
