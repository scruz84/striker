package engine

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// Formation balance and squad-fit calibration (openspec match-simulation "Formation balance" and
// "Squad fit"): the blocks below run from TestTacticsStudy (TACTICS_SIM=goals,fit,fitmc,real or
// TACTICS_SIM=1) and assert the requirements' bounds.

// specialistProfile turns n random outfielders of a squad into specialists at a level (see
// "Specialist profiles"); n == 0 with a capped profile instead caps one line for every player.
type specialistProfile struct {
	name  string
	n     int
	apply func(p *common.Player, level int)
}

// pureSpecialist sets every outfield skill to 25 and then the role skills to the level.
func pureSpecialist(role func(p *common.Player, level int)) func(p *common.Player, level int) {
	return func(p *common.Player, level int) {
		p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot = 25, 25, 25, 25, 25, 25
		role(p, level)
	}
}

// homeProfiles is each formation's specialist profile ("Specialist profiles"): the players it fields
// more of than a 4-4-2, or for 4-5-1 a squad without strikers.
var homeProfiles = map[int]specialistProfile{
	common.FormationDefensive: {"3 pure centre-backs", 3, pureSpecialist(func(p *common.Player, l int) { p.Defense = l })},
	common.FormationAttacking: {"3 pure strikers", 3, pureSpecialist(func(p *common.Player, l int) { p.Shot, p.Dribbling = l, l })},
	common.FormationFourThreeThree: {"2 pure wingers", 2, pureSpecialist(func(p *common.Player, l int) {
		p.Lateral, p.Shot, p.Dribbling = l, l, l
	})},
	common.FormationThreeFiveTwo: {"2 pure full-backs", 2, pureSpecialist(func(p *common.Player, l int) { p.Lateral, p.Defense = l, l })},
	common.FormationFourFiveOne: {"Shot and Dribbling capped at 30", 0, func(p *common.Player, _ int) {
		p.Shot, p.Dribbling = minInt(p.Shot, 30), minInt(p.Dribbling, 30)
	}},
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// bestKeeperIndex is the squad member most likely to keep goal, never made a specialist.
func bestKeeperIndex(squad []*common.Player) int {
	best := 0
	for i, p := range squad {
		if p.GoalKeeper > squad[best].GoalKeeper {
			best = i
		}
	}
	return best
}

// withProfile is a copy of squad with profile applied at level.
func withProfile(rng *rand.Rand, squad []*common.Player, profile specialistProfile, level int) []*common.Player {
	out := cloneSquad(squad, 0)
	keeper := bestKeeperIndex(out)
	if profile.n == 0 {
		for i, p := range out {
			if i != keeper {
				profile.apply(p, level)
			}
		}
		return out
	}
	done := 0
	for _, i := range rng.Perm(len(out)) {
		if done == profile.n {
			break
		}
		if i != keeper {
			profile.apply(out[i], level)
			done++
		}
	}
	return out
}

// realisticSquad is a random squad (see randomSquad) plus 2 to 4 partial specialists of random
// types: one role skill drawn from 60 to 80 and the other outfield skills 15 lower.
func realisticSquad(rng *rand.Rand, firstId int64) []*common.Player {
	squad := randomSquad(rng, firstId)
	keeper := bestKeeperIndex(squad)
	n, done := 2+rng.Intn(3), 0
	for _, i := range rng.Perm(len(squad)) {
		if done == n {
			break
		}
		if i == keeper {
			continue
		}
		p, level := squad[i], 60+rng.Intn(21)
		for _, v := range []*int{&p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot} {
			*v = maxInt(20, *v-15)
		}
		switch rng.Intn(6) {
		case 0:
			p.Defense = level
		case 1:
			p.Midfield, p.Pass = level, level
		case 2:
			p.Shot, p.Dribbling = level, level
		case 3:
			p.Lateral, p.Shot, p.Dribbling = level, level, level
		case 4:
			p.Lateral, p.Midfield, p.Pass = level, level, level
		default:
			p.Lateral, p.Defense = level, level
		}
		done++
	}
	return squad
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func bestLineupOf(squad []*common.Player, formation int) *common.Lineup {
	return &common.Lineup{Formation: formation, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(squad, formation))}
}

// projectedAgainstOwn442 is each formation's projected goal difference for squad against its own
// best 4-4-2 XI (reference squad: reference, usually squad itself), and the best formation.
func projectedAgainstOwn442(squad []*common.Player, reference []*common.Player) (map[int]float64, int) {
	ref := ZoneDuelSide{Lineup: bestLineupOf(reference, common.FormationBalanced), Squad: reference}
	diffs := map[int]float64{}
	best, bestDiff := -1, math.Inf(-1)
	for _, f := range common.AllFormations {
		xg := ProjectExpectedGoals(ZoneDuelSide{Lineup: bestLineupOf(squad, f), Squad: squad}, ref, 1, true)
		diffs[f] = xg.Own - xg.Opponent
		if diffs[f] > bestDiff+1e-9 {
			best, bestDiff = f, diffs[f]
		}
	}
	return diffs, best
}

// calibrateGeneralists is "Random squads": generalist squads' mean projected goal difference per
// formation against their own best 4-4-2 XI.
func calibrateGeneralists(t *testing.T, rng *rand.Rand) {
	const squads = 400
	mean := map[int]float64{}
	for i := 0; i < squads; i++ {
		sq := randomSquad(rng, 1)
		diffs, _ := projectedAgainstOwn442(sq, sq)
		for f, d := range diffs {
			mean[f] += d / squads
		}
	}
	line := ""
	for _, f := range common.AllFormations {
		line += fmt.Sprintf(" %s%+.3f", formationNames[f], mean[f])
		if math.Abs(mean[f]) > 0.05 {
			t.Errorf("Formation balance: generalist squads expect %+.3f goals in %s against their own 4-4-2, want within ±0.05", mean[f], formationNames[f])
		}
	}
	t.Logf("=== generalist squads (%d), mean projected goal difference against own 4-4-2:%s", squads, line)
}

// calibrateGoalVolume is "Goal volume": an even 4-4-2 mirror scores 3.0 to 3.4 goals per match.
func calibrateGoalVolume(t *testing.T, rng *rand.Rand) {
	const matches = 6000
	base := uniformSquad(1, 50, 70)
	var tl tally
	for i := 0; i < matches; i++ {
		hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(base, 0)}, simSide{squad: cloneSquad(base, 1000)}, 1)
		tl.add(hg, ag)
	}
	goals := float64(tl.gf+tl.ga) / float64(tl.n)
	t.Logf("=== goal volume, 4-4-2 mirror with every skill at 50 (%d matches): %.2f goals per match", matches, goals)
	if goals < 3.0 || goals > 3.4 {
		t.Errorf("Goal volume: %.2f goals per match, want 3.0 to 3.4", goals)
	}
}

// calibrateFit is "Specialist profiles" and "Better specialists, bigger edge".
func calibrateFit(t *testing.T, rng *rand.Rand) {
	const squads = 100
	base := make([][]*common.Player, squads)
	for i := range base {
		base[i] = randomSquad(rng, 1)
	}
	t.Log("=== squad fit: share of squads for which the formation is the best projected one, and its edge over the same squad's 4-4-2")
	for _, f := range []int{common.FormationDefensive, common.FormationAttacking, common.FormationFourThreeThree, common.FormationFourFiveOne, common.FormationThreeFiveTwo} {
		profile := homeProfiles[f]
		edge := map[int]float64{}
		share := 0
		for _, level := range []int{75, 40} {
			prng := rand.New(rand.NewSource(int64(f)*100 + int64(level)))
			for _, sq := range base {
				ps := withProfile(prng, sq, profile, level)
				diffs, best := projectedAgainstOwn442(ps, sq)
				edge[level] += (diffs[f] - diffs[common.FormationBalanced]) / squads
				if level == 75 && best == f {
					share++
				}
			}
			if profile.n == 0 {
				break // a capped profile has no level
			}
		}
		t.Logf("%s with %s: best for %d%%, edge %+.3f at 75 and %+.3f at 40", formationNames[f], profile.name, share, edge[75], edge[40])
		if share < 15 {
			t.Errorf("Squad fit: %s is the best shape for %d%% of squads with %s, want at least 15%%", formationNames[f], share, profile.name)
		}
		if profile.n > 0 && edge[75] <= edge[40] {
			t.Errorf("Squad fit: %s's edge over 4-4-2 is %+.3f with specialists at 75 and %+.3f at 40, want it larger at 75", formationNames[f], edge[75], edge[40])
		}
	}
}

// calibrateFiveAtTheBack is "Five at the back with good and poor centre-backs".
func calibrateFiveAtTheBack(t *testing.T, rng *rand.Rand) {
	const squads, matches = 150, 12
	base := make([][]*common.Player, squads)
	for i := range base {
		base[i] = randomSquad(rng, 1)
	}
	profile := homeProfiles[common.FormationDefensive]
	ppg := map[[2]int]float64{}
	for _, level := range []int{75, 40} {
		for _, f := range []int{common.FormationDefensive, common.FormationBalanced} {
			var tl tally
			prng := rand.New(rand.NewSource(int64(level)))
			for _, sq := range base {
				ps := withProfile(prng, sq, profile, level)
				for m := 0; m < matches; m++ {
					hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(ps, 0), formation: f}, simSide{squad: cloneSquad(sq, 1000)}, 1)
					tl.add(hg, ag)
				}
			}
			ppg[[2]int{level, f}] = tl.ppg()
		}
	}
	good := ppg[[2]int{75, common.FormationDefensive}] - ppg[[2]int{75, common.FormationBalanced}]
	poor := ppg[[2]int{40, common.FormationDefensive}] - ppg[[2]int{40, common.FormationBalanced}]
	t.Logf("=== five at the back: centre-backs at 75: 5-3-2 %.2f vs 4-4-2 %.2f (%+.2f); at 40: %.2f vs %.2f (%+.2f)",
		ppg[[2]int{75, common.FormationDefensive}], ppg[[2]int{75, common.FormationBalanced}], good,
		ppg[[2]int{40, common.FormationDefensive}], ppg[[2]int{40, common.FormationBalanced}], poor)
	if good < 0.08 {
		t.Errorf("Squad fit: with good centre-backs a 5-3-2 takes %+.2f points per match over 4-4-2, want at least +0.08", good)
	}
	if poor > 0.03 {
		t.Errorf("Squad fit: with poor centre-backs a 5-3-2 takes %+.2f points per match over 4-4-2, want at most +0.03", poor)
	}
}

// calibrateRealistic is "Realistic squads in full matches" and "Variety across realistic squads".
func calibrateRealistic(t *testing.T, rng *rand.Rand) {
	const squads, matches = 1000, 10
	base := make([][]*common.Player, squads)
	shares := map[int]int{}
	for i := range base {
		base[i] = realisticSquad(rng, 1)
		_, best := projectedAgainstOwn442(base[i], base[i])
		shares[best]++
	}
	line := ""
	for _, f := range common.AllFormations {
		share := 100 * float64(shares[f]) / squads
		line += fmt.Sprintf(" %s=%.0f%%", formationNames[f], share)
		if share > 30 || share < 8 {
			t.Errorf("Squad fit: %s is the best projected shape for %.0f%% of realistic squads, want 8%% to 30%%", formationNames[f], share)
		}
	}
	t.Logf("=== realistic squads (%d), best projected formation:%s", squads, line)

	ppg := map[int]float64{}
	for _, f := range common.AllFormations {
		var tl tally
		for _, sq := range base {
			for m := 0; m < matches; m++ {
				hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(sq, 0), formation: f}, simSide{squad: cloneSquad(sq, 1000)}, 1)
				tl.add(hg, ag)
			}
		}
		ppg[f] = tl.ppg()
		t.Logf("realistic squads, %s vs the same squad in 4-4-2 (%d matches): %s", formationNames[f], tl.n, tl)
	}
	for _, f := range common.AllFormations {
		if d := ppg[f] - ppg[common.FormationBalanced]; math.Abs(d) > 0.05 {
			t.Errorf("Formation balance: realistic squads take %.3f points per match in %s against %.3f for the 4-4-2 mirror, want within 0.05", ppg[f], formationNames[f], ppg[common.FormationBalanced])
		}
	}
}
