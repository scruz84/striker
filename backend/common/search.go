package common

import "strconv"

// Search powers the frontend's global search page: look up a team, player, coach, or stadium by
// (partial, case-insensitive) name and jump straight to wherever that thing is already shown -
// there's no dedicated coach/stadium page, so those two results link to the team that employs/
// owns them, same as TeamDetailsPage.StadiumCard and CoachCard already do.

// SearchResultLimit caps each category independently so a broad query (e.g. a single common
// letter) still returns promptly and the results list stays scannable.
const SearchResultLimit = 20

type TeamSearchResult struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	DivisionName string `json:"divisionName"`
}

type PlayerSearchResult struct {
	Id        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	TeamId    int64  `json:"teamId"`
	TeamName  string `json:"teamName"`
}

// CoachSearchResult only ever covers employed trainers - an unemployed candidate on the open
// market has no team page to land on.
type CoachSearchResult struct {
	Id        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	TeamId    int64  `json:"teamId"`
	TeamName  string `json:"teamName"`
}

type StadiumSearchResult struct {
	TeamId      int64  `json:"teamId"`
	TeamName    string `json:"teamName"`
	StadiumName string `json:"stadiumName"`
}

type SearchResults struct {
	Teams    []TeamSearchResult    `json:"teams"`
	Players  []PlayerSearchResult  `json:"players"`
	Coaches  []CoachSearchResult   `json:"coaches"`
	Stadiums []StadiumSearchResult `json:"stadiums"`
}

// is_national_team excluded explicitly (unlike most other team listings in this codebase, this
// one has no league join that would already filter it out for free - see
// init71_national_teams.sql) - a national team isn't a club a player can be scouted to or
// transferred from, so it has no business showing up in a player/team/stadium search.
var SearchTeamsStmtSql = "SELECT t.team_id, t.name, t.country, COALESCE(l.name, '') " +
	"FROM teams t LEFT JOIN leagues l ON l.league_id = t.league_id " +
	"WHERE t.name ILIKE $1 AND t.is_national_team = false ORDER BY t.name LIMIT " + strconv.Itoa(SearchResultLimit)

var SearchPlayersStmtSql = "SELECT p.player_id, p.first_name, p.last_name, COALESCE(p.team_id, 0), COALESCE(t.name, '') " +
	"FROM players p LEFT JOIN teams t ON t.team_id = p.team_id " +
	"WHERE p.first_name ILIKE $1 OR p.last_name ILIKE $1 OR (p.first_name || ' ' || p.last_name) ILIKE $1 " +
	"ORDER BY p.last_name, p.first_name LIMIT " + strconv.Itoa(SearchResultLimit)

var SearchCoachesStmtSql = "SELECT tr.trainer_id, tr.first_name, tr.last_name, tr.team_id, t.name " +
	"FROM trainers tr JOIN teams t ON t.team_id = tr.team_id " +
	"WHERE tr.team_id IS NOT NULL AND (tr.first_name ILIKE $1 OR tr.last_name ILIKE $1 OR (tr.first_name || ' ' || tr.last_name) ILIKE $1) " +
	"ORDER BY tr.last_name, tr.first_name LIMIT " + strconv.Itoa(SearchResultLimit)

var SearchStadiumsStmtSql = "SELECT team_id, name, stadium_name FROM teams " +
	"WHERE stadium_name IS NOT NULL AND stadium_name ILIKE $1 ORDER BY stadium_name LIMIT " + strconv.Itoa(SearchResultLimit)

// Search runs the four by-name lookups behind the frontend's search page. query is wrapped in
// SQL wildcards here so callers just pass the raw user-typed text.
func Search(dbCtx *DatabaseContext, query string) (*SearchResults, error) {
	pattern := "%" + query + "%"
	results := &SearchResults{
		Teams:    []TeamSearchResult{},
		Players:  []PlayerSearchResult{},
		Coaches:  []CoachSearchResult{},
		Stadiums: []StadiumSearchResult{},
	}

	teamRows, err := dbCtx.Db.Query(SearchTeamsStmtSql, pattern)
	if err != nil {
		return nil, err
	}
	defer teamRows.Close()
	for teamRows.Next() {
		var r TeamSearchResult
		if err := teamRows.Scan(&r.Id, &r.Name, &r.Country, &r.DivisionName); err != nil {
			return nil, err
		}
		results.Teams = append(results.Teams, r)
	}
	if err := teamRows.Err(); err != nil {
		return nil, err
	}

	playerRows, err := dbCtx.Db.Query(SearchPlayersStmtSql, pattern)
	if err != nil {
		return nil, err
	}
	defer playerRows.Close()
	for playerRows.Next() {
		var r PlayerSearchResult
		if err := playerRows.Scan(&r.Id, &r.FirstName, &r.LastName, &r.TeamId, &r.TeamName); err != nil {
			return nil, err
		}
		results.Players = append(results.Players, r)
	}
	if err := playerRows.Err(); err != nil {
		return nil, err
	}

	coachRows, err := dbCtx.Db.Query(SearchCoachesStmtSql, pattern)
	if err != nil {
		return nil, err
	}
	defer coachRows.Close()
	for coachRows.Next() {
		var r CoachSearchResult
		if err := coachRows.Scan(&r.Id, &r.FirstName, &r.LastName, &r.TeamId, &r.TeamName); err != nil {
			return nil, err
		}
		results.Coaches = append(results.Coaches, r)
	}
	if err := coachRows.Err(); err != nil {
		return nil, err
	}

	stadiumRows, err := dbCtx.Db.Query(SearchStadiumsStmtSql, pattern)
	if err != nil {
		return nil, err
	}
	defer stadiumRows.Close()
	for stadiumRows.Next() {
		var r StadiumSearchResult
		if err := stadiumRows.Scan(&r.TeamId, &r.TeamName, &r.StadiumName); err != nil {
			return nil, err
		}
		results.Stadiums = append(results.Stadiums, r)
	}
	if err := stadiumRows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
