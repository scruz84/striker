package common

import (
	"errors"
	"time"
)

// The instant archive (openspec/specs/match-data-retention): an optional, admin-enabled pass that
// deletes the per-minute instants of fixtures older than the seasons kept, so the three instants
// tables stop growing without bound. Every other piece of a match - result, events, ratings,
// lineups, flavor - stays. Each archived fixture is flagged instants_archived (see
// init118_instant_archive.sql), so no later pass looks at it again.

// InstantArchiveMaxKeepSeasons bounds the seasons kept (0 archives everything but the current
// season).
const InstantArchiveMaxKeepSeasons = 10

// InstantArchiveBatch is how many fixtures one archive statement handles, so each statement's
// transaction (about 96 instant rows per fixture) stays short.
const InstantArchiveBatch = 200

var ErrInvalidInstantArchiveKeepSeasons = errors.New("seasons kept must be between 0 and 10")

// InstantArchiveSettings is the global archive setting: whether the daily pass runs, and how many
// finished seasons keep their instants.
type InstantArchiveSettings struct {
	Enabled     bool `json:"enabled"`
	KeepSeasons int  `json:"keepSeasons"`
}

const GetInstantArchiveSettingsStmtSql = "SELECT archive_instants_enabled, archive_keep_seasons FROM season"
const SetInstantArchiveSettingsStmtSql = "UPDATE season SET archive_instants_enabled = $1, archive_keep_seasons = $2"

func GetInstantArchiveSettings(dbCtx *DatabaseContext) (InstantArchiveSettings, error) {
	var s InstantArchiveSettings
	err := dbCtx.Db.QueryRow(GetInstantArchiveSettingsStmtSql).Scan(&s.Enabled, &s.KeepSeasons)
	return s, err
}

// SetInstantArchiveSettings saves the setting. Turning it off only stops future passes: instants
// already deleted are not restored.
func SetInstantArchiveSettings(dbCtx *DatabaseContext, s InstantArchiveSettings) error {
	if s.KeepSeasons < 0 || s.KeepSeasons > InstantArchiveMaxKeepSeasons {
		return ErrInvalidInstantArchiveKeepSeasons
	}
	_, err := dbCtx.Db.Exec(SetInstantArchiveSettingsStmtSql, s.Enabled, s.KeepSeasons)
	return err
}

// Each archive statement flags a batch of eligible fixtures archived and deletes their instants in
// one statement, returning how many fixtures it flagged and how many instant rows it deleted. Rows
// another replica holds are skipped, so concurrent passes split the work. In the league and cup
// statements $1 is the seasons kept and $2 the batch size.

// ArchiveLeagueMatchInstantsStmtSql: played league matches of season <= the country's current
// season - 1 - kept.
const ArchiveLeagueMatchInstantsStmtSql = "WITH batch AS (" +
	"UPDATE matches SET instants_archived = true WHERE match_id IN (" +
	"SELECT m.match_id FROM matches m JOIN leagues l ON l.league_id = m.league_id JOIN countries c ON c.name = l.country " +
	"WHERE m.played AND NOT m.instants_archived AND m.season <= c.current_season - 1 - $1 " +
	"ORDER BY m.match_id LIMIT $2 FOR UPDATE OF m SKIP LOCKED) RETURNING match_id), " +
	"del AS (DELETE FROM match_instants WHERE match_id IN (SELECT match_id FROM batch) RETURNING 1) " +
	"SELECT (SELECT count(*) FROM batch), (SELECT count(*) FROM del)"

// ArchiveCupTieInstantsStmtSql: played ties of completed cups. A domestic cup or playoff counts
// its country's season, a regional or international cup its edition against its region's newest.
const ArchiveCupTieInstantsStmtSql = "WITH batch AS (" +
	"UPDATE cup_ties SET instants_archived = true WHERE tie_id IN (" +
	"SELECT ct.tie_id FROM cup_ties ct JOIN cups cu ON cu.cup_id = ct.cup_id LEFT JOIN countries c ON c.name = cu.country " +
	"WHERE ct.played AND NOT ct.instants_archived AND cu.completed AND (" +
	"(cu.country IS NOT NULL AND cu.season <= c.current_season - 1 - $1) OR " +
	"(cu.region IS NOT NULL AND cu.season <= (SELECT max(r.season) FROM cups r WHERE r.region = cu.region) - 1 - $1)) " +
	"ORDER BY ct.tie_id LIMIT $2 FOR UPDATE OF ct SKIP LOCKED) RETURNING tie_id), " +
	"del AS (DELETE FROM cup_tie_instants WHERE tie_id IN (SELECT tie_id FROM batch) RETURNING 1) " +
	"SELECT (SELECT count(*) FROM batch), (SELECT count(*) FROM del)"

// FriendlyArchiveCutoffsStmtSql is, per country, the first league kickoff of its oldest kept
// season (current season - kept): a friendly its clubs hosted before it is archived. A country
// with no league match in that season has no cutoff, so none of its friendlies is archived.
const FriendlyArchiveCutoffsStmtSql = "SELECT l.country, min(m.match_date) FROM matches m " +
	"JOIN leagues l ON l.league_id = m.league_id JOIN countries c ON c.name = l.country " +
	"WHERE m.season = c.current_season - $1 GROUP BY l.country"

// ArchiveFriendlyInstantsStmtSql: played friendlies whose home team's country has a cutoff ($2
// countries, $3 cutoffs, paired by position) later than their kickoff. $1 is the batch size: the
// seasons kept are already in the cutoffs.
const ArchiveFriendlyInstantsStmtSql = "WITH batch AS (" +
	"UPDATE friendly_matches SET instants_archived = true WHERE match_id IN (" +
	"SELECT f.match_id FROM friendly_matches f JOIN teams t ON t.team_id = f.home_team_id " +
	"JOIN unnest($2::varchar[], $3::timestamptz[]) AS cut(country, start_at) ON cut.country = t.country " +
	"WHERE f.played AND NOT f.instants_archived AND f.match_date < cut.start_at " +
	"ORDER BY f.match_id LIMIT $1 FOR UPDATE OF f SKIP LOCKED) RETURNING match_id), " +
	"del AS (DELETE FROM friendly_instants WHERE match_id IN (SELECT match_id FROM batch) RETURNING 1) " +
	"SELECT (SELECT count(*) FROM batch), (SELECT count(*) FROM del)"

// InstantArchiveResult counts what one pass archived: fixtures per kind and instant rows deleted.
type InstantArchiveResult struct {
	LeagueMatches int64
	CupTies       int64
	Friendlies    int64
	Instants      int64
}

// archiveBatches runs one archive statement until a batch flags no fixture, adding to the totals.
func archiveBatches(dbCtx *DatabaseContext, stmt string, fixtures *int64, instants *int64, args ...any) error {
	for {
		var flagged, deleted int64
		if err := dbCtx.Db.QueryRow(stmt, args...).Scan(&flagged, &deleted); err != nil {
			return err
		}
		*fixtures += flagged
		*instants += deleted
		if flagged == 0 {
			return nil
		}
	}
}

// friendlyArchiveCutoffs reads FriendlyArchiveCutoffsStmtSql as two parallel slices.
func friendlyArchiveCutoffs(dbCtx *DatabaseContext, keepSeasons int) ([]string, []time.Time, error) {
	rows, err := dbCtx.Db.Query(FriendlyArchiveCutoffsStmtSql, keepSeasons)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var countries []string
	var cutoffs []time.Time
	for rows.Next() {
		var country string
		var cutoff time.Time
		if err := rows.Scan(&country, &cutoff); err != nil {
			return nil, nil, err
		}
		countries = append(countries, country)
		cutoffs = append(cutoffs, cutoff)
	}
	return countries, cutoffs, rows.Err()
}

// ArchiveOldInstants is one archive pass with keepSeasons finished seasons kept: league matches,
// then cup ties, then friendlies, each batch by batch until nothing is left. Safe to run on
// several replicas at once. It returns what it archived, even when it stops on an error.
func ArchiveOldInstants(dbCtx *DatabaseContext, keepSeasons int) (InstantArchiveResult, error) {
	var res InstantArchiveResult
	if keepSeasons < 0 || keepSeasons > InstantArchiveMaxKeepSeasons {
		return res, ErrInvalidInstantArchiveKeepSeasons
	}
	if err := archiveBatches(dbCtx, ArchiveLeagueMatchInstantsStmtSql, &res.LeagueMatches, &res.Instants, keepSeasons, InstantArchiveBatch); err != nil {
		return res, err
	}
	if err := archiveBatches(dbCtx, ArchiveCupTieInstantsStmtSql, &res.CupTies, &res.Instants, keepSeasons, InstantArchiveBatch); err != nil {
		return res, err
	}
	countries, cutoffs, err := friendlyArchiveCutoffs(dbCtx, keepSeasons)
	if err != nil || len(countries) == 0 {
		return res, err
	}
	err = archiveBatches(dbCtx, ArchiveFriendlyInstantsStmtSql, &res.Friendlies, &res.Instants, InstantArchiveBatch, countries, cutoffs)
	return res, err
}
