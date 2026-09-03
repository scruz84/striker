package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

// RetirementForcedAge is the age at which a player retires unconditionally.
const RetirementForcedAge = 40

// RetirementEligibleAge is the earliest age a player might retire, with the chance rising
// each year after it (see ShouldRetire) up to RetirementForcedAge.
const RetirementEligibleAge = 33
const RetirementChancePerYear = 0.15

// ShouldRetire decides whether a player calls it a career this season, given their age
// *after* this season's aging. Below RetirementEligibleAge it's never; at/after
// RetirementForcedAge it's always; in between, the chance rises 15 percentage points per year.
func ShouldRetire(age int, r *rand.Rand) bool {
	if age >= RetirementForcedAge {
		return true
	}
	if age < RetirementEligibleAge {
		return false
	}
	chance := float32(age-RetirementEligibleAge+1) * RetirementChancePerYear
	return r.Float32() < chance
}

// ageTrendFor is the typical per-season skill delta for a player's age: young players
// improve, players past their prime decline, and those in between hold roughly steady.
func ageTrendFor(age int) int {
	switch {
	case age < 24:
		return 1
	case age < 30:
		return 0
	default:
		return -1
	}
}

// driftSkill applies a player's age trend plus a little noise (-1..+1) to one skill value,
// clamped to a sane [1, 99] range.
func driftSkill(value int, age int, r *rand.Rand) int {
	value += ageTrendFor(age) + r.Intn(3) - 1
	if value < 1 {
		value = 1
	}
	if value > 99 {
		value = 99
	}
	return value
}

// ProgressPlayers ages every player on every team by one season: retirees are removed, the
// rest have their skills drift per ageTrendFor, and any team left short of a full squad is
// topped back up from the youth academy (see common.CreateAcademyPlayers). Meant to run once
// per season, from AdvanceSeasonIfComplete.
func ProgressPlayers(dbCtx *common.DatabaseContext) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	teamIds, err := common.GetAllTeamIds(dbCtx)
	if err != nil {
		log.Errorf("error listing teams for player progression: %s", err)
		return
	}

	for _, teamId := range teamIds {
		players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting players for team %d progression: %s", teamId, err)
			continue
		}

		for _, player := range *players {
			newAge := player.Age + 1
			if ShouldRetire(newAge, r) {
				if err := common.RetirePlayer(dbCtx, player.Id); err != nil {
					log.Errorf("error retiring player %d: %s", player.Id, err)
				}
				continue
			}

			goalKeeper := driftSkill(player.GoalKeeper, newAge, r)
			defense := driftSkill(player.Defense, newAge, r)
			lateral := driftSkill(player.Lateral, newAge, r)
			pass := driftSkill(player.Pass, newAge, r)
			midfield := driftSkill(player.Midfield, newAge, r)
			dribbling := driftSkill(player.Dribbling, newAge, r)
			shot := driftSkill(player.Shot, newAge, r)
			if err := common.UpdatePlayerProgression(dbCtx, player.Id, newAge,
				goalKeeper, defense, lateral, pass, midfield, dribbling, shot); err != nil {
				log.Errorf("error updating progression for player %d: %s", player.Id, err)
			}
		}

		topUpAcademy(dbCtx, teamId)
	}
}

func topUpAcademy(dbCtx *common.DatabaseContext, teamId int64) {
	count, err := common.CountTeamPlayers(dbCtx, teamId)
	if err != nil {
		log.Errorf("error counting players for team %d: %s", teamId, err)
		return
	}
	if count >= common.SquadSize {
		return
	}
	needed := common.SquadSize - count
	log.Infof("team %d down to %d players, generating %d academy prospects", teamId, count, needed)
	if err := common.CreateAcademyPlayers(dbCtx, teamId, needed); err != nil {
		log.Errorf("error generating academy players for team %d: %s", teamId, err)
	}
}
