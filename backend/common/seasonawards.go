package common

import (
	"encoding/json"
	"sort"
)

// SeasonAward is a league's season-end award winner (Golden Boot or Player of the Season) - see
// GetPlayerOfTheSeason. Golden Boot itself is just GetTopScorers' #1 result, so it needs no
// separate type.
type SeasonAward struct {
	PlayerId  int64
	FirstName string
	LastName  string
	TeamId    int64
	TeamName  string
	Score     int
}

const getSeasonAwardEventsStmtSql = "SELECT me.event_type, me.team_players FROM match_events me " +
	"JOIN matches m ON m.match_id = me.match_id " +
	"WHERE m.league_id = $1 AND m.season = $2 AND me.event_type IN ($3, $4, $5)"

// Deliberately simple weights for "Player of the Season" - not a real performance model, just
// enough to make it feel earned: rewards goals, docks cards, nothing more sophisticated than that.
const (
	playerOfSeasonGoalWeight       = 3
	playerOfSeasonYellowCardWeight = -1
	playerOfSeasonRedCardWeight    = -3
)

// tallySeasonAwardScores sums each player's weighted goal/card score from raw match_events.
// team_players rows (see tallyGoalsByPlayer, which this generalizes to more than one event type).
// Pure so it's testable without a database.
func tallySeasonAwardScores(events []struct {
	EventType       int
	TeamPlayersJson string
}) map[int64]int {
	scores := make(map[int64]int)
	for _, event := range events {
		var weight int
		switch event.EventType {
		case GOAL:
			weight = playerOfSeasonGoalWeight
		case YELLOW_CARD:
			weight = playerOfSeasonYellowCardWeight
		case RED_CARD:
			weight = playerOfSeasonRedCardWeight
		default:
			continue
		}
		var playerIds []int64
		if err := json.Unmarshal([]byte(event.TeamPlayersJson), &playerIds); err != nil || len(playerIds) == 0 {
			continue
		}
		scores[playerIds[0]] += weight
	}
	return scores
}

// rankSeasonAwardScores orders every player with a positive score, highest first, ties broken by
// lowest player id for a stable result - mirrors rankScorerStandings' tie-break. A player with a
// score of 0 or below never qualifies (an empty season, or one where cards outweighed every goal
// scored - nobody "stood out" enough to deserve the award).
func rankSeasonAwardScores(scores map[int64]int) []int64 {
	ids := make([]int64, 0, len(scores))
	for id, score := range scores {
		if score > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] != scores[ids[j]] {
			return scores[ids[i]] > scores[ids[j]]
		}
		return ids[i] < ids[j]
	})
	return ids
}

// GetPlayerOfTheSeason returns the league's Player of the Season for the given season by a
// simple goals/cards weighted score (see the playerOfSeason* weights above), or nil if nobody
// qualifies yet.
func GetPlayerOfTheSeason(dbCtx *DatabaseContext, leagueId int64, season int) (*SeasonAward, error) {
	rows, err := dbCtx.Db.Query(getSeasonAwardEventsStmtSql, leagueId, season, GOAL, YELLOW_CARD, RED_CARD)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]struct {
		EventType       int
		TeamPlayersJson string
	}, 0)
	for rows.Next() {
		var eventType int
		var teamPlayersJson string
		if err := rows.Scan(&eventType, &teamPlayersJson); err != nil {
			return nil, err
		}
		events = append(events, struct {
			EventType       int
			TeamPlayersJson string
		}{eventType, teamPlayersJson})
	}

	scores := tallySeasonAwardScores(events)
	// Skip a player no longer resolvable to a current team (retired, released to free agency -
	// same "current squads only" limitation GetTopScorers has) in favor of the next-best score,
	// rather than failing the whole award for one unlucky pick.
	for _, playerId := range rankSeasonAwardScores(scores) {
		award := SeasonAward{PlayerId: playerId, Score: scores[playerId]}
		// GetPlayerTeamForScorerStmtSql also selects birth_country (for the top-scorers flag), which
		// an award has no use for.
		var birthCountry string
		if err := dbCtx.Db.QueryRow(GetPlayerTeamForScorerStmtSql, playerId).Scan(&award.FirstName, &award.LastName, &award.TeamId, &award.TeamName, &birthCountry); err != nil {
			continue
		}
		return &award, nil
	}
	return nil, nil
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
