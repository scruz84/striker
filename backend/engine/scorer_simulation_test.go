package engine

import (
	"github.com/scruz84/striker/common"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGetPlayerForChance_ReturnsSentinelWhenNoOutfieldPlayersLeft(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
	}

	if player := getPlayerForChance(r, &positions); player != -1 {
		t.Fatalf("expected -1 when striker/midfield/defense are all empty, got %d", player)
	}
}

func TestGetPlayerForChance_FallsBackToDefenseWhenAttackIsEmpty(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.DEFENSE: {42},
	}

	for i := 0; i < 20; i++ {
		if player := getPlayerForChance(r, &positions); player != 42 {
			t.Fatalf("expected the only remaining defender (42), got %d", player)
		}
	}
}

func TestChanceSuccessProbability_EvenMatchupReturnsBaseline(t *testing.T) {
	got := chanceSuccessProbability(100, 100)
	if math.Abs(float64(got-ConversionBaseline)) > 0.001 {
		t.Fatalf("expected an even matchup to land on the baseline %.2f, got %.4f", ConversionBaseline, got)
	}
}

func TestChanceSuccessProbability_BoundedForExtremeMismatch(t *testing.T) {
	dominant := chanceSuccessProbability(100000, 1)
	if dominant > ConversionMax {
		t.Fatalf("expected a dominant attack to be capped at %.2f, got %.4f", ConversionMax, dominant)
	}
	overwhelmed := chanceSuccessProbability(1, 100000)
	if overwhelmed < ConversionMin {
		t.Fatalf("expected an overwhelmed attack to be floored at %.2f, got %.4f", ConversionMin, overwhelmed)
	}
}

func TestChanceSuccessProbability_StrongerAttackScoresMoreOften(t *testing.T) {
	weak := chanceSuccessProbability(40, 60)
	strong := chanceSuccessProbability(60, 40)
	if !(weak < ConversionBaseline && strong > ConversionBaseline && weak < strong) {
		t.Fatalf("expected weak < baseline < strong, got weak=%.4f baseline=%.2f strong=%.4f", weak, ConversionBaseline, strong)
	}
}

func TestHomeChanceShare_EqualMidfieldIsFiftyFifty(t *testing.T) {
	match := &common.MatchSimulation{HomeQua: &common.TeamQualification{Midfield: 50}, AwayQua: &common.TeamQualification{Midfield: 50}}
	if got := homeChanceShare(match); math.Abs(float64(got-0.5)) > 0.001 {
		t.Fatalf("expected 0.5 for equal midfields, got %.4f", got)
	}
}

func TestHomeChanceShare_BoundedForExtremeDominance(t *testing.T) {
	match := &common.MatchSimulation{HomeQua: &common.TeamQualification{Midfield: 100}, AwayQua: &common.TeamQualification{Midfield: 0.0001}}
	if got := homeChanceShare(match); got > PossessionMaxShare {
		t.Fatalf("expected home share to be capped at %.2f, got %.4f", PossessionMaxShare, got)
	}
}

func TestFlankMidfieldSupport_EvenMidfieldBattleIsNeutral(t *testing.T) {
	if got := flankMidfieldSupport(50, 50); got != 0 {
		t.Fatalf("expected an even flank midfield battle to add nothing, got %v", got)
	}
}

func TestFlankMidfieldSupport_WinningTheFlankAddsAModestBoost(t *testing.T) {
	got := flankMidfieldSupport(70, 50)
	want := float32(FlankMidfieldSupportWeight) * 20
	if math.Abs(float64(got-want)) > 0.001 {
		t.Fatalf("expected a %v-weighted boost of %v, got %v", FlankMidfieldSupportWeight, want, got)
	}
	if got <= 0 {
		t.Fatalf("expected winning the flank's midfield battle to boost attack, got %v", got)
	}
}

func TestFlankMidfieldSupport_LosingTheFlankIsAPenalty(t *testing.T) {
	if got := flankMidfieldSupport(40, 60); got >= 0 {
		t.Fatalf("expected losing the flank's midfield battle to penalize attack, got %v", got)
	}
}

func TestChanceQualityMultiplier_OnlyEverReturnsTheTwoTiers(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		got := chanceQualityMultiplier(r, 60, 40)
		if got != ClearCutQualityMultiplier && got != HalfChanceQualityMultiplier {
			t.Fatalf("expected either the clear-cut or half-chance multiplier, got %.2f", got)
		}
	}
}

func TestClassifyChance_SharesMatchTheConstants(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	const rolls = 1000000
	counts := map[int]int{}
	for i := 0; i < rolls; i++ {
		counts[classifyChance(r)]++
	}
	penalty := float64(counts[common.SET_PIECE_PENALTY]) / rolls
	freeKick := float64(counts[common.SET_PIECE_DIRECT_FREE_KICK]) / rolls
	if math.Abs(penalty-ProbPenaltyPerChance) > 0.001 {
		t.Fatalf("expected penalty share %.3f ± 0.001, got %.4f", ProbPenaltyPerChance, penalty)
	}
	if math.Abs(freeKick-ProbDirectFreeKickPerChance) > 0.0015 {
		t.Fatalf("expected free-kick share %.3f ± 0.0015, got %.4f", ProbDirectFreeKickPerChance, freeKick)
	}
}

func TestBestOnPitchOutfielder_PicksHighestScoreExcludingGoalkeeper(t *testing.T) {
	players := map[int64]common.Player{
		1: {Shot: 99}, // goalkeeper: never a taker
		2: {Shot: 40},
		3: {Shot: 70},
		4: {Shot: 55},
	}
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
		common.DEFENSE:    {2},
		common.MIDFIELD:   {3},
		common.STRIKER:    {4},
	}
	byShot := func(p common.Player) float32 { return float32(p.Shot) }

	if got := bestOnPitchOutfielder(&players, &positions, byShot); got != 3 {
		t.Fatalf("expected the highest-Shot outfielder (3), got %d", got)
	}
}

func TestBestOnPitchOutfielder_TieGoesToLowestId(t *testing.T) {
	players := map[int64]common.Player{5: {Shot: 60}, 3: {Shot: 60}, 9: {Shot: 60}}
	positions := map[int][]int64{common.MIDFIELD: {5, 9}, common.STRIKER: {3}}

	for i := 0; i < 20; i++ { // map iteration order is random; the result must not depend on it
		got := bestOnPitchOutfielder(&players, &positions, func(p common.Player) float32 { return float32(p.Shot) })
		if got != 3 {
			t.Fatalf("expected the lowest id (3) on a tie, got %d", got)
		}
	}
}

func TestBestOnPitchOutfielder_SkipsPlayersRemovedFromThePitch(t *testing.T) {
	players := map[int64]common.Player{2: {Shot: 90}, 3: {Shot: 50}}
	// player 2 (best finisher) was sent off, so he's no longer in the by-position map
	positions := map[int][]int64{common.STRIKER: {3}}

	if got := bestOnPitchOutfielder(&players, &positions, func(p common.Player) float32 { return float32(p.Shot) }); got != 3 {
		t.Fatalf("expected the best remaining player (3), got %d", got)
	}
}

func TestBestOnPitchOutfielder_ReturnsSentinelWhenOnlyTheGoalkeeperIsLeft(t *testing.T) {
	players := map[int64]common.Player{1: {Shot: 90}}
	positions := map[int][]int64{common.GOALKEEPER: {1}}

	if got := bestOnPitchOutfielder(&players, &positions, func(p common.Player) float32 { return float32(p.Shot) }); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
}

func TestPenaltyConversionProbability(t *testing.T) {
	cases := []struct {
		name         string
		shot, keeper int
		want         float64
	}{
		{"even taker and keeper", 60, 60, 0.78},
		{"elite taker against weak keeper is capped", 80, 30, 0.92},
		{"weak taker against elite keeper is floored", 25, 85, 0.60},
		{"modest edge stays on the line", 70, 50, 0.78 + 0.30*20.0/99},
	}
	for _, c := range cases {
		if got := float64(penaltyConversionProbability(c.shot, c.keeper)); math.Abs(got-c.want) > 1e-4 {
			t.Errorf("%s: expected %.4f, got %.4f", c.name, c.want, got)
		}
	}
}

func TestFreeKickConversionProbability(t *testing.T) {
	cases := []struct {
		name          string
		value, keeper float32
		want          float64
	}{
		{"even taker and keeper", 50, 50, 0.06},
		{"specialist taker", 0.6*75 + 0.4*70, 40, 0.06 + 0.10*33.0/99},
		{"capped at the top", 99, 0, 0.15},
		{"floored at the bottom", 0, 99, 0.02},
	}
	for _, c := range cases {
		if got := float64(freeKickConversionProbability(c.value, int(c.keeper))); math.Abs(got-c.want) > 1e-4 {
			t.Errorf("%s: expected %.4f, got %.4f", c.name, c.want, got)
		}
	}
}

// setPieceTestCache is a map-backed common.Cache so processSetPiece can resolve player names
// without Redis.
type setPieceTestCache struct{ values map[string]string }

func (c *setPieceTestCache) GetValue(key string) (string, error) { return c.values[key], nil }
func (c *setPieceTestCache) SetValue(key string, value string)   { c.values[key] = value }
func (c *setPieceTestCache) SetValueLongDuration(key string, value string) {
	c.values[key] = value
}
func (c *setPieceTestCache) SetValueNoExpiry(key string, value string) { c.values[key] = value }
func (c *setPieceTestCache) SetValueFor(key string, value string, _ time.Duration) {
	c.values[key] = value
}
func (c *setPieceTestCache) DeleteValue(key string) { delete(c.values, key) }

func newSetPieceTestMatch() (*common.MatchSimulation, *common.DatabaseContext) {
	home := map[int64]common.Player{
		1: {GoalKeeper: 50}, 2: {Shot: 40, Pass: 40}, 3: {Shot: 80, Pass: 50}, 4: {Shot: 60, Pass: 90},
	}
	away := map[int64]common.Player{
		11: {GoalKeeper: 50}, 12: {Shot: 30, Pass: 30}, 13: {Shot: 90, Pass: 90},
	}
	homePos := map[int][]int64{common.GOALKEEPER: {1}, common.DEFENSE: {2}, common.STRIKER: {3}, common.MIDFIELD: {4}}
	awayPos := map[int][]int64{common.GOALKEEPER: {11}, common.DEFENSE: {12}, common.STRIKER: {13}}
	events := []common.Event{}
	match := &common.MatchSimulation{
		Home: 100, Away: 200, Minute: 33,
		HomePlayers: &home, AwayPlayers: &away,
		EffectiveHomePlayers: &home, EffectiveAwayPlayers: &away,
		HomePlayersByPosition: &homePos, AwayPlayersByPosition: &awayPos,
		Events: &events,
	}
	cache := &setPieceTestCache{values: map[string]string{
		PlayerNameKey + "3": "Home Striker", PlayerNameKey + "4": "Home Passer", PlayerNameKey + "11": "Away Keeper",
		PlayerNameKey + "1": "Home Keeper", PlayerNameKey + "13": "Away Striker",
		PlayerNameKey + "2": "Home Defender", PlayerNameKey + "12": "Away Defender",
	}}
	return match, &common.DatabaseContext{ObjectCache: cache}
}

func TestProcessSetPiece_PenaltyUsesBestShotAndRecordsTheKeeper(t *testing.T) {
	match, dbCtx := newSetPieceTestMatch()
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}
	for i := 0; i < 50; i++ {
		processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_PENALTY)
	}

	if len(*match.Events) != 50 {
		t.Fatalf("expected 50 events, got %d", len(*match.Events))
	}
	goals := 0
	for _, e := range *match.Events {
		if e.SetPiece != common.SET_PIECE_PENALTY || len(e.TeamOwnerPlayers) != 1 || e.TeamOwnerPlayers[0] != 3 ||
			len(e.OtherTeamPlayers) != 1 || e.OtherTeamPlayers[0] != 11 || e.OtherTeamPlayersNames[0] != "Away Keeper" ||
			e.TeamOwner != match.Home || e.OtherTeam != match.Away || e.Minute != 33 {
			t.Fatalf("unexpected penalty event %+v", e)
		}
		if e.EventType == common.GOAL {
			goals++
		}
	}
	if goals != match.HomeGoals || match.AwayGoals != 0 {
		t.Fatalf("score must follow the GOAL events: %d goals vs score %d-%d", goals, match.HomeGoals, match.AwayGoals)
	}
	if goals < 25 { // 0.78 + 0.30*30/99 ≈ 0.87 per penalty
		t.Fatalf("expected most of 50 penalties to be scored, got %d", goals)
	}
}

func TestProcessSetPiece_FreeKickUsesShotAndPassBlend(t *testing.T) {
	match, dbCtx := newSetPieceTestMatch()
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}
	processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_DIRECT_FREE_KICK)

	e := (*match.Events)[0]
	// player 4: 0.6*60 + 0.4*90 = 72 beats player 3: 0.6*80 + 0.4*50 = 68
	if e.SetPiece != common.SET_PIECE_DIRECT_FREE_KICK || e.TeamOwnerPlayers[0] != 4 || e.OtherTeamPlayers[0] != 11 {
		t.Fatalf("unexpected free-kick event %+v", e)
	}
}

func TestProcessSetPiece_AwaySideAttacksTheHomeKeeper(t *testing.T) {
	match, dbCtx := newSetPieceTestMatch()
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}
	processSetPiece(s, dbCtx, match, match.Away, common.SET_PIECE_PENALTY)

	e := (*match.Events)[0]
	if e.TeamOwner != match.Away || e.TeamOwnerPlayers[0] != 13 || e.OtherTeam != match.Home || e.OtherTeamPlayers[0] != 1 {
		t.Fatalf("unexpected away penalty event %+v", e)
	}
}

func TestProcessSetPiece_DiscardedWhenNoOutfielderIsLeft(t *testing.T) {
	match, dbCtx := newSetPieceTestMatch()
	onlyKeeper := map[int][]int64{common.GOALKEEPER: {1}}
	match.HomePlayersByPosition = &onlyKeeper
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}

	processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_PENALTY)
	processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_DIRECT_FREE_KICK)

	if len(*match.Events) != 0 || match.HomeGoals != 0 {
		t.Fatalf("expected the set pieces to be discarded, got %d events and %d goals", len(*match.Events), match.HomeGoals)
	}
}

// TestSetPieceCalibration_GoalsPerChanceStayPutInAnEvenMatchup checks the set pieces carved out of
// the open-play chances don't change the goals a match produces: in an even matchup (equal attack
// and defense, taker Shot = keeper GoalKeeper) the blended goals per chance stay within 0.004 of
// open play alone, i.e. ≈ ±0.05 goals per match at ~12.6 chances.
func TestSetPieceCalibration_GoalsPerChanceStayPutInAnEvenMatchup(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	const chances = 200000
	const chancesPerMatch = 12.6
	openPlayGoals, blendedGoals, penalties := 0.0, 0.0, 0
	for i := 0; i < chances; i++ {
		// open-play expectation for this chance (even attack/defense of 100)
		quality := chanceQualityMultiplier(r, 100, 100)
		openProb := float64(chanceSuccessProbability(100*quality, 100))
		openPlayGoals += openProb
		switch classifyChance(r) {
		case common.SET_PIECE_PENALTY:
			penalties++
			blendedGoals += float64(penaltyConversionProbability(60, 60))
		case common.SET_PIECE_DIRECT_FREE_KICK:
			blendedGoals += float64(freeKickConversionProbability(60, 60))
		default:
			blendedGoals += openProb
		}
	}
	openPerChance := openPlayGoals / chances
	blendedPerChance := blendedGoals / chances
	if math.Abs(blendedPerChance-openPerChance) > 0.004 {
		t.Fatalf("goals per chance moved from %.4f to %.4f (limit ±0.004)", openPerChance, blendedPerChance)
	}
	penaltiesPerMatch := float64(penalties) / chances * chancesPerMatch
	if math.Abs(penaltiesPerMatch-0.28) > 0.02 {
		t.Fatalf("expected 0.28 ± 0.02 penalties per match, got %.3f", penaltiesPerMatch)
	}
}

// openPlayEvents runs processChance until it has n open-play events for side (set pieces are
// skipped - processSetPiece has its own tests) and returns them.
func openPlayEvents(t *testing.T, side func(*common.MatchSimulation) int64, n int) ([]common.Event, *common.MatchSimulation) {
	t.Helper()
	match, dbCtx := newSetPieceTestMatch()
	match.HomeQua = &common.TeamQualification{Attack: 50, Defense: 50}
	match.AwayQua = &common.TeamQualification{Attack: 50, Defense: 50}
	s := ScorerSimulation{r: rand.New(rand.NewSource(7))}
	var open []common.Event
	for i := 0; i < 1000 && len(open) < n; i++ {
		*match.Events = (*match.Events)[:0]
		processChance(s, dbCtx, match, side(match))
		for _, e := range *match.Events {
			if e.SetPiece == common.SET_PIECE_NONE {
				open = append(open, e)
			}
		}
	}
	if len(open) < n {
		t.Fatalf("expected %d open-play events, got %d", n, len(open))
	}
	return open, match
}

func TestProcessChance_OpenPlayRecordsTheAwayKeeper(t *testing.T) {
	events, _ := openPlayEvents(t, func(m *common.MatchSimulation) int64 { return m.Home }, 20)
	for _, e := range events {
		if len(e.OtherTeamPlayers) != 1 || e.OtherTeamPlayers[0] != 11 ||
			len(e.OtherTeamPlayersNames) != 1 || e.OtherTeamPlayersNames[0] != "Away Keeper" {
			t.Fatalf("home open-play chance must name the away keeper, got %+v", e)
		}
	}
}

func TestProcessChance_OpenPlayRecordsTheHomeKeeper(t *testing.T) {
	events, _ := openPlayEvents(t, func(m *common.MatchSimulation) int64 { return m.Away }, 20)
	for _, e := range events {
		if len(e.OtherTeamPlayers) != 1 || e.OtherTeamPlayers[0] != 1 || e.OtherTeamPlayersNames[0] != "Home Keeper" {
			t.Fatalf("away open-play chance must name the home keeper, got %+v", e)
		}
	}
}
