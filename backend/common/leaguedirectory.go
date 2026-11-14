package common

// LeagueDirectoryEntry is one league in the browsable directory of every league in the game
// world - just enough to list and link into it (see handlers.LeagueDirectoryHandler and
// GetDivisionDetailsStmtSql for the actual classification/teams once picked).
type LeagueDirectoryEntry struct {
	LeagueId    int64  `json:"leagueId"`
	Name        string `json:"name"`
	Tier        int    `json:"tier"`
	GroupNumber int    `json:"groupNumber"`
}

// CountryLeagueDirectory groups a country's leagues for the directory - one entry per active
// country, in GetLeagueDirectory's country order.
type CountryLeagueDirectory struct {
	Country string                 `json:"country"`
	Leagues []LeagueDirectoryEntry `json:"leagues"`
}

const getLeagueDirectoryStmtSql = "SELECT league_id, country, name, tier, group_number " +
	"FROM leagues WHERE archived_season IS NULL ORDER BY country, tier, group_number"

// GetLeagueDirectory lists every league in the game world, grouped by country - lets any
// logged-in user browse into a league they don't play in via GetDivisionDetailsStmtSql, not just
// their own. Deliberately public information (standings and rosters, like real football) as
// opposed to a team's finances or contract terms, which stay owner-only - see
// handlers.PlayerDetailsHandler and GetTeamDetailsById.
func GetLeagueDirectory(dbCtx *DatabaseContext) ([]CountryLeagueDirectory, error) {
	rows, err := dbCtx.Db.Query(getLeagueDirectoryStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	directory := make([]CountryLeagueDirectory, 0)
	var current *CountryLeagueDirectory
	for rows.Next() {
		var entry LeagueDirectoryEntry
		var country string
		if err := rows.Scan(&entry.LeagueId, &country, &entry.Name, &entry.Tier, &entry.GroupNumber); err != nil {
			return nil, err
		}
		if current == nil || current.Country != country {
			directory = append(directory, CountryLeagueDirectory{Country: country})
			current = &directory[len(directory)-1]
		}
		current.Leagues = append(current.Leagues, entry)
	}
	return directory, rows.Err()
}
