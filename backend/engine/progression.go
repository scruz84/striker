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

// trainerBonus converts a hired head coach's skill rating into a flat per-season attribute-
// growth bonus, on top of ageTrendFor: nothing below TrainerSkillMin (so a mediocre trainer is
// no better than having none), +1 for a solid one, +2 for an elite one. A team with no trainer
// passes skill 0 here, so this is fully backward compatible with the pre-trainer behavior.
func trainerBonus(trainerSkill int) int {
	switch {
	case trainerSkill >= 80:
		return 2
	case trainerSkill >= common.TrainerSkillMin:
		return 1
	default:
		return 0
	}
}

// driftSkill applies a player's age trend, a hired trainer's bonus (0 with no trainer), and a
// little noise (-1..+1) to one skill value, clamped to a sane [1, 99] range.
func driftSkill(value int, age int, trainerSkill int, r *rand.Rand) int {
	value += ageTrendFor(age) + trainerBonus(trainerSkill) + r.Intn(3) - 1
	if value < 1 {
		value = 1
	}
	if value > 99 {
		value = 99
	}
	return value
}

// primaryPosition picks the tactical position a player spent the most minutes at this season
// (see common.GetTeamPlayerPositionMinutes), ties broken by the lowest position id for
// determinism, and sums their total minutes across every position they occupied. ok is false
// only when the player has no recorded minutes at all - i.e. they never played.
func primaryPosition(minutesByPosition map[int]int) (position int, totalMinutes int, ok bool) {
	position, bestMinutes := -1, -1
	for p, minutes := range minutesByPosition {
		totalMinutes += minutes
		if minutes > bestMinutes || (minutes == bestMinutes && p < position) {
			position, bestMinutes = p, minutes
		}
	}
	return position, totalMinutes, position != -1
}

// driftSkillsForPosition is driftSkill restricted to whichever of a player's seven numeric
// skills actually get used playing `position` - mirroring common.PositionValue's own
// position->attribute weighting, so what improves tracks exactly what match minutes at that
// position would plausibly sharpen (a defender's shooting doesn't get better from defending).
// Every relevant skill independently rolls playChance (see common.FullSeasonMinutesReference)
// before drifting; skills outside the relevant set are always left untouched.
func driftSkillsForPosition(p common.Player, position int, playChance float32, age int, trainerSkill int, r *rand.Rand) (
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot int) {
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot =
		p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot

	maybeDrift := func(current int) int {
		if r.Float32() >= playChance {
			return current
		}
		return driftSkill(current, age, trainerSkill, r)
	}

	switch position {
	case common.GOALKEEPER:
		goalKeeper = maybeDrift(goalKeeper)
	case common.DEFENSE:
		defense = maybeDrift(defense)
	case common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT:
		lateral, defense = maybeDrift(lateral), maybeDrift(defense)
	case common.MIDFIELD:
		midfield, pass = maybeDrift(midfield), maybeDrift(pass)
	case common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT:
		lateral, midfield, pass = maybeDrift(lateral), maybeDrift(midfield), maybeDrift(pass)
	case common.STRIKER:
		shot, pass, dribbling = maybeDrift(shot), maybeDrift(pass), maybeDrift(dribbling)
	case common.LATERAL_STRIKER_LEFT, common.LATERAL_STRIKER_RIGHT:
		lateral, shot, pass, dribbling = maybeDrift(lateral), maybeDrift(shot), maybeDrift(pass), maybeDrift(dribbling)
	}
	return
}

// ProgressPlayers ages every player on every team by one season: retirees are removed, and
// their contract ticks down by one season (see common.AdvancePlayerContract - a contract
// reaching zero releases the player to free agency) regardless of anything below. Skills,
// though, only change for a player who actually played a competitive minute this season (see
// common.GetTeamPlayerPositionMinutes) - someone nobody fielded keeps every attribute exactly as
// it was, growth or decline alike. For a player who did play, only the skills relevant to
// whichever position they spent the most minutes at drift (see driftSkillsForPosition -
// mirrors common.PositionValue's own position->attribute weighting), each independently, with a
// chance that scales with their total minutes played this season up to
// common.FullSeasonMinutesReference (see driftSkill for the age trend and a hired head coach's
// trainerBonus, from common.GetTeamTrainer, that drift then applies). Any team left short of a
// full squad (by either retirement or an expired contract) is topped back up from the youth
// academy (see common.CreateAcademyPlayers, also trainer-boosted). Also tops up the open trainer
// market (see common.TopUpTrainerMarket) and clears the season's minutes accumulator once it's
// been read. Meant to run once per season, from AdvanceSeasonIfComplete.
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

		trainerSkill := 0
		if trainer, err := common.GetTeamTrainer(dbCtx, teamId); err != nil {
			log.Errorf("error getting trainer for team %d progression: %s", teamId, err)
		} else if trainer != nil {
			trainerSkill = trainer.Skill
		}

		minutesByPlayer, err := common.GetTeamPlayerPositionMinutes(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting position minutes for team %d progression: %s", teamId, err)
			minutesByPlayer = nil // treat as "nobody played" rather than block the season rollover
		}

		for _, player := range *players {
			newAge := player.Age + 1
			if ShouldRetire(newAge, r) {
				if err := common.RetirePlayer(dbCtx, player.Id); err != nil {
					log.Errorf("error retiring player %d: %s", player.Id, err)
				}
				continue
			}

			goalKeeper, defense, lateral, pass, midfield, dribbling, shot :=
				player.GoalKeeper, player.Defense, player.Lateral, player.Pass, player.Midfield, player.Dribbling, player.Shot

			if position, totalMinutes, played := primaryPosition(minutesByPlayer[player.Id]); played {
				playChance := float32(totalMinutes) / float32(common.FullSeasonMinutesReference)
				if playChance > 1 {
					playChance = 1
				}
				goalKeeper, defense, lateral, pass, midfield, dribbling, shot =
					driftSkillsForPosition(*player, position, playChance, newAge, trainerSkill, r)
			}

			if err := common.UpdatePlayerProgression(dbCtx, player.Id, newAge,
				goalKeeper, defense, lateral, pass, midfield, dribbling, shot); err != nil {
				log.Errorf("error updating progression for player %d: %s", player.Id, err)
			}

			released, err := common.AdvancePlayerContract(dbCtx, player.Id)
			if err != nil {
				log.Errorf("error advancing contract for player %d: %s", player.Id, err)
			} else if released {
				log.Infof("player %d (%s %s) released to free agency: contract expired", player.Id, player.FirstName, player.LastName)
			}
		}

		topUpAcademy(dbCtx, teamId, trainerSkill)
		autoManageLineupIfUnowned(dbCtx, teamId)
	}

	if err := common.TopUpTrainerMarket(dbCtx); err != nil {
		log.Errorf("error topping up trainer market: %s", err)
	}

	if err := common.ClearPlayerPositionMinutes(dbCtx); err != nil {
		log.Errorf("error clearing player position minutes after progression: %s", err)
	}
}

// autoManageLineupIfUnowned keeps a team nobody manages (no owner) fielding its current best XI
// every season - the only "intelligence" it has is always picking its best available players
// (see common.SelectBestLineup), same as at team creation. Without this, an unowned team's
// lineup would go stale as retirees and academy arrivals (both just above) change its squad:
// retirees are already dropped from default_lineup_positions (see RetirePlayer), leaving the
// team fielding fewer than 11 players, and new academy players would never be selected at all.
func autoManageLineupIfUnowned(dbCtx *common.DatabaseContext, teamId int64) {
	owner, err := common.GetTeamOwner(dbCtx, teamId)
	if err != nil || owner > 0 {
		return
	}

	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad for unowned team %d lineup refresh: %s", teamId, err)
		return
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, common.FlattenLineup(common.SelectBestLineup(*players))); err != nil {
		log.Errorf("error refreshing lineup for unowned team %d: %s", teamId, err)
	}
}

func topUpAcademy(dbCtx *common.DatabaseContext, teamId int64, trainerSkill int) {
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
	if err := common.CreateAcademyPlayers(dbCtx, teamId, needed, trainerSkill); err != nil {
		log.Errorf("error generating academy players for team %d: %s", teamId, err)
	}
}
