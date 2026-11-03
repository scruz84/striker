package common

import (
	"database/sql"
	"errors"
	"math"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// Emergent rivalries (see openspec/specs/rivalries): every pair of clubs of the same country has one
// score, 0 to RivalryMaxScore, grown by their shared history (see AddRivalryEvent's callers) and
// multiplied by RivalryDecayFactor at each season rollover. The score sets the pair's level, which
// a match takes at kickoff (see GetRivalryLevel) and keeps for good (matches.rivalry_level,
// cup_ties.rivalry_level).

const (
	RIVALRY_NONE    = 0
	RIVALRY_RIVAL   = 1
	RIVALRY_CLASICO = 2

	RivalryMaxScore         = 40.0
	RivalryRivalThreshold   = 8.0
	RivalryClasicoThreshold = 16.0
	// RivalryDecayFactor multiplies every pair score at its country's rollover; a pair that falls
	// below RivalryDecayFloor is reset to 0.
	RivalryDecayFactor = 0.6
	RivalryDecayFloor  = 0.5
	// RivalryHumanBonus multiplies the points of any event between two human-owned clubs.
	RivalryHumanBonus = 1.5

	// Season and transfer events.
	RivalryTitleRacePoints     = 4.0
	RivalryKeyPlayerSalePoints = 3.0
)

// RivalryLevel is the level of a pair at score: none below 8, Rival from 8, Clásico from 16.
func RivalryLevel(score float64) int {
	switch {
	case score >= RivalryClasicoThreshold:
		return RIVALRY_CLASICO
	case score >= RivalryRivalThreshold:
		return RIVALRY_RIVAL
	default:
		return RIVALRY_NONE
	}
}

// DecayRivalryScore is one rollover's decay of a pair score (see DecayCountryRivalriesStmtSql,
// which applies the same rule in SQL).
func DecayRivalryScore(score float64) float64 {
	decayed := score * RivalryDecayFactor
	if decayed < RivalryDecayFloor {
		return 0
	}
	return decayed
}

// TeamRivalry is one of a club's rivals as its team page lists it.
type TeamRivalry struct {
	TeamId   int64   `json:"teamId"`
	TeamName string  `json:"teamName"`
	Score    float64 `json:"score"`
	Level    int     `json:"level"`
}

const (
	rivalryPairStmtSql = "SELECT a.country, b.country, a.is_national_team, b.is_national_team, " +
		"a.owner_id IS NOT NULL AND a.owner_id > 0, b.owner_id IS NOT NULL AND b.owner_id > 0 " +
		"FROM teams a, teams b WHERE a.team_id = $1 AND b.team_id = $2"
	ensureRivalryStmtSql = "INSERT INTO team_rivalries (team_low_id, team_high_id, country) VALUES ($1, $2, $3) " +
		"ON CONFLICT (team_low_id, team_high_id) DO NOTHING"
	lockRivalryStmtSql   = "SELECT score FROM team_rivalries WHERE team_low_id = $1 AND team_high_id = $2 FOR UPDATE"
	updateRivalryStmtSql = "UPDATE team_rivalries SET score = $3, updated_at = now() WHERE team_low_id = $1 AND team_high_id = $2"
	// GetRivalryScoreStmtSql reads a pair's score (no row: 0).
	GetRivalryScoreStmtSql = "SELECT score FROM team_rivalries WHERE team_low_id = $1 AND team_high_id = $2"
	// GetTopRivalriesStmtSql lists a club's pairs with a level, highest score first.
	GetTopRivalriesStmtSql = "SELECT CASE WHEN r.team_low_id = $1 THEN r.team_high_id ELSE r.team_low_id END AS other, t.name, r.score " +
		"FROM team_rivalries r JOIN teams t ON t.team_id = CASE WHEN r.team_low_id = $1 THEN r.team_high_id ELSE r.team_low_id END " +
		"WHERE (r.team_low_id = $1 OR r.team_high_id = $1) AND r.score >= $2 ORDER BY r.score DESC, other LIMIT $3"
	// DecayCountryRivalriesStmtSql is one rollover's decay of every pair of a country, in one
	// statement (see DecayRivalryScore).
	DecayCountryRivalriesStmtSql = "UPDATE team_rivalries SET score = CASE WHEN score * $2 < $3 THEN 0 ELSE score * $2 END, updated_at = now() " +
		"WHERE country = $1 AND score > 0"
	isCupFinalStmtSql = "SELECT count(*) = 1 FROM cup_ties WHERE cup_id = $1 AND round = $2"
)

func rivalryPairIds(a int64, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}

// rivalryPair reports whether a and b can have a rivalry - two different clubs (never a national
// team) of the same country - with that country and whether both have a human owner now.
func rivalryPair(dbCtx *DatabaseContext, a int64, b int64) (country string, bothHuman bool, ok bool, err error) {
	if a == b {
		return "", false, false, nil
	}
	var countryA, countryB sql.NullString
	var nationalA, nationalB, humanA, humanB bool
	err = dbCtx.Db.QueryRow(rivalryPairStmtSql, a, b).Scan(&countryA, &countryB, &nationalA, &nationalB, &humanA, &humanB)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, false, nil
	}
	if err != nil {
		return "", false, false, err
	}
	if nationalA || nationalB || !countryA.Valid || countryA.String != countryB.String {
		return "", false, false, nil
	}
	return countryA.String, humanA && humanB, true, nil
}

// AddRivalryPoints adds points to the pair (a, b) of country, capped at RivalryMaxScore, and returns
// its score before and after. The row is created if missing and locked for the read-modify-write,
// so concurrent additions to one pair never lose an update.
func AddRivalryPoints(dbCtx *DatabaseContext, a int64, b int64, country string, points float64) (float64, float64, error) {
	low, high := rivalryPairIds(a, b)
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(ensureRivalryStmtSql, low, high, country); err != nil {
		return 0, 0, err
	}
	var old float64
	if err := tx.QueryRow(lockRivalryStmtSql, low, high).Scan(&old); err != nil {
		return 0, 0, err
	}
	updated := math.Max(0, math.Min(RivalryMaxScore, old+points))
	if _, err := tx.Exec(updateRivalryStmtSql, low, high, updated); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return old, updated, nil
}

// AddRivalryEvent applies one rivalry event worth basePoints to the pair (a, b): nothing for a pair
// that can't have a rivalry (see rivalryPair), × RivalryHumanBonus when both clubs have a human
// owner, and a NOTIFICATION_RIVALRY to each human owner when it raises the pair's level.
// Best-effort: an error is logged, never returned, since the event that earned the points has
// already happened. A match's caller runs this only after the match's replay (see
// engine.applyPostMatchEffects), so the notification never spoils it.
func AddRivalryEvent(dbCtx *DatabaseContext, a int64, b int64, basePoints float64) {
	if old, updated, ok := AddRivalryEventQuietly(dbCtx, a, b, basePoints); ok {
		NotifyRivalryRise(dbCtx, a, b, old, updated)
	}
}

// AddRivalryEventQuietly is AddRivalryEvent without the notification, returning the pair's score
// before and after (ok false when nothing was added) - for the season's title race, whose rise is
// only announced if it survives that rollover's decay (see engine.applyTitleRaceRivalries).
func AddRivalryEventQuietly(dbCtx *DatabaseContext, a int64, b int64, basePoints float64) (float64, float64, bool) {
	if basePoints <= 0 {
		return 0, 0, false
	}
	country, bothHuman, ok, err := rivalryPair(dbCtx, a, b)
	if err != nil {
		log.Errorf("error reading rivalry pair %d-%d: %s", a, b, err)
		return 0, 0, false
	}
	if !ok {
		return 0, 0, false
	}
	points := basePoints
	if bothHuman {
		points *= RivalryHumanBonus
	}
	old, updated, err := AddRivalryPoints(dbCtx, a, b, country, points)
	if err != nil {
		log.Errorf("error adding %.1f rivalry points to pair %d-%d: %s", points, a, b, err)
		return 0, 0, false
	}
	return old, updated, true
}

// NotifyRivalryRise tells both clubs' human owners about the pair's new level when going from score
// old to score updated raised it; a level that stays or falls is never notified.
func NotifyRivalryRise(dbCtx *DatabaseContext, a int64, b int64, old float64, updated float64) {
	if level := RivalryLevel(updated); level > RivalryLevel(old) {
		notifyRivalryRise(dbCtx, a, b, level)
		notifyRivalryRise(dbCtx, b, a, level)
	}
}

// notifyRivalryRise tells teamId's owner (if any) that otherId is now its Rival or Clásico.
func notifyRivalryRise(dbCtx *DatabaseContext, teamId int64, otherId int64, level int) {
	other, err := GetTeamById(otherId, dbCtx)
	if err != nil {
		return
	}
	msg := NotificationMessage{
		Text:   other.TeamName + " is now your rival",
		Key:    "notifications.messages.rivalryRival",
		Params: map[string]any{"team": other.TeamName, "teamId": otherId},
	}
	if level == RIVALRY_CLASICO {
		msg.Text = "Your matches against " + other.TeamName + " are now a Clásico"
		msg.Key = "notifications.messages.rivalryClasico"
	}
	_ = NotifyTeamOwnerWithLink(dbCtx, teamId, NOTIFICATION_RIVALRY, msg, "/team/"+strconv.FormatInt(otherId, 10))
}

// GetRivalryScore is the pair's current score (0 when they have none).
func GetRivalryScore(dbCtx *DatabaseContext, a int64, b int64) (float64, error) {
	low, high := rivalryPairIds(a, b)
	var score float64
	err := dbCtx.Db.QueryRow(GetRivalryScoreStmtSql, low, high).Scan(&score)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return score, err
}

// GetRivalryLevel is the pair's current level - what a match between them kicks off with.
func GetRivalryLevel(dbCtx *DatabaseContext, a int64, b int64) (int, error) {
	if a == b {
		return RIVALRY_NONE, nil
	}
	score, err := GetRivalryScore(dbCtx, a, b)
	if err != nil {
		return RIVALRY_NONE, err
	}
	return RivalryLevel(score), nil
}

// KickoffRivalryLevel is GetRivalryLevel for a match about to kick off, logging and playing at
// level none when it can't be read: a rivalry must never stop a match.
func KickoffRivalryLevel(dbCtx *DatabaseContext, home int64, away int64) int {
	level, err := GetRivalryLevel(dbCtx, home, away)
	if err != nil {
		log.Errorf("error reading the rivalry level of %d-%d: %s", home, away, err)
		return RIVALRY_NONE
	}
	return level
}

// GetTopRivalries lists up to limit of teamId's pairs with a level, highest score first.
func GetTopRivalries(dbCtx *DatabaseContext, teamId int64, limit int) ([]TeamRivalry, error) {
	rows, err := dbCtx.Db.Query(GetTopRivalriesStmtSql, teamId, RivalryRivalThreshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TeamRivalry, 0, limit)
	for rows.Next() {
		var r TeamRivalry
		if err := rows.Scan(&r.TeamId, &r.TeamName, &r.Score); err != nil {
			return nil, err
		}
		r.Level = RivalryLevel(r.Score)
		out = append(out, r)
	}
	return out, rows.Err()
}

// DecayCountryRivalries applies one rollover's decay to every pair of country (see
// engine.advanceCountrySeasonIfComplete, which runs it under the country's rollover lock). A level
// that falls is never notified.
func DecayCountryRivalries(dbCtx *DatabaseContext, country string) error {
	_, err := dbCtx.Db.Exec(DecayCountryRivalriesStmtSql, country, RivalryDecayFactor, RivalryDecayFloor)
	return err
}

// IsCupFinal reports whether round is cupId's final: the only tie of its round.
func IsCupFinal(dbCtx *DatabaseContext, cupId int64, round int) (bool, error) {
	var final bool
	err := dbCtx.Db.QueryRow(isCupFinalStmtSql, cupId, round).Scan(&final)
	return final, err
}

// fixtureRivalrySql is the rivalry level a fixture row shows (alias a, with home_team_id,
// away_team_id, played and rivalry_level columns): the level it kicked off with once played, the
// pair's current level before. The points a match earns are only added after its replay, so a
// level change it causes can never show on it or on the next fixture before then.
func fixtureRivalrySql(a string) string {
	current := "(SELECT CASE WHEN r.score >= " + strconv.FormatFloat(RivalryClasicoThreshold, 'f', -1, 64) + " THEN " + strconv.Itoa(RIVALRY_CLASICO) +
		" WHEN r.score >= " + strconv.FormatFloat(RivalryRivalThreshold, 'f', -1, 64) + " THEN " + strconv.Itoa(RIVALRY_RIVAL) + " ELSE " + strconv.Itoa(RIVALRY_NONE) + " END " +
		"FROM team_rivalries r WHERE r.team_low_id = LEAST(" + a + ".home_team_id, " + a + ".away_team_id) AND r.team_high_id = GREATEST(" + a + ".home_team_id, " + a + ".away_team_id))"
	return "CASE WHEN " + a + ".played THEN " + a + ".rivalry_level ELSE COALESCE(" + current + ", " + strconv.Itoa(RIVALRY_NONE) + ") END"
}

// GetFixtureRivalry is the rivalry level a league match (kind MatchKindLeague) or cup tie (MatchKindCup)
// shows - see fixtureRivalrySql; RIVALRY_NONE for anything else or when it can't be read.
func GetFixtureRivalry(dbCtx *DatabaseContext, kind string, id int64) int {
	table, idColumn := "matches", "match_id"
	switch kind {
	case MatchKindLeague:
	case MatchKindCup:
		table, idColumn = "cup_ties", "tie_id"
	default:
		return RIVALRY_NONE
	}
	var level int
	err := dbCtx.Db.QueryRow("SELECT "+fixtureRivalrySql("f")+" FROM "+table+" f WHERE f."+idColumn+" = $1 AND f.home_team_id IS NOT NULL AND f.away_team_id IS NOT NULL", id).Scan(&level)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Errorf("error reading the rivalry level of %s %d: %s", kind, id, err)
	}
	return level
}
