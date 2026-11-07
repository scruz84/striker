package common

import (
	"database/sql"
	"time"
)

// WorldCupEveryInternationalCycles is how many international cup cycles pass between World Cups:
// one starts whenever the international cup counter reaches a multiple of this (see
// engine.StartWorldCupForNewSeason, whose WorldCupFrequencySeasons is this value).
const WorldCupEveryInternationalCycles = 4

// InternationalCupCycle is the real time between two international cycles, the World Cup's clock
// (see engine.StartWorldCupIfDue, whose WorldCupCycleCadence is this value). Regional club cups do
// not follow it (see engine.StartRegionalCupsIfDue).
const InternationalCupCycle = 60 * 24 * time.Hour

const getWorldCupHistoryStmtSql = "SELECT wc.world_cup_id, wc.season, wc.name, wc.stage, wc.completed, wc.champion_team_id, wc.knockout_cup_id, " +
	"COALESCE(t.name, '') " +
	"FROM world_cups wc LEFT JOIN teams t ON t.team_id = wc.champion_team_id " +
	"ORDER BY wc.world_cup_id DESC"

// GetWorldCupHistory lists every World Cup, newest first - the roll of honour on the World Cup
// tab. ChampionTeamName is empty until a World Cup is completed.
func GetWorldCupHistory(dbCtx *DatabaseContext) ([]WorldCup, error) {
	rows, err := dbCtx.Db.Query(getWorldCupHistoryStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]WorldCup, 0)
	for rows.Next() {
		wc, err := scanWorldCup(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, *wc)
	}
	return history, rows.Err()
}

const getInternationalCupCounterStmtSql = "SELECT international_cup_season, last_international_cup_check FROM season LIMIT 1"

// nextWorldCupStart estimates when the next World Cup starts: the international cup counter
// advances once per InternationalCupCycle from lastCheck, and a World Cup starts when it reaches
// the next multiple of WorldCupEveryInternationalCycles above counter.
func nextWorldCupStart(counter int, lastCheck time.Time) time.Time {
	cycles := WorldCupEveryInternationalCycles - counter%WorldCupEveryInternationalCycles
	return lastCheck.Add(time.Duration(cycles) * InternationalCupCycle)
}

// GetNextWorldCupEstimate is nextWorldCupStart for the current counter, or nil when fewer than
// MinWorldCupCountries countries are active (no World Cup would start) or the counter is unset.
// Callers only ask while no World Cup is running.
func GetNextWorldCupEstimate(dbCtx *DatabaseContext) (*time.Time, error) {
	active, err := GetActiveCountries(dbCtx)
	if err != nil {
		return nil, err
	}
	if len(active) < MinWorldCupCountries {
		return nil, nil
	}
	var counter int
	var lastCheck sql.NullTime
	err = dbCtx.Db.QueryRow(getInternationalCupCounterStmtSql).Scan(&counter, &lastCheck)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	from := time.Now()
	if lastCheck.Valid {
		from = lastCheck.Time
	}
	estimate := nextWorldCupStart(counter, from)
	return &estimate, nil
}
