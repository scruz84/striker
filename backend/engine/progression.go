package engine

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
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
// to one skill value, clamped to [1, max]. max is that player's individually-rolled lifetime
// ceiling for this specific skill (see common.rollSkillCap/GetPlayerSkillCaps) rather than a
// universal 99 - once a player reaches their own ceiling, further positive drift simply has no
// more room to go, same as a real player topping out their potential. The trainer's bonus (0 with
// no trainer) only applies while the age trend is still flat or rising (ageTrendFor(age) >= 0) -
// a coach accelerates a player's development, but can't reverse the physical decline of someone
// already past their prime, so a veteran's skills merely hold steady at best rather than climbing
// indefinitely no matter how good their coach is.
func driftSkill(value int, max int, age int, trainerSkill int, r *rand.Rand) int {
	trend := ageTrendFor(age)
	bonus := 0
	if trend >= 0 {
		bonus = trainerBonus(trainerSkill)
	}
	value += trend + bonus + r.Intn(3) - 1
	if value < 1 {
		value = 1
	}
	if value > max {
		value = max
	}
	return value
}

// ageTrendForStamina is the typical per-season Stamina delta purely from age - unlike
// ageTrendFor (the seven core footballing skills, which hold flat through a player's late 20s),
// physical conditioning starts fading earlier and keeps declining faster the older a player gets,
// matching the feature's own spec ("a mayor edad sera esperable que la resistencia baje").
func ageTrendForStamina(age int) int {
	switch {
	case age < 24:
		return 1
	case age < 30:
		return 0
	case age < 34:
		return -1
	default:
		return -2
	}
}

// driftStamina applies a player's Stamina-specific age trend (see ageTrendForStamina) plus a
// little noise (-1..+1), clamped to [1, 99] - same shape as driftSkill, but with no trainer bonus
// (a hired coach develops footballing skill, not raw physical conditioning) since nothing in the
// feature's own spec ties Stamina's drift to coaching.
func driftStamina(value int, age int, r *rand.Rand) int {
	value += ageTrendForStamina(age) + r.Intn(3) - 1
	return clampAttribute(value)
}

// FormRisePerMatch is the baseline Form bump a player gets just for featuring in a match, however
// well or badly they played - "el estado de forma deberia subir cuando el jugador juegue
// partidos" from the feature's own spec.
const FormRisePerMatch = 2

// FormPerformanceWeight scales how much a match rating above/below the average, unremarkable
// matchRatingBaseline (6.0 - mirrors common.matchRatingBase) pushes Form further up or down on
// top of FormRisePerMatch - "tambien si tiene buenas actuaciones ... al contrario deberia bajar
// si ... sus actuaciones son malas": a poor enough rating can outweigh FormRisePerMatch entirely
// and net Form down even though the player did feature.
const FormPerformanceWeight = 2.0

// matchRatingBaseline mirrors common.matchRatingBase (unexported in that package) - the
// unremarkable, dead-even rating driftFormPlayed measures a performance against.
const matchRatingBaseline = 6.0

// FormDecayPerMatchBenched is how much Form falls for a squad player who sat out a match
// entirely - "deberia bajar si no juega partidos" from the feature's own spec.
const FormDecayPerMatchBenched = 1

// driftFormPlayed updates a player's Form after a match they featured in: a baseline rise for
// simply playing (FormRisePerMatch), adjusted up or down by how far their own match rating (see
// common.ComputeMatchRatings, recomputed in-memory for this exact match by
// matchRatingsForProgression) landed from matchRatingBaseline. hasRating is false only when the
// match had no computed ratings at all (missing lineup/event data) - plain featuring still counts
// then, just without a performance adjustment either way.
func driftFormPlayed(form int, rating float64, hasRating bool) int {
	delta := FormRisePerMatch
	if hasRating {
		delta += int(math.Round((rating - matchRatingBaseline) * FormPerformanceWeight))
	}
	return clampAttribute(form + delta)
}

// driftFormBenched is driftFormPlayed's counterpart for a squad player who didn't feature at all
// this match.
func driftFormBenched(form int) int {
	return clampAttribute(form - FormDecayPerMatchBenched)
}

// clampAttribute bounds a drifted Stamina/Form value to the same [1, 99] range every other
// numeric player attribute uses (see driftSkill).
func clampAttribute(value int) int {
	if value < 1 {
		return 1
	}
	if value > 99 {
		return 99
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
func driftSkillsForPosition(p common.Player, caps common.PlayerSkillCaps, position int, playChance float32, age int, trainerSkill int, r *rand.Rand) (
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot int) {
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot =
		p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot

	maybeDrift := func(current int, max int) int {
		if r.Float32() >= playChance {
			return current
		}
		return driftSkill(current, max, age, trainerSkill, r)
	}

	switch position {
	case common.GOALKEEPER:
		goalKeeper = maybeDrift(goalKeeper, caps.GoalKeeperMax)
	case common.DEFENSE:
		defense = maybeDrift(defense, caps.DefenseMax)
	case common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT:
		lateral, defense = maybeDrift(lateral, caps.LateralMax), maybeDrift(defense, caps.DefenseMax)
	case common.MIDFIELD:
		midfield, pass = maybeDrift(midfield, caps.MidfieldMax), maybeDrift(pass, caps.PassMax)
	case common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT:
		lateral, midfield, pass = maybeDrift(lateral, caps.LateralMax), maybeDrift(midfield, caps.MidfieldMax), maybeDrift(pass, caps.PassMax)
	case common.STRIKER:
		shot, pass, dribbling = maybeDrift(shot, caps.ShotMax), maybeDrift(pass, caps.PassMax), maybeDrift(dribbling, caps.DribblingMax)
	case common.LATERAL_STRIKER_LEFT, common.LATERAL_STRIKER_RIGHT:
		lateral, shot, pass, dribbling = maybeDrift(lateral, caps.LateralMax), maybeDrift(shot, caps.ShotMax), maybeDrift(pass, caps.PassMax), maybeDrift(dribbling, caps.DribblingMax)
	}
	return
}

// defaultSkillCaps is the fallback used when a player has no PlayerSkillCaps row yet (e.g. a
// player created before this feature existed, caught between app deploy and the schema backfill
// finishing) - the universal [1, 99] range every player used to share, so such a player is never
// stuck unable to grow just because their individual ceilings haven't been rolled yet.
var defaultSkillCaps = common.PlayerSkillCaps{
	GoalKeeperMax: 99, DefenseMax: 99, LateralMax: 99, PassMax: 99,
	MidfieldMax: 99, DribblingMax: 99, ShotMax: 99,
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
// updateSkillsAfterPlayedMatch right after any match is simulated - league, cup, or friendly (see
// engine.simulateMatch/simulateCupTie/simulateFriendly) - it drifts skills for every player who
// featured, using that match's own in-memory minutesByPosition (MatchSimulation.
// HomeMinutesByPosition/AwayMinutesByPosition) rather than a season-long accumulator - this fully
// replaces the old season-end drift that used to run once per season in ProgressPlayers (removed
// - keeping both would double-count the same minutes toward growth twice). Age, and a hired head
// coach's skill (see common.GetTeamTrainer), work exactly as they did in the season-end system
// (see driftSkill) - only the trigger and the per-event magnitude changed, not what influences
// growth. A friendly counts the same as a competitive match here - it carries no financial stakes
// (see UpdateTeamFinancesAfterMatch), but the players on the pitch still develop from playing it.
// season is the current one, stamped on each common.LogPlayerProgression row (the same audit
// trail the season-end system used to write to, just with many more, smaller rows now).
func updatePlayerSkillsAfterMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation, season int) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	ratings := matchRatingsForProgression(match)
	updateTeamPlayersAfterMatch(dbCtx, match.Home, match.HomePlayers, match.HomeMinutesByPosition, ratings, season, r)
	updateTeamPlayersAfterMatch(dbCtx, match.Away, match.AwayPlayers, match.AwayMinutesByPosition, ratings, season, r)
}

// matchRatingsForProgression computes this match's own per-player ratings in-memory, the exact
// same way common.WeekMatchById does on every read (see common.ComputeMatchRatings) - purely a
// function of the match's own id, events and result, so this produces byte-identical numbers to
// whatever the player will later see on the match's own report page, with no extra database
// round-trip. Drives Form's post-match update (see driftFormPlayed) - a player's Form moves on how
// well they actually played, not merely on whether they featured. Returns an empty map (never nil)
// when ComputeMatchRatings itself has nothing to go on, so callers can index it unconditionally.
func matchRatingsForProgression(match *common.MatchSimulation) map[int64]float64 {
	ratings := common.ComputeMatchRatings(&common.WeekMatch{
		MatchId:     match.Id,
		HomeGoals:   match.HomeGoals,
		AwayGoals:   match.AwayGoals,
		Events:      match.Events,
		HomeLineup:  match.HomeLineup,
		AwayLineup:  match.AwayLineup,
		HomePlayers: match.HomePlayers,
		AwayPlayers: match.AwayPlayers,
	})
	if ratings == nil {
		return map[int64]float64{}
	}
	return *ratings
}

// updateTeamPlayersAfterMatch is updatePlayerSkillsAfterMatch's per-team half - separated out
// since a match always has two independent sides, each with its own trainer and its own subset
// of minutesByPosition. Besides the seven core skills, it also drifts every squad player's
// Stamina and Form here: Stamina only for whoever actually played (see driftStamina), Form for
// every squad player regardless - up for whoever played (more so with a good rating, see
// driftFormPlayed), down for whoever sat out the match entirely (see driftFormBenched). When at
// least one player's core skills actually evolve, the team's owner gets a NOTIFICATION_PROGRESSION
// notification linking to /players/evolutions?logIds=... - the exact player_progression_log rows
// this match produced (see common.GetPlayerEvolutionsForTeamByLogIds and the frontend's
// LatestEvolutionsPage), so the report only ever shows the player(s) who actually evolved after
// this specific match rather than every squad player's own all-time latest row. Stamina/Form drift
// silently (no notification) - unlike the seven core skills, they're expected to move after
// nearly every match, so a notification for them would lose all signal value.
func updateTeamPlayersAfterMatch(dbCtx *common.DatabaseContext, teamId int64, players *map[int64]common.Player,
	minutesByPosition map[int64]map[int]int, ratings map[int64]float64, season int, r *rand.Rand) {
	if players == nil || len(*players) == 0 {
		return
	}

	trainerSkill := 0
	if trainer, err := common.GetTeamTrainer(dbCtx, teamId); err != nil {
		log.Errorf("error getting trainer for team %d match progression: %s", teamId, err)
	} else if trainer != nil {
		trainerSkill = trainer.Skill
	}

	playerIds := make([]int64, 0, len(minutesByPosition))
	for playerId := range minutesByPosition {
		playerIds = append(playerIds, playerId)
	}
	skillCaps, err := common.GetPlayerSkillCaps(dbCtx, playerIds)
	if err != nil {
		log.Errorf("error getting skill caps for team %d match progression: %s", teamId, err)
		skillCaps = map[int64]common.PlayerSkillCaps{}
	}

	evolvedLogIds := make([]int64, 0)

	for playerId, player := range *players {
		minutesAtPosition, playedThisMatch := minutesByPosition[playerId]
		if !playedThisMatch {
			if newForm := driftFormBenched(player.Form); newForm != player.Form {
				if err := common.UpdatePlayerStaminaAndForm(dbCtx, player.Id, player.Stamina, newForm); err != nil {
					log.Errorf("error updating benched form for player %d: %s", player.Id, err)
				}
			}
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

		caps, ok := skillCaps[playerId]
		if !ok {
			caps = defaultSkillCaps
		}

		goalKeeper, defense, lateral, pass, midfield, dribbling, shot :=
			driftSkillsForPosition(player, caps, position, playChance, player.Age, trainerSkill, r)

		newStamina := player.Stamina
		if r.Float32() < playChance {
			newStamina = driftStamina(player.Stamina, player.Age, r)
		}
		rating, hasRating := ratings[playerId]
		newForm := driftFormPlayed(player.Form, rating, hasRating)
		if newStamina != player.Stamina || newForm != player.Form {
			if err := common.UpdatePlayerStaminaAndForm(dbCtx, player.Id, newStamina, newForm); err != nil {
				log.Errorf("error updating stamina/form for player %d: %s", player.Id, err)
			}
		}

		if goalKeeper == player.GoalKeeper && defense == player.Defense && lateral == player.Lateral &&
			pass == player.Pass && midfield == player.Midfield && dribbling == player.Dribbling && shot == player.Shot {
			continue // nothing actually drifted this match - not worth a write or a log row
		}

		if err := common.UpdatePlayerProgression(dbCtx, player.Id, player.Age,
			goalKeeper, defense, lateral, pass, midfield, dribbling, shot); err != nil {
			log.Errorf("error updating match progression for player %d: %s", player.Id, err)
			continue
		}
		logId, err := common.LogPlayerProgression(dbCtx, player.Id, season, player.Age, player.Age,
			player.GoalKeeper, goalKeeper, player.Defense, defense, player.Lateral, lateral,
			player.Pass, pass, player.Midfield, midfield, player.Dribbling, dribbling,
			player.Shot, shot)
		if err != nil {
			log.Errorf("error logging match progression for player %d: %s", player.Id, err)
			continue
		}
		evolvedLogIds = append(evolvedLogIds, logId)
	}

	if len(evolvedLogIds) == 0 {
		return
	}
	evolvedCount := len(evolvedLogIds)
	msg := common.NotificationMessage{
		Text: fmt.Sprintf("%d of your players developed after the match - check their latest evolutions", evolvedCount),
		Key:  "notifications.messages.playerEvolutions",
		Params: map[string]any{
			"count": evolvedCount,
		},
	}
	link := "/players/evolutions?logIds=" + joinInt64s(evolvedLogIds)
	if err := common.NotifyTeamOwnerWithLink(dbCtx, teamId, common.NOTIFICATION_PROGRESSION, msg, link); err != nil {
		log.Errorf("error notifying team %d owner of player evolutions: %s", teamId, err)
	}
}

// joinInt64s renders evolvedLogIds as a comma-separated query-string value - the
// "/players/evolutions?logIds=..." link identifies exactly the player_progression_log rows this
// one match's progression run produced (see GetPlayerEvolutionsForTeamByLogIds), rather than every
// squad player's all-time latest row, which can span many unrelated matches.
func joinInt64s(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
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
// (and, via refreshAiTeamLineup, the best formation its AI level can find for that squad) every
// season. Without this, an unowned team's lineup would go stale as retirees and academy arrivals
// (both just above) change its squad: retirees are already dropped from default_lineup_positions
// (see RetirePlayer), leaving the team fielding fewer than 11 players, and new academy players
// would never be selected at all. Delegates to refreshAiTeamLineup (engine/aiteams.go) - the same
// function every per-match/per-tick AI reaction calls (see aiTeamBaselineReaction) - rather than
// duplicating its own copy of the lineup/formation selection, so the two can never drift into
// picking differently for the same squad. Unlike RunAITeamMatchReaction/RunAITeamManagement,
// this doesn't gate on isAiManagedTeam (AiLevel >=
// AiLevelCaretaker): a team that's unowned but hasn't been assigned an AI level yet (AiLevel 0 -
// e.g. one not yet reached by BackfillAiLevels) still needs its lineup refreshed here or it would
// simply go stale forever, and refreshAiTeamLineup/formationRankForAiLevel already treat AiLevel 0
// the same as Caretaker's "almost the best" behavior, not as an error case.
func autoManageLineupIfUnowned(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil || team.OwnerId > 0 {
		return
	}
	refreshAiTeamLineup(dbCtx, teamId, team.AiLevel)
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
