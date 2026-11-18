package common

import "math/rand"

// LegendNameChance is the probability that a newly generated player is given a legend name
// (see legend_names) instead of a name drawn from the country's regular name pool. Kept low - a
// legend name is meant to be a rare, memorable easter egg, not a common occurrence.
const LegendNameChance = 0.005

// GetRandomLegendNameStmtSql draws one (first_name, last_name) pair from legend_names (see
// resources/sql/history/init58_legend_names.sql) - a single random-order query rather than two
// independent ones (contrast GetRandomName's separate names_<suffix>/lastnames_<suffix> draws),
// since a legend's first and last name have to stay paired ("Lionel" always with "Messi", never
// mixed with an unrelated last name).
const GetRandomLegendNameStmtSql = "SELECT first_name, last_name FROM legend_names ORDER BY random() LIMIT 1"

// GetPlayerName picks a name for a newly generated player: with LegendNameChance odds it draws
// one from legend_names, otherwise it falls back to GetRandomName's usual country-pool draw.
// Callers that generate players (doCreatePlayers, InjectFreeAgents) should use this instead of
// calling GetRandomName directly, so legends can turn up anywhere a new player is created.
func GetPlayerName(dbCtx *DatabaseContext, country string) (string, string, error) {
	if rand.Float64() < LegendNameChance {
		var firstName, lastName string
		if err := dbCtx.Db.QueryRow(GetRandomLegendNameStmtSql).Scan(&firstName, &lastName); err == nil {
			return firstName, lastName, nil
		}
		// legend_names is a fixed, always-populated seed table (see the migration above), so a
		// query error here should be impossible in practice - but a new player is never worth
		// blocking on this easter egg, so fall through to the ordinary country-pool draw instead
		// of propagating the error.
	}
	return GetRandomName(dbCtx, country)
}
