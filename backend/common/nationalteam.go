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

// SetNationalTeamLineup picks country's current best eligible XI (see GetPlayersByBirthCountry/
// SelectBestLineup) and persists it as teamId's override lineup for matchId (see
// SetFriendlyLineup) - always an explicit override, since a national team has no real roster of
// its own to fall back to the way a club's BestLineupForTeam does (see GetLineupsForFriendly).
// This saved lineup is also where international caps are counted from later (see
// GetPlayerCareerStats) - a player appears in it regardless of which club actually employs them.
func SetNationalTeamLineup(dbCtx *DatabaseContext, matchId int64, teamId int64, country string) error {
	players, err := GetPlayersByBirthCountry(dbCtx, country)
	if err != nil {
		return err
	}
	positions := FlattenLineup(SelectBestLineup(players))
	if err := SetFriendlyLineup(dbCtx, matchId, teamId, FormationBalanced, positions, []LineupSubstitution{}); err != nil {
		return err
	}
	notifyCalledUpPlayers(dbCtx, players, positions, country)
	return nil
}

// notifyCalledUpPlayers tells each selected player's actual club (not the national team, which
// has no owner) that their player was called up - best-effort, same as every other notification
// site, since a failed notification must never undo the lineup that was just saved.
func notifyCalledUpPlayers(dbCtx *DatabaseContext, players []*Player, positions []LineupPosition, country string) {
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
		if err := NotifyTeamOwnerWithLink(dbCtx, player.TeamId, NOTIFICATION_CALL_UP, msg, "/player/"+strconv.FormatInt(player.Id, 10)); err != nil {
			log.Errorf("error notifying team %d of call-up for player %d: %s", player.TeamId, player.Id, err)
		}
	}
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
func ScheduleNationalTeamFriendlies(dbCtx *DatabaseContext) error {
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
