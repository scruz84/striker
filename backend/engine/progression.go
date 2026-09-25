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

// FreeAgencyRetirementSeasons is how many consecutive seasons a player can go unsigned before
// they're forced into retirement regardless of age (see ProgressFreeAgents) - long enough that a
// released player has a real chance to find a new club, short enough that free agency doesn't
// become a de facto permanent parking spot for players nobody wants.
const FreeAgencyRetirementSeasons = 2

// ShouldRetire decides whether a player calls it a career this season, given their current age
// (age itself no longer changes at season end - see ProcessBirthdays). Below
// RetirementEligibleAge it's never; at/after RetirementForcedAge it's always; in between, the
// chance rises 15 percentage points per year.
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

// driftSkill applies a player's age trend, a hired trainer's bonus, and a little noise (-1..+1)
// to one skill value, clamped to a sane [1, 99] range. The trainer's bonus (0 with no trainer)
// only applies while the age trend is still flat or rising (ageTrendFor(age) >= 0) - a coach
// accelerates a player's development, but can't reverse the physical decline of someone already
// past their prime, so a veteran's skills merely hold steady at best rather than climbing
// indefinitely no matter how good their coach is.
func driftSkill(value int, age int, trainerSkill int, r *rand.Rand) int {
	trend := ageTrendFor(age)
	bonus := 0
	if trend >= 0 {
		bonus = trainerBonus(trainerSkill)
	}
	value += trend + bonus + r.Intn(3) - 1
	if value < 1 {
		value = 1
	}
	if value > 99 {
		value = 99
	}
	return value
}

// primaryPosition picks the tactical position with the most minutes in a minutesByPosition map -
// a single match's (see updatePlayerSkillsAfterMatch) or, previously, a whole season's - ties
// broken by the lowest position id for determinism, and sums total minutes across every position
// occupied. ok is false only when there are no recorded minutes at all - i.e. the player never
// took the field.
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
// Every relevant skill independently rolls playChance (see updatePlayerSkillsAfterMatch's
// MatchSkillGrowthMinutesReference) before drifting; skills outside the relevant set are always
// left untouched.
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

// MatchSkillGrowthMinutesReference is the minutes-played benchmark
// updatePlayerSkillsAfterMatch treats as "a full match" (90 minutes) for skill-growth purposes,
// but scaled by the same 10-full-matches budget the old season-end system used (see this file's
// git history for the season-long FullSeasonMinutesReference this replaces): a player's chance
// of any given relevant skill drifting after one match is that match's minutes divided by this
// reference, so someone who starts and finishes every match still only has roughly a 1-in-10
// chance of a given skill moving per game they play, rather than nearly guaranteed growth every
// single match - across a full season of starts that lands in the same ballpark as the old
// once-a-season roll, just spread out match-by-match instead of saved up for one big drift.
const MatchSkillGrowthMinutesReference = 900

// updatePlayerSkillsAfterMatch is skill growth's actual trigger: called from
// updateSkillsAfterCompetitiveMatch right after a competitive match is simulated (see
// engine.simulateMatch/simulateCupTie), it drifts skills for every player who featured, using
// that match's own in-memory minutesByPosition (MatchSimulation.HomeMinutesByPosition/
// AwayMinutesByPosition) rather than a season-long accumulator - this fully replaces the old
// season-end drift that used to run once per season in ProgressPlayers (removed - keeping both
// would double-count the same minutes toward growth twice). Age, and a hired head coach's skill
// (see common.GetTeamTrainer), work exactly as they did in the season-end system (see
// driftSkill) - only the trigger and the per-event magnitude changed, not what influences
// growth. Deliberately never called for friendly matches (see engine.simulateFriendly) - see
// updateSkillsAfterCompetitiveMatch's own doc comment for why. season is the current one,
// stamped on each common.LogPlayerProgression row (the same audit trail the season-end system
// used to write to, just with many more, smaller rows now).
func updatePlayerSkillsAfterMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation, season int) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	updateTeamPlayersAfterMatch(dbCtx, match.Home, match.HomePlayers, match.HomeMinutesByPosition, season, r)
	updateTeamPlayersAfterMatch(dbCtx, match.Away, match.AwayPlayers, match.AwayMinutesByPosition, season, r)
}

// updateTeamPlayersAfterMatch is updatePlayerSkillsAfterMatch's per-team half - separated out
// since a match always has two independent sides, each with its own trainer and its own subset
// of minutesByPosition.
func updateTeamPlayersAfterMatch(dbCtx *common.DatabaseContext, teamId int64, players *map[int64]common.Player,
	minutesByPosition map[int64]map[int]int, season int, r *rand.Rand) {
	if players == nil || len(minutesByPosition) == 0 {
		return
	}

	trainerSkill := 0
	if trainer, err := common.GetTeamTrainer(dbCtx, teamId); err != nil {
		log.Errorf("error getting trainer for team %d match progression: %s", teamId, err)
	} else if trainer != nil {
		trainerSkill = trainer.Skill
	}

	for playerId, minutesAtPosition := range minutesByPosition {
		player, ok := (*players)[playerId]
		if !ok {
			continue
		}

		position, totalMinutes, played := primaryPosition(minutesAtPosition)
		if !played {
			continue
		}
		playChance := float32(totalMinutes) / float32(MatchSkillGrowthMinutesReference)
		if playChance > 1 {
			playChance = 1
		}

		goalKeeper, defense, lateral, pass, midfield, dribbling, shot :=
			driftSkillsForPosition(player, position, playChance, player.Age, trainerSkill, r)

		if goalKeeper == player.GoalKeeper && defense == player.Defense && lateral == player.Lateral &&
			pass == player.Pass && midfield == player.Midfield && dribbling == player.Dribbling && shot == player.Shot {
			continue // nothing actually drifted this match - not worth a write or a log row
		}

		if err := common.UpdatePlayerProgression(dbCtx, player.Id, player.Age,
			goalKeeper, defense, lateral, pass, midfield, dribbling, shot); err != nil {
			log.Errorf("error updating match progression for player %d: %s", player.Id, err)
			continue
		}
		if err := common.LogPlayerProgression(dbCtx, player.Id, season, player.Age, player.Age,
			player.GoalKeeper, goalKeeper, player.Defense, defense, player.Lateral, lateral,
			player.Pass, pass, player.Midfield, midfield, player.Dribbling, dribbling,
			player.Shot, shot); err != nil {
			log.Errorf("error logging match progression for player %d: %s", player.Id, err)
		}
	}
}

// ProgressPlayers advances every player on every team by one season: retirees are removed, and
// their contract ticks down by one season (see common.AdvancePlayerContract - a contract
// reaching zero releases the player to free agency) regardless of anything below. Age itself is
// no longer touched here - it advances on its own periodic cycle (see ProcessBirthdays) - this
// just reads each player's current age for the retirement check. Skills are no longer touched
// here at all: that used to be a season-end drift keyed off a whole season's accumulated
// minutes-by-position, but it's now applied match-by-match instead, right after each match a
// player actually features in (see updatePlayerSkillsAfterMatch) - a season-end pass over the
// same minutes would double-count exactly the growth those per-match updates already applied,
// so this function's only remaining jobs are retirement, contracts, and squad upkeep. Any team
// left short of a full squad (by either retirement or an expired contract) is topped back up
// from the youth academy (see common.CreateAcademyPlayers, trainer-boosted). Also tops up the
// open trainer market (see common.TopUpTrainerMarket). Meant to run once per season, from
// AdvanceSeasonIfComplete.
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

		for _, player := range *players {
			if ShouldRetire(player.Age, r) {
				if err := common.RetirePlayer(dbCtx, player.Id); err != nil {
					log.Errorf("error retiring player %d: %s", player.Id, err)
				}
				continue
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
}

// ProgressFreeAgents advances every currently-unowned player by one season - ProgressPlayers only
// iterates players belonging to a team, so without this a free agent would never retire or
// otherwise change at all no matter how long they sat unsigned. Skills are never drifted here: a
// free agent has no club, no matches, and no coach to develop (or decline) them. Age isn't
// touched here either - like ProgressPlayers, it just reads each player's current age (which
// advances independently - see ProcessBirthdays); only the free-agency clock (see
// common.AdvanceFreeAgencyClock) actually changes.
//
// A free agent retires exactly like a rostered player (see ShouldRetire) if their current age
// calls for it, and also, regardless of age, once they've now gone FreeAgencyRetirementSeasons
// consecutive seasons without a club - nobody stays a free agent forever. Either way they don't
// simply vanish: common.RetireFreeAgentAsTrainer converts them into a fresh, unemployed trainer
// candidate on the open market first, the same way a long, unresolved job search might end in a
// career change rather than nothing at all. Meant to run once per season from
// AdvanceSeasonIfComplete, alongside ProgressPlayers.
func ProgressFreeAgents(dbCtx *common.DatabaseContext) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	freeAgents, err := common.GetFreeAgents(dbCtx)
	if err != nil {
		log.Errorf("error listing free agents for progression: %s", err)
		return
	}

	for _, player := range freeAgents {
		newAge := player.Age

		if ShouldRetire(newAge, r) {
			if err := common.RetireFreeAgentAsTrainer(dbCtx, player); err != nil {
				log.Errorf("error retiring free agent %d as a trainer: %s", player.Id, err)
			}
			continue
		}

		seasonsUnsigned, err := common.AdvanceFreeAgencyClock(dbCtx, player.Id, newAge)
		if err != nil {
			log.Errorf("error advancing free agency clock for player %d: %s", player.Id, err)
			continue
		}
		if seasonsUnsigned >= FreeAgencyRetirementSeasons {
			log.Infof("player %d (%s %s) retiring after %d seasons unsigned",
				player.Id, player.FirstName, player.LastName, seasonsUnsigned)
			if err := common.RetireFreeAgentAsTrainer(dbCtx, player); err != nil {
				log.Errorf("error retiring free agent %d as a trainer: %s", player.Id, err)
			}
		}
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
