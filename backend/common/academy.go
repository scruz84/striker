package common

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CanteraSize is how many prospects a team's youth academy pool ("cantera") holds at once -
// enough to give a manager a real choice without this turning into its own squad-management
// screen. See GetCantera.
const CanteraSize = 3

var ErrPlayerNotInAcademy = errors.New("player is not in this team's academy")
var ErrAcademyPromotionAlreadyUsedThisWeek = errors.New("a player has already been promoted from the academy this week")

// Excludes what a senior-squad query would show (see GetPlayersFromTeamStmtSql) - a cantera
// prospect is deliberately not part of that list until promoted.
const GetCanteraStmtSql = "SELECT player_id, team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country " +
	"FROM players WHERE team_id = $1 AND in_academy = true ORDER BY player_id"

const CountCanteraStmtSql = "SELECT count(*) FROM players WHERE team_id = $1 AND in_academy = true"

// GetCantera returns teamId's current academy prospects, generating fresh ones first if the
// pool has fewer than CanteraSize - lazily, so a team is never left with nothing to consider,
// including one that predates this feature entirely and has never had a cantera row created for
// it at all.
func GetCantera(dbCtx *DatabaseContext, teamId int64) ([]*Player, error) {
	if err := topUpCantera(dbCtx, teamId); err != nil {
		return nil, err
	}

	rows, err := dbCtx.Db.Query(GetCanteraStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]*Player, 0, CanteraSize)
	for rows.Next() {
		var p Player
		var birthDay int64
		var avatar string
		var birthCountry sql.NullString
		if err := rows.Scan(&p.Id, &p.TeamId, &p.FirstName, &p.LastName, &birthDay,
			&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot,
			&p.Header, &p.Quick, &p.Creative, &p.Technical, &p.Wage, &p.Age, &avatar, &p.ContractYearsRemaining, &birthCountry); err != nil {
			return nil, err
		}
		p.BirthDay = time.UnixMilli(birthDay)
		p.Avatar = GetAvatarFromString(avatar)
		p.BirthCountry = birthCountry.String
		players = append(players, &p)
	}
	return players, nil
}

// topUpCantera generates fresh prospects if teamId's academy pool has fewer than CanteraSize.
// A hired trainer raises the prospect skill ceiling here exactly as it does for an automatic
// squad backfill (see academySkillMax) - investing in coaching pays off in the pipeline too.
func topUpCantera(dbCtx *DatabaseContext, teamId int64) error {
	var count int
	if err := dbCtx.Db.QueryRow(CountCanteraStmtSql, teamId).Scan(&count); err != nil {
		return err
	}
	if count >= CanteraSize {
		return nil
	}

	trainerSkill := 0
	if trainer, err := GetTeamTrainer(dbCtx, teamId); err != nil {
		return err
	} else if trainer != nil {
		trainerSkill = trainer.Skill
	}
	return createCanteraProspects(dbCtx, teamId, CanteraSize-count, trainerSkill)
}

// createCanteraProspects generates `count` young prospects into teamId's cantera (in_academy =
// true) - same skill/age ranges and trainer bonus as an automatic senior-squad backfill (see
// CreateAcademyPlayers), but held separately until a manager promotes one (see
// PromoteFromAcademy) instead of joining the senior squad immediately.
func createCanteraProspects(dbCtx *DatabaseContext, teamId int64, count int, trainerSkill int) error {
	if count <= 0 {
		return nil
	}
	country, err := GetTeamCountry(dbCtx, teamId)
	if err != nil {
		return err
	}
	insertStmt, err := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, count, AcademySkillMin, academySkillMax(trainerSkill), AcademyAgeMin, AcademyAgeMax, country, true)
	return nil
}

const GetLastAcademyPromotionStmtSql = "SELECT last_academy_promotion_at FROM teams WHERE team_id = $1"
const SetLastAcademyPromotionStmtSql = "UPDATE teams SET last_academy_promotion_at = $1 WHERE team_id = $2"
const PromotePlayerFromAcademyStmtSql = "UPDATE players SET in_academy = false, wage = $1, contract_years_remaining = $2 WHERE player_id = $3"
const GetPlayerAcademyOwnershipForUpdateStmtSql = "SELECT team_id, in_academy FROM players WHERE player_id = $1 " + ForUpdate

// CanPromoteFromAcademy reports whether teamId's weekly academy promotion is still available,
// and if not, when the next one opens up (the coming Monday 00:00 UTC - see StartOfWeek), so the
// frontend can show when to come back.
func CanPromoteFromAcademy(dbCtx *DatabaseContext, teamId int64) (canPromote bool, nextAvailableAt time.Time, err error) {
	var lastPromotion sql.NullTime
	if err := dbCtx.Db.QueryRow(GetLastAcademyPromotionStmtSql, teamId).Scan(&lastPromotion); err != nil {
		return false, time.Time{}, err
	}
	canPromote, nextAvailableAt = canPromoteFromAcademyAt(lastPromotion, time.Now())
	return canPromote, nextAvailableAt, nil
}

// canPromoteFromAcademyAt is CanPromoteFromAcademy's actual week-boundary decision, pulled out
// as a pure function (same reasoning as nextAllowedFriendlyDate in friendlies.go) so the
// once-per-real-calendar-week rule can be unit tested without a live database.
func canPromoteFromAcademyAt(lastPromotion sql.NullTime, now time.Time) (canPromote bool, nextAvailableAt time.Time) {
	weekStart := StartOfWeek(now)
	if !lastPromotion.Valid || lastPromotion.Time.Before(weekStart) {
		return true, time.Time{}
	}
	return false, weekStart.AddDate(0, 0, 7)
}

// PromoteFromAcademy moves one cantera prospect onto teamId's senior squad, granting them a
// fresh professional contract and market wage exactly as SignFreeAgent does for a free agent -
// promotion is, mechanically, their first professional signing, with no other requirement beyond
// the weekly cap (see CanPromoteFromAcademy) and actually owning the prospect. Tops the cantera
// back up to CanteraSize afterward so the manager immediately sees a full pool again next time.
func PromoteFromAcademy(dbCtx *DatabaseContext, playerId int64, teamId int64) error {
	canPromote, _, err := CanPromoteFromAcademy(dbCtx, teamId)
	if err != nil {
		return err
	}
	if !canPromote {
		return ErrAcademyPromotionAlreadyUsedThisWeek
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentTeamId sql.NullInt64
	var inAcademy bool
	err = tx.QueryRow(GetPlayerAcademyOwnershipForUpdateStmtSql, playerId).Scan(&currentTeamId, &inAcademy)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayerNotFound
	}
	if err != nil {
		return err
	}
	if !currentTeamId.Valid || currentTeamId.Int64 != teamId || !inAcademy {
		return ErrPlayerNotInAcademy
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	wage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	contractYears := RandomContractYears()

	if _, err := tx.Exec(PromotePlayerFromAcademyStmtSql, wage, contractYears, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(SetLastAcademyPromotionStmtSql, time.Now(), teamId); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	AcademyPromotionsCounter.Add(context.Background(), 1)

	playerName := player.FirstName + " " + player.LastName
	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_ACADEMY_PROMOTION,
		"Promoted "+playerName+" from the academy to the first team")

	if err := topUpCantera(dbCtx, teamId); err != nil {
		// Best-effort: the promotion itself already succeeded and committed. GetCantera will
		// retry the top-up on the next read regardless.
		return nil
	}
	return nil
}
