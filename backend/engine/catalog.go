package engine

import (
	"encoding/json"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const ClassificationKey = "classDet_key"
const CurrentSeasonKeyPrefix = "currentSeason_key_"
const LeagueWeekKey = "leagueWeek_key"
const LeagueSeasonTableKey = "leagueSeasonTable_key_"
const FullTeamDetailsKey = "fullTeamDet_key_"
const MatchDetailsKey = "matchDet_key_"
const PlayersFromTeamKey = "playTeam_key_"
const PlayerNameKey = "playerName_key_"
const TeamDetailsKey = "teamDet_key_"
const TeamOwnerKey = "teamOwner_key_"

// InvalidateTeamDetailsCache drops both the owner and non-owner cached views of a team (see
// GetTeamDetails) so a direct DB write - e.g. an admin override - is reflected on the next read
// instead of waiting out the cache's TTL.
func InvalidateTeamDetailsCache(dbCtx *common.DatabaseContext, teamId int64) {
	idStr := strconv.FormatInt(teamId, 10)
	dbCtx.ObjectCache.DeleteValue(FullTeamDetailsKey + idStr)
	dbCtx.ObjectCache.DeleteValue(TeamDetailsKey + idStr)
}

// InvalidateTeamCaches drops everything cached about a club whose ownership or squad just changed
// (released, reset, owner deleted): both views of its details, its cached owner id and its cached
// player list - otherwise notifications and pages keep addressing the previous owner.
func InvalidateTeamCaches(dbCtx *common.DatabaseContext, teamId int64) {
	InvalidateTeamDetailsCache(dbCtx, teamId)
	idStr := strconv.FormatInt(teamId, 10)
	dbCtx.ObjectCache.DeleteValue(TeamOwnerKey + idStr)
	dbCtx.ObjectCache.DeleteValue(PlayersFromTeamKey + idStr)
}

// CurrentSeasonKey is this cache's key for one country's current season - current_season now
// lives on `countries` (see init75_season_per_country.sql), one counter per country, so the
// cache key needs the country baked in too.
func CurrentSeasonKey(country string) string {
	return CurrentSeasonKeyPrefix + country
}

func GetCurrentSeason(dbCtx *common.DatabaseContext, country string) (int, error) {
	key := CurrentSeasonKey(country)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		vint, err := strconv.Atoi(v)
		if err == nil {
			return vint, nil
		}
	}
	vint, err := common.GetCurrentSeasonForCountry(dbCtx, country)
	if err != nil {
		return 0, err
	}
	dbCtx.ObjectCache.SetValueLongDuration(key, strconv.Itoa(vint))
	return vint, nil
}

// ResetCurrentSeasonCache overwrites the cached "current season" for one country (see
// GetCurrentSeason above) with the value a caller just wrote straight to the database - the
// same "refresh the cache immediately after a direct write" pattern
// AdvanceSeasonsIfComplete already uses after a normal season rollover.
// handlers.InitializeGame and handlers.RecreateLeaguesKeepingUsers both reset every country's
// season to 1 directly in Postgres; without this, a season number cached from a previous run
// (most plausible with an external Redis instance kept across a database reset - see issue #8)
// would silently outlive the reset for up to the cache's own TTL. Worse than a transient
// mismatch: GenerateMatches (called once, synchronously, right after) would tag the freshly
// generated fixtures with that stale season number, while every other read gets the correct one
// straight from the database - and nothing ever revisits "zero matches for the season the
// database says it is" as a bug needing recovery, so the fixtures stay permanently orphaned
// rather than the mismatch resolving itself once the cache entry expires.
func ResetCurrentSeasonCache(dbCtx *common.DatabaseContext, country string, season int) {
	dbCtx.ObjectCache.SetValueLongDuration(CurrentSeasonKey(country), strconv.Itoa(season))
}

func GetPlayersFromTeam(dbCtx *common.DatabaseContext, teamId int64) (*[]*common.Player, error) {
	key := PlayersFromTeamKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		var players *[]*common.Player
		err := json.Unmarshal([]byte(v), &players)
		if err == nil {
			return players, nil
		}
	}
	teamDetails, _ := common.GetPlayersFromTeamId(dbCtx, teamId)
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

// resolveMatchTeamPlayers is runMatchSimulation's own GetPlayersForMatchTeam - teamId's cached
// squad (see GetPlayersFromTeam) for an ordinary club, or, when teamId is a national team, lineup's
// called-up players fetched by id (see common.GetPlayersByIds), since a national team keeps no
// roster of its own under its own team_id for GetPlayersFromTeam's cache to ever find (see
// common.SetNationalTeamLineup). Kept separate from common.GetPlayersForMatchTeam so a club match
// still benefits from GetPlayersFromTeam's cache; only a national-team match skips it, since its
// called-up squad is resolved fresh from that week's lineup anyway.
func resolveMatchTeamPlayers(dbCtx *common.DatabaseContext, teamId int64, lineup *common.Lineup) (*[]*common.Player, error) {
	_, isNationalTeam, err := common.GetNationalTeamCountry(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	if !isNationalTeam {
		return GetPlayersFromTeam(dbCtx, teamId)
	}
	return common.GetPlayersByIds(dbCtx, common.LineupPlayerIds(lineup))
}

func GetPlayerFullName(dbCtx *common.DatabaseContext, playerId int64) (string, error) {
	key := PlayerNameKey + strconv.FormatInt(playerId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		return v, nil
	}
	name, e := common.GetPlayerFullName(dbCtx, playerId)
	if e != nil {
		return "", e
	}
	dbCtx.ObjectCache.SetValue(key, name)
	return name, nil
}

func GetDivisionDetails(dbCtx *common.DatabaseContext, leagueId int64) (*common.DivisionDetails, error) {
	key := ClassificationKey + strconv.FormatInt(leagueId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var divisionDetails *common.DivisionDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &divisionDetails)
		if err == nil {
			return divisionDetails, nil
		}
	}
	divisionDetails, err := common.GetDivisionDetails(dbCtx, leagueId)
	if err != nil {
		log.Errorf("error retrieving division details from: %d. %s", leagueId, err)
		return nil, err
	}
	value, _ := json.Marshal(divisionDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return divisionDetails, nil
}

// GetLeagueWeekMatches returns one matchday of a league's season, cached like GetDivisionDetails
// (1 minute) so browsing the league page's matchdays doesn't hit the database on every tap. The
// result is never nil - a week past the season's last one is an empty slice.
func GetLeagueWeekMatches(dbCtx *common.DatabaseContext, leagueId int64, season int, week int) ([]common.WeekMatch, error) {
	key := LeagueWeekKey + strconv.FormatInt(leagueId, 10) + "_" + strconv.Itoa(season) + "_" + strconv.Itoa(week)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		var cached []common.WeekMatch
		if err := json.Unmarshal([]byte(v), &cached); err == nil && cached != nil {
			return cached, nil
		}
	}
	matches, err := common.WeekMatches(dbCtx, leagueId, season, week)
	if err != nil {
		log.Errorf("error retrieving week %d matches of league %d: %s", week, leagueId, err)
		return nil, err
	}
	result := make([]common.WeekMatch, 0)
	if matches != nil {
		result = *matches
	}
	value, _ := json.Marshal(result)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return result, nil
}

// GetLeagueSeasonTable returns a league's final table for a past season (see
// common.GetLeagueSeasonTable), cached with no expiry: a finished season's table never changes, and
// league ids are never reused after a world reset. nil when the season has no table; that is not
// cached, so a season that rolls over later is picked up.
func GetLeagueSeasonTable(dbCtx *common.DatabaseContext, leagueId int64, season int) (*common.LeagueSeasonTable, error) {
	key := LeagueSeasonTableKey + strconv.FormatInt(leagueId, 10) + "_" + strconv.Itoa(season)
	if v, _ := dbCtx.ObjectCache.GetValue(key); v != "" {
		var cached common.LeagueSeasonTable
		if err := json.Unmarshal([]byte(v), &cached); err == nil && len(cached.Positions) > 0 {
			return &cached, nil
		}
	}
	table, err := common.GetLeagueSeasonTable(dbCtx, leagueId, season)
	if err != nil || table == nil {
		return table, err
	}
	value, _ := json.Marshal(table)
	dbCtx.ObjectCache.SetValueNoExpiry(key, string(value))
	return table, nil
}

func GetTeamDetails(dbCtx *common.DatabaseContext, owner bool, teamId int64) (*common.TeamDetails, error) {
	var key string
	if owner {
		key = FullTeamDetailsKey + strconv.FormatInt(teamId, 10)
	} else {
		key = TeamDetailsKey + strconv.FormatInt(teamId, 10)
	}
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var teamDetails *common.TeamDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &teamDetails)
		if err == nil {
			return teamDetails, nil
		}
	}
	var err error
	teamDetails, err = common.GetTeamDetailsById(teamId, owner, dbCtx)
	if err != nil {
		return nil, err
	}
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

func GetUserOfTeamId(dbCtx *common.DatabaseContext, teamId int64) (int64, error) {
	key := TeamOwnerKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		return common.StringToInt64(v)
	}
	userId, err := common.GetTeamOwner(dbCtx, teamId)
	if err != nil {
		return -1, err
	}
	dbCtx.ObjectCache.SetValueLongDuration(key, strconv.FormatInt(userId, 10))
	return userId, nil
}

func GetMatchDetails(dbCtx *common.DatabaseContext, matchId int64) (*common.WeekMatch, error) {
	key := MatchDetailsKey + strconv.FormatInt(matchId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var matchDetails *common.WeekMatch
	if v != "" {
		err := json.Unmarshal([]byte(v), &matchDetails)
		if err == nil {
			return matchDetails, nil
		}
	}
	var err error
	matchDetails, err = common.WeekMatchById(dbCtx, matchId)
	if err != nil {
		return nil, err
	}
	if matchDetails == nil {
		return nil, nil
	}
	value, _ := json.Marshal(matchDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return matchDetails, nil
}
