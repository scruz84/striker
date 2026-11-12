package engine

import (
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// openspec add-team-mentality: an Attacking side's attack ratings x1.08 and defense x0.92 (a
// Defensive side's the other way round), +/-1 point of chance probability per side, and a 0.33 /
// 0.27 maximum fatigue penalty (match-simulation "Team mentality", "Chance generation",
// "In-match fatigue").

func near(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestApplyMentality_AttackingSide(t *testing.T) {
	q := &common.TeamQualification{Attack: 200, LeftAttack: 100, RightAttach: 50, Defense: 180, LeftDefense: 90, RightDefense: 45, Midfield: 150, Goalkeeper: 70}
	applyMentality(q, common.MENTALITY_ATTACKING)
	if !near(float64(q.Attack), 216) || !near(float64(q.Defense), 165.6) {
		t.Errorf("central attack %.2f defense %.2f, want 216 and 165.6", q.Attack, q.Defense)
	}
	if !near(float64(q.LeftAttack), 108) || !near(float64(q.RightAttach), 54) || !near(float64(q.LeftDefense), 82.8) || !near(float64(q.RightDefense), 41.4) {
		t.Errorf("flanks %+v, want every attack x1.08 and defense x0.92", *q)
	}
	if q.Midfield != 150 || q.Goalkeeper != 70 {
		t.Errorf("midfield %.2f goalkeeper %.2f changed", q.Midfield, q.Goalkeeper)
	}
}

func TestApplyMentality_DefensiveSideAndBalanced(t *testing.T) {
	q := &common.TeamQualification{Attack: 200, Defense: 180}
	applyMentality(q, common.MENTALITY_DEFENSIVE)
	if !near(float64(q.Attack), 184) || !near(float64(q.Defense), 194.4) {
		t.Errorf("defensive: attack %.2f defense %.2f, want 184 and 194.4", q.Attack, q.Defense)
	}
	b := &common.TeamQualification{Attack: 200, Defense: 180, Midfield: 3}
	before := *b
	applyMentality(b, common.MENTALITY_BALANCED)
	if *b != before {
		t.Errorf("a Balanced side changed: %+v", *b)
	}
}

func chanceAt(home, away int) float32 {
	return chanceProbability(&common.MatchSimulation{
		HomeQua: &common.TeamQualification{Midfield: 50}, AwayQua: &common.TeamQualification{Midfield: 50},
		HomeMentality: home, AwayMentality: away,
	})
}

func TestChanceProbability_Mentality(t *testing.T) {
	if got := chanceAt(common.MENTALITY_BALANCED, common.MENTALITY_BALANCED); got != ProbChance {
		t.Errorf("two Balanced sides: %.4f, want exactly %.4f", got, ProbChance)
	}
	if got := chanceAt(common.MENTALITY_ATTACKING, common.MENTALITY_ATTACKING); !near(float64(got), 0.16) {
		t.Errorf("two Attacking sides: %.4f, want 0.16", got)
	}
	if got := chanceAt(common.MENTALITY_ATTACKING, common.MENTALITY_DEFENSIVE); !near(float64(got), float64(ProbChance)) {
		t.Errorf("Attacking against Defensive: %.4f, want %.4f", got, ProbChance)
	}
	if got := chanceAt(common.MENTALITY_DEFENSIVE, common.MENTALITY_DEFENSIVE); !near(float64(got), 0.12) {
		t.Errorf("two Defensive sides: %.4f, want 0.12", got)
	}
}

func TestApplyFatigue_MentalitySetsTheMaximumPenalty(t *testing.T) {
	players := map[int64]common.Player{1: {Id: 1, Stamina: 0, Defense: 100, Shot: 50}}
	minutes := map[int64]map[int]int{1: {common.DEFENSE: 90}}
	for _, c := range []struct {
		mentality     int
		defense, shot int
	}{
		{common.MENTALITY_BALANCED, 70, 35},
		{common.MENTALITY_ATTACKING, 67, 34}, // 50 x 0.67 = 33.5 rounds to 34
		{common.MENTALITY_DEFENSIVE, 73, 37}, // 50 x 0.73 = 36.5 rounds to 37
	} {
		got := (*applyFatigue(&players, minutes, c.mentality))[1]
		if got.Defense != c.defense || got.Shot != c.shot {
			t.Errorf("mentality %d: defense %d shot %d, want %d and %d", c.mentality, got.Defense, got.Shot, c.defense, c.shot)
		}
	}
}

// Golden check: with both sides Balanced, every minute of a 90-minute match rates both sides
// exactly as the pipeline did before mentality existed - the old fatigue formula
// (common.FatigueMultiplier), the zone ratings with only the home advantage on top, and the old
// chance probability. (The chance draw itself uses the process-wide random source, which can't be
// seeded, so the check is made on everything the draw depends on.)
func TestBalancedSidesPlayExactlyAsBefore(t *testing.T) {
	dbCtx, match := newEvenMatch(homeAdvantageMultiplier(1, 2))
	_, reference := newEvenMatch(homeAdvantageMultiplier(1, 2))
	for minute := 1; minute <= 90; minute++ {
		match.Minute, reference.Minute = minute, minute
		StaminaSimulation{}.Apply(dbCtx, match)
		InitialSimulation{}.Apply(dbCtx, match)

		oldFatigue := func(players *map[int64]common.Player, byPosition map[int64]map[int]int) *map[int64]common.Player {
			out := make(map[int64]common.Player, len(*players))
			for id, p := range *players {
				m := common.FatigueMultiplier(p.Stamina, minutesPlayedSoFar(byPosition[id]))
				p.GoalKeeper, p.Defense, p.Lateral = scaleSkill(p.GoalKeeper, m), scaleSkill(p.Defense, m), scaleSkill(p.Lateral, m)
				p.Pass, p.Midfield, p.Dribbling, p.Shot = scaleSkill(p.Pass, m), scaleSkill(p.Midfield, m), scaleSkill(p.Dribbling, m), scaleSkill(p.Shot, m)
				out[id] = p
			}
			return &out
		}
		reference.EffectiveHomePlayers = oldFatigue(reference.HomePlayers, reference.HomeMinutesByPosition)
		reference.EffectiveAwayPlayers = oldFatigue(reference.AwayPlayers, reference.AwayMinutesByPosition)
		calculateQua(reference.HomeQua, reference.HomeLineup, reference.HomePlayersByPosition, reference.EffectiveHomePlayers)
		calculateQua(reference.AwayQua, reference.AwayLineup, reference.AwayPlayersByPosition, reference.EffectiveAwayPlayers)
		applyHomeAdvantage(reference.HomeQua, reference.HomeAdvantage)

		if *match.HomeQua != *reference.HomeQua || *match.AwayQua != *reference.AwayQua {
			t.Fatalf("minute %d: ratings %+v / %+v, want %+v / %+v", minute, *match.HomeQua, *match.AwayQua, *reference.HomeQua, *reference.AwayQua)
		}
		dominance := (match.HomeQua.Midfield - match.AwayQua.Midfield) / (match.HomeQua.Midfield + match.AwayQua.Midfield)
		if dominance < 0 {
			dominance = -dominance
		}
		if got := chanceProbability(match); got != ProbChance+dominance*ProbChanceDominanceBonus {
			t.Fatalf("minute %d: chance probability %.6f changed", minute, got)
		}
		trackMinutesByPosition(match.HomePlayersByPosition, match.HomeMinutesByPosition)
		trackMinutesByPosition(match.AwayPlayersByPosition, match.AwayMinutesByPosition)
		trackMinutesByPosition(reference.HomePlayersByPosition, reference.HomeMinutesByPosition)
		trackMinutesByPosition(reference.AwayPlayersByPosition, reference.AwayMinutesByPosition)
	}
}

// measureMentality plays matches between identical squads on neutral ground, the home side with
// mentality and the away side Balanced, and returns the home side's average goals for and against.
func measureMentality(matches int, seed int64, mentality int) (goalsFor, goalsAgainst float64) {
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < matches; i++ {
		dbCtx, match := newEvenMatch(1)
		match.HomeMentality = mentality
		pipeline := SimulationPipelineImpl{simulations: []Simulation{
			SubstitutionSimulation{}, StaminaSimulation{}, InitialSimulation{},
			ScorerSimulation{r: rng}, DisciplineSimulation{r: rng},
		}}
		playMatchMinutes(dbCtx, match, pipeline)
		goalsFor += float64(match.HomeGoals)
		goalsAgainst += float64(match.AwayGoals)
	}
	return goalsFor / float64(matches), goalsAgainst / float64(matches)
}

// TestMentalityCalibration (match-simulation "Mentality effect sizes") simulates 20,000 matches
// per mentality between identical squads and checks an Attacking side scores and concedes 8-20%
// more than a Balanced side against a Balanced side, and a Defensive side 8-20% fewer. It takes a
// while, so it only runs with MENTALITY_SIM=1:
//
//	MENTALITY_SIM=1 go test ./engine -run TestMentalityCalibration -v
func TestMentalityCalibration(t *testing.T) {
	if os.Getenv("MENTALITY_SIM") != "1" {
		t.Skip("set MENTALITY_SIM=1 to run the mentality calibration")
	}
	const matches = 20000
	baseFor, baseAgainst := measureMentality(matches, 11, common.MENTALITY_BALANCED)
	base := (baseFor + baseAgainst) / 2
	t.Logf("balanced vs balanced: %.3f - %.3f", baseFor, baseAgainst)
	for _, c := range []struct {
		name      string
		mentality int
		low, high float64
	}{
		{"attacking", common.MENTALITY_ATTACKING, 1.08, 1.20},
		{"defensive", common.MENTALITY_DEFENSIVE, 0.80, 0.92},
	} {
		gf, ga := measureMentality(matches, 11, c.mentality)
		t.Logf("%s vs balanced: %.3f - %.3f (for x%.3f, against x%.3f)", c.name, gf, ga, gf/base, ga/base)
		for label, ratio := range map[string]float64{"goals for": gf / base, "goals against": ga / base} {
			if ratio < c.low || ratio > c.high {
				t.Errorf("%s side: %s x%.3f, want %.2f-%.2f", c.name, label, ratio, c.low, c.high)
			}
		}
	}
}
