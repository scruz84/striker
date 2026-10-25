package common

import (
	"database/sql"
	"time"
)

// SkillDelta is one attribute's value before and after a progression run - After equals Before
// for a skill that didn't drift (see engine.driftSkillsForPosition's independent per-skill
// chance), which is itself meaningful to show (that attribute held steady this season).
type SkillDelta struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

// PlayerEvolution is one player's captured skill/age change from a single progression run,
// applied right after a competitive match they featured in (see engine.updatePlayerSkillsAfterMatch)
// - the randomized drift itself can't be recomputed after the fact, so this is the only record
// of what actually changed and by how much.
type PlayerEvolution struct {
	PlayerId   int64      `json:"playerId"`
	Season     int        `json:"season"`
	Age        SkillDelta `json:"age"`
	GoalKeeper SkillDelta `json:"goalKeeper"`
	Defense    SkillDelta `json:"defense"`
	Lateral    SkillDelta `json:"lateral"`
	Pass       SkillDelta `json:"pass"`
	Midfield   SkillDelta `json:"midfield"`
	Dribbling  SkillDelta `json:"dribbling"`
	Shot       SkillDelta `json:"shot"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// TeamPlayerEvolution is one row of a squad-wide "latest evolutions" view - a PlayerEvolution
// plus enough of the player's identity to list and link to them (see GetLatestPlayerEvolutionsForTeam).
type TeamPlayerEvolution struct {
	PlayerEvolution
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	// BirthCountry lets the list show the player's flag; empty for a legacy player without one.
	BirthCountry string `json:"birthCountry,omitempty"`
}

const InsertPlayerProgressionLogStmtSql = "INSERT INTO player_progression_log " +
	"(player_id, season, age_before, age_after, goal_keeper_before, goal_keeper_after, " +
	"defense_before, defense_after, lateral_before, lateral_after, pass_before, pass_after, " +
	"midfield_before, midfield_after, dribbling_before, dribbling_after, shot_before, shot_after) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) " +
	"RETURNING id"

// playerEvolutionColumns is shared by GetLatestPlayerEvolution and
// GetLatestPlayerEvolutionsForTeam - same column order both scan against.
const playerEvolutionColumns = "player_id, season, age_before, age_after, " +
	"goal_keeper_before, goal_keeper_after, defense_before, defense_after, " +
	"lateral_before, lateral_after, pass_before, pass_after, " +
	"midfield_before, midfield_after, dribbling_before, dribbling_after, " +
	"shot_before, shot_after, created_at"

const GetLatestPlayerEvolutionStmtSql = "SELECT " + playerEvolutionColumns +
	" FROM player_progression_log WHERE player_id = $1 ORDER BY id DESC LIMIT 1"

// GetLatestPlayerEvolutionsForTeamStmtSql picks each of a team's players' single most recent
// progression row (DISTINCT ON requires its leading ORDER BY columns to match - see the inner
// query), then re-sorts the result for display: most recently evolved first, alphabetically
// among those from the same run.
const GetLatestPlayerEvolutionsForTeamStmtSql = "SELECT * FROM (" +
	"SELECT DISTINCT ON (ppl.player_id) p.first_name, p.last_name, COALESCE(p.birth_country, '') AS birth_country, " + playerEvolutionColumnsAliased +
	" FROM player_progression_log ppl JOIN players p ON p.player_id = ppl.player_id" +
	" WHERE p.team_id = $1 ORDER BY ppl.player_id, ppl.id DESC" +
	") latest ORDER BY latest.created_at DESC, latest.last_name"

const playerEvolutionColumnsAliased = "ppl.player_id, ppl.season, ppl.age_before, ppl.age_after, " +
	"ppl.goal_keeper_before, ppl.goal_keeper_after, ppl.defense_before, ppl.defense_after, " +
	"ppl.lateral_before, ppl.lateral_after, ppl.pass_before, ppl.pass_after, " +
	"ppl.midfield_before, ppl.midfield_after, ppl.dribbling_before, ppl.dribbling_after, " +
	"ppl.shot_before, ppl.shot_after, ppl.created_at"

// LogPlayerProgression records one player's before/after age and seven core skills from a single
// progression run - called from engine.updateTeamPlayersAfterMatch at the exact point both are
// in scope, right alongside UpdatePlayerProgression itself. Returns the new row's id so the caller
// can tie a NOTIFICATION_PROGRESSION notification to exactly the rows one specific match produced
// (see GetPlayerEvolutionsForTeamByLogIds) rather than every player's all-time latest row, which
// could span many unrelated matches once progression started running match-by-match instead of
// once a season.
func LogPlayerProgression(dbCtx *DatabaseContext, playerId int64, season int, ageBefore, ageAfter int,
	goalKeeperBefore, goalKeeperAfter, defenseBefore, defenseAfter, lateralBefore, lateralAfter,
	passBefore, passAfter, midfieldBefore, midfieldAfter, dribblingBefore, dribblingAfter,
	shotBefore, shotAfter int) (int64, error) {
	var id int64
	err := dbCtx.Db.QueryRow(InsertPlayerProgressionLogStmtSql, playerId, season, ageBefore, ageAfter,
		goalKeeperBefore, goalKeeperAfter, defenseBefore, defenseAfter, lateralBefore, lateralAfter,
		passBefore, passAfter, midfieldBefore, midfieldAfter, dribblingBefore, dribblingAfter,
		shotBefore, shotAfter).Scan(&id)
	return id, err
}

func scanPlayerEvolution(scan func(dest ...any) error) (PlayerEvolution, error) {
	var e PlayerEvolution
	err := scan(&e.PlayerId, &e.Season, &e.Age.Before, &e.Age.After,
		&e.GoalKeeper.Before, &e.GoalKeeper.After, &e.Defense.Before, &e.Defense.After,
		&e.Lateral.Before, &e.Lateral.After, &e.Pass.Before, &e.Pass.After,
		&e.Midfield.Before, &e.Midfield.After, &e.Dribbling.Before, &e.Dribbling.After,
		&e.Shot.Before, &e.Shot.After, &e.CreatedAt)
	return e, err
}

// GetLatestPlayerEvolution returns a player's most recent progression run, or nil if they've
// never been through one (never played a competitive minute since this table started tracking,
// or the player is new).
func GetLatestPlayerEvolution(dbCtx *DatabaseContext, playerId int64) (*PlayerEvolution, error) {
	row := dbCtx.Db.QueryRow(GetLatestPlayerEvolutionStmtSql, playerId)
	evolution, err := scanPlayerEvolution(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &evolution, nil
}

// GetLatestPlayerEvolutionsForTeam returns every current squad player's most recent progression
// run - the squad-wide "/players/evolutions" view. A player never progressed (too new, or never
// played) simply has no row here. Only used when a viewer opens that page with no specific match
// in mind (see GetPlayerEvolutionsForTeamByLogIds for the notification's own, precise view) - each
// player's own "most recent" row can come from a different match, so this is a browsing view, not
// a report of any one match's results.
func GetLatestPlayerEvolutionsForTeam(dbCtx *DatabaseContext, teamId int64) ([]TeamPlayerEvolution, error) {
	rows, err := dbCtx.Db.Query(GetLatestPlayerEvolutionsForTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTeamPlayerEvolutions(rows)
}

const GetPlayerEvolutionsForTeamByLogIdsStmtSql = "SELECT p.first_name, p.last_name, COALESCE(p.birth_country, ''), " +
	playerEvolutionColumnsAliased +
	" FROM player_progression_log ppl JOIN players p ON p.player_id = ppl.player_id" +
	" WHERE p.team_id = $1 AND ppl.id = ANY($2) ORDER BY p.last_name, p.first_name"

// GetPlayerEvolutionsForTeamByLogIds returns exactly the player_progression_log rows named by
// logIds, scoped to teamId's current squad (so one manager can never pull up another team's
// players just by guessing ids) - the precise set one NOTIFICATION_PROGRESSION notification
// reported, as opposed to GetLatestPlayerEvolutionsForTeam's "whatever each player's own latest
// row happens to be", which drifts away from a specific match's results as soon as a different
// match later updates some (but not all) of the same squad.
func GetPlayerEvolutionsForTeamByLogIds(dbCtx *DatabaseContext, teamId int64, logIds []int64) ([]TeamPlayerEvolution, error) {
	if len(logIds) == 0 {
		return []TeamPlayerEvolution{}, nil
	}
	rows, err := dbCtx.Db.Query(GetPlayerEvolutionsForTeamByLogIdsStmtSql, teamId, logIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTeamPlayerEvolutions(rows)
}

// SkillTrendFromEvolution reduces one progression run to a single signed figure - positive if the
// seven core skills' combined total rose, negative if it fell, 0 for a nil evolution (a player who
// hasn't been through a progression run yet). Used by PlayerSkillTrend and kept as its own pure
// function, independent of any database read, so a contract negotiation's trend-reaction math (see
// ExpectedWageWithIntelligence/MaxAcceptableRenewalYearsWithIntelligence) is unit-testable against
// a fabricated PlayerEvolution rather than a live player_progression_log row.
func SkillTrendFromEvolution(e *PlayerEvolution) int {
	if e == nil {
		return 0
	}
	before := e.GoalKeeper.Before + e.Defense.Before + e.Lateral.Before + e.Pass.Before +
		e.Midfield.Before + e.Dribbling.Before + e.Shot.Before
	after := e.GoalKeeper.After + e.Defense.After + e.Lateral.After + e.Pass.After +
		e.Midfield.After + e.Dribbling.After + e.Shot.After
	return after - before
}

// PlayerSkillTrend is how much a player's combined seven core skills changed in their most
// recently logged progression run (see GetLatestPlayerEvolution) - the signal a contract
// negotiation reacts to, scaled by the player's own intelligence tier (see
// ExpectedWageWithIntelligence/MaxAcceptableRenewalYearsWithIntelligence). 0 for a player who's
// never been through a progression run (too new, or never featured in a match since this table
// started tracking) - same as a flat trend, so their negotiation behaves exactly as it did before
// this feature shipped.
func PlayerSkillTrend(dbCtx *DatabaseContext, playerId int64) (int, error) {
	evolution, err := GetLatestPlayerEvolution(dbCtx, playerId)
	if err != nil {
		return 0, err
	}
	return SkillTrendFromEvolution(evolution), nil
}

func scanTeamPlayerEvolutions(rows *sql.Rows) ([]TeamPlayerEvolution, error) {
	evolutions := make([]TeamPlayerEvolution, 0)
	for rows.Next() {
		var t TeamPlayerEvolution
		if err := rows.Scan(&t.FirstName, &t.LastName, &t.BirthCountry, &t.PlayerId, &t.Season, &t.Age.Before, &t.Age.After,
			&t.GoalKeeper.Before, &t.GoalKeeper.After, &t.Defense.Before, &t.Defense.After,
			&t.Lateral.Before, &t.Lateral.After, &t.Pass.Before, &t.Pass.After,
			&t.Midfield.Before, &t.Midfield.After, &t.Dribbling.Before, &t.Dribbling.After,
			&t.Shot.Before, &t.Shot.After, &t.CreatedAt); err != nil {
			return nil, err
		}
		evolutions = append(evolutions, t)
	}
	return evolutions, rows.Err()
}
