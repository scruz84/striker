package common

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Cup is a single-elimination competition for one country's teams in one season.
type Cup struct {
	CupId            int64  `json:"cupId"`
	Country          string `json:"country"`
	Season           int    `json:"season"`
	Name             string `json:"name"`
	Completed        bool   `json:"completed"`
	ChampionTeamId   *int64 `json:"championTeamId"`
	ChampionTeamName string `json:"championTeamName"`
}

// CupTie is one bracket slot - a match between two teams, or, if AwayTeamId is nil, an
// automatic bye advancing HomeTeamId without a match being played.
type CupTie struct {
	TieId           int64      `json:"tieId"`
	CupId           int64      `json:"cupId"`
	Round           int        `json:"round"`
	SlotNumber      int        `json:"slotNumber"`
	HomeTeamId      *int64     `json:"homeTeamId"`
	HomeTeamName    string     `json:"homeTeamName"`
	AwayTeamId      *int64     `json:"awayTeamId"`
	AwayTeamName    string     `json:"awayTeamName"`
	MatchDate       *time.Time `json:"matchDate"`
	Played          bool       `json:"played"`
	HomeGoals       int        `json:"homeGoals"`
	AwayGoals       int        `json:"awayGoals"`
	WentToPenalties bool       `json:"wentToPenalties"`
	WinnerTeamId    *int64     `json:"winnerTeamId"`
}

const InsertCupStmtSql = "INSERT INTO cups (country, season, name) VALUES ($1, $2, $3) RETURNING cup_id"

const InsertCupTieStmtSql = "INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played, winner_team_id) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING tie_id"

const GetActiveCupsStmtSql = "SELECT cup_id, country, season, name, completed, champion_team_id FROM cups WHERE completed = false"

const GetCupStmtSql = "SELECT c.cup_id, c.country, c.season, c.name, c.completed, c.champion_team_id, t.name " +
	"FROM cups c LEFT JOIN teams t ON t.team_id = c.champion_team_id WHERE c.cup_id = $1"

const GetLatestCupForCountryStmtSql = "SELECT cup_id FROM cups WHERE country = $1 ORDER BY cup_id DESC LIMIT 1"

const GetCupTiesStmtSql = "SELECT ct.tie_id, ct.cup_id, ct.round, ct.slot_number, ct.home_team_id, th.name, " +
	"ct.away_team_id, ta.name, ct.match_date, ct.played, ct.home_goals, ct.away_goals, ct.went_to_penalties, ct.winner_team_id " +
	"FROM cup_ties ct LEFT JOIN teams th ON th.team_id = ct.home_team_id LEFT JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE ct.cup_id = $1 ORDER BY ct.round, ct.slot_number"

const GetMaxCupRoundStmtSql = "SELECT COALESCE(MAX(round), 0) FROM cup_ties WHERE cup_id = $1"

const GetUnresolvedCupTieCountStmtSql = "SELECT count(*) FROM cup_ties WHERE cup_id = $1 AND round = $2 AND winner_team_id IS NULL"

const GetCupRoundWinnersStmtSql = "SELECT winner_team_id FROM cup_ties WHERE cup_id = $1 AND round = $2 ORDER BY slot_number"

const SetCupCompletedStmtSql = "UPDATE cups SET completed = true, champion_team_id = $1 WHERE cup_id = $2"

const GetSimulableCupTiesStmtSql = "SELECT tie_id, cup_id, home_team_id, away_team_id FROM cup_ties " +
	"WHERE match_date is not null AND match_date <= now() AND played = false AND in_simulation = false " +
	"AND home_team_id IS NOT NULL AND away_team_id IS NOT NULL " + ForUpdate

const SetCupTiePlayedStmtSql = "UPDATE cup_ties SET played = true, in_simulation = false, home_goals = $1, away_goals = $2, " +
	"went_to_penalties = $3, winner_team_id = $4 WHERE tie_id = $5"

const InsertCupTieEventStmtSql = "INSERT INTO cup_tie_events (tie_id, event_type, team_owner, other_team, minute, " +
	"team_players, team_player_names, home_goals, away_goals) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"

const GetCupTieEventsStmtSql = "SELECT event_type, team_owner, other_team, minute, team_players, team_player_names, home_goals, away_goals " +
	"FROM cup_tie_events WHERE tie_id = $1 ORDER BY minute ASC"

// CreateCup starts a new single-elimination cup for a country's season.
func CreateCup(dbCtx *DatabaseContext, country string, season int, name string) (int64, error) {
	var cupId int64
	err := dbCtx.Db.QueryRow(InsertCupStmtSql, country, season, name).Scan(&cupId)
	return cupId, err
}

// CreateCupTie inserts one bracket slot. A nil awayTeamId with a non-nil homeTeamId is a bye:
// it's recorded as already won by the home team, with no match to simulate.
func CreateCupTie(dbCtx *DatabaseContext, cupId int64, round int, slotNumber int, homeTeamId *int64, awayTeamId *int64, matchDate *time.Time) error {
	var winnerTeamId *int64
	played := false
	if homeTeamId != nil && awayTeamId == nil {
		winnerTeamId = homeTeamId
		played = true
	}
	_, err := dbCtx.Db.Exec(InsertCupTieStmtSql, cupId, round, slotNumber, homeTeamId, awayTeamId, matchDate, played, winnerTeamId)
	return err
}

func GetActiveCups(dbCtx *DatabaseContext) ([]Cup, error) {
	rows, err := dbCtx.Db.Query(GetActiveCupsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cups := make([]Cup, 0)
	for rows.Next() {
		var cup Cup
		var championTeamId sql.NullInt64
		if err := rows.Scan(&cup.CupId, &cup.Country, &cup.Season, &cup.Name, &cup.Completed, &championTeamId); err != nil {
			return nil, err
		}
		if championTeamId.Valid {
			cup.ChampionTeamId = &championTeamId.Int64
		}
		cups = append(cups, cup)
	}
	return cups, nil
}

func GetCup(dbCtx *DatabaseContext, cupId int64) (*Cup, error) {
	var cup Cup
	var championTeamId sql.NullInt64
	var championTeamName sql.NullString
	err := dbCtx.Db.QueryRow(GetCupStmtSql, cupId).Scan(&cup.CupId, &cup.Country, &cup.Season, &cup.Name,
		&cup.Completed, &championTeamId, &championTeamName)
	if err != nil {
		return nil, err
	}
	if championTeamId.Valid {
		cup.ChampionTeamId = &championTeamId.Int64
	}
	cup.ChampionTeamName = championTeamName.String
	return &cup, nil
}

func GetLatestCupForCountry(dbCtx *DatabaseContext, country string) (int64, error) {
	var cupId int64
	err := dbCtx.Db.QueryRow(GetLatestCupForCountryStmtSql, country).Scan(&cupId)
	return cupId, err
}

func GetCupTies(dbCtx *DatabaseContext, cupId int64) ([]CupTie, error) {
	rows, err := dbCtx.Db.Query(GetCupTiesStmtSql, cupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ties := make([]CupTie, 0)
	for rows.Next() {
		tie, err := scanCupTie(rows)
		if err != nil {
			return nil, err
		}
		ties = append(ties, *tie)
	}
	return ties, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanCupTie(row rowScanner) (*CupTie, error) {
	var tie CupTie
	var homeTeamId, awayTeamId, winnerTeamId sql.NullInt64
	var homeTeamName, awayTeamName sql.NullString
	var matchDate sql.NullTime
	err := row.Scan(&tie.TieId, &tie.CupId, &tie.Round, &tie.SlotNumber, &homeTeamId, &homeTeamName,
		&awayTeamId, &awayTeamName, &matchDate, &tie.Played, &tie.HomeGoals, &tie.AwayGoals,
		&tie.WentToPenalties, &winnerTeamId)
	if err != nil {
		return nil, err
	}
	if homeTeamId.Valid {
		tie.HomeTeamId = &homeTeamId.Int64
	}
	tie.HomeTeamName = homeTeamName.String
	if awayTeamId.Valid {
		tie.AwayTeamId = &awayTeamId.Int64
	}
	tie.AwayTeamName = awayTeamName.String
	if matchDate.Valid {
		tie.MatchDate = &matchDate.Time
	}
	if winnerTeamId.Valid {
		tie.WinnerTeamId = &winnerTeamId.Int64
	}
	return &tie, nil
}

// GetMaxCupRound returns the highest round number a cup has generated ties for (0 if none).
func GetMaxCupRound(dbCtx *DatabaseContext, cupId int64) (int, error) {
	var round int
	err := dbCtx.Db.QueryRow(GetMaxCupRoundStmtSql, cupId).Scan(&round)
	return round, err
}

// IsCupRoundResolved reports whether every tie in a round has a winner (played, or a bye).
func IsCupRoundResolved(dbCtx *DatabaseContext, cupId int64, round int) (bool, error) {
	var unresolved int
	err := dbCtx.Db.QueryRow(GetUnresolvedCupTieCountStmtSql, cupId, round).Scan(&unresolved)
	if err != nil {
		return false, err
	}
	return unresolved == 0, nil
}

// GetCupRoundWinners returns a round's winners in slot order, ready to feed into
// engine.BuildNextRoundPairings.
func GetCupRoundWinners(dbCtx *DatabaseContext, cupId int64, round int) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetCupRoundWinnersStmtSql, cupId, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	winners := make([]int64, 0)
	for rows.Next() {
		var winnerTeamId int64
		if err := rows.Scan(&winnerTeamId); err != nil {
			return nil, err
		}
		winners = append(winners, winnerTeamId)
	}
	return winners, nil
}

func SetCupCompleted(dbCtx *DatabaseContext, cupId int64, championTeamId int64) error {
	_, err := dbCtx.Db.Exec(SetCupCompletedStmtSql, championTeamId, cupId)
	return err
}

// GetSimulableCupTies returns ties whose match_date has arrived and both teams are known
// (i.e. not a still-unresolved future round, and not a bye - a bye never gets a match_date).
func GetSimulableCupTies(dbCtx *DatabaseContext) ([]CupTie, error) {
	rows, err := dbCtx.Db.Query(GetSimulableCupTiesStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ties := make([]CupTie, 0)
	for rows.Next() {
		var tie CupTie
		var homeTeamId, awayTeamId int64
		if err := rows.Scan(&tie.TieId, &tie.CupId, &homeTeamId, &awayTeamId); err != nil {
			return nil, err
		}
		tie.HomeTeamId = &homeTeamId
		tie.AwayTeamId = &awayTeamId
		ties = append(ties, tie)
	}
	return ties, nil
}

// SetCupTiePlayed records a simulated tie's result and its events.
func SetCupTiePlayed(dbCtx *DatabaseContext, tieId int64, homeGoals int, awayGoals int, wentToPenalties bool, winnerTeamId int64, events []Event) error {
	if _, err := dbCtx.Db.Exec(SetCupTiePlayedStmtSql, homeGoals, awayGoals, wentToPenalties, winnerTeamId, tieId); err != nil {
		return err
	}

	insertEvent, err := dbCtx.Db.Prepare(InsertCupTieEventStmtSql)
	if err != nil {
		return err
	}
	defer insertEvent.Close()

	for _, e := range events {
		teamPlayersJson, _ := json.Marshal(e.TeamOwnerPlayers)
		teamPlayerNamesJson, _ := json.Marshal(e.TeamOwnerPlayersNames)
		if _, err := insertEvent.Exec(tieId, e.EventType, e.TeamOwner, e.OtherTeam, e.Minute,
			teamPlayersJson, teamPlayerNamesJson, e.HomeGoals, e.AwayGoals); err != nil {
			return err
		}
	}
	return nil
}

func GetCupTieEvents(dbCtx *DatabaseContext, tieId int64) ([]Event, error) {
	rows, err := dbCtx.Db.Query(GetCupTieEventsStmtSql, tieId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var e Event
		var teamPlayers, teamPlayerNames string
		if err := rows.Scan(&e.EventType, &e.TeamOwner, &e.OtherTeam, &e.Minute, &teamPlayers, &teamPlayerNames,
			&e.HomeGoals, &e.AwayGoals); err != nil {
			return nil, err
		}
		if teamPlayers != "" {
			_ = json.Unmarshal([]byte(teamPlayers), &e.TeamOwnerPlayers)
		}
		if teamPlayerNames != "" {
			_ = json.Unmarshal([]byte(teamPlayerNames), &e.TeamOwnerPlayersNames)
		}
		events = append(events, e)
	}
	return events, nil
}
