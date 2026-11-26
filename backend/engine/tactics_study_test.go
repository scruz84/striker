package engine

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/scruz84/striker/common"
)

// Tactics study harness: simulates many matches to compare formations, mentalities and the weight
// of each attribute per lineup slot. Opt-in, since a full run takes several minutes:
//
//	TACTICS_SIM=1 go test ./engine -run TestTactics -v -timeout 30m
//
// TACTICS_SIM can instead name the blocks to run, comma-separated: rr (formation round robin),
// zones, xg, mcrand, ment, weights, shooters, and the calibration blocks of
// formation_calibration_test.go: goals, fit, fitmc, real. rr, xg, goals, fit, fitmc and real
// assert the openspec match-simulation "Formation balance" and "Squad fit" bounds.

var formationNames = map[int]string{
	common.FormationBalanced: "4-4-2", common.FormationDefensive: "5-3-2", common.FormationAttacking: "3-4-3",
	common.FormationFourThreeThree: "4-3-3", common.FormationFourFiveOne: "4-5-1", common.FormationThreeFiveTwo: "3-5-2",
}
var mentalityNames = map[int]string{-1: "DEF", 0: "BAL", 1: "ATT"}

type simSide struct {
	squad     []*common.Player
	formation int
	mentality int
	lineup    map[int][]int64 // optional explicit lineup
}

func cloneSquad(src []*common.Player, idOffset int64) []*common.Player {
	out := make([]*common.Player, len(src))
	for i, p := range src {
		c := *p
		c.Id = p.Id + idOffset
		out[i] = &c
	}
	return out
}

func playSides(rng *rand.Rand, home simSide, away simSide, homeAdv float32) (int, int, *common.MatchSimulation) {
	hl := home.lineup
	if hl == nil {
		hl = common.SelectBestLineupForFormation(home.squad, home.formation)
	}
	al := away.lineup
	if al == nil {
		al = common.SelectBestLineupForFormation(away.squad, away.formation)
	}
	homeLineup, awayLineup := lineupFor(1, home.formation, hl), lineupFor(2, away.formation, al)
	homeLineup.Mentality, awayLineup.Mentality = home.mentality, away.mentality
	names := map[string]string{}
	for _, p := range append(append([]*common.Player(nil), home.squad...), away.squad...) {
		names[PlayerNameKey+strconv.FormatInt(p.Id, 10)] = "P"
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
		HomeMinutesByPosition: map[int64]map[int]int{}, AwayMinutesByPosition: map[int64]map[int]int{},
		HomeAdvantage: homeAdv,
		HomeMentality: home.mentality, AwayMentality: away.mentality,
	}
	pipeline := SimulationPipelineImpl{simulations: []Simulation{
		SubstitutionSimulation{}, StaminaSimulation{}, InitialSimulation{},
		ScorerSimulation{r: rng}, DisciplineSimulation{r: rng},
	}}
	playMatchMinutes(dbCtx, match, pipeline)
	return match.HomeGoals, match.AwayGoals, match
}

type tally struct {
	n, w, d, l int
	gf, ga     int
}

func (t *tally) add(gf, ga int) {
	t.n++
	t.gf += gf
	t.ga += ga
	switch {
	case gf > ga:
		t.w++
	case gf < ga:
		t.l++
	default:
		t.d++
	}
}
func (t tally) ppg() float64 { return float64(3*t.w+t.d) / float64(t.n) }
func (t tally) String() string {
	n := float64(t.n)
	return fmt.Sprintf("W%4.1f D%4.1f L%4.1f  %.2f-%.2f  ppg %.2f", 100*float64(t.w)/n, 100*float64(t.d)/n, 100*float64(t.l)/n,
		float64(t.gf)/n, float64(t.ga)/n, t.ppg())
}

// uniformSquad: keeper + 16 identical outfielders at skill s, all fit.
func uniformSquad(firstId int64, s int, stamina int) []*common.Player {
	squad := []*common.Player{{Id: firstId, GoalKeeper: s, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10, Stamina: stamina, Age: 27}}
	for i := int64(1); i <= 16; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: 10, Defense: s, Lateral: s, Pass: s, Midfield: s, Dribbling: s, Shot: s, Stamina: stamina, Age: 27})
	}
	return squad
}

// randomSquad mimics doCreatePlayers: 18 players, each skill uniform in [30,60], stamina [50,90].
func randomSquad(rng *rand.Rand, firstId int64) []*common.Player {
	r := func() int { return 30 + rng.Intn(31) }
	squad := make([]*common.Player, 0, 18)
	for i := int64(0); i < 18; i++ {
		squad = append(squad, &common.Player{Id: firstId + i, GoalKeeper: r(), Defense: r(), Lateral: r(), Pass: r(), Midfield: r(), Dribbling: r(), Shot: r(), Stamina: 50 + rng.Intn(41), Age: 27})
	}
	return squad
}

func TestTacticsStudy(t *testing.T) {
	if os.Getenv("TACTICS_SIM") == "" {
		t.Skip("set TACTICS_SIM=1")
	}
	which := os.Getenv("TACTICS_SIM")
	blocks := map[string]bool{}
	for _, name := range strings.Split(which, ",") {
		blocks[strings.TrimSpace(name)] = true
	}
	run := func(name string) bool { return which == "1" || blocks[name] }
	rng := rand.New(rand.NewSource(42))

	// ---------------------------------------------------------------- 1. formation round robin, identical uniform squads
	if run("rr") {
		const N = 3000
		base := uniformSquad(1, 50, 70)
		t.Log("=== 1. Formation round robin, identical squads (all skills 50, stamina 70), neutral, BAL v BAL, N=", N)
		ppg := map[int]float64{}
		header := "         "
		for _, b := range common.AllFormations {
			header += fmt.Sprintf("%9s", formationNames[b])
		}
		t.Log(header + "   avg ppg")
		for _, a := range common.AllFormations {
			line := fmt.Sprintf("%-8s ", formationNames[a])
			total := 0.0
			for _, b := range common.AllFormations {
				var tl tally
				for i := 0; i < N; i++ {
					hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(base, 0), formation: a}, simSide{squad: cloneSquad(base, 1000), formation: b}, 1)
					tl.add(hg, ag)
				}
				line += fmt.Sprintf("%9.2f", tl.ppg())
				total += tl.ppg()
			}
			ppg[a] = total / float64(len(common.AllFormations))
			t.Logf("%s   %.2f", line, ppg[a])
		}
		mean := 0.0
		for _, v := range ppg {
			mean += v / float64(len(ppg))
		}
		for f, v := range ppg {
			if math.Abs(v-mean) > 0.06 {
				t.Errorf("Formation balance: %s averages %.2f points per match in the round robin, want within 0.06 of the mean %.2f", formationNames[f], v, mean)
			}
		}
	}

	// ---------------------------------------------------------------- 1b. zone ratings per formation
	if run("zones") {
		base := uniformSquad(1, 50, 99)
		t.Log("=== 1b. Minute-1 zone ratings per formation, identical squads (all 50)")
		t.Log("form     LDef   Def  RDef  LMid   Mid  RMid  LAtt   Att  RAtt   sum")
		for _, f := range common.AllFormations {
			l := &common.Lineup{Formation: f, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(base, f))}
			d := ProjectZoneDuel(ZoneDuelSide{Lineup: l, Squad: base}, ZoneDuelSide{Lineup: l, Squad: base}, 1, true)
			q := d.Own
			t.Logf("%-6s %6.1f%6.1f%6.1f%6.1f%6.1f%6.1f%6.1f%6.1f%6.1f%6.0f", formationNames[f], q.LeftDefense, q.Defense, q.RightDefense, q.LeftMidfield, q.Midfield, q.RightMidfield, q.LeftAttack, q.Attack, q.RightAttach,
				q.LeftDefense+q.Defense+q.RightDefense+q.LeftMidfield+q.Midfield+q.RightMidfield+q.LeftAttack+q.Attack+q.RightAttach)
		}
	}

	// ---------------------------------------------------------------- 2. random realistic squads: xG projection round robin
	if run("xg") {
		const K = 400
		t.Log("=== 2. Random realistic squads (skills U[30,60]): same squad, formation A vs B, projected xG difference (A-B), mean over", K, "squads")
		diff := map[[2]int]float64{}
		bestCount := map[int]int{}
		for k := 0; k < K; k++ {
			sq := randomSquad(rng, 1)
			lineups := map[int]*common.Lineup{}
			for _, f := range common.AllFormations {
				lineups[f] = &common.Lineup{Formation: f, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(sq, f))}
			}
			for _, a := range common.AllFormations {
				for _, b := range common.AllFormations {
					xg := ProjectExpectedGoals(ZoneDuelSide{lineups[a], sq}, ZoneDuelSide{lineups[b], sq}, 1, true)
					diff[[2]int{a, b}] += (xg.Own - xg.Opponent) / K
				}
			}
			// which formation the CPU's static ranking (PositionValue sum) would rank first
			byId := map[int64]*common.Player{}
			for _, p := range sq {
				byId[p.Id] = p
			}
			best, bestS := -1, -1
			for _, f := range common.AllFormations {
				if s := common.LineupStrength(lineups[f], byId); s > bestS {
					best, bestS = f, s
				}
			}
			bestCount[best]++
		}
		header := "         "
		for _, b := range common.AllFormations {
			header += fmt.Sprintf("%8s", formationNames[b])
		}
		t.Log(header)
		for _, a := range common.AllFormations {
			line := fmt.Sprintf("%-8s ", formationNames[a])
			for _, b := range common.AllFormations {
				line += fmt.Sprintf("%+8.2f", diff[[2]int{a, b}])
			}
			t.Log(line)
		}
		t.Logf("CPU static ranking (LineupStrength) picks as #1: %v", namedCounts(bestCount))
	}

	if run("xg") {
		calibrateGeneralists(t, rng)
	}
	if run("goals") {
		calibrateGoalVolume(t, rng)
	}
	if run("fit") {
		calibrateFit(t, rng)
	}
	if run("fitmc") {
		calibrateFiveAtTheBack(t, rng)
	}
	if run("real") {
		calibrateRealistic(t, rng)
	}

	// ---------------------------------------------------------------- 2b. random squads Monte Carlo: each formation vs 4-4-2
	if run("mcrand") {
		const K, M = 300, 10
		t.Logf("=== 2b. Random squads, Monte Carlo: formation X vs 4-4-2 with the same squad, neutral, %d squads x %d matches", K, M)
		tl := map[int]*tally{}
		for _, f := range common.AllFormations {
			tl[f] = &tally{}
		}
		for k := 0; k < K; k++ {
			sq := randomSquad(rng, 1)
			for _, f := range common.AllFormations {
				for m := 0; m < M; m++ {
					hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(sq, 0), formation: f}, simSide{squad: cloneSquad(sq, 1000), formation: common.FormationBalanced}, 1)
					tl[f].add(hg, ag)
				}
			}
		}
		for _, f := range common.AllFormations {
			t.Logf("%-6s vs 4-4-2: %s", formationNames[f], tl[f])
		}
	}

	// ---------------------------------------------------------------- 3. mentality
	if run("ment") {
		const N = 6000
		t.Log("=== 3a. Mentality matrix, identical squads (50), 4-4-2 mirror, neutral, N=", N)
		base := uniformSquad(1, 50, 70)
		for _, ma := range []int{-1, 0, 1} {
			line := fmt.Sprintf("%s: ", mentalityNames[ma])
			for _, mb := range []int{-1, 0, 1} {
				var tl tally
				for i := 0; i < N; i++ {
					hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(base, 0), mentality: ma}, simSide{squad: cloneSquad(base, 1000), mentality: mb}, 1)
					tl.add(hg, ag)
				}
				line += fmt.Sprintf(" | vs %s %s", mentalityNames[mb], tl)
			}
			t.Log(line)
		}
		t.Log("=== 3b. Stronger (55) vs weaker (45) side, 4-4-2, neutral: who benefits from which mentality (result for the strong side)")
		strong, weak := uniformSquad(1, 55, 70), uniformSquad(1000, 45, 70)
		for _, ms := range []int{-1, 0, 1} {
			line := fmt.Sprintf("strong %s: ", mentalityNames[ms])
			for _, mw := range []int{-1, 0, 1} {
				var tl tally
				for i := 0; i < N; i++ {
					hg, ag, _ := playSides(rng, simSide{squad: cloneSquad(strong, 0), mentality: ms}, simSide{squad: cloneSquad(weak, 0), mentality: mw}, 1)
					tl.add(hg, ag)
				}
				line += fmt.Sprintf(" | weak %s %s", mentalityNames[mw], tl)
			}
			t.Log(line)
		}
	}

	// ---------------------------------------------------------------- 4. attribute weight per position (xG marginal)
	if run("weights") {
		t.Log("=== 4. Marginal value of +10 in one attribute for ONE player at one slot (4-4-2 and 4-3-3 mirror, all 50). Projected xG diff change x100")
		attrs := []string{"GK", "Def", "Lat", "Pass", "Mid", "Drib", "Shot"}
		bump := func(p *common.Player, a string) {
			switch a {
			case "GK":
				p.GoalKeeper += 10
			case "Def":
				p.Defense += 10
			case "Lat":
				p.Lateral += 10
			case "Pass":
				p.Pass += 10
			case "Mid":
				p.Midfield += 10
			case "Drib":
				p.Dribbling += 10
			case "Shot":
				p.Shot += 10
			}
		}
		posNames := map[int]string{common.GOALKEEPER: "GK", common.DEFENSE: "CB", common.LATERAL_DEFENSE_LEFT: "LB", common.MIDFIELD: "CM",
			common.LATERAL_MIDFIELD_LEFT: "LM", common.STRIKER: "ST", common.LATERAL_STRIKER_LEFT: "LW"}
		for _, f := range []int{common.FormationBalanced, common.FormationFourThreeThree} {
			base := uniformSquad(1, 50, 99)
			base[0].GoalKeeper = 50
			lmap := common.SelectBestLineupForFormation(base, f)
			lineup := &common.Lineup{Formation: f, Positions: common.FlattenLineup(lmap)}
			ref := ProjectExpectedGoals(ZoneDuelSide{lineup, base}, ZoneDuelSide{lineup, base}, 1, true)
			refDiff := ref.Own - ref.Opponent
			t.Logf("-- %s (base xG %.2f-%.2f)", formationNames[f], ref.Own, ref.Opponent)
			t.Log("slot  " + fmt.Sprintf("%7s%7s%7s%7s%7s%7s%7s", "GK", "Def", "Lat", "Pass", "Mid", "Drib", "Shot") + "   | CPU PositionValue gain")
			for _, pos := range []int{common.GOALKEEPER, common.DEFENSE, common.LATERAL_DEFENSE_LEFT, common.MIDFIELD, common.LATERAL_MIDFIELD_LEFT, common.STRIKER, common.LATERAL_STRIKER_LEFT} {
				if len(lmap[pos]) == 0 {
					continue
				}
				id := lmap[pos][0]
				line := fmt.Sprintf("%-5s", posNames[pos])
				pv := ""
				for _, a := range attrs {
					sq := cloneSquad(base, 0)
					var pl *common.Player
					for _, p := range sq {
						if p.Id == id {
							pl = p
						}
					}
					before := common.PositionValue(pos, *pl)
					bump(pl, a)
					pv += fmt.Sprintf("%3d", common.PositionValue(pos, *pl)-before)
					xg := ProjectExpectedGoals(ZoneDuelSide{lineup, sq}, ZoneDuelSide{lineup, base}, 1, true)
					line += fmt.Sprintf("%+7.1f", 100*((xg.Own-xg.Opponent)-refDiff))
				}
				t.Log(line + "   | " + pv)
			}
		}
	}

	// ---------------------------------------------------------------- 5. shooters: who scores open play goals
	if run("shooters") {
		t.Log("=== 5. Who takes open-play shots / scores, by lineup slot (4-3-3 and 4-4-2, identical squads)")
		base := uniformSquad(1, 50, 70)
		for _, f := range []int{common.FormationBalanced, common.FormationFourThreeThree} {
			shots, goals := map[int]int{}, map[int]int{}
			for i := 0; i < 2000; i++ {
				_, _, m := playSides(rng, simSide{squad: cloneSquad(base, 0), formation: f}, simSide{squad: cloneSquad(base, 1000), formation: f}, 1)
				posOf := map[int64]int{}
				for _, p := range m.HomeLineup.Positions {
					posOf[p.PlayerId] = p.Position
				}
				for _, e := range *m.Events {
					if e.TeamOwner != 1 || (e.EventType != common.GOAL && e.EventType != common.NO_GOAL) || len(e.TeamOwnerPlayers) == 0 {
						continue
					}
					pos := posOf[e.TeamOwnerPlayers[0]]
					if e.SetPiece != common.SET_PIECE_NONE {
						pos = 100 // set piece
					}
					shots[pos]++
					if e.EventType == common.GOAL {
						goals[pos]++
					}
				}
			}
			t.Logf("-- %s: shots %v  goals %v", formationNames[f], shots, goals)
		}
	}
}

func namedCounts(m map[int]int) string {
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", formationNames[k], m[k]))
	}
	return strings.Join(parts, " ")
}

func TestTacticsDirectorPick(t *testing.T) {
	if os.Getenv("TACTICS_SIM") == "" {
		t.Skip("set TACTICS_SIM=1")
	}
	rng := rand.New(rand.NewSource(7))
	counts := map[int]int{}
	for k := 0; k < 400; k++ {
		own, opp := randomSquad(rng, 1), randomSquad(rng, 1000)
		ol := &common.Lineup{Formation: common.FormationBalanced, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(opp, common.FormationBalanced))}
		counts[chooseFormationAgainst(own, ZoneDuelSide{Lineup: ol, Squad: opp}, 1, true)]++
	}
	t.Logf("Director picks vs a 4-4-2: %s", namedCounts(counts))
}
