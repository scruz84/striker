package engine

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/scruz84/striker/common"
)

// tiredSquad is a keeper (first id) and ten outfield starters at skill 70 with Stamina 0, plus a
// bench of a keeper and five fresh (Stamina 99) outfielders at skill 66 - the case a Director's
// planned substitutions are meant to help with.
func tiredSquad(firstId int64) []*common.Player {
	squad := []*common.Player{{Id: firstId, GoalKeeper: 90, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10, Stamina: 0, Age: 28}}
	for i := int64(1); i <= 10; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: 10, Defense: 70, Lateral: 70, Pass: 70, Midfield: 70, Dribbling: 70, Shot: 70, Stamina: 0, Age: 28})
	}
	squad = append(squad, &common.Player{Id: firstId + 11, GoalKeeper: 60, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10, Stamina: 99, Age: 28})
	for i := int64(12); i <= 16; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: 10, Defense: 66, Lateral: 66, Pass: 66, Midfield: 66, Dribbling: 66, Shot: 66, Stamina: 99, Age: 28})
	}
	return squad
}

func lineupFor(teamId int64, formation int, lineup map[int][]int64) *common.Lineup {
	return &common.Lineup{TeamId: teamId, Formation: formation, Positions: common.FlattenLineup(lineup)}
}

// playSeededMatch plays one 90-minute match between two identical tired squads, with the home side
// using homeSubstitutions and the away side none, and returns the home goal difference and the
// home side's summed zone qualifications over the match. Goals, cards and chances draw from rng
// (injuries are left out: they are symmetrical and need the database).
func playSeededMatch(rng *rand.Rand, homeSquad []*common.Player, awaySquad []*common.Player, formation int, homeSubstitutions []common.LineupSubstitution) (int, float64) {
	homeLineupMap := common.SelectBestLineupForFormation(homeSquad, formation)
	awayLineupMap := common.SelectBestLineupForFormation(awaySquad, formation)
	names := map[string]string{}
	for _, p := range append(append([]*common.Player(nil), homeSquad...), awaySquad...) {
		names[PlayerNameKey+strconv.FormatInt(p.Id, 10)] = "P" + strconv.FormatInt(p.Id, 10)
	}
	dbCtx := &common.DatabaseContext{ObjectCache: &setPieceTestCache{values: names}}

	events := make([]common.Event, 0)
	instants := make([]common.MatchInstant, 0)
	homeLineup, awayLineup := lineupFor(1, formation, homeLineupMap), lineupFor(2, formation, awayLineupMap)
	match := &common.MatchSimulation{
		Home: 1, Away: 2,
		HomeLineup: homeLineup, AwayLineup: awayLineup,
		HomeQua: &common.TeamQualification{}, AwayQua: &common.TeamQualification{},
		HomePlayers: playersToMapById(&homeSquad), AwayPlayers: playersToMapById(&awaySquad),
		HomePlayersByPosition: playersByPosition(homeLineup), AwayPlayersByPosition: playersByPosition(awayLineup),
		Events: &events, MatchInstants: &instants,
		YellowCards: map[int64]bool{}, SentOff: map[int64]bool{},
		HomeSubstitutions:     homeSubstitutions,
		HomeMinutesByPosition: map[int64]map[int]int{}, AwayMinutesByPosition: map[int64]map[int]int{},
	}
	pipeline := SimulationPipelineImpl{simulations: []Simulation{
		SubstitutionSimulation{}, StaminaSimulation{}, InitialSimulation{},
		ScorerSimulation{r: rng}, DisciplineSimulation{r: rng},
	}}

	strength := 0.0
	for minute := 1; minute <= 90; minute++ {
		match.Minute = minute
		pipeline.Simulate(dbCtx, match)
		trackMinutesByPosition(match.HomePlayersByPosition, match.HomeMinutesByPosition)
		trackMinutesByPosition(match.AwayPlayersByPosition, match.AwayMinutesByPosition)
		q := match.HomeQua
		strength += float64(q.Defense + q.Midfield + q.LeftAttack + q.RightAttach + q.Attack)
	}
	return match.HomeGoals - match.AwayGoals, strength
}

// TestDirectorPlannedSubstitutions_AreNotWorseOverSeededMatches plays 500 seeded matches of a
// Director's tired side with and without its planned substitutions, against an identical side that
// makes none. The planned changes must raise the side's summed strength over the match (that part
// is exact) and must not cost goals beyond noise.
func TestDirectorPlannedSubstitutions_AreNotWorseOverSeededMatches(t *testing.T) {
	const matches = 500
	home, away := tiredSquad(1), tiredSquad(101)
	plan := common.PlanAiMatchLineup(home, nil, nil, common.AiLevelDirector, false, rand.New(rand.NewSource(1)))
	if len(plan.Substitutions) == 0 {
		t.Fatalf("expected the Director to plan substitutions for a tired side")
	}

	withGoals, withoutGoals := 0, 0
	withStrength, withoutStrength := 0.0, 0.0
	for i := 0; i < matches; i++ {
		goals, strength := playSeededMatch(rand.New(rand.NewSource(int64(i))), home, away, plan.Formation, plan.Substitutions)
		withGoals += goals
		withStrength += strength
		goals, strength = playSeededMatch(rand.New(rand.NewSource(int64(i))), home, away, plan.Formation, nil)
		withoutGoals += goals
		withoutStrength += strength
	}

	if withStrength <= withoutStrength {
		t.Fatalf("expected planned substitutions to raise the side's strength, got %.1f vs %.1f", withStrength, withoutStrength)
	}
	meanWith, meanWithout := float64(withGoals)/matches, float64(withoutGoals)/matches
	// Goals are noisy (about 0.11 standard error on a difference of two means of 500 matches), so
	// only a clear loss counts as "worse".
	if meanWith < meanWithout-0.35 {
		t.Fatalf("expected planned substitutions not to cost goals: mean goal difference %.3f with vs %.3f without", meanWith, meanWithout)
	}
	t.Logf("mean goal difference %.3f with planned substitutions, %.3f without; strength %.0f vs %.0f", meanWith, meanWithout, withStrength, withoutStrength)
}
