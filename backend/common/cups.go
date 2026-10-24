package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Cup is a single-elimination competition for one season - either one country's own tier-1
// teams (Country set, Region empty) or, for an international cup (issue #64), one team per
// country from every active country in one region (Region set, Country empty). Exactly one of
// the two is ever set - see resources/sql/init52_international_cup.sql's CHECK constraint.
type Cup struct {
	CupId            int64  `json:"cupId"`
	Country          string `json:"country"`
	Region           string `json:"region"`
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

const InsertCupStmtSql = "INSERT INTO cups (country, region, season, name) VALUES ($1, NULL, $2, $3) RETURNING cup_id"

const InsertInternationalCupStmtSql = "INSERT INTO cups (country, region, season, name) VALUES (NULL, $1, $2, $3) RETURNING cup_id"

const InsertCupTieStmtSql = "INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played, winner_team_id) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING tie_id"

const GetActiveCupsStmtSql = "SELECT cup_id, country, region, season, name, completed, champion_team_id FROM cups WHERE completed = false"

const GetCupStmtSql = "SELECT c.cup_id, c.country, c.region, c.season, c.name, c.completed, c.champion_team_id, t.name " +
	"FROM cups c LEFT JOIN teams t ON t.team_id = c.champion_team_id WHERE c.cup_id = $1"

const GetLatestCupForCountryStmtSql = "SELECT cup_id FROM cups WHERE country = $1 ORDER BY cup_id DESC LIMIT 1"

const GetLatestCupForRegionStmtSql = "SELECT cup_id FROM cups WHERE region = $1 ORDER BY cup_id DESC LIMIT 1"

const GetCupTiesStmtSql = "SELECT ct.tie_id, ct.cup_id, ct.round, ct.slot_number, ct.home_team_id, th.name, " +
	"ct.away_team_id, ta.name, ct.match_date, ct.played, ct.home_goals, ct.away_goals, ct.went_to_penalties, ct.winner_team_id " +
	"FROM cup_ties ct LEFT JOIN teams th ON th.team_id = ct.home_team_id LEFT JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE ct.cup_id = $1 ORDER BY ct.round, ct.slot_number"

const GetCupTieByIdStmtSql = "SELECT ct.tie_id, ct.cup_id, ct.round, ct.slot_number, ct.home_team_id, th.name, " +
	"ct.away_team_id, ta.name, ct.match_date, ct.played, ct.home_goals, ct.away_goals, ct.went_to_penalties, ct.winner_team_id " +
	"FROM cup_ties ct LEFT JOIN teams th ON th.team_id = ct.home_team_id LEFT JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE ct.tie_id = $1"

const GetMaxCupRoundStmtSql = "SELECT COALESCE(MAX(round), 0) FROM cup_ties WHERE cup_id = $1"

// A tie with a winner is still counted as unresolved while its match sits inside the
// live-broadcast window (see LiveBroadcastWindow) - a bye (match_date IS NULL) is exempt since
// there's no broadcast to wait out. Without this, the next round's pairing (and therefore the
// winning team's name) would be created and revealed the instant the tie is simulated, before
// that tie's own page stops showing it as "LIVE" (issue: match results review).
const GetUnresolvedCupTieCountStmtSql = "SELECT count(*) FROM cup_ties WHERE cup_id = $1 AND round = $2 " +
	"AND (winner_team_id IS NULL OR (match_date IS NOT NULL AND match_date > $3))"

const GetCupRoundWinnersStmtSql = "SELECT winner_team_id FROM cup_ties WHERE cup_id = $1 AND round = $2 ORDER BY slot_number"

const SetCupCompletedStmtSql = "UPDATE cups SET completed = true, champion_team_id = $1 WHERE cup_id = $2"

const GetSimulableCupTiesStmtSql = "SELECT tie_id, cup_id, home_team_id, away_team_id, match_date FROM cup_ties " +
	"WHERE match_date is not null AND match_date <= now() AND played = false AND in_simulation = false " +
	"AND home_team_id IS NOT NULL AND away_team_id IS NOT NULL " + ForUpdate

const SetCupTiePlayedStmtSql = "UPDATE cup_ties SET played = true, in_simulation = false, home_goals = $1, away_goals = $2, " +
	"went_to_penalties = $3, winner_team_id = $4 WHERE tie_id = $5"

const InsertCupTieEventStmtSql = "INSERT INTO cup_tie_events (tie_id, event_type, team_owner, other_team, minute, " +
	"team_players, team_player_names, home_goals, away_goals, set_piece) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"

const InsertCupTieInstantStmtSql = "INSERT INTO cup_tie_instants(tie_id, minute, home_goals, away_goals, home_qua, away_qua, " +
	"midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"

const GetCupTieInstantsStmtSql = "SELECT tie_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, " +
	"homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff " +
	"FROM cup_tie_instants WHERE tie_id = $1 order by minute asc"

// GetCupTieFlavorStmtSql reads a cup tie's report flavor plus the home team's stadium name in one
// go (a bye or a national team has no home stadium, so that column can be NULL).
const GetCupTieFlavorStmtSql = "SELECT ct.weather, ct.attendance, ct.referee_name, ct.assistant1_name, ct.assistant2_name, t.stadium_name " +
	"FROM cup_ties ct LEFT JOIN teams t ON t.team_id = ct.home_team_id WHERE ct.tie_id = $1"

const GetCupTieEventsStmtSql = "SELECT event_type, team_owner, other_team, minute, team_players, team_player_names, home_goals, away_goals, set_piece " +
	"FROM cup_tie_events WHERE tie_id = $1 ORDER BY minute ASC"

// CreateCup starts a new single-elimination cup for a country's season.
func CreateCup(dbCtx *DatabaseContext, country string, season int, name string) (int64, error) {
	var cupId int64
	err := dbCtx.Db.QueryRow(InsertCupStmtSql, country, season, name).Scan(&cupId)
	return cupId, err
}

// CreateInternationalCup starts a new single-elimination cup for a region's season - the
// international-cup counterpart to CreateCup (issue #64).
func CreateInternationalCup(dbCtx *DatabaseContext, region string, season int, name string) (int64, error) {
	var cupId int64
	err := dbCtx.Db.QueryRow(InsertInternationalCupStmtSql, region, season, name).Scan(&cupId)
	return cupId, err
}

// CreateCupTie inserts one bracket slot and returns its tie id. A nil awayTeamId with a non-nil
// homeTeamId is a bye: it's recorded as already won by the home team, with no match to simulate.
// The returned tie id lets a caller that just created a real (non-bye) tie immediately act on it
// - see engine.createRoundTies, which uses it to auto-select a lineup for either side that turns
// out to be a national team (see GetNationalTeamCountry/SetNationalTeamCupTieLineup).
func CreateCupTie(dbCtx *DatabaseContext, cupId int64, round int, slotNumber int, homeTeamId *int64, awayTeamId *int64, matchDate *time.Time) (int64, error) {
	var winnerTeamId *int64
	played := false
	if homeTeamId != nil && awayTeamId == nil {
		winnerTeamId = homeTeamId
		played = true
	}
	var tieId int64
	err := dbCtx.Db.QueryRow(InsertCupTieStmtSql, cupId, round, slotNumber, homeTeamId, awayTeamId, matchDate, played, winnerTeamId).Scan(&tieId)
	return tieId, err
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
		var country, region sql.NullString
		var championTeamId sql.NullInt64
		if err := rows.Scan(&cup.CupId, &country, &region, &cup.Season, &cup.Name, &cup.Completed, &championTeamId); err != nil {
			return nil, err
		}
		cup.Country = country.String
		cup.Region = region.String
		if championTeamId.Valid {
			cup.ChampionTeamId = &championTeamId.Int64
		}
		cups = append(cups, cup)
	}
	return cups, nil
}

func GetCup(dbCtx *DatabaseContext, cupId int64) (*Cup, error) {
	var cup Cup
	var country, region sql.NullString
	var championTeamId sql.NullInt64
	var championTeamName sql.NullString
	err := dbCtx.Db.QueryRow(GetCupStmtSql, cupId).Scan(&cup.CupId, &country, &region, &cup.Season, &cup.Name,
		&cup.Completed, &championTeamId, &championTeamName)
	if err != nil {
		return nil, err
	}
	cup.Country = country.String
	cup.Region = region.String
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

// GetLatestCupForRegion is GetLatestCupForCountry's international-cup counterpart (issue #64).
func GetLatestCupForRegion(dbCtx *DatabaseContext, region string) (int64, error) {
	var cupId int64
	err := dbCtx.Db.QueryRow(GetLatestCupForRegionStmtSql, region).Scan(&cupId)
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

// GetCupTieById fetches a single tie, or sql.ErrNoRows if tieId doesn't exist.
func GetCupTieById(dbCtx *DatabaseContext, tieId int64) (*CupTie, error) {
	return scanCupTie(dbCtx.Db.QueryRow(GetCupTieByIdStmtSql, tieId))
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

// IsCupRoundResolved reports whether every tie in a round has a winner (played, or a bye) whose
// match, if any, is no longer inside its live-broadcast window (see LiveBroadcastWindow).
func IsCupRoundResolved(dbCtx *DatabaseContext, cupId int64, round int) (bool, error) {
	var unresolved int
	err := dbCtx.Db.QueryRow(GetUnresolvedCupTieCountStmtSql, cupId, round, LiveBroadcastCutoff()).Scan(&unresolved)
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
		var matchDate time.Time
		if err := rows.Scan(&tie.TieId, &tie.CupId, &homeTeamId, &awayTeamId, &matchDate); err != nil {
			return nil, err
		}
		tie.HomeTeamId = &homeTeamId
		tie.AwayTeamId = &awayTeamId
		tie.MatchDate = &matchDate
		ties = append(ties, tie)
	}
	return ties, nil
}

// SetCupTiePlayed records a simulated tie's result and its events.
func SetCupTiePlayed(dbCtx *DatabaseContext, tieId int64, homeGoals int, awayGoals int, wentToPenalties bool, winnerTeamId int64, events []Event, instants []MatchInstant) error {
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
			teamPlayersJson, teamPlayerNamesJson, e.HomeGoals, e.AwayGoals, e.SetPiece); err != nil {
			return err
		}
	}

	insertInstant, err := dbCtx.Db.Prepare(InsertCupTieInstantStmtSql)
	if err != nil {
		return err
	}
	defer insertInstant.Close()

	for _, i := range instants {
		homeQuaJson, _ := json.Marshal(i.HomeQua)
		awayQuaJson, _ := json.Marshal(i.AwayQua)
		if _, err := insertInstant.Exec(tieId, i.Minute, i.HomeGoals, i.AwayGoals, homeQuaJson, awayQuaJson, i.MidfieldDiff,
			i.HomeAttackRightDiff, i.HomeAttackLeftDiff, i.HomeAttackCenterDiff,
			i.AwayAttackRightDiff, i.AwayAttackLeftDiff, i.AwayAttackCenterDiff); err != nil {
			return err
		}
	}
	return nil
}

// GetCupTieInstants reads back a played cup tie's per-minute zone instants (see SetCupTiePlayed) -
// the cup equivalent of GetMatchInstants / GetFriendlyInstants.
func GetCupTieInstants(dbCtx *DatabaseContext, tieId int64) (*[]MatchInstant, error) {
	rows, err := dbCtx.Db.Query(GetCupTieInstantsStmtSql, tieId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	instants := make([]MatchInstant, 0)
	for rows.Next() {
		var i MatchInstant
		var homeQua, awayQua string
		if err := rows.Scan(&i.MatchId, &i.Minute, &i.HomeGoals, &i.AwayGoals, &homeQua, &awayQua, &i.MidfieldDiff,
			&i.HomeAttackRightDiff, &i.HomeAttackLeftDiff, &i.HomeAttackCenterDiff,
			&i.AwayAttackRightDiff, &i.AwayAttackLeftDiff, &i.AwayAttackCenterDiff); err != nil {
			return nil, err
		}
		if homeQua != "" {
			_ = json.Unmarshal([]byte(homeQua), &i.HomeQua)
		}
		if awayQua != "" {
			_ = json.Unmarshal([]byte(awayQua), &i.AwayQua)
		}
		instants = append(instants, i)
	}
	return &instants, rows.Err()
}

const InsertCupTieLineupStmtSql = "INSERT INTO cup_tie_lineup (tie_id, team_id, formation) VALUES ($1, $2, $3) " +
	"ON CONFLICT (tie_id, team_id) DO UPDATE SET formation = EXCLUDED.formation RETURNING lineup_id"
const DeleteCupTieLineupPositionsStmtSql = "DELETE FROM cup_tie_lineup_positions WHERE lineup_id = $1"
const InsertCupTieLineupPositionStmtSql = "INSERT INTO cup_tie_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)"

// SetCupTieLineup replaces teamId's lineup override for one cup tie, along with any
// substitutions planned specifically for it (see cup_tie_lineup_substitutions) - identical shape
// to SetFriendlyLineup/SetLineupForMatch (upsert the (tie_id, team_id) row, replace its positions
// and substitutions outright, guarded by a non-blocking advisory lock scoped to this tie), but
// against cup_tie_lineup/cup_tie_lineup_positions rather than lineup/lineup_positions -
// cup_ties.tie_id is its own id sequence, independent of matches.match_id, so a tie can never
// safely share the league-match-keyed tables (see resources/sql/init38_match_lineup_overrides.sql).
func SetCupTieLineup(dbCtx *DatabaseContext, tieId int64, teamId int64, formation int, positions []LineupPosition, substitutions []LineupSubstitution) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1, $2)", tieId, teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	var lineupId int64
	if err := tx.QueryRow(InsertCupTieLineupStmtSql, tieId, teamId, formation).Scan(&lineupId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteCupTieLineupPositionsStmtSql, lineupId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertCupTieLineupPositionStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	for _, p := range positions {
		if _, err := insertStmt.Exec(lineupId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(DeleteCupTieLineupSubstitutionsStmtSql, tieId, teamId); err != nil {
		return err
	}
	insertSubStmt, err := tx.Prepare(InsertCupTieLineupSubstitutionStmtSql)
	if err != nil {
		return err
	}
	defer insertSubStmt.Close()
	for i, s := range substitutions {
		if _, err := insertSubStmt.Exec(tieId, teamId, i+1, s.PlayerOutId, s.PlayerInId, s.Minute); err != nil {
			return err
		}
	}

	return tx.Commit()
}

const GetCupTieLineupStmtSql = "SELECT lineup_id, formation FROM cup_tie_lineup WHERE tie_id = $1 AND team_id = $2"
const GetCupTieLineupPositionsStmtSql = "SELECT player_id, position FROM cup_tie_lineup_positions WHERE lineup_id = $1"

// GetCupTieLineup returns teamId's saved override for one cup tie, including any substitutions
// planned specifically for it (see GetCupTieLineupSubstitutions). hasOverride is false (with a
// zero Lineup) when the team hasn't saved one for this tie yet.
func GetCupTieLineup(dbCtx *DatabaseContext, tieId int64, teamId int64) (Lineup, bool, error) {
	lineup := Lineup{MatchId: tieId, TeamId: teamId, Positions: make([]LineupPosition, 0), Substitutions: make([]LineupSubstitution, 0)}

	var lineupId int64
	err := dbCtx.Db.QueryRow(GetCupTieLineupStmtSql, tieId, teamId).Scan(&lineupId, &lineup.Formation)
	if errors.Is(err, sql.ErrNoRows) {
		return lineup, false, nil
	}
	if err != nil {
		return Lineup{}, false, err
	}

	rows, err := dbCtx.Db.Query(GetCupTieLineupPositionsStmtSql, lineupId)
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

	substitutions, err := GetCupTieLineupSubstitutions(dbCtx, tieId, teamId)
	if err != nil {
		return Lineup{}, false, err
	}
	lineup.Substitutions = substitutions

	return lineup, true, nil
}

// GetLineupsForCupTie resolves both teams' lineups for one cup tie: a team's saved override (see
// SetCupTieLineup) if it has one, falling back to its standing default lineup otherwise - unlike
// GetLineupsForFriendly, which instead falls back to a freshly computed best XI (see
// BestLineupForTeam), since a friendly should be fulfilled by default with the strongest squad
// currently available rather than whatever was last saved for competitive play. Used by both
// simulateCupTie (engine/cups.go) and
// GetCupTieDetails (the played-tie "Lineups" view) - replaces the previous simulateCupTie call to
// GetLineupsForMatch(dbCtx, tie.TieId, ...), which was only ever safe while nothing wrote to
// lineup/lineup_positions (see this file's SetCupTieLineup doc comment).
func GetLineupsForCupTie(dbCtx *DatabaseContext, tieId int64, teamIds []int64) (*map[int64]*Lineup, error) {
	lineups := make(map[int64]*Lineup, len(teamIds))
	for _, teamId := range teamIds {
		override, hasOverride, err := GetCupTieLineup(dbCtx, tieId, teamId)
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
		defaultLineup.MatchId = tieId
		lineups[teamId] = defaultLineup
	}
	return &lineups, nil
}

// CupTieDetails is the full profile shown on a cup tie's page - the tie itself, its event
// timeline, and both teams' lineups/rosters, the cup equivalent of WeekMatch. Only meaningful
// once the tie has two real teams (not a bye) - see handlers.CupTieDetailsHandler.
type CupTieDetails struct {
	CupTie
	Events      []Event          `json:"events"`
	HomeLineup  *Lineup          `json:"homeLineup"`
	AwayLineup  *Lineup          `json:"awayLineup"`
	HomePlayers map[int64]Player `json:"homePlayers"`
	AwayPlayers map[int64]Player `json:"awayPlayers"`
	// Ratings is computed on read (see ComputeMatchRatings), same as WeekMatch.Ratings/
	// FriendlyMatch.Ratings, so a played tie's "Jugadores" tab shows the same per-player ratings a
	// league match or friendly does instead of only the lineups pitch view (issue: cup/friendly
	// match pages missing the league match page's info).
	Ratings *map[int64]float64 `json:"ratings,omitempty"`
	// MatchInstants and the flavor fields below are only filled for a played tie, and only for one
	// simulated after resources/sql/init83_match_viewer_parity.sql (an older tie has none, and the
	// viewer then shows no zone duel / flavor lines) - the cup equivalent of WeekMatch's fields.
	MatchInstants *[]MatchInstant `json:"matchInstants,omitempty"`
	// HomeStadiumName is read at fetch time from the home team, like WeekMatch.HomeStadiumName; empty
	// for a national team, which has none.
	HomeStadiumName string `json:"homeStadiumName,omitempty"`
	Weather         string `json:"weather,omitempty"`
	Attendance      int    `json:"attendance,omitempty"`
	RefereeName     string `json:"refereeName,omitempty"`
	Assistant1Name  string `json:"assistant1Name,omitempty"`
	Assistant2Name  string `json:"assistant2Name,omitempty"`
}

// GetCupTieDetails assembles one cup tie's full detail view - see CupTieDetails.
func GetCupTieDetails(dbCtx *DatabaseContext, tieId int64) (*CupTieDetails, error) {
	tie, err := GetCupTieById(dbCtx, tieId)
	if err != nil {
		return nil, err
	}
	details := &CupTieDetails{CupTie: *tie}

	events, err := GetCupTieEvents(dbCtx, tieId)
	if err != nil {
		return nil, err
	}
	details.Events = events

	if tie.HomeTeamId == nil || tie.AwayTeamId == nil {
		return details, nil
	}

	var weather, refereeName, assistant1, assistant2, stadiumName sql.NullString
	var attendance sql.NullInt64
	if err := dbCtx.Db.QueryRow(GetCupTieFlavorStmtSql, tieId).Scan(&weather, &attendance, &refereeName,
		&assistant1, &assistant2, &stadiumName); err != nil {
		return nil, err
	}
	details.HomeStadiumName = stadiumName.String
	if tie.Played {
		details.Weather = weather.String
		details.Attendance = int(attendance.Int64)
		details.RefereeName = refereeName.String
		details.Assistant1Name = assistant1.String
		details.Assistant2Name = assistant2.String
		instants, err := GetCupTieInstants(dbCtx, tieId)
		if err != nil {
			return nil, err
		}
		if len(*instants) > 0 {
			details.MatchInstants = instants
		}
	}

	lineups, err := GetLineupsForCupTie(dbCtx, tieId, []int64{*tie.HomeTeamId, *tie.AwayTeamId})
	if err != nil {
		return nil, err
	}
	details.HomeLineup = (*lineups)[*tie.HomeTeamId]
	details.AwayLineup = (*lineups)[*tie.AwayTeamId]

	homePlayers, err := GetPlayersForMatchTeam(dbCtx, *tie.HomeTeamId, details.HomeLineup)
	if err != nil {
		return nil, err
	}
	details.HomePlayers = playerListToMap(homePlayers)

	awayPlayers, err := GetPlayersForMatchTeam(dbCtx, *tie.AwayTeamId, details.AwayLineup)
	if err != nil {
		return nil, err
	}
	details.AwayPlayers = playerListToMap(awayPlayers)

	// Mirrors EnrichPlayedFriendly's identical guard - only a played tie has meaningful ratings to
	// show, and ComputeMatchRatings itself needs both lineups to attribute events to a side.
	if tie.Played && details.HomeLineup != nil && details.AwayLineup != nil {
		events := details.Events
		details.Ratings = ComputeMatchRatings(&WeekMatch{
			MatchId:     tie.TieId,
			HomeGoals:   tie.HomeGoals,
			AwayGoals:   tie.AwayGoals,
			Events:      &events,
			HomeLineup:  details.HomeLineup,
			AwayLineup:  details.AwayLineup,
			HomePlayers: &details.HomePlayers,
			AwayPlayers: &details.AwayPlayers,
		})
	}

	return details, nil
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
			&e.HomeGoals, &e.AwayGoals, &e.SetPiece); err != nil {
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
