package common

import (
	"context"
	"time"
)

// RegionalCupCountryState is one country's standing towards its region's next international club
// cup edition (see engine.StartRegionalCupsIfDue, openspec/specs/international-cup).
type RegionalCupCountryState struct {
	Country string
	// LastEnteredSeason is the newest domestic season whose champion this country already sent to
	// the cup (international_cup_entries), 0 if it never entered.
	LastEnteredSeason int
	// ChampionSeason/ChampionTeamId are its most recent tier-1 champion, 0 when it has not
	// finished a season yet.
	ChampionSeason int
	ChampionTeamId int64
	// RunnerUpTeamId is the runner-up (final_position 2) of that same league and season, 0 when
	// none is recorded.
	RunnerUpTeamId int64
	// UnusedChampions counts its tier-1 seasons with a champion newer than LastEnteredSeason.
	UnusedChampions int
}

// RegionalCupEntrant is one team a fresh country sends to an edition: its champion
// (FinalPosition 1) or runner-up (2) of the tier-1 season Season.
type RegionalCupEntrant struct {
	Country       string
	Season        int
	TeamId        int64
	FinalPosition int
}

// Fresh reports whether the country has a champion that has not played the cup yet.
func (s RegionalCupCountryState) Fresh() bool {
	return s.ChampionTeamId != 0 && s.ChampionSeason > s.LastEnteredSeason
}

// GetRegionalCupCountryStatesStmtSql returns one row per country in $2 that has a tier-1 ($1)
// league. The latest champion is the newest season's final_position = 1 row, lowest league_id
// among several tier-1 groups; the runner-up is that league and season's final_position = 2 row.
const GetRegionalCupCountryStatesStmtSql = `
WITH tier1 AS (
	SELECT DISTINCT country FROM leagues WHERE tier = $1 AND country = ANY($2) AND archived_season IS NULL
), champions AS (
	SELECT l.country, sh.season, sh.team_id, l.league_id
	FROM season_history sh JOIN leagues l ON l.league_id = sh.league_id
	WHERE l.tier = $1 AND sh.final_position = 1 AND l.country = ANY($2)
), entered AS (
	SELECT country, max(season) AS season FROM international_cup_entries
	WHERE country = ANY($2) GROUP BY country
)
SELECT t.country, COALESCE(e.season, 0), COALESCE(latest.season, 0), COALESCE(latest.team_id, 0),
	COALESCE(runner_up.team_id, 0),
	(SELECT count(DISTINCT c.season) FROM champions c WHERE c.country = t.country AND c.season > COALESCE(e.season, 0))
FROM tier1 t
LEFT JOIN entered e ON e.country = t.country
LEFT JOIN LATERAL (
	SELECT c.season, c.team_id, c.league_id FROM champions c WHERE c.country = t.country
	ORDER BY c.season DESC, c.league_id ASC LIMIT 1
) latest ON true
LEFT JOIN LATERAL (
	SELECT sh.team_id FROM season_history sh
	WHERE sh.league_id = latest.league_id AND sh.season = latest.season AND sh.final_position = 2
	ORDER BY sh.team_id LIMIT 1
) runner_up ON true
ORDER BY t.country`

// GetRegionalCupCountryStates reads RegionalCupCountryState for every country in countries that
// has a league of the given tier; a country without one is left out.
func GetRegionalCupCountryStates(dbCtx *DatabaseContext, tier int, countries []string) ([]RegionalCupCountryState, error) {
	rows, err := dbCtx.Db.Query(GetRegionalCupCountryStatesStmtSql, tier, countries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make([]RegionalCupCountryState, 0, len(countries))
	for rows.Next() {
		var s RegionalCupCountryState
		if err := rows.Scan(&s.Country, &s.LastEnteredSeason, &s.ChampionSeason, &s.ChampionTeamId, &s.RunnerUpTeamId, &s.UnusedChampions); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

const HasActiveRegionalCupStmtSql = "SELECT EXISTS (SELECT 1 FROM cups WHERE region = $1 AND completed = false)"

// HasActiveRegionalCup reports whether region has an edition that has not finished yet.
func HasActiveRegionalCup(dbCtx *DatabaseContext, region string) (bool, error) {
	var active bool
	err := dbCtx.Db.QueryRow(HasActiveRegionalCupStmtSql, region).Scan(&active)
	return active, err
}

// InsertRegionalCupStmtSql numbers the edition within its region: one more than the region's
// highest season so far.
const InsertRegionalCupStmtSql = "INSERT INTO cups (country, region, season, name) " +
	"SELECT NULL, $1::varchar, COALESCE(max(season), 0) + 1, $2 FROM cups WHERE region = $1::varchar RETURNING cup_id, season"

const InsertInternationalCupEntryStmtSql = "INSERT INTO international_cup_entries (cup_id, country, season, team_id, final_position) VALUES ($1, $2, $3, $4, $5)"

// RegionalCupTieSeed is one first-round bracket slot: AwayTeamId nil is a bye for HomeTeamId.
// MatchDate is the tie's kickoff (20:45 in the home side's country, see engine.cupTieKickoff),
// ignored for a bye.
type RegionalCupTieSeed struct {
	HomeTeamId int64
	AwayTeamId *int64
	MatchDate  time.Time
}

// CreateRegionalCup writes a new edition of region's cup in one transaction: the cups row
// (numbered within the region), one international_cup_entries row per entrant and the first
// round's ties, real ties kicking off at their own MatchDate and byes stored as already won (as
// CreateCupTie does). A country may send two entrants (its champion and runner-up). It returns the cup id and its edition number. Nothing is left behind on
// failure, so a half-written edition can never block its region (see HasActiveRegionalCup).
func CreateRegionalCup(dbCtx *DatabaseContext, region string, name string, entrants []RegionalCupEntrant, ties []RegionalCupTieSeed) (int64, int, error) {
	tx, err := dbCtx.Db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var cupId int64
	var edition int
	if err := tx.QueryRow(InsertRegionalCupStmtSql, region, name).Scan(&cupId, &edition); err != nil {
		return 0, 0, err
	}
	for _, e := range entrants {
		if _, err := tx.Exec(InsertInternationalCupEntryStmtSql, cupId, e.Country, e.Season, e.TeamId, e.FinalPosition); err != nil {
			return 0, 0, err
		}
	}
	for i, tie := range ties {
		home := tie.HomeTeamId
		var scheduledAt *time.Time
		var winner *int64
		played := false
		if tie.AwayTeamId != nil {
			matchDate := tie.MatchDate
			scheduledAt = &matchDate
		} else {
			winner, played = &home, true
		}
		var tieId int64
		if err := tx.QueryRow(InsertCupTieStmtSql, cupId, 1, i+1, home, tie.AwayTeamId, scheduledAt, played, winner).Scan(&tieId); err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return cupId, edition, nil
}

const GetInternationalCupEntriesStmtSql = "SELECT country, season, team_id, final_position FROM international_cup_entries " +
	"WHERE cup_id = $1 ORDER BY country, final_position"

// InternationalCupEntry is one recorded entrant of a regional cup edition: FinalPosition 1 is its
// country's champion, 2 its runner-up.
type InternationalCupEntry struct {
	Country       string
	Season        int
	TeamId        int64
	FinalPosition int
}

// GetInternationalCupEntries lists cupId's recorded entrants, by country and then final position.
func GetInternationalCupEntries(dbCtx *DatabaseContext, cupId int64) ([]InternationalCupEntry, error) {
	rows, err := dbCtx.Db.Query(GetInternationalCupEntriesStmtSql, cupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]InternationalCupEntry, 0)
	for rows.Next() {
		var e InternationalCupEntry
		if err := rows.Scan(&e.Country, &e.Season, &e.TeamId, &e.FinalPosition); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
