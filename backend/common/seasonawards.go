package common

import (
	"database/sql"
	"errors"
	"fmt"
)

// SeasonAward is a league's season-end award winner (Golden Boot or Player of the Season) - see
// GetPlayerOfTheSeason. Golden Boot itself is just GetTopScorersFinal's #1 result, so it needs no
// separate type.
type SeasonAward struct {
	PlayerId  int64
	FirstName string
	LastName  string
	TeamId    int64
	TeamName  string
	Score     int
}

// Deliberately simple weights for "Player of the Season" - not a real performance model, just
// enough to make it feel earned: rewards goals, docks cards, nothing more sophisticated than that.
const (
	playerOfSeasonGoalWeight       = 3
	playerOfSeasonYellowCardWeight = -1
	playerOfSeasonRedCardWeight    = -3
)

// getPlayerOfTheSeasonStmtSql scores every player of a league season's league matches - every
// one, the final round's still on air included ($1 league, $2 season, $3 cutoff, passed as NULL)
// - and keeps the best positive score, ties to the lower player id. A player who retired in this
// very rollover (ProgressPlayers runs before the awards) is still found, through retired_players.
var getPlayerOfTheSeasonStmtSql = "SELECT t.player_id, " + statsPlayerNameColumnsSql + ", t.team_id, COALESCE(tm.name, ''), " +
	fmt.Sprintf("t.goals * %d + t.yellow * %d + t.red * %d AS score ", playerOfSeasonGoalWeight, playerOfSeasonYellowCardWeight, playerOfSeasonRedCardWeight) +
	"FROM " + leagueTotalsSql + statsPlayerNameJoinsSql + "LEFT JOIN teams tm ON tm.team_id = t.team_id " +
	"ORDER BY score DESC, t.player_id LIMIT 1"

// GetPlayerOfTheSeason returns the league's Player of the Season for the given season by a
// simple goals/cards weighted score (see the playerOfSeason* weights above), or nil if nobody
// has a positive one.
func GetPlayerOfTheSeason(dbCtx *DatabaseContext, leagueId int64, season int) (*SeasonAward, error) {
	var award SeasonAward
	var birthCountry string
	var retired bool
	err := dbCtx.Db.QueryRow(getPlayerOfTheSeasonStmtSql, leagueId, season, nil).Scan(&award.PlayerId, &award.FirstName, &award.LastName,
		&birthCountry, &retired, &award.TeamId, &award.TeamName, &award.Score)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if award.Score <= 0 {
		return nil, nil
	}
	return &award, nil
}

// Award codes stored in player_season_awards.award.
const (
	AwardGoldenBoot     = 1
	AwardPlayerOfSeason = 2
)

const insertPlayerSeasonAwardStmtSql = "INSERT INTO player_season_awards (league_id, season, award, player_id, value) " +
	"VALUES ($1, $2, $3, $4, $5) ON CONFLICT (league_id, season, award) DO NOTHING"

// RecordPlayerSeasonAward permanently records playerId as leagueId's winner of award for season -
// see engine.awardSeasonHonors. Idempotent per (league, season, award): a season is only ever
// awarded once, so a repeat call is a no-op rather than overwriting the first winner.
func RecordPlayerSeasonAward(dbCtx *DatabaseContext, leagueId int64, season int, award int, playerId int64, value int) error {
	_, err := dbCtx.Db.Exec(insertPlayerSeasonAwardStmtSql, leagueId, season, award, playerId, value)
	return err
}

// PlayerSeasonAwards counts how many times a player has won each season-end honor - see
// GetPlayerSeasonAwards.
type PlayerSeasonAwards struct {
	GoldenBoots     int `json:"goldenBoots"`
	PlayerOfSeasons int `json:"playerOfSeasons"`
}

const getPlayerSeasonAwardsStmtSql = "SELECT award, count(*) FROM player_season_awards WHERE player_id = $1 GROUP BY award"

// GetPlayerSeasonAwards returns how many Golden Boots and Player of the Season awards playerId
// has won across every league and season.
func GetPlayerSeasonAwards(dbCtx *DatabaseContext, playerId int64) (PlayerSeasonAwards, error) {
	var awards PlayerSeasonAwards
	rows, err := dbCtx.Db.Query(getPlayerSeasonAwardsStmtSql, playerId)
	if err != nil {
		return awards, err
	}
	defer rows.Close()
	for rows.Next() {
		var award, count int
		if err := rows.Scan(&award, &count); err != nil {
			return awards, err
		}
		switch award {
		case AwardGoldenBoot:
			awards.GoldenBoots = count
		case AwardPlayerOfSeason:
			awards.PlayerOfSeasons = count
		}
	}
	return awards, rows.Err()
}
