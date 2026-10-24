package common

import (
	"database/sql"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// NationalSquadPoolSize is how many of a country's best eligible players (by birth country,
// excluding academy prospects and free agents - see GetPlayersByBirthCountry) are considered when
// picking a national team's starting XI for one friendly (see SetNationalTeamLineup) - generous
// enough to cover every position's depth without scanning a country's entire player pool.
const NationalSquadPoolSize = 40

// scheduleNationalTeamFriendliesLockKey is an arbitrary Postgres advisory lock key (see
// pg_advisory_lock and common.schemaMigrationLockKey, the same convention), unique only in the
// sense that nothing else in this app uses it - guards ScheduleNationalTeamFriendlies against two
// --engine replicas running it at the same moment (see that function's own doc comment).
const scheduleNationalTeamFriendliesLockKey = 8817225

// NationalTeamFriendlyKickoffHour/NationalTeamFriendlyKickoffMinute is the local kickoff time a
// weekly national-team friendly (or World Cup group fixture, see engine.startWorldCupGroup)
// targets in its home country's own timezone (see NextLocalSaturdayKickoff) - a plausible
// Saturday-evening international window, chosen so two countries paired in the same weekly round
// never appear to kick off at literally the same real-world instant just because they share a
// kickoff delay the way a same-day club friendly does.
const (
	NationalTeamFriendlyKickoffHour   = 21
	NationalTeamFriendlyKickoffMinute = 0
)

const InsertNationalTeamStmtSql = "INSERT INTO teams (country, name, budget, stadium_capacity, is_national_team) " +
	"VALUES ($1, $2, 0, 0, true) RETURNING team_id"
const GetNationalTeamIdStmtSql = "SELECT team_id FROM teams WHERE country = $1 AND is_national_team = true"
const InsertNationalTeamFriendlyMatchStmtSql = InsertFriendlyMatchStmtSql + " RETURNING match_id"

const GetPlayersByBirthCountryStmtSql = "SELECT player_id, team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"COALESCE(shirt_number, 0), stamina, form " +
	"FROM players " +
	"WHERE birth_country = $1 AND in_academy = false AND team_id IS NOT NULL " +
	"ORDER BY (goal_keeper + defense + \"lateral\" + pass + midfield + dribbling + shot) DESC " +
	"LIMIT $2"

// nationalTeamName looks up country's display name in KnownCountries (e.g. "Spain") - falling
// back to the bare upper-cased code for anything not found there, which should never actually
// happen since only a known, active country ever gets one. Deliberately just the country's own
// name, with no "National Team" suffix: this is the raw value stored in `teams.name` and used
// verbatim by backend-authored text (notifications, this file's own log lines), but every list a
// manager actually reads renders a national team's name through the frontend's own
// localizedTeamName/COUNTRY_NAME_KEYS lookup by country code instead, so it shows in whichever
// language the UI is already in rather than always in English.
func nationalTeamName(country string) string {
	for _, c := range KnownCountries {
		if c.Code == country {
			return c.Name
		}
	}
	return strings.ToUpper(country)
}

// GetNationalTeamId looks up country's national team - see CreateNationalTeam.
func GetNationalTeamId(dbCtx *DatabaseContext, country string) (int64, error) {
	var teamId int64
	err := dbCtx.Db.QueryRow(GetNationalTeamIdStmtSql, country).Scan(&teamId)
	return teamId, err
}

// CreateNationalTeam materializes country's national team as a real, unowned `teams` row (see
// init71_national_teams.sql for why it's deliberately financially/competitively inert) - a real
// row rather than some separate concept so it can reuse the entire friendly_matches/
// friendly_lineup machinery unchanged, just fed a virtual squad instead of a club's own roster
// (see SetNationalTeamLineup). On a unique-violation (teams_national_team_country_idx - a race
// against another caller creating the same country's team at the same moment, see
// BackfillNationalTeams/AdminAddCountryHandler), returns whichever row won instead of erroring.
func CreateNationalTeam(dbCtx *DatabaseContext, country string) (int64, error) {
	var teamId int64
	err := dbCtx.Db.QueryRow(InsertNationalTeamStmtSql, country, nationalTeamName(country)).Scan(&teamId)
	if err != nil && isUniqueViolation(err) {
		return GetNationalTeamId(dbCtx, country)
	}
	return teamId, err
}

// ensureNationalTeam gets country's national team id, lazily creating it if the startup backfill
// somehow hasn't reached it yet (defensive - BackfillNationalTeams/AdminAddCountryHandler already
// cover the normal paths).
func ensureNationalTeam(dbCtx *DatabaseContext, country string) (int64, error) {
	teamId, err := GetNationalTeamId(dbCtx, country)
	if errors.Is(err, sql.ErrNoRows) {
		return CreateNationalTeam(dbCtx, country)
	}
	return teamId, err
}

// BackfillNationalTeams creates a national team for every currently active country that doesn't
// have one yet - called once at service startup (see fmservice.go), same convention as
// BackfillAiLevels/TopUpTrainerMarket, to cover a world that existed before this feature shipped.
func BackfillNationalTeams(dbCtx *DatabaseContext) error {
	active, err := GetActiveCountries(dbCtx)
	if err != nil {
		return err
	}
	for _, country := range active {
		if _, err := ensureNationalTeam(dbCtx, country); err != nil {
			return err
		}
	}
	return nil
}

// GetPlayersByBirthCountry returns the pick of country's eligible player pool (currently on a
// club's senior squad, not an academy prospect or free agent) for national-team selection,
// ordered by overall skill and capped at NationalSquadPoolSize - a deliberate simplification (the
// strongest-by-raw-skill rather than any notion of current form/fitness/call-up history), same
// spirit as the rest of this engine's other skill-based approximations.
func GetPlayersByBirthCountry(dbCtx *DatabaseContext, country string) ([]*Player, error) {
	rows, err := dbCtx.Db.Query(GetPlayersByBirthCountryStmtSql, country, NationalSquadPoolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]*Player, 0)
	var playerId, teamId int64
	var firstName, lastName string
	var birthDay int64
	var goalKeeper, defense, lateral, pass, midfield, dribbling, shot int
	var header, quick, creative, technical bool
	var wage, age int
	var avatar string
	var contractYearsRemaining int
	var birthCountry sql.NullString
	var shirtNumber, stamina, form int
	for rows.Next() {
		if err := rows.Scan(&playerId, &teamId, &firstName, &lastName, &birthDay, &goalKeeper, &defense, &lateral, &pass,
			&midfield, &dribbling, &shot, &header, &quick, &creative, &technical, &wage, &age, &avatar,
			&contractYearsRemaining, &birthCountry, &shirtNumber, &stamina, &form); err != nil {
			return nil, err
		}
		players = append(players, &Player{
			Id:                     playerId,
			TeamId:                 teamId,
			FirstName:              firstName,
			LastName:               lastName,
			BirthDay:               time.UnixMilli(birthDay),
			GoalKeeper:             goalKeeper,
			Defense:                defense,
			Lateral:                lateral,
			Pass:                   pass,
			Midfield:               midfield,
			Dribbling:              dribbling,
			Shot:                   shot,
			Header:                 header,
			Quick:                  quick,
			Creative:               creative,
			Technical:              technical,
			Wage:                   wage,
			Age:                    age,
			Avatar:                 GetAvatarFromString(avatar),
			ContractYearsRemaining: contractYearsRemaining,
			BirthCountry:           birthCountry.String,
			ShirtNumber:            shirtNumber,
			Stamina:                stamina,
			Form:                   form,
		})
	}
	return players, rows.Err()
}

// selectNationalSquadLineup is SetNationalTeamLineup/SetNationalTeamCupTieLineup's shared first
// step: fetch country's current eligible pool and pick its best XI (see GetPlayersByBirthCountry/
// SelectBestLineup). Returned alongside the flattened positions so the caller can still look up
// each selected player's name/club for the call-up notification without a second query.
func selectNationalSquadLineup(dbCtx *DatabaseContext, country string) ([]*Player, []LineupPosition, error) {
	players, err := GetPlayersByBirthCountry(dbCtx, country)
	if err != nil {
		return nil, nil, err
	}
	return players, FlattenLineup(SelectBestLineup(players)), nil
}

// SetNationalTeamLineup picks country's current best eligible XI and persists it as teamId's
// override lineup for matchId, a friendly (see SetFriendlyLineup) - always an explicit override,
// since a national team has no real roster of its own to fall back to the way a club's
// BestLineupForTeam does (see GetLineupsForFriendly). This saved lineup is also where
// international caps are counted from later (see GetPlayerCareerStats) - a player appears in it
// regardless of which club actually employs them.
func SetNationalTeamLineup(dbCtx *DatabaseContext, matchId int64, teamId int64, country string) error {
	players, positions, err := selectNationalSquadLineup(dbCtx, country)
	if err != nil {
		return err
	}
	if err := SetFriendlyLineup(dbCtx, matchId, teamId, FormationBalanced, positions, []LineupSubstitution{}); err != nil {
		return err
	}
	notifyCalledUpPlayers(dbCtx, players, positions, country, "/friendlies/"+strconv.FormatInt(matchId, 10))
	return nil
}

// SetNationalTeamCupTieLineup is SetNationalTeamLineup's World Cup knockout-stage counterpart:
// same squad selection, persisted instead as teamId's override lineup for cup tie tieId (see
// SetCupTieLineup) - called from engine.createRoundTies for every tie a national team is part of,
// covering every knockout round the same way SetNationalTeamLineup covers every weekly friendly.
func SetNationalTeamCupTieLineup(dbCtx *DatabaseContext, tieId int64, teamId int64, country string) error {
	players, positions, err := selectNationalSquadLineup(dbCtx, country)
	if err != nil {
		return err
	}
	if err := SetCupTieLineup(dbCtx, tieId, teamId, FormationBalanced, positions, []LineupSubstitution{}); err != nil {
		return err
	}
	notifyCalledUpPlayers(dbCtx, players, positions, country, "/cup/ties/"+strconv.FormatInt(tieId, 10))
	return nil
}

// notifyCalledUpPlayers tells each selected player's actual club (not the national team, which
// has no owner) that their player was called up - best-effort, same as every other notification
// site, since a failed notification must never undo the lineup that was just saved. link points
// at the match itself (a friendly or a cup tie, whichever the caller is setting a lineup for)
// rather than the player's own page, so a click leads straight to the fixture they were called up
// for.
func notifyCalledUpPlayers(dbCtx *DatabaseContext, players []*Player, positions []LineupPosition, country string, link string) {
	playersById := make(map[int64]*Player, len(players))
	for _, player := range players {
		playersById[player.Id] = player
	}

	teamName := nationalTeamName(country)
	for _, position := range positions {
		player, ok := playersById[position.PlayerId]
		if !ok {
			continue
		}
		playerName := player.FirstName + " " + player.LastName
		msg := NotificationMessage{
			Text:   playerName + " was called up to " + teamName + " for this week's international friendly",
			Key:    "notifications.messages.calledUp",
			Params: map[string]any{"player": playerName, "team": teamName},
		}
		if err := NotifyTeamOwnerWithLink(dbCtx, player.TeamId, NOTIFICATION_CALL_UP, msg, link); err != nil {
			log.Errorf("error notifying team %d of call-up for player %d: %s", player.TeamId, player.Id, err)
		}
	}
}

// GetPlayersForMatchTeam resolves every player who should count as teamId's roster for one
// match, for both simulation and match-detail display - an ordinary club's full squad (see
// GetPlayersFromTeamId), or, when teamId is a national team (see GetNationalTeamCountry), the
// specific players the saved lineup called up, fetched by id (see GetPlayersByIds) since they
// stay on their own real clubs' rosters rather than ever moving onto the national team's own
// team_id. Without this, GetPlayersFromTeamId silently comes back empty for a national team,
// leaving a played friendly or World Cup tie with a saved lineup but no player records to resolve
// it against - see engine.runMatchSimulation/EnrichPlayedFriendly/GetCupTieDetails.
//
// For an ordinary club, lineup's own players are also backfilled by id (see GetPlayersByIds) on
// top of the current squad - a played match's lineup is historical, so a player it named may
// since have been sold, released or retired off teamId's roster, and would otherwise be missing
// from a past match's report.
func GetPlayersForMatchTeam(dbCtx *DatabaseContext, teamId int64, lineup *Lineup) (*[]*Player, error) {
	_, isNationalTeam, err := GetNationalTeamCountry(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	if isNationalTeam {
		return GetPlayersByIds(dbCtx, LineupPlayerIds(lineup))
	}

	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	have := make(map[int64]bool, len(*players))
	for _, p := range *players {
		have[p.Id] = true
	}
	missing := make([]int64, 0)
	for _, id := range LineupPlayerIds(lineup) {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return players, nil
	}
	extra, err := GetPlayersByIds(dbCtx, missing)
	if err != nil {
		return nil, err
	}
	*players = append(*players, *extra...)
	return players, nil
}

const getNationalTeamCountryStmtSql = "SELECT country FROM teams WHERE team_id = $1 AND is_national_team = true"

// GetNationalTeamCountry reports whether teamId is a national team and, if so, its country - used
// by engine.createRoundTies to decide, for any cup tie it creates, whether either side needs its
// lineup auto-selected (see SetNationalTeamCupTieLineup) rather than left for a manager to set.
func GetNationalTeamCountry(dbCtx *DatabaseContext, teamId int64) (country string, isNationalTeam bool, err error) {
	err = dbCtx.Db.QueryRow(getNationalTeamCountryStmtSql, teamId).Scan(&country)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return country, err == nil, err
}

// Both exclude any match already linked into world_cup_group_matches: a group fixture is a real
// friendly_matches row under the hood (see engine.startWorldCupGroup), but it belongs on the
// World Cup tab's own group cards (see GetWorldCupGroups), not here - without this filter, a
// country mid-group-stage has its whole round-robin (every group opponent, several weeks out, all
// created at once) show up on the Amistosos tab at the same time, looking exactly like the
// double-booking bug this file already guards against elsewhere even though every fixture is
// legitimate (see GetCountriesWithPendingNationalTeamFriendly's own unrelated guard, and
// ScheduleNationalTeamFriendlies pausing entirely during an active World Cup).
const getUpcomingNationalTeamFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE ht.is_national_team = true AND fm.match_date > now() " +
	"AND NOT EXISTS (SELECT 1 FROM world_cup_group_matches wcgm WHERE wcgm.match_id = fm.match_id) " +
	"ORDER BY fm.match_date ASC LIMIT $1"

const getRecentNationalTeamFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE ht.is_national_team = true AND fm.played = true " +
	"AND NOT EXISTS (SELECT 1 FROM world_cup_group_matches wcgm WHERE wcgm.match_id = fm.match_id) " +
	"ORDER BY fm.match_date DESC LIMIT $1"

// GetUpcomingNationalTeamFriendlies lists every national team's upcoming ordinary friendly,
// soonest first - the global, any-country counterpart to GetUpcomingFriendlies (which is scoped
// to one club), since no single club "owns" a national-team fixture the friendlies page could
// otherwise filter by. World Cup group fixtures are deliberately excluded (see this query's own
// doc comment) - they're a different section of the same page (WorldCupSection's group cards).
func GetUpcomingNationalTeamFriendlies(dbCtx *DatabaseContext, limit int) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(getUpcomingNationalTeamFriendliesStmtSql, limit)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

// GetRecentNationalTeamFriendlies is GetUpcomingNationalTeamFriendlies' played counterpart, most
// recent first.
func GetRecentNationalTeamFriendlies(dbCtx *DatabaseContext, limit int) ([]FriendlyMatch, error) {
	rows, err := dbCtx.Db.Query(getRecentNationalTeamFriendliesStmtSql, limit)
	if err != nil {
		return nil, err
	}
	return scanFriendlyMatches(rows)
}

// NationalTeamCallUp is one row in a player's "recent call-ups" list on their profile page - a
// national-team friendly or World Cup knockout tie this player started for their birth country,
// most recent first. Kind tells the frontend which detail page to link to ("friendly" ->
// /friendlies/:id, "cupTie" -> /cup/ties/:id), since the two live in entirely separate tables
// (friendly_matches vs cup_ties) - see GetRecentNationalTeamCallUps.
type NationalTeamCallUp struct {
	Kind         string    `json:"kind"`
	MatchId      int64     `json:"matchId"`
	TeamName     string    `json:"teamName"`
	OpponentName string    `json:"opponentName"`
	MatchDate    time.Time `json:"matchDate"`
	Played       bool      `json:"played"`
	GoalsFor     int       `json:"goalsFor"`
	GoalsAgainst int       `json:"goalsAgainst"`
}

// getRecentNationalTeamCallUpsStmtSql unions playerId's starting call-ups from both places a
// national team ever plays - a weekly friendly and a World Cup knockout tie (see
// SetNationalTeamLineup/SetNationalTeamCupTieLineup, the only writers of friendly_lineup_positions/
// cup_tie_lineup_positions for a team flagged is_national_team) - each side resolved to
// "team/opponent/goals for/against" from that player's own national team's perspective rather
// than home/away, the same reorientation GetPlayerRecentMatches does for league matches.
const getRecentNationalTeamCallUpsStmtSql = "SELECT kind, match_id, team_name, opponent_name, match_date, played, goals_for, goals_against FROM (" +
	"SELECT 'friendly' AS kind, fm.match_id AS match_id, t.name AS team_name, " +
	"CASE WHEN fl.team_id = fm.home_team_id THEN ta.name ELSE th.name END AS opponent_name, " +
	"fm.match_date AS match_date, fm.played AS played, " +
	"CASE WHEN fl.team_id = fm.home_team_id THEN fm.home_goals ELSE fm.away_goals END AS goals_for, " +
	"CASE WHEN fl.team_id = fm.home_team_id THEN fm.away_goals ELSE fm.home_goals END AS goals_against " +
	"FROM friendly_lineup_positions flp " +
	"JOIN friendly_lineup fl ON fl.lineup_id = flp.lineup_id " +
	"JOIN friendly_matches fm ON fm.match_id = fl.match_id " +
	"JOIN teams t ON t.team_id = fl.team_id " +
	"JOIN teams th ON th.team_id = fm.home_team_id JOIN teams ta ON ta.team_id = fm.away_team_id " +
	"WHERE flp.player_id = $1 AND t.is_national_team = true " +
	"UNION ALL " +
	"SELECT 'cupTie' AS kind, ct.tie_id AS match_id, t.name AS team_name, " +
	"CASE WHEN ctl.team_id = ct.home_team_id THEN ta.name ELSE th.name END AS opponent_name, " +
	"ct.match_date AS match_date, ct.played AS played, " +
	"CASE WHEN ctl.team_id = ct.home_team_id THEN ct.home_goals ELSE ct.away_goals END AS goals_for, " +
	"CASE WHEN ctl.team_id = ct.home_team_id THEN ct.away_goals ELSE ct.home_goals END AS goals_against " +
	"FROM cup_tie_lineup_positions ctlp " +
	"JOIN cup_tie_lineup ctl ON ctl.lineup_id = ctlp.lineup_id " +
	"JOIN cup_ties ct ON ct.tie_id = ctl.tie_id " +
	"JOIN teams t ON t.team_id = ctl.team_id " +
	"JOIN teams th ON th.team_id = ct.home_team_id JOIN teams ta ON ta.team_id = ct.away_team_id " +
	"WHERE ctlp.player_id = $1 AND t.is_national_team = true" +
	") combined ORDER BY match_date DESC LIMIT $2"

// GetRecentNationalTeamCallUps lists playerId's most recent national-team call-ups (friendly or
// World Cup knockout tie, most recent first, capped at limit) - see NationalTeamCallUp.
func GetRecentNationalTeamCallUps(dbCtx *DatabaseContext, playerId int64, limit int) ([]NationalTeamCallUp, error) {
	rows, err := dbCtx.Db.Query(getRecentNationalTeamCallUpsStmtSql, playerId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	callUps := make([]NationalTeamCallUp, 0)
	for rows.Next() {
		var c NationalTeamCallUp
		if err := rows.Scan(&c.Kind, &c.MatchId, &c.TeamName, &c.OpponentName, &c.MatchDate, &c.Played, &c.GoalsFor, &c.GoalsAgainst); err != nil {
			return nil, err
		}
		callUps = append(callUps, c)
	}
	return callUps, rows.Err()
}

// pairNationalTeamCountries pairs up countries for one week's national-team friendlies,
// preferring same-region pairings (see RegionForCountry) the same way the international club cup
// already groups by region - realistic travel/rivalry grouping rather than pure chance. Any
// country left over within its own region (an odd region size) is pooled across regions and
// paired off from there, so at most one country site-wide sits out a given week (only when the
// total active country count is itself odd). Region keys are sorted before use so a fixed r
// always produces the same pairing - map iteration order is not itself random-seeded, so leaving
// it unsorted would make this untestable despite r being fixed.
func pairNationalTeamCountries(countries []string, r *rand.Rand) [][2]string {
	byRegion := make(map[string][]string)
	for _, country := range countries {
		byRegion[RegionForCountry(country)] = append(byRegion[RegionForCountry(country)], country)
	}
	regions := make([]string, 0, len(byRegion))
	for region := range byRegion {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	var pairs [][2]string
	var leftover []string
	for _, region := range regions {
		shuffled := append([]string(nil), byRegion[region]...)
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		for i := 0; i+1 < len(shuffled); i += 2 {
			pairs = append(pairs, [2]string{shuffled[i], shuffled[i+1]})
		}
		if len(shuffled)%2 == 1 {
			leftover = append(leftover, shuffled[len(shuffled)-1])
		}
	}

	r.Shuffle(len(leftover), func(i, j int) { leftover[i], leftover[j] = leftover[j], leftover[i] })
	for i := 0; i+1 < len(leftover); i += 2 {
		pairs = append(pairs, [2]string{leftover[i], leftover[i+1]})
	}
	return pairs
}

const getCountriesWithPendingNationalTeamFriendlyStmtSql = "SELECT DISTINCT country FROM teams WHERE is_national_team = true AND team_id IN (" +
	"SELECT home_team_id FROM friendly_matches WHERE played = false " +
	"UNION SELECT away_team_id FROM friendly_matches WHERE played = false)"

// GetCountriesWithPendingNationalTeamFriendly returns every country whose national team is
// already part of an unplayed friendly - a weekly round's own fixture, or a World Cup group/
// knockout match (both land in friendly_matches; see startWorldCupGroup's own comment on why a
// group match is "just" a friendly under the hood). ScheduleNationalTeamFriendlies excludes all of
// these from this week's pairing, so a country already waiting on a fixture that hasn't kicked
// off or been simulated yet can never be paired again on top of it - without this guard, that
// country's national team could end up with two friendlies scheduled at once.
func GetCountriesWithPendingNationalTeamFriendly(dbCtx *DatabaseContext) (map[string]bool, error) {
	rows, err := dbCtx.Db.Query(getCountriesWithPendingNationalTeamFriendlyStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pending := make(map[string]bool)
	for rows.Next() {
		var country string
		if err := rows.Scan(&country); err != nil {
			return nil, err
		}
		pending[country] = true
	}
	return pending, rows.Err()
}

const getPendingNationalTeamFriendlyMatchIdsStmtSql = "SELECT fm.match_id FROM friendly_matches fm " +
	"JOIN teams ht ON ht.team_id = fm.home_team_id " +
	"JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE fm.played = false AND ht.is_national_team = true AND at.is_national_team = true " +
	"AND (ht.country = ANY($1) OR at.country = ANY($1))"

const deleteFriendlyLineupPositionsForMatchesStmtSql = "DELETE FROM friendly_lineup_positions WHERE lineup_id IN (" +
	"SELECT lineup_id FROM friendly_lineup WHERE match_id = ANY($1))"
const deleteFriendlyLineupSubstitutionsForMatchesStmtSql = "DELETE FROM friendly_lineup_substitutions WHERE match_id = ANY($1)"
const deleteFriendlyLineupForMatchesStmtSql = "DELETE FROM friendly_lineup WHERE match_id = ANY($1)"
const deleteFriendlyMatchesByIdStmtSql = "DELETE FROM friendly_matches WHERE match_id = ANY($1)"

// CancelPendingNationalTeamFriendlies deletes every still-unplayed national-team friendly
// (a weekly friendly - a World Cup fixture is never "pending" at this point, since
// engine.StartWorldCupForNewSeason calls this before creating any) involving any of countries,
// along with any lineup already saved for it (CallUpNationalTeamSquads can close a call-up up to
// a day before kickoff, so one can exist even for a fixture that's days away). Called by
// engine.StartWorldCupForNewSeason right before entering countries into a fresh World Cup's
// groups: without this, a country that still had an ordinary weekly friendly pending when the
// tournament starts would end up scheduled into that AND its first group match at once - both
// landing on the same home-country Saturday 21:00 local kickoff (see
// NationalTeamFriendlyKickoffHour/Minute), since both are computed the exact same way. A real
// international calendar clears itself for a tournament instead of double-booking a country this
// way (see ScheduleNationalTeamFriendlies's own "skips entirely while a World Cup is running" doc
// comment for the other half of that same rule).
func CancelPendingNationalTeamFriendlies(dbCtx *DatabaseContext, countries []string) error {
	if len(countries) == 0 {
		return nil
	}

	rows, err := dbCtx.Db.Query(getPendingNationalTeamFriendlyMatchIdsStmtSql, countries)
	if err != nil {
		return err
	}
	matchIds := make([]int64, 0)
	for rows.Next() {
		var matchId int64
		if err := rows.Scan(&matchId); err != nil {
			rows.Close()
			return err
		}
		matchIds = append(matchIds, matchId)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(matchIds) == 0 {
		return nil
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(deleteFriendlyLineupPositionsForMatchesStmtSql, matchIds); err != nil {
		return err
	}
	if _, err := tx.Exec(deleteFriendlyLineupSubstitutionsForMatchesStmtSql, matchIds); err != nil {
		return err
	}
	if _, err := tx.Exec(deleteFriendlyLineupForMatchesStmtSql, matchIds); err != nil {
		return err
	}
	if _, err := tx.Exec(deleteFriendlyMatchesByIdStmtSql, matchIds); err != nil {
		return err
	}
	return tx.Commit()
}

// ScheduleNationalTeamFriendlies is the national-team friendly pairing pass (see
// engine.StartNationalTeamFriendlyScheduler, run frequently rather than on a rigid weekly
// timestamp) - pairs up every active country that isn't already waiting on a fixture (see
// GetCountriesWithPendingNationalTeamFriendly/pairNationalTeamCountries) and creates each fixture
// directly with no challenge/accept dance (there is no human on either side to ask - see
// common.AcceptFriendlyChallenge for that flow, which this deliberately bypasses). Running this
// often, instead of once a week, means a country is re-paired for its next fixture within minutes
// of its previous one being marked played rather than however long is left until the next fixed
// weekly slot - so there's (almost) always a future fixture for the match viewer to show, and no
// country is left sitting out for however much of a week its own local kickoff happens to miss a
// rigid weekly cron run by (see GetCountriesWithPendingNationalTeamFriendly's own doc comment).
// Each side's starting XI is deliberately NOT selected here - see CallUpNationalTeamSquads for why
// that's a separate pass closer to kickoff instead.
//
// Each fixture kicks off at NationalTeamFriendlyKickoffHour:Minute on the next Saturday in the
// home side's own local timezone (see NextLocalSaturdayKickoff) - pairNationalTeamCountries'
// shuffle already picks which side of a pair is "home" at random, so which country's timezone
// anchors a given fixture is itself random. Two countries in different timezones consequently
// kick off at different real-world instants even though they're both "Saturday evening", rather
// than every fixture in the round landing on the exact same instant.
//
// Skips entirely while a World Cup is running (see HasActiveWorldCup): a real international
// calendar pauses ordinary friendlies around a tournament rather than double-booking a country's
// national team in the same week, and group-stage/knockout fixtures already give every active
// country as much national-team action as a weekly friendly would anyway.
//
// Guarded by a non-blocking, session-level advisory lock on a dedicated connection, held for the
// whole function body - the same shape advanceCountrySeasonIfComplete/advanceCup use for the same
// reason (see either one's own doc comment): this is a check-then-act (read every country's
// pending state, then insert new fixtures for a computed pairing) with no transaction of its own,
// and two --engine replicas (a real, supported deployment shape - see docker-compose.yml) both
// ticking this job at once - every 5 minutes, see engine.StartNationalTeamFriendlyScheduler -
// could otherwise both read the same "nobody's pending" snapshot and both independently pair and
// insert fixtures for the same countries, landing a country in two different friendlies at once.
// Keyed on a single fixed constant rather than per-country, since the pairing decision itself
// (which countries are even eligible) spans every active country at once, not just one.
//
// engine.StartWorldCupForNewSeason takes the same lock (see WithNationalTeamSchedulingLock), so a
// World Cup can never start between this pass's HasActiveWorldCup check and its inserts.
func ScheduleNationalTeamFriendlies(dbCtx *DatabaseContext) error {
	// When another replica holds the lock it is already scheduling this tick - whatever it leaves
	// behind, the next tick picks up correctly.
	_, err := TryWithSessionAdvisoryLock(dbCtx, scheduleNationalTeamFriendliesLockKey, func() error {
		return scheduleNationalTeamFriendliesLocked(dbCtx)
	})
	return err
}

// WithNationalTeamSchedulingLock runs fn holding the lock ScheduleNationalTeamFriendlies takes,
// waiting for it if a friendlies pass is running - for engine.StartWorldCupForNewSeason, whose
// group fixtures must never be created while a friendlies pass is pairing the same countries.
func WithNationalTeamSchedulingLock(dbCtx *DatabaseContext, fn func() error) error {
	return WithSessionAdvisoryLock(dbCtx, scheduleNationalTeamFriendliesLockKey, fn)
}

func scheduleNationalTeamFriendliesLocked(dbCtx *DatabaseContext) error {
	hasWorldCup, err := HasActiveWorldCup(dbCtx)
	if err != nil {
		return err
	}
	if hasWorldCup {
		return nil
	}

	active, err := GetActiveCountries(dbCtx)
	if err != nil {
		return err
	}

	pending, err := GetCountriesWithPendingNationalTeamFriendly(dbCtx)
	if err != nil {
		return err
	}
	eligible := make([]string, 0, len(active))
	for _, country := range active {
		if !pending[country] {
			eligible = append(eligible, country)
		}
	}
	if len(eligible) < 2 {
		return nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pairs := pairNationalTeamCountries(eligible, r)

	for _, pair := range pairs {
		homeTeamId, err := ensureNationalTeam(dbCtx, pair[0])
		if err != nil {
			log.Errorf("error ensuring national team for %s: %s", pair[0], err)
			continue
		}
		awayTeamId, err := ensureNationalTeam(dbCtx, pair[1])
		if err != nil {
			log.Errorf("error ensuring national team for %s: %s", pair[1], err)
			continue
		}

		kickoff := NextLocalSaturdayKickoff(pair[0], NationalTeamFriendlyKickoffHour, NationalTeamFriendlyKickoffMinute)

		var matchId int64
		if err := dbCtx.Db.QueryRow(InsertNationalTeamFriendlyMatchStmtSql, homeTeamId, awayTeamId, kickoff).Scan(&matchId); err != nil {
			log.Errorf("error creating national-team friendly between %s and %s: %s", pair[0], pair[1], err)
			continue
		}
	}
	return nil
}

const getNationalTeamFriendliesNeedingCallUpStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.country FROM friendly_matches fm " +
	"JOIN teams ht ON ht.team_id = fm.home_team_id " +
	"WHERE ht.is_national_team = true AND fm.played = false AND fm.match_date <= $1 " +
	"AND NOT EXISTS (SELECT 1 FROM friendly_lineup fl WHERE fl.match_id = fm.match_id AND fl.team_id = fm.home_team_id) " +
	"UNION ALL " +
	"SELECT fm.match_id, fm.away_team_id, at.country FROM friendly_matches fm " +
	"JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE at.is_national_team = true AND fm.played = false AND fm.match_date <= $1 " +
	"AND NOT EXISTS (SELECT 1 FROM friendly_lineup fl WHERE fl.match_id = fm.match_id AND fl.team_id = fm.away_team_id)"

// nationalTeamCallUpWindow is how long before kickoff a national team's call-up (its starting XI
// selection, see SetNationalTeamLineup) closes - a real squad is announced shortly before a match,
// not the moment the fixture itself is scheduled (which, see ScheduleNationalTeamFriendlies, can
// be days earlier) - so a player's current form/fitness at call-up time is actually current.
const nationalTeamCallUpWindow = 24 * time.Hour

// CallUpNationalTeamSquads is ScheduleNationalTeamFriendlies' squad-selection counterpart (see
// engine.StartNationalTeamCallUpScheduler) - run frequently, it finds every unplayed
// national-team friendly within nationalTeamCallUpWindow of kickoff that doesn't have a saved
// lineup yet for one side or the other, and calls SetNationalTeamLineup for just that side. The
// "doesn't have a lineup yet" check (a plain SQL NOT EXISTS against friendly_lineup) is what makes
// this idempotent to call on every tick right up to kickoff: once a side's call-up is saved, that
// row stops matching, so a player is never re-notified of the same call-up over and over, and a
// side whose call-up already succeeded is left alone even if the other side's failed and still
// needs retrying.
//
// Runs under a session-level advisory lock: two --engine replicas ticking this job at the same
// instant would otherwise both find the same side without a lineup and both call it up, sending
// every selected player's club the call-up notification twice. The NOT EXISTS query runs inside
// the lock, so the replica that runs second sees the lineups the first one saved.
func CallUpNationalTeamSquads(dbCtx *DatabaseContext) error {
	_, err := TryWithSessionAdvisoryLock(dbCtx, NamespacedLockKey(LockNamespaceNationalTeamCallUp, 0), func() error {
		return callUpNationalTeamSquadsLocked(dbCtx)
	})
	return err
}

func callUpNationalTeamSquadsLocked(dbCtx *DatabaseContext) error {
	cutoff := time.Now().Add(nationalTeamCallUpWindow)
	rows, err := dbCtx.Db.Query(getNationalTeamFriendliesNeedingCallUpStmtSql, cutoff)
	if err != nil {
		return err
	}
	type callUp struct {
		matchId int64
		teamId  int64
		country string
	}
	callUps := make([]callUp, 0)
	for rows.Next() {
		var c callUp
		if err := rows.Scan(&c.matchId, &c.teamId, &c.country); err != nil {
			rows.Close()
			return err
		}
		callUps = append(callUps, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range callUps {
		if err := SetNationalTeamLineup(dbCtx, c.matchId, c.teamId, c.country); err != nil {
			log.Errorf("error setting national-team lineup for %s (match %d): %s", c.country, c.matchId, err)
		}
	}
	return nil
}
