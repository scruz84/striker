package common

import (
	"errors"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

// FreeAgentInjectionSkillMin/Max and FreeAgentInjectionAgeMin/Max bound the randomly generated
// players InjectFreeAgents adds straight to the free agent pool - the same skill range as a
// freshly created team's starting squad (SeniorSkillMin/Max), since these are meant to read as
// plausible transfer-market signings rather than raw academy prospects (AcademySkillMin/Max).
const FreeAgentInjectionSkillMin, FreeAgentInjectionSkillMax = SeniorSkillMin, SeniorSkillMax
const FreeAgentInjectionAgeMin, FreeAgentInjectionAgeMax = 20, 30

// InsertFreeAgentStmtSql mirrors InsertNewPlayerStmtSql but omits team_id entirely (defaulting
// to NULL) rather than taking one - unlike doCreatePlayers, InjectFreeAgents never has a team to
// assign, so there's no shirt number, no club history row (see RecordPlayerClubHistory), and no
// contract (contract_years_remaining = 0, same as a just-released player - see
// ReleaseFromTeamStmtSql).
const InsertFreeAgentStmtSql = "INSERT INTO players (first_name, last_name, birth_day, " +
	"goal_keeper, defense, \"lateral\", pass, midfield, dribbling, shot, \"header\", quick, creative, technical, " +
	"avatar, wage, age, contract_years_remaining, birth_country, stamina, form, " +
	"goal_keeper_max, defense_max, lateral_max, pass_max, midfield_max, dribbling_max, shot_max) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, " +
	"$22, $23, $24, $25, $26, $27, $28) " +
	"RETURNING player_id"

// InjectFreeAgents adds count brand-new, never-employed players straight to the free agent pool
// (see GetFreeAgents/GetFreeAgentListings, which simply query players WHERE team_id IS NULL - no
// separate "market" table for free agents to appear on), spread randomly across the world's
// currently active countries. Unlike a released player, an injected one has no last_team_id, so
// signing them never charges a poaching premium (see SignFreeAgent). This exists purely to keep
// the transfer market's free agents tab from running dry in a long-running world where the only
// other sources of free agents are a contract expiring (AdvancePlayerContract) or an early
// release (ReleasePlayer) - see engine.InjectFreeAgentsJob, which runs this weekly with the
// admin-configured count from GetFreeAgentInjectionsPerWeek.
func InjectFreeAgents(dbCtx *DatabaseContext, count int) error {
	if count <= 0 {
		return nil
	}
	countries, err := GetActiveCountries(dbCtx)
	if err != nil {
		return err
	}
	if len(countries) == 0 {
		return nil
	}

	insertStmt, err := dbCtx.Db.Prepare(InsertFreeAgentStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for i := 0; i < count; i++ {
		country := countries[rand.Intn(len(countries))]
		firstName, lastName, err := GetPlayerName(dbCtx, country)
		if err != nil {
			log.Errorf("error generating name for injected free agent: %s", err)
			continue
		}
		avatar := RandomAvatarAsString()
		goalKeeper := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		defense := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		lateral := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		pass := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		midfield := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		dribbling := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		shot := FreeAgentInjectionSkillMin + rand.Intn(FreeAgentInjectionSkillMax-FreeAgentInjectionSkillMin+1)
		age := FreeAgentInjectionAgeMin + rand.Intn(FreeAgentInjectionAgeMax-FreeAgentInjectionAgeMin+1)
		wage := PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
		stamina := StaminaGenerationMin + rand.Intn(StaminaGenerationMax-StaminaGenerationMin+1)

		var newPlayerId int64
		err = insertStmt.QueryRow(&firstName, &lastName, time.Now().UnixMilli(),
			goalKeeper, defense, lateral, pass, midfield, dribbling, shot,
			false, false, false, false,
			&avatar, wage, age, 0, &country, stamina, DefaultForm,
			rollSkillCap(goalKeeper), rollSkillCap(defense), rollSkillCap(lateral), rollSkillCap(pass),
			rollSkillCap(midfield), rollSkillCap(dribbling), rollSkillCap(shot)).Scan(&newPlayerId)
		if err != nil {
			log.Errorf("error injecting free agent: %s", err)
		}
	}
	return nil
}

const GetFreeAgentInjectionsPerWeekStmtSql = "SELECT free_agent_injections_per_week FROM season"
const SetFreeAgentInjectionsPerWeekStmtSql = "UPDATE season SET free_agent_injections_per_week = $1"

var ErrInvalidFreeAgentInjectionCount = errors.New("free agent injections per week must be zero or greater")

// GetFreeAgentInjectionsPerWeek returns the admin-configured count of new players
// InjectFreeAgents adds to the free agent pool each real calendar week - see
// handlers.AdminSetFreeAgentInjectionsPerWeekHandler.
func GetFreeAgentInjectionsPerWeek(dbCtx *DatabaseContext) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow(GetFreeAgentInjectionsPerWeekStmtSql).Scan(&n)
	return n, err
}

// SetFreeAgentInjectionsPerWeek updates the count. Zero is allowed (turns injection off
// entirely - free agents then only ever come from a contract expiring or an early release).
func SetFreeAgentInjectionsPerWeek(dbCtx *DatabaseContext, n int) error {
	if n < 0 {
		return ErrInvalidFreeAgentInjectionCount
	}
	_, err := dbCtx.Db.Exec(SetFreeAgentInjectionsPerWeekStmtSql, n)
	return err
}
