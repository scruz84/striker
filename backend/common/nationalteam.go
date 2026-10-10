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

// NationalTeamFriendlyKickoffDelay mirrors friendlyScheduleDelay (friendlies.go) - how soon after
// being scheduled a national-team friendly kicks off, long enough to clear the simulation
// scheduler's polling window (see GetSimulableFriendlies).
const NationalTeamFriendlyKickoffDelay = 5 * time.Minute

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

// nationalTeamName looks up country's display name in KnownCountries for a readable team name
// (e.g. "Spain National Team") - falling back to the bare code for anything not found there,
// which should never actually happen since only a known, active country ever gets one.
func nationalTeamName(country string) string {
	for _, c := range KnownCountries {
		if c.Code == country {
			return c.Name + " National Team"
		}
	}
	return strings.ToUpper(country) + " National Team"
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

const getUpcomingNationalTeamFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE ht.is_national_team = true AND fm.match_date > now() ORDER BY fm.match_date ASC LIMIT $1"

const getRecentNationalTeamFriendliesStmtSql = "SELECT fm.match_id, fm.home_team_id, ht.name, fm.away_team_id, at.name, fm.match_date, fm.played, fm.home_goals, fm.away_goals " +
	"FROM friendly_matches fm JOIN teams ht ON ht.team_id = fm.home_team_id JOIN teams at ON at.team_id = fm.away_team_id " +
	"WHERE ht.is_national_team = true AND fm.played = true ORDER BY fm.match_date DESC LIMIT $1"

// GetUpcomingNationalTeamFriendlies lists every national team's upcoming friendly, soonest
// first - the global, any-country counterpart to GetUpcomingFriendlies (which is scoped to one
// club), since no single club "owns" a national-team fixture the friendlies page could otherwise
// filter by.
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

// ScheduleNationalTeamFriendlies is the weekly national-team friendly round (see
// engine.StartNationalTeamFriendlyScheduler) - pairs up every active country (see
// pairNationalTeamCountries), creates each fixture directly with no challenge/accept dance (there
// is no human on either side to ask - see common.AcceptFriendlyChallenge for that flow, which this
// deliberately bypasses), and immediately persists each side's starting XI (see
// SetNationalTeamLineup) so there's a lineup ready for the normal friendly simulation scheduler to
// pick up, and to count international caps from once it's played.
//
// Skips the whole week while a World Cup is running (see HasActiveWorldCup): a real international
// calendar pauses ordinary friendlies around a tournament rather than double-booking a country's
// national team in the same week, and group-stage/knockout fixtures already give every active
// country as much national-team action as a weekly friendly would anyway.
func ScheduleNationalTeamFriendlies(dbCtx *DatabaseContext) error {
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
	if len(active) < 2 {
		return nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pairs := pairNationalTeamCountries(active, r)
	kickoff := time.Now().Add(NationalTeamFriendlyKickoffDelay)

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

		var matchId int64
		if err := dbCtx.Db.QueryRow(InsertNationalTeamFriendlyMatchStmtSql, homeTeamId, awayTeamId, kickoff).Scan(&matchId); err != nil {
			log.Errorf("error creating national-team friendly between %s and %s: %s", pair[0], pair[1], err)
			continue
		}

		if err := SetNationalTeamLineup(dbCtx, matchId, homeTeamId, pair[0]); err != nil {
			log.Errorf("error setting national-team lineup for %s (match %d): %s", pair[0], matchId, err)
		}
		if err := SetNationalTeamLineup(dbCtx, matchId, awayTeamId, pair[1]); err != nil {
			log.Errorf("error setting national-team lineup for %s (match %d): %s", pair[1], matchId, err)
		}
	}
	return nil
}
