package common

import (
	"strconv"

	log "github.com/sirupsen/logrus"
)

// League and CPU team creation, shared by a new world or country (see handlers.InitializeGame,
// handlers.AdminAddCountryHandler) and a restructure that adds groups and tops a tier up with new
// CPU teams (see engine.applyRestructure), so every team is created the same way.

const insertLeagueStmtSql = "INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, $2, $3, $4) RETURNING league_id"

// LeagueName is a group's league name: the tier name and the group number, e.g. "III.2".
func LeagueName(tierName string, group int) string {
	return tierName + "." + strconv.Itoa(group)
}

// CreateLeague inserts one group of tier for country, named LeagueName(tierName, group). It gets
// no schedule override: its kickoff follows the country's league kickoff for its tier (see
// EffectiveLeagueSchedule).
func CreateLeague(dbCtx *DatabaseContext, country string, tierName string, tier int, group int) (int64, error) {
	var leagueId int64
	err := dbCtx.Db.QueryRow(insertLeagueStmtSql, country, group, LeagueName(tierName, group), tier).Scan(&leagueId)
	return leagueId, err
}

// CreateLeagues inserts groups 1..groups of tier for country. A group that fails to insert is
// logged and left out.
func CreateLeagues(dbCtx *DatabaseContext, country string, tierName string, tier int, groups int) []int64 {
	log.Infof("initializing country %s league %s tier %d with %d group(s)", country, tierName, tier, groups)
	leagues := make([]int64, 0, groups)
	for g := 1; g <= groups; g++ {
		leagueId, err := CreateLeague(dbCtx, country, tierName, tier, g)
		if err != nil {
			log.Errorf("error inserting league %s of %s: %s", LeagueName(tierName, g), country, err)
			continue
		}
		leagues = append(leagues, leagueId)
	}
	return leagues
}

// CreateCountryLeagues creates one country's leagues and teams, shaped by tiers.
func CreateCountryLeagues(dbCtx *DatabaseContext, country string, tiers []TierConfig) {
	for _, tierConfig := range tiers {
		leagueIds := CreateLeagues(dbCtx, country, tierConfig.Name, tierConfig.Tier, tierConfig.Groups)
		CreateTeamsForTier(dbCtx, country, leagueIds, tierConfig.Tier, tierConfig.TeamsPerLeague)
	}
}

// CreateTeamsForTier fills each of leagueIds with teamsPerLeague new CPU teams and their
// classification rows.
func CreateTeamsForTier(dbCtx *DatabaseContext, country string, leagueIds []int64, tier int, teamsPerLeague int) {
	for _, leagueId := range leagueIds {
		for i := 0; i < teamsPerLeague; i++ {
			if _, err := CreateTeam(dbCtx, country, leagueId, tier); err != nil {
				log.Errorf("error creating a team in league %d: %s", leagueId, err)
			}
		}
		CreateClassification(dbCtx, leagueId)
	}
}

const insertTeamStmtSql = "INSERT INTO teams (country, league_id, name, stadium_name, stadium_capacity, ai_level) " +
	"VALUES ($1, $2, $3, $4, $5, $6) RETURNING team_id"

// CreateTeam creates one unowned CPU team in leagueId: a random name and stadium, an AI level for
// its tier and a CPU style, a full squad, trainer candidates and a default lineup. It does not add
// the team's classification row (see CreateClassification / InsertTeamsToClassification).
func CreateTeam(dbCtx *DatabaseContext, country string, leagueId int64, tier int) (int64, error) {
	var teamId int64
	// Every team starts unowned (see AssignTeam for how a human later claims one), so it gets an
	// AI level right away - see RandomAiLevel.
	if err := dbCtx.Db.QueryRow(insertTeamStmtSql, country, leagueId, RandomTeamName(country), RandomStadiumName(country),
		DefaultStadiumCapacity, RandomAiLevel(tier)).Scan(&teamId); err != nil {
		return 0, err
	}
	if err := AssignAiStyle(dbCtx, teamId); err != nil {
		log.Errorf("error assigning a CPU style to team %d: %s", teamId, err)
	}
	CreatePlayers(dbCtx, teamId, country)
	if err := GenerateTrainerCandidates(dbCtx, country, TrainerCandidatesPerTeam); err != nil {
		log.Errorf("error generating trainer candidates for team %d: %s", teamId, err)
	}
	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching newly created squad for team %d: %s", teamId, err)
	} else if err := InsertLineup(dbCtx, teamId, SelectBestLineup(*players), true); err != nil {
		log.Errorf("error seeding default lineup for team %d: %s", teamId, err)
	}
	return teamId, nil
}

// CreateClassification adds a season-1 classification row for every team of leagueId (a season
// rollover's ResetClassificationForSeason later moves the rows to the current season).
func CreateClassification(dbCtx *DatabaseContext, leagueId int64) {
	InsertTeamsToClassification(dbCtx, leagueId, 1, GetDivisionTeams(dbCtx, leagueId))
}
