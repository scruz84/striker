package common

import (
	"database/sql"
	"sort"
	"time"
)

// Competitions of a player_match_stats row (openspec/specs/player-stats).
const (
	StatsCompetitionLeague           = 1
	StatsCompetitionDomesticCup      = 2
	StatsCompetitionPlayoff          = 3
	StatsCompetitionInternationalCup = 4
	StatsCompetitionWorldCup         = 5
	StatsCompetitionClubFriendly     = 6
	StatsCompetitionNationalFriendly = 7
)

// statsFixtureContext is what every player_match_stats row of one fixture shares. Season is nil
// when it is not the fixture's own (a regional cup, the World Cup or a friendly): each row then
// takes its team's country's current season.
type statsFixtureContext struct {
	Kickoff       time.Time
	Season        *int
	Competition   int
	CompetitionId *int64
}

const getLeagueStatsContextStmtSql = "SELECT match_date, season, league_id FROM matches WHERE match_id = $1"

const getCupStatsContextStmtSql = "SELECT COALESCE(ct.match_date, now()), " +
	"CASE WHEN c.country IS NOT NULL THEN c.season END, " +
	"CASE WHEN c.kind = 'playoff' THEN 3 WHEN c.country IS NOT NULL THEN 2 WHEN c.region = 'world' THEN 5 ELSE 4 END, " +
	"CASE WHEN c.region = 'world' THEN wc.world_cup_id ELSE c.cup_id END " +
	"FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id " +
	"LEFT JOIN world_cups wc ON wc.knockout_cup_id = c.cup_id " +
	"WHERE ct.tie_id = $1"

const getFriendlyStatsContextStmtSql = "SELECT fm.match_date, " +
	"CASE WHEN g.world_cup_id IS NOT NULL THEN 5 WHEN h.is_national_team THEN 7 ELSE 6 END, g.world_cup_id " +
	"FROM friendly_matches fm JOIN teams h ON h.team_id = fm.home_team_id " +
	"LEFT JOIN world_cup_group_matches wgm ON wgm.match_id = fm.match_id " +
	"LEFT JOIN world_cup_groups g ON g.group_id = wgm.group_id " +
	"WHERE fm.match_id = $1"

const getTeamCountrySeasonStmtSql = "SELECT c.current_season FROM teams t JOIN countries c ON c.name = t.country WHERE t.team_id = $1"

func getStatsFixtureContextTx(tx *sql.Tx, kind string, id int64) (statsFixtureContext, error) {
	var ctx statsFixtureContext
	var season sql.NullInt64
	var competitionId sql.NullInt64
	var err error
	switch kind {
	case SimulationKindLeague:
		ctx.Competition = StatsCompetitionLeague
		err = tx.QueryRow(getLeagueStatsContextStmtSql, id).Scan(&ctx.Kickoff, &season, &competitionId)
	case SimulationKindCup:
		err = tx.QueryRow(getCupStatsContextStmtSql, id).Scan(&ctx.Kickoff, &season, &ctx.Competition, &competitionId)
	default:
		err = tx.QueryRow(getFriendlyStatsContextStmtSql, id).Scan(&ctx.Kickoff, &ctx.Competition, &competitionId)
	}
	if err != nil {
		return ctx, err
	}
	if season.Valid {
		s := int(season.Int64)
		ctx.Season = &s
	}
	if competitionId.Valid {
		c := competitionId.Int64
		ctx.CompetitionId = &c
	}
	return ctx, nil
}

// PlayerMatchStatLine is one player's line of one fixture, as player_match_stats stores it.
type PlayerMatchStatLine struct {
	PlayerId int64
	TeamId   int64
	Appeared bool
	Started  bool
	Minutes  int
	Rated    bool
	Rating   float64
	// Position is the tactical position of his rating (see PlayerRating.Position); nil without an
	// appearance.
	Position     *int
	Goals        int
	PenaltyGoals int
	YellowCards  int
	RedCards     int
}

// BuildPlayerMatchStatLines turns a fixture's stored ratings (who featured, for how long, how well)
// and its events (goals and cards, each naming its one player first in TeamOwnerPlayers) into one
// line per player, sorted by player id. A player with an event but no rating - never the case for
// a simulated match - still gets a line, without an appearance. Pure so it is testable without a
// database.
func BuildPlayerMatchStatLines(ratings []PlayerRating, events []Event) []PlayerMatchStatLine {
	lines := make(map[int64]*PlayerMatchStatLine, len(ratings))
	for _, r := range ratings {
		position := r.Position
		lines[r.PlayerId] = &PlayerMatchStatLine{
			Position: &position,
			PlayerId: r.PlayerId,
			TeamId:   r.TeamId,
			Appeared: true,
			Started:  r.CameOn == nil,
			Minutes:  r.Minutes,
			Rated:    r.Rated,
			Rating:   r.Rating,
		}
	}
	for _, e := range events {
		if e.EventType != GOAL && e.EventType != YELLOW_CARD && e.EventType != RED_CARD {
			continue
		}
		if len(e.TeamOwnerPlayers) == 0 {
			continue
		}
		playerId := e.TeamOwnerPlayers[0]
		line, ok := lines[playerId]
		if !ok {
			line = &PlayerMatchStatLine{PlayerId: playerId, TeamId: e.TeamOwner}
			lines[playerId] = line
		}
		switch e.EventType {
		case GOAL:
			line.Goals++
			if e.SetPiece == SET_PIECE_PENALTY {
				line.PenaltyGoals++
			}
		case YELLOW_CARD:
			line.YellowCards++
		case RED_CARD:
			line.RedCards++
		}
	}
	out := make([]PlayerMatchStatLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, *line)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerId < out[j].PlayerId })
	return out
}

const deletePlayerMatchStatsStmtSql = "DELETE FROM player_match_stats WHERE kind = $1 AND match_id = $2"

const insertPlayerMatchStatsStmtSql = "INSERT INTO player_match_stats " +
	"(kind, match_id, player_id, team_id, kickoff, season, competition, competition_id, appeared, started, minutes, rated, rating, " +
	"goals, penalty_goals, yellow_cards, red_cards, position) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)"

// replacePlayerMatchStatsTx stores a fixture's player match stats inside its result transaction,
// replacing any left by an earlier failed attempt. A default result stores none: it has no
// ratings and no goal or card.
func replacePlayerMatchStatsTx(tx *sql.Tx, kind string, id int64, result MatchResult) error {
	if _, err := tx.Exec(deletePlayerMatchStatsStmtSql, kind, id); err != nil {
		return err
	}
	lines := BuildPlayerMatchStatLines(result.PlayerRatings, result.Events)
	if len(lines) == 0 {
		return nil
	}
	ctx, err := getStatsFixtureContextTx(tx, kind, id)
	if err != nil {
		return err
	}
	insert, err := tx.Prepare(insertPlayerMatchStatsStmtSql)
	if err != nil {
		return err
	}
	defer insert.Close()
	teamSeasons := make(map[int64]int)
	for _, line := range lines {
		season := 1
		if ctx.Season != nil {
			season = *ctx.Season
		} else if s, ok := teamSeasons[line.TeamId]; ok {
			season = s
		} else {
			if err := tx.QueryRow(getTeamCountrySeasonStmtSql, line.TeamId).Scan(&season); err != nil && err != sql.ErrNoRows {
				return err
			}
			teamSeasons[line.TeamId] = season
		}
		if _, err := insert.Exec(kind, id, line.PlayerId, line.TeamId, ctx.Kickoff, season, ctx.Competition, ctx.CompetitionId,
			line.Appeared, line.Started, line.Minutes, line.Rated, line.Rating,
			line.Goals, line.PenaltyGoals, line.YellowCards, line.RedCards, line.Position); err != nil {
			return err
		}
	}
	return nil
}
