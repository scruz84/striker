package engine

// Economy simulator - NOT a regression test. It is skipped unless ECON_SIM is set, and exists to
// answer "is the in-game economy balanced?" with numbers instead of a gut feeling. It reuses the
// engine's real income/expense functions (CalculateAttendance, optimalTicketPrice,
// shouldUpgradeStadium, driftSocios, sponsorshipAmount, prizeMoneyForPosition,
// merchandisingForPosition, fanbaseCommercialFactor, driftFanbase*, StadiumMaintenanceCost,
// common.PlayerWageFromSkills, ...) wherever a scenario doesn't override them, so the "shipped"
// scenario is the game exactly as it is. Its output is the acceptance check for the economy
// rebalance (issue 262): passive-team income/expense ratios near 1.0 / 1.1-1.2 / 1.3 for
// divisions 3 / 2 / 1, and a world money supply that stops growing.
//
// Run with:
//
//	ECON_SIM=1 go test ./engine -run TestEconomySimulation -v -count=1
//
// Optional env: ECON_SIM_SCENARIOS=baseline,B-socios+tier-wages (comma list, default all),
// ECON_SIM_SEASONS=40, ECON_SIM_SEED=1. See openspec/changes/archive/2026-10-19-rebalance-economy-socios/design.md
// for what it found and what each scenario means.

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

// simScenario holds the levers the simulator can vary on top of the shipped game: wages by
// division tier and a per-win squad bonus. Sponsorship, prize money, merchandising, season tickets,
// every stadium cost, the socios attendance model and the CPU's pricing/expansion rules always use
// the shipped code.
type simScenario struct {
	name string

	wageTierMult map[int]float64 // multiplier on the wage bill by division tier (nil = the shipped common.DivisionWageFactor)
	winBonus     map[int]int64   // squad bonus paid per league win, by tier (nil = none)
}

func shippedScenario() simScenario {
	return simScenario{name: "shipped"}
}

// ---------------------------------------------------------------- demand model

func (t *simTeam) formPointsAndMatches() (int, int) {
	points := 0
	for _, p := range t.form {
		points += p
	}
	return points, len(t.form)
}

// attendance draws one home crowd with the real CalculateAttendance, and records whether it sold
// the ground out (league matches only) in the rolling window the CPU expansion rule reads.
func (sc simScenario) attendance(t, opp *simTeam, price, cupRound int, r *rand.Rand) int {
	points, matches := t.formPointsAndMatches()
	att := CalculateAttendance(t.capacity, price, t.socios, points, matches, opp.fanbase, cupRound, common.RandomWeather(), r)
	if cupRound == 0 {
		t.sellouts = append(t.sellouts, att >= t.capacity)
		if len(t.sellouts) > common.RecentSelloutMatches {
			t.sellouts = t.sellouts[1:]
		}
	}
	return att
}

func (t *simTeam) recentSellouts() (int, int) {
	n := 0
	for _, s := range t.sellouts {
		if s {
			n++
		}
	}
	return n, len(t.sellouts)
}

func (sc simScenario) bestPrice(t *simTeam) int {
	points, matches := t.formPointsAndMatches()
	return optimalTicketPrice(t.capacity, t.socios, points, matches)
}

// ---------------------------------------------------------------- world

type simClass int

const (
	classHumanPassive simClass = iota // default ticket price, never upgrades, balanced sponsor deal
	classHumanOptimal                 // best price + upgrades stadium whenever affordable + hires a coach
	classCaretaker
	classManager
	classDirector
)

func (c simClass) String() string {
	return [...]string{"human-passive", "human-optimal", "AI-caretaker", "AI-manager", "AI-director"}[c]
}

type simTeam struct {
	tier        int
	class       simClass
	budget      int64
	capacity    int
	price       int
	fanbase     int
	socios      int
	form        []int // league points from the last (up to) five matches
	deal        common.SponsorDeal
	wageBill    int64 // per round, players only
	wageCap     int64 // wage growth from skill progression saturates here (skill caps)
	trainerWage int64
	strength    float64
	pendingCap  int
	pendingDay  int
	attEMA      float64 // smoothed attendance/capacity at home matches
	sellouts    []bool  // whether each of the last home league matches sold the ground out

	ledger map[string]int64 // current season
}

func newSimTeam(tier int, class simClass, sc simScenario, r *rand.Rand) *simTeam {
	t := &simTeam{
		tier: tier, class: class, budget: 400000, capacity: common.DefaultStadiumCapacity,
		price: common.DefaultTicketPrice, fanbase: common.DefaultFanbase,
		deal: common.SponsorDealBalanced, ledger: map[string]int64{}, attEMA: 0.5,
	}
	t.socios = common.DefaultSocios
	total, skillSum := 0, 0.0
	for p := 0; p < common.SquadSize; p++ {
		var s [7]int
		for k := range s {
			s[k] = common.SeniorSkillMin + r.Intn(common.SeniorSkillMax-common.SeniorSkillMin+1)
			skillSum += float64(s[k])
		}
		total += common.PlayerWageFromSkills(s[0], s[1], s[2], s[3], s[4], s[5], s[6])
	}
	t.wageBill = int64(float64(total) * sc.wageMultiplier(tier))
	t.wageCap = int64(float64(t.wageBill) * 1.25) // avg skill ceiling is ~+12 above the starting level
	t.strength = skillSum/float64(common.SquadSize*7) + r.NormFloat64()*3
	return t
}

func (sc simScenario) wageMultiplier(tier int) float64 {
	if sc.wageTierMult == nil {
		return common.DivisionWageFactor(tier)
	}
	return sc.wageTierMult[tier]
}

func (t *simTeam) add(key string, amount int64) {
	t.budget += amount
	t.ledger[key] += amount
}

func (t *simTeam) recordForm(points int) {
	t.form = append(t.form, points)
	if len(t.form) > 5 {
		t.form = t.form[1:]
	}
}

type simLeague struct {
	tier  int
	teams []*simTeam
}

// pickClass mirrors common.aiLevelWeightsByTier.
func pickClass(tier int, r *rand.Rand) simClass {
	weights := [3]int{60, 30, 10}
	switch tier {
	case 1:
		weights = [3]int{10, 30, 60}
	case 2:
		weights = [3]int{30, 50, 20}
	}
	roll := r.Intn(100)
	switch {
	case roll < weights[0]:
		return classCaretaker
	case roll < weights[0]+weights[1]:
		return classManager
	default:
		return classDirector
	}
}

// buildWorld creates the default world (1 + 4 + 16 leagues of 10 teams). In every league, team 0 is
// a passive human and team 1 an optimising human - two reference points a real manager lands
// between; the other eight are CPU teams with the shipped AI-level mix.
func buildWorld(sc simScenario, r *rand.Rand) []*simLeague {
	var leagues []*simLeague
	for _, cfg := range common.DefaultWorldTiers {
		for g := 0; g < cfg.Groups; g++ {
			l := &simLeague{tier: cfg.Tier}
			for i := 0; i < cfg.TeamsPerLeague; i++ {
				class := pickClass(cfg.Tier, r)
				switch i {
				case 0:
					class = classHumanPassive
				case 1:
					class = classHumanOptimal
				}
				l.teams = append(l.teams, newSimTeam(cfg.Tier, class, sc, r))
			}
			leagues = append(leagues, l)
		}
	}
	return leagues
}

// roundRobin returns circle-method pairings for n (even) teams: a double round-robin of 2(n-1)
// rounds, each a list of [home, away] index pairs.
func roundRobin(n int) [][][2]int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	rounds := make([][][2]int, 0, 2*(n-1))
	for round := 0; round < n-1; round++ {
		pairs := make([][2]int, 0, n/2)
		for i := 0; i < n/2; i++ {
			a, b := idx[i], idx[n-1-i]
			if round%2 == 1 {
				a, b = b, a
			}
			pairs = append(pairs, [2]int{a, b})
		}
		rounds = append(rounds, pairs)
		idx = append([]int{idx[0], idx[n-1]}, idx[1:n-1]...)
	}
	for round := 0; round < n-1; round++ {
		mirrored := make([][2]int, 0, n/2)
		for _, p := range rounds[round] {
			mirrored = append(mirrored, [2]int{p[1], p[0]})
		}
		rounds = append(rounds, mirrored)
	}
	return rounds
}

func poisson(lambda float64, r *rand.Rand) int {
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

func playMatch(home, away *simTeam, r *rand.Rand) (int, int) {
	diff := (home.strength - away.strength) * 0.05
	return poisson(math.Max(0.2, 1.7+diff), r), poisson(math.Max(0.2, 1.35-diff), r)
}

func sponsor(t *simTeam, out matchOutcome) int64 {
	amount := sponsorshipAmount(sponsorshipForTier(t.tier), t.deal, out)
	return int64(float64(amount) * fanbaseCommercialFactor(t.fanbase))
}

func outcomePoints(o matchOutcome) int {
	switch o {
	case outcomeWin:
		return 3
	case outcomeDraw:
		return 1
	default:
		return 0
	}
}

// ---------------------------------------------------------------- run

type seasonLedger map[string]int64

// simStats aggregates ledgers per "class|tier", per season.
type simStats struct {
	teams    int
	bySeason []seasonLedger
	homeAtt  int64 // total home league attendance, for the average-crowd column
	homeCap  int64
	homeGms  int64
}

type snapshot struct {
	season       int
	totalBudget  int64
	medianBudget int64
	byClassTier  map[string][3]float64 // avg budget, avg capacity, avg socios
}

type simOutput struct {
	stats map[string]*simStats
	snaps []snapshot
}

func runEconomySim(sc simScenario, seasons int, seed int64) simOutput {
	r := rand.New(rand.NewSource(seed))
	leagues := buildWorld(sc, r)
	out := simOutput{stats: map[string]*simStats{}}
	keyOf := func(t *simTeam) string { return fmt.Sprintf("%s|T%d", t.class, t.tier) }
	for _, l := range leagues {
		for _, t := range l.teams {
			k := keyOf(t)
			if out.stats[k] == nil {
				out.stats[k] = &simStats{}
			}
			out.stats[k].teams++
		}
	}

	rr := roundRobin(common.TeamsPerLeague)
	day := 0
	for season := 0; season < seasons; season++ {
		points := map[*simTeam]int{}
		for round := 0; round < len(rr); round++ {
			for _, l := range leagues {
				for _, pair := range rr[round] {
					home, away := l.teams[pair[0]], l.teams[pair[1]]
					hg, ag := playMatch(home, away, r)

					att := sc.attendance(home, away, home.price, 0, r)
					home.add("ticket", TicketRevenue(att, home.price))
					home.attEMA = 0.9*home.attEMA + 0.1*float64(att)/float64(home.capacity)
					if season > 0 {
						st := out.stats[keyOf(home)]
						st.homeAtt += int64(att)
						st.homeCap += int64(home.capacity)
						st.homeGms++
					}

					for _, side := range []struct {
						team   *simTeam
						gf, ga int
					}{{home, hg, ag}, {away, ag, hg}} {
						t := side.team
						o := outcomeFor(side.gf, side.ga)
						points[t] += outcomePoints(o)
						t.recordForm(outcomePoints(o))
						t.add("sponsorship", sponsor(t, o))
						t.fanbase = driftFanbaseForResult(t.fanbase, o, r)
						t.fanbase = driftFanbaseTierPull(t.fanbase, t.tier, r)

						t.add("wages", -t.wageBill)
						t.add("trainer", -t.trainerWage)
						t.add("maintenance", -StadiumMaintenanceCost(t.capacity))
						if o == outcomeWin && sc.winBonus != nil {
							t.add("winBonus", -sc.winBonus[t.tier])
						}
					}
				}
			}
			for _, l := range leagues {
				for _, t := range l.teams {
					sc.aiDailyTick(t, day)
				}
			}
			day++
		}

		// rollover: prize money, season tickets, merchandising, cup, socios renewal
		for _, l := range leagues {
			sorted := append([]*simTeam(nil), l.teams...)
			sort.SliceStable(sorted, func(i, j int) bool { return points[sorted[i]] > points[sorted[j]] })
			n := len(sorted)
			for i, t := range sorted {
				t.add("prize", prizeMoneyForPosition(prizePoolForTier(l.tier), i+1, n))
				factor := fanbaseCommercialFactor(t.fanbase)
				t.add("seasonTickets", int64(float64(t.capacity*SeasonTicketRevenuePerSeat)*factor))
				t.add("merch", int64(float64(merchandisingForPosition(merchandisingPoolForTier(l.tier), i+1, n))*factor))
				t.socios = driftSocios(t.socios, t.fanbase, i+1, n)
			}
			if l.tier == CupTier {
				sc.simCup(l.teams, r)
			}
		}
		// Skill progression nudges the wage bill up, saturating at each player's skill ceiling.
		for _, l := range leagues {
			for _, t := range l.teams {
				growth := 1.004 // untrained: age trend only
				if t.trainerWage > 0 {
					growth = 1.03 // an 80+ coach adds +2 per drift event for under-30s
				}
				t.wageBill = int64(math.Min(float64(t.wageBill)*growth, float64(t.wageCap)))
			}
		}
		day++ // gap day before the next season's fixtures

		// fold ledgers
		for _, l := range leagues {
			for _, t := range l.teams {
				st := out.stats[keyOf(t)]
				for len(st.bySeason) <= season {
					st.bySeason = append(st.bySeason, seasonLedger{})
				}
				for k, v := range t.ledger {
					st.bySeason[season][k] += v
				}
				t.ledger = map[string]int64{}
			}
		}
		out.snaps = append(out.snaps, takeSnapshot(season+1, leagues))
	}
	return out
}

func takeSnapshot(season int, leagues []*simLeague) snapshot {
	snap := snapshot{season: season, byClassTier: map[string][3]float64{}}
	counts := map[string]float64{}
	var budgets []int64
	for _, l := range leagues {
		for _, t := range l.teams {
			snap.totalBudget += t.budget
			budgets = append(budgets, t.budget)
			key := fmt.Sprintf("%s|T%d", t.class, t.tier)
			v := snap.byClassTier[key]
			v[0] += float64(t.budget)
			v[1] += float64(t.capacity)
			v[2] += float64(t.socios)
			snap.byClassTier[key] = v
			counts[key]++
		}
	}
	for k, v := range snap.byClassTier {
		snap.byClassTier[k] = [3]float64{v[0] / counts[k], v[1] / counts[k], v[2] / counts[k]}
	}
	sort.Slice(budgets, func(i, j int) bool { return budgets[i] < budgets[j] })
	snap.medianBudget = budgets[len(budgets)/2]
	return snap
}

// aiDailyTick applies the economically relevant slice of RunAITeamManagement (aiteams.go):
// Manager+ sets its best ticket price, Director+ hires a trainer and expands the stadium once its
// budget clears upgradeStadiumIfAffordable's cushion. The optimising human does the same, minus
// the cushion. Stadium construction takes StadiumUpgradeDuration (7 days = 7 daily rounds).
func (sc simScenario) aiDailyTick(t *simTeam, day int) {
	if t.pendingCap > 0 && day >= t.pendingDay {
		t.capacity = t.pendingCap
		t.pendingCap = 0
	}
	switch t.class {
	case classManager, classDirector, classHumanOptimal:
		t.price = sc.bestPrice(t)
	}
	if (t.class == classDirector || t.class == classHumanOptimal) && t.trainerWage == 0 {
		t.trainerWage = int64(float64(common.TrainerWageFromSkill(85)) * sc.wageMultiplier(t.tier))
	}
	if t.class != classDirector && t.class != classHumanOptimal {
		return
	}
	if t.pendingCap > 0 {
		return
	}
	if soldOut, total := t.recentSellouts(); !shouldUpgradeStadium(soldOut, total) {
		return
	}
	cost := int64(t.capacity) * common.StadiumUpgradeCostPerCurrentSeat
	cushion := int64(stadiumUpgradeBudgetCushionMultiplier)
	if t.class == classHumanOptimal {
		cushion = 1
	}
	if t.budget < cost*cushion {
		return
	}
	t.add("stadiumUpgrade", -cost)
	t.pendingCap = t.capacity + common.StadiumUpgradeSeats
	t.pendingDay = day + int(common.StadiumUpgradeDuration.Hours()/24)
}

// simCup plays the tier-1 domestic cup as a single-elimination bracket with byes: CupPrizePerRound
// x round to every tie winner plus CupChampionBonus, plus the home side's gate and both sides'
// sponsorship (cup ties go through UpdateTeamFinancesAfterMatch but never pay wages).
func (sc simScenario) simCup(teams []*simTeam, r *rand.Rand) {
	alive := append([]*simTeam(nil), teams...)
	r.Shuffle(len(alive), func(i, j int) { alive[i], alive[j] = alive[j], alive[i] })
	for round := 1; len(alive) > 1; round++ {
		target := 1
		for target*2 < len(alive) {
			target *= 2
		}
		pairs := len(alive) - target
		if len(alive) == target {
			pairs = len(alive) / 2
		}
		next := append([]*simTeam(nil), alive[pairs*2:]...)
		for i := 0; i < pairs; i++ {
			home, away := alive[2*i], alive[2*i+1]
			hg, ag := playMatch(home, away, r)
			winner := home
			if ag > hg || (ag == hg && r.Intn(2) == 0) {
				winner = away
			}
			att := sc.attendance(home, away, home.price, round, r)
			home.add("cupTicket", TicketRevenue(att, home.price))
			home.add("cupSponsorship", sponsor(home, outcomeFor(hg, ag)))
			away.add("cupSponsorship", sponsor(away, outcomeFor(ag, hg)))
			winner.add("cupPrize", int64(CupPrizePerRound*round))
			next = append(next, winner)
		}
		alive = next
	}
	alive[0].add("cupPrize", CupChampionBonus)
}

// ---------------------------------------------------------------- reporting

var incomeKeys = []string{"ticket", "sponsorship", "prize", "seasonTickets", "merch", "cupTicket", "cupSponsorship", "cupPrize"}
var expenseKeys = []string{"wages", "winBonus", "trainer", "maintenance", "stadiumUpgrade"}

func sumKeys(l seasonLedger, keys []string, sign int64) int64 {
	var total int64
	for _, k := range keys {
		total += l[k] * sign
	}
	return total
}

func windowLedger(st *simStats, from, to int) seasonLedger {
	agg := seasonLedger{}
	for s := from; s < to && s < len(st.bySeason); s++ {
		for k, v := range st.bySeason[s] {
			agg[k] += v
		}
	}
	return agg
}

func report(sc simScenario, out simOutput, seasons int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== scenario: %s (%d seasons) ===\n", sc.name, seasons)
	keys := make([]string, 0, len(out.stats))
	for k := range out.stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Per team, per season. "S1" = the first season; "S2-N" = the average of the rest.
	fmt.Fprintf(&b, "%-20s %5s | %-26s | %-26s | %-11s | %s\n", "class|tier", "teams", "S1  income expense ratio",
		"S2-N income expense ratio", "crowd  fill", "S2-N main income -> main expense")
	for _, k := range keys {
		st := out.stats[k]
		per := func(l seasonLedger, n int) (int64, int64) {
			return sumKeys(l, incomeKeys, 1) / int64(st.teams*n), sumKeys(l, expenseKeys, -1) / int64(st.teams*n)
		}
		i1, e1 := per(windowLedger(st, 0, 1), 1)
		rest := windowLedger(st, 1, seasons)
		iN, eN := per(rest, seasons-1)
		crowd, fill := int64(0), 0.0
		if st.homeGms > 0 {
			crowd = st.homeAtt / st.homeGms
			fill = float64(st.homeAtt) / float64(st.homeCap)
		}
		fmt.Fprintf(&b, "%-20s %5d | %9d %9d %5.2f | %9d %9d %5.2f | %6d %3.0f%% | %s -> %s\n", k, st.teams, i1, e1, ratio(i1, e1),
			iN, eN, ratio(iN, eN), crowd, fill*100, topShare(rest, incomeKeys, 1), topShare(rest, expenseKeys, -1))
	}

	fmt.Fprintf(&b, "\nmoney supply across all teams (start = %d):\n", 400000*210)
	step := int(math.Max(1, float64(len(out.snaps))/6))
	for i := step - 1; i < len(out.snaps); i += step {
		s := out.snaps[i]
		fmt.Fprintf(&b, "  after season %3d: total=%14d  median team budget=%12d\n", s.season, s.totalBudget, s.medianBudget)
	}
	last := out.snaps[len(out.snaps)-1]
	fmt.Fprintf(&b, "\nfinal average budget / stadium capacity / socios by class|tier:\n")
	ck := make([]string, 0)
	for k := range last.byClassTier {
		ck = append(ck, k)
	}
	sort.Strings(ck)
	for _, k := range ck {
		v := last.byClassTier[k]
		fmt.Fprintf(&b, "  %-20s budget=%12.0f capacity=%8.0f socios=%6.0f\n", k, v[0], v[1], v[2])
	}
	return b.String()
}

func ratio(i, e int64) float64 {
	if e == 0 {
		return math.Inf(1)
	}
	return float64(i) / float64(e)
}

func topShare(l seasonLedger, keys []string, sign int64) string {
	type kv struct {
		k string
		v int64
	}
	var all []kv
	var total int64
	for _, k := range keys {
		v := l[k] * sign
		all = append(all, kv{k, v})
		total += v
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	if total <= 0 {
		return "-"
	}
	var parts []string
	for i := 0; i < 2 && i < len(all); i++ {
		parts = append(parts, fmt.Sprintf("%s %d%%", all[i].k, all[i].v*100/total))
	}
	return strings.Join(parts, ", ")
}

// ---------------------------------------------------------------- scenarios

// scenarios lists every variant the simulator can run.
func scenarios() []simScenario {
	noWageFactor := shippedScenario()
	noWageFactor.name = "no-wage-factor"
	noWageFactor.wageTierMult = map[int]float64{1: 1, 2: 1, 3: 1}

	// Variant C from the issue: a smaller factor plus a squad bonus per league win.
	bonus := shippedScenario()
	bonus.name = "smaller-factor+win-bonus"
	bonus.wageTierMult = map[int]float64{1: 3.3, 2: 1.8, 3: 1.05}
	bonus.winBonus = map[int]int64{1: 9000, 2: 3500, 3: 300}

	return []simScenario{shippedScenario(), noWageFactor, bonus}
}

func TestEconomySimulation(t *testing.T) {
	if os.Getenv("ECON_SIM") == "" {
		t.Skip("economy simulator: set ECON_SIM=1 to run (see economysim_test.go)")
	}
	seasons := 40
	if v := os.Getenv("ECON_SIM_SEASONS"); v != "" {
		seasons, _ = strconv.Atoi(v)
	}
	seed := int64(1)
	if v := os.Getenv("ECON_SIM_SEED"); v != "" {
		seed, _ = strconv.ParseInt(v, 10, 64)
	}
	wanted := map[string]bool{}
	if v := os.Getenv("ECON_SIM_SCENARIOS"); v != "" {
		for _, n := range strings.Split(v, ",") {
			wanted[strings.TrimSpace(n)] = true
		}
	}
	for _, sc := range scenarios() {
		if len(wanted) > 0 && !wanted[sc.name] {
			continue
		}
		fmt.Print(report(sc, runEconomySim(sc, seasons, seed), seasons))
	}
}
