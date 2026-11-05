package engine

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "AI level calibration": each CPU level plays a Caretaker with the same squad
// on a neutral ground, both sides planning their kickoff exactly as planAiKickoff does at a league
// match (PlanAiMatchLineup, then PickAiMentality against the other XI). It is what later CPU changes
// are tuned against, and it only runs with AI_LEVEL_SIM=1:
//
//	AI_LEVEL_SIM=1 go test ./engine -run TestAiLevelCalibration -v

// calibrationSquad is a random but plausible 18-man squad drawn from rng: 2 keepers, 6 defenders,
// 6 midfielders and 4 forwards with uneven skills, Stamina and ages, so formation choice,
// substitutions and mentality all have something to work with. firstId offsets the ids so two
// copies of the same squad can face each other.
func calibrationSquad(rng *rand.Rand, firstId int64) []*common.Player {
	squad := make([]*common.Player, 0, 18)
	id := firstId
	skill := func(lo, hi int) int { return lo + rng.Intn(hi-lo+1) }
	add := func(p *common.Player) {
		p.Id = id
		p.Stamina = skill(20, 90)
		p.Age = skill(18, 34)
		squad = append(squad, p)
		id++
	}
	for i := 0; i < 2; i++ {
		add(&common.Player{GoalKeeper: skill(40, 75), Defense: 10, Lateral: 10, Pass: 15, Midfield: 10, Dribbling: 10, Shot: 10})
	}
	for i := 0; i < 6; i++ {
		add(&common.Player{GoalKeeper: 5, Defense: skill(35, 75), Lateral: skill(20, 70), Pass: skill(20, 50), Midfield: skill(20, 45), Dribbling: skill(10, 40), Shot: skill(10, 30)})
	}
	for i := 0; i < 6; i++ {
		add(&common.Player{GoalKeeper: 5, Defense: skill(20, 45), Lateral: skill(20, 70), Pass: skill(35, 75), Midfield: skill(35, 75), Dribbling: skill(25, 60), Shot: skill(20, 50)})
	}
	for i := 0; i < 4; i++ {
		add(&common.Player{GoalKeeper: 5, Defense: skill(10, 30), Lateral: skill(20, 70), Pass: skill(30, 60), Midfield: skill(20, 45), Dribbling: skill(35, 75), Shot: skill(35, 80)})
	}
	return squad
}

// calibrationSide is one side's kickoff plan, as planAiKickoff would make it.
type calibrationSide struct {
	squad []*common.Player
	plan  common.AiMatchPlan
}

func planCalibrationSide(squad []*common.Player, level int, seed int64) calibrationSide {
	return calibrationSide{squad: squad, plan: common.PlanAiMatchLineup(squad, nil, nil, level, false, rand.New(rand.NewSource(seed)))}
}

func (s calibrationSide) lineup(teamId int64, mentality int) *common.Lineup {
	return &common.Lineup{TeamId: teamId, Formation: s.plan.Formation, Positions: s.plan.Positions,
		Substitutions: s.plan.Substitutions, Orders: s.plan.Orders, Mentality: mentality, LineupRoles: s.plan.Roles}
}

func (s calibrationSide) strength() int {
	byId := make(map[int64]*common.Player, len(s.squad))
	for _, p := range s.squad {
		byId[p.Id] = p
	}
	return common.LineupStrength(&common.Lineup{Positions: s.plan.Positions}, byId)
}

// playCalibrationMatch plays one neutral 90-minute match (no injuries: they need the database)
// between two planned sides of the given levels and returns the goals.
func playCalibrationMatch(rng *rand.Rand, home calibrationSide, homeLevel int, away calibrationSide, awayLevel int) (int, int) {
	homeStrength, awayStrength := home.strength(), away.strength()
	homeLineup := home.lineup(1, common.PickAiMentality(homeLevel, homeStrength, awayStrength, true, false))
	awayLineup := away.lineup(2, common.PickAiMentality(awayLevel, awayStrength, homeStrength, false, false))
	names := map[string]string{}
	for _, p := range append(append([]*common.Player(nil), home.squad...), away.squad...) {
		names[PlayerNameKey+strconv.FormatInt(p.Id, 10)] = "P" + strconv.FormatInt(p.Id, 10)
	}
	dbCtx := &common.DatabaseContext{ObjectCache: &setPieceTestCache{values: names}}
	events := make([]common.Event, 0)
	instants := make([]common.MatchInstant, 0)
	match := &common.MatchSimulation{
		Home: 1, Away: 2,
		HomeLineup: homeLineup, AwayLineup: awayLineup,
		HomeQua: &common.TeamQualification{}, AwayQua: &common.TeamQualification{},
		HomePlayers: playersToMapById(&home.squad), AwayPlayers: playersToMapById(&away.squad),
		HomePlayersByPosition: playersByPosition(homeLineup), AwayPlayersByPosition: playersByPosition(awayLineup),
		Events: &events, MatchInstants: &instants,
		YellowCards: map[int64]bool{}, SentOff: map[int64]bool{},
		HomeSubstitutions: home.plan.Substitutions, AwaySubstitutions: away.plan.Substitutions,
		HomeOrders: homeLineup.Orders, AwayOrders: awayLineup.Orders,
		HomeMentality: homeLineup.Mentality, AwayMentality: awayLineup.Mentality,
		HomeMinutesByPosition: map[int64]map[int]int{}, AwayMinutesByPosition: map[int64]map[int]int{},
		HomeAdvantage: 1,
	}
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

// levelRecord is one side's results over a calibration run.
type levelRecord struct{ wins, draws, losses int }

func (r levelRecord) matches() int { return r.wins + r.draws + r.losses }

func (r levelRecord) pointsPerMatch() float64 {
	return float64(3*r.wins+r.draws) / float64(r.matches())
}

// measureAiLevel plays matches games of level against a Caretaker, alternating the venue, with a
// fresh random squad (the same for both sides) every match, and returns level's record and the
// Caretaker's.
func measureAiLevel(matches int, seed int64, level int) (levelRecord, levelRecord) {
	rng := rand.New(rand.NewSource(seed))
	var own, caretaker levelRecord
	for i := 0; i < matches; i++ {
		squadSeed := rng.Int63()
		ownSide := planCalibrationSide(calibrationSquad(rand.New(rand.NewSource(squadSeed)), 1), level, int64(i))
		careSide := planCalibrationSide(calibrationSquad(rand.New(rand.NewSource(squadSeed)), 101), common.AiLevelCaretaker, int64(i)+1)
		var ownGoals, careGoals int
		if i%2 == 0 {
			ownGoals, careGoals = playCalibrationMatch(rng, ownSide, level, careSide, common.AiLevelCaretaker)
		} else {
			careGoals, ownGoals = playCalibrationMatch(rng, careSide, common.AiLevelCaretaker, ownSide, level)
		}
		switch {
		case ownGoals > careGoals:
			own.wins++
			caretaker.losses++
		case ownGoals < careGoals:
			own.losses++
			caretaker.wins++
		default:
			own.draws++
			caretaker.draws++
		}
	}
	return own, caretaker
}

func TestAiLevelCalibration(t *testing.T) {
	if os.Getenv("AI_LEVEL_SIM") != "1" {
		t.Skip("set AI_LEVEL_SIM=1 to run the AI level calibration")
	}
	const matches = 4000
	report := func(label string, r levelRecord) {
		n := float64(r.matches())
		t.Logf("%-22s won %.1f%%  drew %.1f%%  lost %.1f%%  points per match %.3f", label,
			100*float64(r.wins)/n, 100*float64(r.draws)/n, 100*float64(r.losses)/n, r.pointsPerMatch())
	}
	manager, managerOpp := measureAiLevel(matches, 11, common.AiLevelManager)
	report("Manager vs Caretaker", manager)
	report("  its Caretaker", managerOpp)
	director, directorOpp := measureAiLevel(matches, 11, common.AiLevelDirector)
	report("Director vs Caretaker", director)
	report("  its Caretaker", directorOpp)

	if manager.pointsPerMatch() <= managerOpp.pointsPerMatch() {
		t.Errorf("a Manager takes %.3f points per match against a Caretaker's %.3f, want more", manager.pointsPerMatch(), managerOpp.pointsPerMatch())
	}
	if director.pointsPerMatch() <= manager.pointsPerMatch() {
		t.Errorf("a Director takes %.3f points per match against a Caretaker, a Manager %.3f, want the Director ahead", director.pointsPerMatch(), manager.pointsPerMatch())
	}
}
