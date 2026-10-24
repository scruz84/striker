package common

import "time"

// MaxFailingFixturesListed caps the failing-fixture list of the admin overview.
const MaxFailingFixturesListed = 50

// FixtureKindCounts is one count per kind of fixture a simulation can fail on.
type FixtureKindCounts struct {
	LeagueMatches int `json:"leagueMatches"`
	Friendlies    int `json:"friendlies"`
	CupTies       int `json:"cupTies"`
}

// FailingFixture is an unplayed fixture whose simulation has failed at least once.
type FailingFixture struct {
	Kind      string    `json:"kind"`
	Id        int64     `json:"id"`
	HomeName  string    `json:"homeName"`
	AwayName  string    `json:"awayName"`
	MatchDate time.Time `json:"matchDate"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"lastError"`
}

// SimulationHealth is what the admin overview shows about simulations that did not go through:
// fixtures still claimed long after their claim (a crashed pass), fixtures whose simulation failed
// (with how often and why), and classification rows out of step with the matches played.
type SimulationHealth struct {
	// Stuck counts league matches and friendlies claimed at least StuckSimulationAge ago and still
	// unplayed - a live pass works through its claim within minutes. A cup tie has no claim.
	Stuck int `json:"stuck"`
	// FailingCounts is every unplayed fixture with a recorded failure, per kind; Failing lists up to
	// MaxFailingFixturesListed of them, oldest kickoff first.
	FailingCounts       FixtureKindCounts `json:"failingCounts"`
	Failing             []FailingFixture  `json:"failing"`
	StandingsMismatches int               `json:"standingsMismatches"`
}

const countStuckSimulationsStmtSql = "SELECT " +
	"(SELECT count(*) FROM matches WHERE played = false AND in_simulation = true " +
	"AND (in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second'))) + " +
	"(SELECT count(*) FROM friendly_matches WHERE played = false AND in_simulation = true " +
	"AND (in_simulation_since IS NULL OR in_simulation_since <= now() - ($1 * interval '1 second')))"

const countFailingFixturesStmtSql = "SELECT " +
	"(SELECT count(*) FROM matches WHERE played = false AND simulation_attempts > 0 AND last_simulation_error IS NOT NULL), " +
	"(SELECT count(*) FROM friendly_matches WHERE played = false AND simulation_attempts > 0 AND last_simulation_error IS NOT NULL), " +
	"(SELECT count(*) FROM cup_ties WHERE played = false AND simulation_attempts > 0 AND last_simulation_error IS NOT NULL)"

const listFailingFixturesStmtSql = "SELECT kind, id, home_name, away_name, match_date, simulation_attempts, last_simulation_error FROM (" +
	"SELECT 'league' AS kind, m.match_id AS id, ht.name AS home_name, at.name AS away_name, m.match_date, m.simulation_attempts, m.last_simulation_error " +
	"FROM matches m JOIN teams ht ON ht.team_id = m.home_team_id JOIN teams at ON at.team_id = m.away_team_id " +
	"WHERE m.played = false AND m.simulation_attempts > 0 AND m.last_simulation_error IS NOT NULL " +
	"UNION ALL " +
	"SELECT 'friendly', f.match_id, ht.name, at.name, f.match_date, f.simulation_attempts, f.last_simulation_error " +
	"FROM friendly_matches f JOIN teams ht ON ht.team_id = f.home_team_id JOIN teams at ON at.team_id = f.away_team_id " +
	"WHERE f.played = false AND f.simulation_attempts > 0 AND f.last_simulation_error IS NOT NULL " +
	"UNION ALL " +
	"SELECT 'cup', c.tie_id, ht.name, at.name, c.match_date, c.simulation_attempts, c.last_simulation_error " +
	"FROM cup_ties c JOIN teams ht ON ht.team_id = c.home_team_id JOIN teams at ON at.team_id = c.away_team_id " +
	"WHERE c.played = false AND c.simulation_attempts > 0 AND c.last_simulation_error IS NOT NULL" +
	") failing ORDER BY match_date, kind, id LIMIT $1"

// GetSimulationHealth reads the admin overview's simulation figures. Replaces the old count of
// every claimed row, which also counted live claims and missed failed fixtures and cup ties.
func GetSimulationHealth(dbCtx *DatabaseContext) (SimulationHealth, error) {
	health := SimulationHealth{Failing: make([]FailingFixture, 0)}
	if err := dbCtx.Db.QueryRow(countStuckSimulationsStmtSql, StuckSimulationAge.Seconds()).Scan(&health.Stuck); err != nil {
		return health, err
	}
	if err := dbCtx.Db.QueryRow(countFailingFixturesStmtSql).Scan(
		&health.FailingCounts.LeagueMatches, &health.FailingCounts.Friendlies, &health.FailingCounts.CupTies); err != nil {
		return health, err
	}
	rows, err := dbCtx.Db.Query(listFailingFixturesStmtSql, MaxFailingFixturesListed)
	if err != nil {
		return health, err
	}
	defer rows.Close()
	for rows.Next() {
		var f FailingFixture
		if err := rows.Scan(&f.Kind, &f.Id, &f.HomeName, &f.AwayName, &f.MatchDate, &f.Attempts, &f.LastError); err != nil {
			return health, err
		}
		health.Failing = append(health.Failing, f)
	}
	if err := rows.Err(); err != nil {
		return health, err
	}
	health.StandingsMismatches, err = CountStandingsMismatches(dbCtx)
	return health, err
}

// CountPlayedFixtures counts how many of the given fixture ids of kind (one of the
// SimulationKind constants) are now played - what an admin-triggered re-simulation resolved, as
// opposed to merely attempted.
func CountPlayedFixtures(dbCtx *DatabaseContext, kind string, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	table, column := "matches", "match_id"
	switch kind {
	case SimulationKindFriendly:
		table = "friendly_matches"
	case SimulationKindCup:
		table, column = "cup_ties", "tie_id"
	}
	var count int
	err := dbCtx.Db.QueryRow("SELECT count(*) FROM " + table + " WHERE played = true AND " + column + " IN " + int64SliceToInClause(ids)).Scan(&count)
	return count, err
}
