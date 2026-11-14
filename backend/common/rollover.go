package common

import (
	"database/sql"
	"encoding/json"
)

// A country's season-rollover progress marker (countries.rollover_season/rollover_step/
// rollover_state, see init116_resumable_season_rollover.sql and openspec/specs/engine-replicas
// "Resumable season rollover"). Only engine.advanceCountrySeasonIfComplete reads or writes it,
// always under the country's rollover lock.

// RolloverProgress is the marker as stored: Season is 0 when no rollover is in progress, Step is
// the last completed step (0 for none) and State the JSON the engine keeps between steps.
type RolloverProgress struct {
	Season int
	Step   int
	State  json.RawMessage
}

const getRolloverProgressStmtSql = "SELECT COALESCE(rollover_season, 0), rollover_step, rollover_state FROM countries WHERE name = $1"
const setRolloverProgressStmtSql = "UPDATE countries SET rollover_season = $2, rollover_step = $3, rollover_state = $4 WHERE name = $1"
const clearRolloverProgressStmtSql = "UPDATE countries SET rollover_season = NULL, rollover_step = 0, rollover_state = NULL WHERE name = $1"

// advanceSeasonRecordingStepStmtSql moves the country's current season and records the season
// bump as completed in the same statement, so the two can never disagree after a stop.
const advanceSeasonRecordingStepStmtSql = "UPDATE countries SET current_season = $2, rollover_step = $3 WHERE name = $1"

// GetRolloverProgress reads country's rollover progress marker.
func GetRolloverProgress(dbCtx *DatabaseContext, country string) (RolloverProgress, error) {
	var progress RolloverProgress
	var state []byte
	err := dbCtx.Db.QueryRow(getRolloverProgressStmtSql, country).Scan(&progress.Season, &progress.Step, &state)
	if err != nil {
		return RolloverProgress{}, err
	}
	if state != nil {
		progress.State = json.RawMessage(state)
	}
	return progress, nil
}

// SetRolloverProgress records that country's rollover of season has completed step, with the
// state later steps need (nil for none).
func SetRolloverProgress(dbCtx *DatabaseContext, country string, season int, step int, state json.RawMessage) error {
	var stateValue any
	if state != nil {
		stateValue = []byte(state)
	}
	_, err := dbCtx.Db.Exec(setRolloverProgressStmtSql, country, season, step, stateValue)
	return err
}

// AdvanceSeasonRecordingStep sets country's current season to newSeason and records step (the
// season bump) as completed, in one statement.
func AdvanceSeasonRecordingStep(dbCtx *DatabaseContext, country string, newSeason int, step int) error {
	result, err := dbCtx.Db.Exec(advanceSeasonRecordingStepStmtSql, country, newSeason, step)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ClearRolloverProgress removes country's marker once its rollover has finished.
func ClearRolloverProgress(dbCtx *DatabaseContext, country string) error {
	_, err := dbCtx.Db.Exec(clearRolloverProgressStmtSql, country)
	return err
}
