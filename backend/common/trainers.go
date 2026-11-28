package common

import (
	"database/sql"
	"errors"
	"math/rand"
)

// TrainerSkillMin/Max bound the randomly assigned rating of a freshly generated trainer
// candidate (see GenerateTrainerCandidates) - the same 1-99 scale as a player skill, so a
// trainer's quality reads consistently against everything else in the game.
const TrainerSkillMin, TrainerSkillMax = 30, 90

// TrainerCandidatesPerTeam is how many fresh unassigned trainer candidates are added to a
// country's open market for every team created there (see handlers/initializer.go's createTeam),
// so the coaching pool grows in step with the world's size instead of ever running dry.
const TrainerCandidatesPerTeam = 1

// TrainerWagePerSkillPointNumerator/Denominator price a trainer steeper per skill point than a
// single player's WagePerSkillPointNumerator/Denominator, since one trainer's effect (see
// engine.ProgressPlayers) spans the whole squad rather than one position. Divided by
// TrainerWagePerSkillPointDenominator for the same reason WagePerSkillPoint was split into a
// numerator/denominator pair (see common.PlayerWageFromSkills): the trainer wage is charged
// alongside player wages, once per round (see engine.deductTrainerWage), so the old flat
// per-skill-point rate needs the same correction to avoid being 7x too high under the default
// daily match schedule.
const TrainerWagePerSkillPointNumerator = 8
const TrainerWagePerSkillPointDenominator = 7

func TrainerWageFromSkill(skill int) int {
	return skill * TrainerWagePerSkillPointNumerator / TrainerWagePerSkillPointDenominator
}

var ErrTrainerNotFound = errors.New("trainer not found")
var ErrTrainerNotAvailable = errors.New("trainer is already employed")
var ErrTrainerNotOwnedByTeam = errors.New("trainer does not work for your team")
var ErrTeamAlreadyHasTrainer = errors.New("team already has a trainer - fire them first")

const trainerColumns = "trainer_id, first_name, last_name, skill, wage"

const InsertTrainerStmtSql = "INSERT INTO trainers (first_name, last_name, skill, wage) VALUES ($1, $2, $3, $4)"
const GetAvailableTrainersStmtSql = "SELECT " + trainerColumns + " FROM trainers WHERE team_id IS NULL ORDER BY skill DESC"
const GetTeamTrainerStmtSql = "SELECT " + trainerColumns + " FROM trainers WHERE team_id = $1"
const GetTrainerByIdStmtSql = "SELECT " + trainerColumns + " FROM trainers WHERE trainer_id = $1"
const GetTrainerOwnershipForUpdateStmtSql = "SELECT team_id FROM trainers WHERE trainer_id = $1 " + ForUpdate
const TeamHasTrainerForUpdateStmtSql = "SELECT EXISTS(SELECT 1 FROM trainers WHERE team_id = $1) " + ForUpdate
const HireTrainerStmtSql = "UPDATE trainers SET team_id = $1 WHERE trainer_id = $2"
const FireTrainerStmtSql = "UPDATE trainers SET team_id = NULL WHERE trainer_id = $1"

// Trainer is a team's head coach - see trainers.sql. A hired trainer's Skill accelerates player
// attribute growth and raises the youth academy's prospect ceiling (see engine.ProgressPlayers).
type Trainer struct {
	Id        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Skill     int    `json:"skill"`
	Wage      int    `json:"wage"`
}

func scanTrainer(row interface{ Scan(...any) error }) (*Trainer, error) {
	var t Trainer
	if err := row.Scan(&t.Id, &t.FirstName, &t.LastName, &t.Skill, &t.Wage); err != nil {
		return nil, err
	}
	return &t, nil
}

// GenerateTrainerCandidates adds `count` unassigned trainer candidates to the open market for
// country - called once per team created (see handlers/initializer.go's createTeam), so the
// pool of available coaches grows in step with the world's size.
func GenerateTrainerCandidates(dbCtx *DatabaseContext, country string, count int) error {
	insertStmt, err := dbCtx.Db.Prepare(InsertTrainerStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for i := 0; i < count; i++ {
		firstName, lastName, err := GetRandomName(dbCtx, country)
		if err != nil {
			return err
		}
		skill := TrainerSkillMin + rand.Intn(TrainerSkillMax-TrainerSkillMin+1)
		wage := TrainerWageFromSkill(skill)
		if _, err := insertStmt.Exec(firstName, lastName, skill, wage); err != nil {
			return err
		}
	}
	return nil
}

// TopUpTrainerMarket tops up the open trainer market so there's roughly one candidate available
// per team in the world - called once at service startup (so an existing world immediately gets
// a market to browse, without waiting on new teams being created or a season rolling over) and
// once per season from engine.ProgressPlayers (so supply keeps pace as trainers get hired). Uses
// "en" as a safe fallback name pool since, like GetFreeAgents, the market isn't scoped by
// country - a trainer generated for one country's teams can be hired by any team.
func TopUpTrainerMarket(dbCtx *DatabaseContext) error {
	teamIds, err := GetAllTeamIds(dbCtx)
	if err != nil {
		return err
	}
	available, err := GetAvailableTrainers(dbCtx)
	if err != nil {
		return err
	}
	shortfall := len(teamIds) - len(available)
	if shortfall <= 0 {
		return nil
	}
	return GenerateTrainerCandidates(dbCtx, "en", shortfall)
}

// GetAvailableTrainers lists every unemployed trainer candidate on the open market, best first.
func GetAvailableTrainers(dbCtx *DatabaseContext) ([]*Trainer, error) {
	rows, err := dbCtx.Db.Query(GetAvailableTrainersStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trainers := make([]*Trainer, 0)
	for rows.Next() {
		t, err := scanTrainer(rows)
		if err != nil {
			return nil, err
		}
		trainers = append(trainers, t)
	}
	return trainers, nil
}

// GetTeamTrainer returns teamId's current head coach, or nil if they haven't hired one.
func GetTeamTrainer(dbCtx *DatabaseContext, teamId int64) (*Trainer, error) {
	t, err := scanTrainer(dbCtx.Db.QueryRow(GetTeamTrainerStmtSql, teamId))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// HireTrainer assigns an unemployed trainer to teamId, at no signing fee - only the wage a team
// takes on is the cost, the same convention SignFreeAgent uses for players. A team can only
// employ one trainer at a time (ErrTeamAlreadyHasTrainer); fire the current one first.
func HireTrainer(dbCtx *DatabaseContext, trainerId int64, teamId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var hasTrainer bool
	if err := tx.QueryRow(TeamHasTrainerForUpdateStmtSql, teamId).Scan(&hasTrainer); err != nil {
		return err
	}
	if hasTrainer {
		return ErrTeamAlreadyHasTrainer
	}

	var currentTeamId sql.NullInt64
	err = tx.QueryRow(GetTrainerOwnershipForUpdateStmtSql, trainerId).Scan(&currentTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTrainerNotFound
	}
	if err != nil {
		return err
	}
	if currentTeamId.Valid {
		return ErrTrainerNotAvailable
	}

	if _, err := tx.Exec(HireTrainerStmtSql, teamId, trainerId); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if trainer, err := scanTrainer(dbCtx.Db.QueryRow(GetTrainerByIdStmtSql, trainerId)); err == nil {
		trainerName := trainer.FirstName + " " + trainer.LastName
		_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRAINER, NotificationMessage{
			Text:   "Hired " + trainerName + " as head coach",
			Key:    "notifications.messages.trainerHired",
			Params: map[string]any{"name": trainerName},
		})
	}
	return nil
}

// FireTrainer releases teamId's current trainer back to the open market immediately, at no
// cost - unlike releasing a player early, there's no contract/severance model for coaching
// staff, just an ongoing wage that stops the moment they're let go.
func FireTrainer(dbCtx *DatabaseContext, trainerId int64, teamId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentTeamId sql.NullInt64
	err = tx.QueryRow(GetTrainerOwnershipForUpdateStmtSql, trainerId).Scan(&currentTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTrainerNotFound
	}
	if err != nil {
		return err
	}
	if !currentTeamId.Valid || currentTeamId.Int64 != teamId {
		return ErrTrainerNotOwnedByTeam
	}

	if _, err := tx.Exec(FireTrainerStmtSql, trainerId); err != nil {
		return err
	}
	return tx.Commit()
}

// trainerSkillFromPlayerSkills converts a retiring player's seven core attributes into a
// trainer candidate's skill rating (see RetireFreeAgentAsTrainer): the plain average, clamped
// to the normal TrainerSkillMin/TrainerSkillMax range so neither a legendary player nor a weak
// one ends up with an implausible coaching rating.
func trainerSkillFromPlayerSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot int) int {
	avg := (goalKeeper + defense + lateral + pass + midfield + dribbling + shot) / 7
	switch {
	case avg < TrainerSkillMin:
		return TrainerSkillMin
	case avg > TrainerSkillMax:
		return TrainerSkillMax
	default:
		return avg
	}
}

// RetireFreeAgentAsTrainer removes player from the game the same way RetirePlayer does (their
// lineup positions, if any, are cleared first, then the row itself deleted), but not without a
// trace - their playing career becomes a fresh, unemployed trainer candidate on the open market
// first, keeping their own name (see trainerSkillFromPlayerSkills for how their skill rating is
// derived). The only caller is engine.ProgressFreeAgents, for a player who has either aged out
// or gone too long without a club - conceptually, someone who can no longer get a playing
// contract moving into coaching instead of vanishing outright.
func RetireFreeAgentAsTrainer(dbCtx *DatabaseContext, player *Player) error {
	skill := trainerSkillFromPlayerSkills(player.GoalKeeper, player.Defense, player.Lateral,
		player.Pass, player.Midfield, player.Dribbling, player.Shot)
	wage := TrainerWageFromSkill(skill)

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(InsertTrainerStmtSql, player.FirstName, player.LastName, skill, wage); err != nil {
		return err
	}
	if err := recordRetiredPlayerTx(tx, *player); err != nil {
		return err
	}
	if err := clearRetiringPlayerLineupsTx(tx, player.Id); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerStmtSql, player.Id); err != nil {
		return err
	}
	return tx.Commit()
}
