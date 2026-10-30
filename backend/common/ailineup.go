package common

import (
	"math/rand"
	"sort"
)

// AiMatchPlan is what a CPU team fields in one match: its formation, XI and the substitutions it
// schedules (see PlanAiMatchLineup).
type AiMatchPlan struct {
	Formation     int
	Positions     []LineupPosition
	Substitutions []LineupSubstitution
}

// Minutes and thresholds of the CPU's planned substitutions (see planCompetitiveSubstitutions and
// planFriendlyRotation). They are the numbers in openspec/specs/cpu-team-management.
const (
	// aiCaretakerSubMinute and aiCaretakerMaxStamina: a Caretaker's one change is at minute 70
	// and only for a starter with Stamina below 50, when the best bench player for his slot is
	// worth at least aiCaretakerMinValueRatio of him.
	aiCaretakerSubMinute     = 70
	aiCaretakerMaxStamina    = 50
	aiCaretakerMinValueRatio = 0.80
	// aiManagerSubMinute and aiManagerMaxSubs: a Manager makes at most 2 changes, at minute 70,
	// each worth at least aiManagerMinGain of the starter's value over the remaining minutes. It
	// leaves one of the 3 substitutions free for an injury.
	aiManagerSubMinute = 70
	aiManagerMaxSubs   = 2
	aiManagerMinGain   = 0.05
	// A Director makes up to 3 changes, each at its best minute from aiDirectorFirstMinute to
	// aiDirectorLastMinute (every aiDirectorMinuteStep), worth at least aiDirectorMinGain of the
	// starter's value over the remaining minutes. Its last change is never before
	// aiDirectorLastSubMinimum, so an early injury can still be covered.
	aiDirectorFirstMinute    = 55
	aiDirectorLastMinute     = 85
	aiDirectorMinuteStep     = 5
	aiDirectorMinGain        = 0.02
	aiDirectorLastSubMinimum = 75
	aiFriendlySubMinute      = 60
	aiFriendlyMaxSubs        = 3
	aiFriendlyRestedRegulars = 5
	aiFriendlyYoungPlayerAge = 21
	aiMatchLength            = FatigueMatchLength
	aiLineupSize             = 11
)

// PlanAiMatchLineup decides a CPU team's formation, XI and scheduled substitutions for one match,
// from its AI level (see AiLevelCaretaker/Manager/Director). It is pure and deterministic given
// rng, so the engine can seed rng per match and get the same plan when a match is re-simulated.
//
// leagueUnavailable is every squad player serving an injury or suspension (what bars a league
// match or cup tie); matchUnavailable is who is barred from this match (injuries only, for a
// friendly). A Manager or Director builds its XI from available players only; a Caretaker keeps
// picking from the whole squad and relies on the kickoff backfill (see BackfillEmptyPositions),
// but never schedules an unavailable player to come on.
//
// In a competitive match the XI is the team's regulars and the substitutions come from the
// fatigue-gain model (planCompetitiveSubstitutions). In a friendly the team rotates so its
// reserves play (planFriendlyRotation).
func PlanAiMatchLineup(squad []*Player, leagueUnavailable map[int64]bool, matchUnavailable map[int64]bool,
	aiLevel int, isFriendly bool, rng *rand.Rand) AiMatchPlan {
	regularsPool := squad
	if aiLevel >= AiLevelManager {
		regularsPool = playersWithout(squad, leagueUnavailable)
	}
	formation, regulars := SelectBestFormationAndLineup(regularsPool, aiLevel, rng)

	if isFriendly {
		return planFriendlyRotation(squad, regulars, formation, matchUnavailable, aiLevel)
	}

	plan := AiMatchPlan{Formation: formation, Positions: orderedLineupPositions(regulars)}
	plan.Substitutions = planCompetitiveSubstitutions(squad, regulars, leagueUnavailable, aiLevel)
	return plan
}

// orderedLineupPositions flattens a position -> players map in positionFillOrder, so the same
// lineup always yields the same slice (FlattenLineup follows map order).
func orderedLineupPositions(lineup map[int][]int64) []LineupPosition {
	positions := make([]LineupPosition, 0, aiLineupSize)
	for _, position := range positionFillOrder {
		for _, id := range lineup[position] {
			positions = append(positions, LineupPosition{PlayerId: id, Position: position})
		}
	}
	return positions
}

func playersWithout(squad []*Player, excluded map[int64]bool) []*Player {
	kept := make([]*Player, 0, len(squad))
	for _, p := range squad {
		if !excluded[p.Id] {
			kept = append(kept, p)
		}
	}
	return kept
}

// aiStarter is one XI member with the slot he plays.
type aiStarter struct {
	player   *Player
	position int
}

// aiStarters lists the XI's members in positionFillOrder (outfield only when outfieldOnly), with
// their Player records, skipping ids the squad does not hold.
func aiStarters(squad []*Player, lineup map[int][]int64, outfieldOnly bool) []aiStarter {
	byId := make(map[int64]*Player, len(squad))
	for _, p := range squad {
		byId[p.Id] = p
	}
	starters := make([]aiStarter, 0, aiLineupSize)
	for _, position := range positionFillOrder {
		if outfieldOnly && position == GOALKEEPER {
			continue
		}
		for _, id := range lineup[position] {
			if p, ok := byId[id]; ok {
				starters = append(starters, aiStarter{p, position})
			}
		}
	}
	return starters
}

// aiBench is every squad player who is not in the XI and may come on (not in unavailable).
func aiBench(squad []*Player, lineup map[int][]int64, unavailable map[int64]bool) []*Player {
	starting := make(map[int64]bool, aiLineupSize)
	for _, ids := range lineup {
		for _, id := range ids {
			starting[id] = true
		}
	}
	bench := make([]*Player, 0, len(squad))
	for _, p := range squad {
		if !starting[p.Id] && !unavailable[p.Id] {
			bench = append(bench, p)
		}
	}
	return bench
}

// aiSubstitutionGain is the expected gain, in value-minutes, of replacing starter (worth V at his
// slot) with bench (worth W there) at minute m: (90 - m) * (W * benchFactor - V * starterFactor).
// The starter has already played m minutes and plays on to 90, so his average fatigue factor is
// the multiplier at the midpoint (m + 90) / 2; the bench player plays 0 to 90 - m, midpoint
// (90 - m) / 2. Both use FatigueMultiplierAt, the engine's own formula.
func aiSubstitutionGain(starter aiStarter, bench *Player, minute int) float64 {
	remaining := float64(aiMatchLength - minute)
	starterFactor := FatigueMultiplierAt(starter.player.Stamina, float64(minute+aiMatchLength)/2)
	benchFactor := FatigueMultiplierAt(bench.Stamina, remaining/2)
	starterValue := float64(PositionValue(starter.position, *starter.player))
	benchValue := float64(PositionValue(starter.position, *bench))
	return remaining * (benchValue*benchFactor - starterValue*starterFactor)
}

// aiPair is one candidate substitution: bench player for starter at minute, worth gain.
type aiPair struct {
	starter aiStarter
	bench   *Player
	minute  int
	gain    float64
	// threshold is the smallest gain the pair needs to be worth making (a share of the starter's
	// value over the remaining minutes).
	threshold float64
}

// pickAiPairs takes the best pairs by gain whose gain reaches their threshold, up to max, never
// using a player in two pairs. Ties go to the lower starter id, then the lower bench id, so the
// result never depends on iteration order.
func pickAiPairs(pairs []aiPair, max int) []aiPair {
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].gain != pairs[j].gain {
			return pairs[i].gain > pairs[j].gain
		}
		if pairs[i].starter.player.Id != pairs[j].starter.player.Id {
			return pairs[i].starter.player.Id < pairs[j].starter.player.Id
		}
		return pairs[i].bench.Id < pairs[j].bench.Id
	})
	used := make(map[int64]bool)
	picked := make([]aiPair, 0, max)
	for _, pair := range pairs {
		if len(picked) == max {
			break
		}
		if pair.gain <= 0 || pair.gain < pair.threshold {
			continue
		}
		if used[pair.starter.player.Id] || used[pair.bench.Id] {
			continue
		}
		used[pair.starter.player.Id], used[pair.bench.Id] = true, true
		picked = append(picked, pair)
	}
	return picked
}

// planCompetitiveSubstitutions schedules a CPU team's changes for a league match or cup tie, by AI
// level, from the fatigue-gain model (aiSubstitutionGain): a Caretaker at most 1, a Manager at
// most 2, a Director at most 3. Only outfield starters are replaced, and only by bench players who
// are not unavailable. It returns an empty slice when nothing is worth changing.
func planCompetitiveSubstitutions(squad []*Player, lineup map[int][]int64, unavailable map[int64]bool, aiLevel int) []LineupSubstitution {
	starters := aiStarters(squad, lineup, true)
	// A Caretaker's XI may name an unavailable player, who the kickoff backfill replaces: he is
	// not worth planning a change for.
	fit := make([]aiStarter, 0, len(starters))
	for _, s := range starters {
		if !unavailable[s.player.Id] {
			fit = append(fit, s)
		}
	}
	bench := aiBench(squad, lineup, unavailable)
	subs := make([]LineupSubstitution, 0, MaxScheduledSubstitutions)
	if len(fit) == 0 || len(bench) == 0 {
		return subs
	}

	switch {
	case aiLevel >= AiLevelDirector:
		return planDirectorSubstitutions(fit, bench)
	case aiLevel >= AiLevelManager:
		pairs := make([]aiPair, 0, len(fit)*len(bench))
		for _, s := range fit {
			for _, b := range bench {
				value := float64(PositionValue(s.position, *s.player))
				pairs = append(pairs, aiPair{
					starter: s, bench: b, minute: aiManagerSubMinute,
					gain:      aiSubstitutionGain(s, b, aiManagerSubMinute),
					threshold: aiManagerMinGain * float64(aiMatchLength-aiManagerSubMinute) * value,
				})
			}
		}
		for _, pair := range pickAiPairs(pairs, aiManagerMaxSubs) {
			subs = append(subs, LineupSubstitution{PlayerOutId: pair.starter.player.Id, PlayerInId: pair.bench.Id, Minute: pair.minute})
		}
		return subs
	default:
		return planCaretakerSubstitution(fit, bench)
	}
}

// planCaretakerSubstitution is the Caretaker's single change: its most tired outfield starter
// (lowest Stamina, below aiCaretakerMaxStamina) comes off at minute 70 for the best bench player
// at his slot, if that player is worth at least 80% of him.
func planCaretakerSubstitution(starters []aiStarter, bench []*Player) []LineupSubstitution {
	subs := make([]LineupSubstitution, 0, 1)
	tired := starters[0]
	for _, s := range starters[1:] {
		if s.player.Stamina < tired.player.Stamina ||
			(s.player.Stamina == tired.player.Stamina && s.player.Id < tired.player.Id) {
			tired = s
		}
	}
	if tired.player.Stamina >= aiCaretakerMaxStamina {
		return subs
	}
	best := bestBenchPlayerFor(tired.position, bench)
	if best == nil {
		return subs
	}
	if float64(PositionValue(tired.position, *best)) < aiCaretakerMinValueRatio*float64(PositionValue(tired.position, *tired.player)) {
		return subs
	}
	return append(subs, LineupSubstitution{PlayerOutId: tired.player.Id, PlayerInId: best.Id, Minute: aiCaretakerSubMinute})
}

// planDirectorSubstitutions gives every pair its best minute (55, 60, ... 85), keeps the top 3 by
// gain that reach 2% of the starter's value over the minutes left, and moves the third to minute
// 75 if it was due earlier.
func planDirectorSubstitutions(starters []aiStarter, bench []*Player) []LineupSubstitution {
	pairs := make([]aiPair, 0, len(starters)*len(bench))
	for _, s := range starters {
		value := float64(PositionValue(s.position, *s.player))
		for _, b := range bench {
			best := aiPair{starter: s, bench: b, gain: -1e18}
			for minute := aiDirectorFirstMinute; minute <= aiDirectorLastMinute; minute += aiDirectorMinuteStep {
				if gain := aiSubstitutionGain(s, b, minute); gain > best.gain {
					best.gain, best.minute = gain, minute
					best.threshold = aiDirectorMinGain * float64(aiMatchLength-minute) * value
				}
			}
			pairs = append(pairs, best)
		}
	}
	picked := pickAiPairs(pairs, MaxScheduledSubstitutions)
	subs := make([]LineupSubstitution, 0, len(picked))
	for i, pair := range picked {
		minute := pair.minute
		if i == MaxScheduledSubstitutions-1 && minute < aiDirectorLastSubMinimum {
			minute = aiDirectorLastSubMinimum
		}
		subs = append(subs, LineupSubstitution{PlayerOutId: pair.starter.player.Id, PlayerInId: pair.bench.Id, Minute: minute})
	}
	return subs
}

// bestBenchPlayerFor is the highest-valued bench player for position (lowest id on a tie), or nil.
func bestBenchPlayerFor(position int, bench []*Player) *Player {
	var best *Player
	for _, b := range bench {
		if best == nil {
			best = b
			continue
		}
		v, bv := PositionValue(position, *b), PositionValue(position, *best)
		if v > bv || (v == bv && b.Id < best.Id) {
			best = b
		}
	}
	return best
}
