package common

import (
	"database/sql"
	"time"
)

const getLatestMatchDateBeforeSeasonStmtSql = "SELECT max(match_date) FROM matches WHERE season < $1"

const getLatestStartedMatchDateStmtSql = "SELECT max(match_date) FROM matches " +
	"WHERE league_id = $1 AND season = $2 AND (played = true OR in_simulation = true)"

// GetLatestMatchDateBeforeSeason returns the latest match_date among league matches of seasons
// before `season`, or nil if there are none. A match flips to played 15 minutes before its
// match_date (see GetSimulableMatches), so a season can close while its last round is still
// dated in the future - the next season's first round has to be scheduled after it, not on top.
func GetLatestMatchDateBeforeSeason(dbCtx *DatabaseContext, season int) (*time.Time, error) {
	return queryOptionalTime(dbCtx, getLatestMatchDateBeforeSeasonStmtSql, season)
}

// GetLatestStartedMatchDate returns the latest match_date among leagueId's already simulated (or
// being simulated) matches of `season`, or nil if none has started - the earliest a rescheduled
// round may be dated without landing on top of one of them.
func GetLatestStartedMatchDate(dbCtx *DatabaseContext, leagueId int64, season int) (*time.Time, error) {
	return queryOptionalTime(dbCtx, getLatestStartedMatchDateStmtSql, leagueId, season)
}

func queryOptionalTime(dbCtx *DatabaseContext, query string, args ...any) (*time.Time, error) {
	var latest sql.NullTime
	if err := dbCtx.Db.QueryRow(query, args...).Scan(&latest); err != nil {
		return nil, err
	}
	if !latest.Valid {
		return nil, nil
	}
	return &latest.Time, nil
}
