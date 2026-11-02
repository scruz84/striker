package engine

import (
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/scruz84/striker/common"
)

// openspec add-home-advantage: the home side's zone ratings are multiplied by
// 1 + HomeAdvantageBase + HomeAdvantageCrowdWeight x fill (match-simulation "Home advantage").

func assertMultiplier(t *testing.T, name string, got float32, want float64) {
	t.Helper()
	if math.Abs(float64(got)-want) > 1e-6 {
		t.Errorf("%s: multiplier %.6f, want %.6f", name, got, want)
	}
}

func TestHomeAdvantageMultiplier_FullStadium(t *testing.T) {
	assertMultiplier(t, "full", homeAdvantageMultiplier(12000, 12000), 1.22)
}

func TestHomeAdvantageMultiplier_EmptyStadium(t *testing.T) {
	assertMultiplier(t, "nobody came", homeAdvantageMultiplier(0, 12000), 1.08)
	assertMultiplier(t, "no capacity", homeAdvantageMultiplier(0, 0), 1.08)
}

func TestHomeAdvantageMultiplier_HalfFullStadium(t *testing.T) {
	assertMultiplier(t, "half", homeAdvantageMultiplier(6000, 12000), 1.15)
}

func TestHomeAdvantageMultiplier_FillIsClampedToOne(t *testing.T) {
	assertMultiplier(t, "over capacity", homeAdvantageMultiplier(15000, 12000), 1.22)
}

// fanbase "Big visitor fills the stadium": 80% -> 92% of capacity moves the multiplier from 1.192
// to about 1.209.
func TestHomeAdvantageMultiplier_BigVisitorFillsTheStadium(t *testing.T) {
	assertMultiplier(t, "80%", homeAdvantageMultiplier(8000, 10000), 1.192)
	assertMultiplier(t, "92%", homeAdvantageMultiplier(9200, 10000), 1.2088)
}

func TestHomeAdvantageFor_WorldCupFixtureIsNeutral(t *testing.T) {
	full := matchdayCrowd{Attendance: 40000, Capacity: 40000}
	assertMultiplier(t, "World Cup", homeAdvantageFor(full, true), 1)
	assertMultiplier(t, "ordinary match", homeAdvantageFor(full, false), 1.22)
}

// evenSquad is a keeper (first id) and ten outfield starters at skill 60, plus a bench of a keeper
// and five outfielders at skill 55, all fit (Stamina 80).
func evenSquad(firstId int64) []*common.Player {
	squad := []*common.Player{{Id: firstId, GoalKeeper: 70, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10, Stamina: 80, Age: 27}}
	for i := int64(1); i <= 10; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: 10, Defense: 60, Lateral: 60, Pass: 60, Midfield: 60, Dribbling: 60, Shot: 60, Stamina: 80, Age: 27})
	}
	squad = append(squad, &common.Player{Id: firstId + 11, GoalKeeper: 60, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10, Stamina: 80, Age: 27})
	for i := int64(12); i <= 16; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: 10, Defense: 55, Lateral: 55, Pass: 55, Midfield: 55, Dribbling: 55, Shot: 55, Stamina: 80, Age: 27})
	}
	return squad
}

// newEvenMatch builds a match between two identical squads in a 4-4-2 with the given home
// advantage, ready for the per-minute pipeline, plus a database context whose cache knows every
// player's name (goal events read it).
func newEvenMatch(homeAdvantage float32) (*common.DatabaseContext, *common.MatchSimulation) {
	homeSquad, awaySquad := evenSquad(1), evenSquad(101)
	homeLineup := lineupFor(1, common.FormationBalanced, common.SelectBestLineupForFormation(homeSquad, common.FormationBalanced))
	awayLineup := lineupFor(2, common.FormationBalanced, common.SelectBestLineupForFormation(awaySquad, common.FormationBalanced))
	names := map[string]string{}
	for _, p := range append(append([]*common.Player(nil), homeSquad...), awaySquad...) {
		names[PlayerNameKey+strconv.FormatInt(p.Id, 10)] = "P" + strconv.FormatInt(p.Id, 10)
	}
	events := make([]common.Event, 0)
	instants := make([]common.MatchInstant, 0)
	match := &common.MatchSimulation{
		Home: 1, Away: 2,
		HomeLineup: homeLineup, AwayLineup: awayLineup,
		HomeQua: &common.TeamQualification{}, AwayQua: &common.TeamQualification{},
		HomePlayers: playersToMapById(&homeSquad), AwayPlayers: playersToMapById(&awaySquad),
		HomePlayersByPosition: playersByPosition(homeLineup), AwayPlayersByPosition: playersByPosition(awayLineup),
		Events: &events, MatchInstants: &instants,
		YellowCards: map[int64]bool{}, SentOff: map[int64]bool{},
		HomeMinutesByPosition: map[int64]map[int]int{}, AwayMinutesByPosition: map[int64]map[int]int{},
		HomeAdvantage: homeAdvantage,
	}
	match.EffectiveHomePlayers = match.HomePlayers
	match.EffectiveAwayPlayers = match.AwayPlayers
	return &common.DatabaseContext{ObjectCache: &setPieceTestCache{values: names}}, match
}

// InitialSimulation multiplies all nine home zone ratings, after the blend, and leaves the away
// side alone; a multiplier of 1 (World Cup) or 0 (unset) changes nothing.
func TestInitialSimulation_HomeAdvantageMultipliesTheNineHomeZones(t *testing.T) {
	dbCtx, neutral := newEvenMatch(1)
	neutral.Minute = 1
	InitialSimulation{}.Apply(dbCtx, neutral)
	_, unset := newEvenMatch(0)
	unset.Minute = 1
	InitialSimulation{}.Apply(dbCtx, unset)
	_, full := newEvenMatch(homeAdvantageMultiplier(1000, 1000))
	full.Minute = 1
	InitialSimulation{}.Apply(dbCtx, full)

	if *unset.HomeQua != *neutral.HomeQua {
		t.Errorf("an unset home advantage changed the ratings: %+v vs %+v", *unset.HomeQua, *neutral.HomeQua)
	}
	if *neutral.HomeQua != *neutral.AwayQua {
		t.Errorf("identical squads with no advantage must rate the same: %+v vs %+v", *neutral.HomeQua, *neutral.AwayQua)
	}
	zones := func(q *common.TeamQualification) []float32 {
		return []float32{q.LeftDefense, q.Defense, q.RightDefense, q.LeftMidfield, q.Midfield, q.RightMidfield, q.LeftAttack, q.Attack, q.RightAttach}
	}
	base, boosted := zones(neutral.HomeQua), zones(full.HomeQua)
	for i := range base {
		if base[i] <= 0 || math.Abs(float64(boosted[i]/base[i])-1.22) > 1e-5 {
			t.Errorf("zone %d: %.4f with a full stadium vs %.4f neutral, want x1.22", i, boosted[i], base[i])
		}
	}
	if *full.AwayQua != *neutral.AwayQua {
		t.Errorf("the away side's ratings changed: %+v vs %+v", *full.AwayQua, *neutral.AwayQua)
	}
	if full.MidfieldDiff <= neutral.MidfieldDiff {
		t.Errorf("the home midfield edge must grow with the advantage: %.4f vs %.4f", full.MidfieldDiff, neutral.MidfieldDiff)
	}
	if stored := (*full.MatchInstants)[0].HomeQua; stored != *full.HomeQua {
		t.Errorf("the stored instant must carry the multiplied ratings: %+v vs %+v", stored, *full.HomeQua)
	}
}

// playEvenMatch plays one 90-minute match between identical squads (goals and cards drawn from rng;
// injuries need the database and are symmetrical, so they are left out) and returns the score.
func playEvenMatch(rng *rand.Rand, homeAdvantage float32) (int, int) {
	dbCtx, match := newEvenMatch(homeAdvantage)
	pipeline := SimulationPipelineImpl{simulations: []Simulation{
		SubstitutionSimulation{}, StaminaSimulation{}, InitialSimulation{},
		ScorerSimulation{r: rng}, DisciplineSimulation{r: rng},
	}}
	for minute := 1; minute <= 90; minute++ {
		match.Minute = minute
		pipeline.Simulate(dbCtx, match)
		trackMinutesByPosition(match.HomePlayersByPosition, match.HomeMinutesByPosition)
		trackMinutesByPosition(match.AwayPlayersByPosition, match.AwayMinutesByPosition)
		if applyForfeitIfUnderstaffed(match) {
			break
		}
	}
	return match.HomeGoals, match.AwayGoals
}

type homeAdvantageRates struct {
	homeWins, draws, awayWins float64
	homeGoals, awayGoals      float64
}

func measureHomeAdvantage(matches int, seed int64, homeAdvantage float32) homeAdvantageRates {
	rng := rand.New(rand.NewSource(seed))
	var r homeAdvantageRates
	for i := 0; i < matches; i++ {
		hg, ag := playEvenMatch(rng, homeAdvantage)
		switch {
		case hg > ag:
			r.homeWins++
		case hg < ag:
			r.awayWins++
		default:
			r.draws++
		}
		r.homeGoals += float64(hg)
		r.awayGoals += float64(ag)
	}
	n := float64(matches)
	r.homeWins, r.draws, r.awayWins = r.homeWins/n, r.draws/n, r.awayWins/n
	return r
}

// TestHomeAdvantageCalibration (match-simulation "Expected effect between even sides") simulates
// 20,000 matches between identical squads at a half-full stadium and checks the home side wins
// 43-49%, the away side 25-31%, and the home side scores 1.15-1.40x as many goals. It takes a
// while, so it only runs with HOME_ADV_SIM=1:
//
//	HOME_ADV_SIM=1 go test ./engine -run TestHomeAdvantageCalibration -v
func TestHomeAdvantageCalibration(t *testing.T) {
	if os.Getenv("HOME_ADV_SIM") != "1" {
		t.Skip("set HOME_ADV_SIM=1 to run the home advantage calibration")
	}
	const matches = 20000
	report := func(label string, r homeAdvantageRates) {
		t.Logf("%-10s home %.1f%%  draw %.1f%%  away %.1f%%  goals %.2f-%.2f (ratio %.2f)", label,
			100*r.homeWins, 100*r.draws, 100*r.awayWins, r.homeGoals/matches, r.awayGoals/matches, r.homeGoals/r.awayGoals)
	}
	report("neutral", measureHomeAdvantage(matches, 7, 1))
	r := measureHomeAdvantage(matches, 7, homeAdvantageMultiplier(1, 2))
	report("half-full", r)

	if r.homeWins < 0.43 || r.homeWins > 0.49 {
		t.Errorf("home side won %.1f%%, want 43-49%%", 100*r.homeWins)
	}
	if r.awayWins < 0.25 || r.awayWins > 0.31 {
		t.Errorf("away side won %.1f%%, want 25-31%%", 100*r.awayWins)
	}
	if ratio := r.homeGoals / r.awayGoals; ratio < 1.15 || ratio > 1.40 {
		t.Errorf("home/away goal ratio %.2f, want 1.15-1.40", ratio)
	}
}
