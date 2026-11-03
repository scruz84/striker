package engine

import (
	"math"
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// Lineup roles in the engine (openspec match-simulation "Penalty resolution", "Direct free-kick
// resolution" and "Captain's calming effect").

func id(v int64) *int64 { return &v }

// namedTestMatch is newSetPieceTestMatch with names for every outfielder, so any of them can be
// recorded as the taker.
func namedTestMatch() (*common.MatchSimulation, *common.DatabaseContext) {
	match, dbCtx := newSetPieceTestMatch()
	cache := dbCtx.ObjectCache.(*setPieceTestCache)
	cache.values[PlayerNameKey+"2"] = "Home Defender"
	cache.values[PlayerNameKey+"12"] = "Away Defender"
	return match, dbCtx
}

func withHomeRoles(match *common.MatchSimulation, roles common.LineupRoles) {
	match.HomeLineup = &common.Lineup{TeamId: match.Home, LineupRoles: roles}
}

// Named taker on the pitch: the passer (Shot 60) takes it over the striker (Shot 80), scored with
// the probability for Shot 60.
func TestProcessSetPiece_NamedPenaltyTakerOnThePitch(t *testing.T) {
	match, dbCtx := namedTestMatch()
	withHomeRoles(match, common.LineupRoles{PenaltyTakerId: id(4)})
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}
	const n = 4000
	for i := 0; i < n; i++ {
		processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_PENALTY)
	}
	goals := 0
	for _, e := range *match.Events {
		if e.TeamOwnerPlayers[0] != 4 {
			t.Fatalf("penalty taken by %d, want the named taker 4", e.TeamOwnerPlayers[0])
		}
		if e.EventType == common.GOAL {
			goals++
		}
	}
	want := float64(penaltyConversionProbability(60, 50))
	if got := float64(goals) / n; math.Abs(got-want) > 0.025 {
		t.Errorf("conversion %.3f, want about %.3f (Shot 60 vs GoalKeeper 50)", got, want)
	}
}

// Named taker on the bench: the automatic rule (best Shot) picks the striker.
func TestProcessSetPiece_NamedPenaltyTakerOnTheBench(t *testing.T) {
	match, dbCtx := namedTestMatch()
	withHomeRoles(match, common.LineupRoles{PenaltyTakerId: id(99)})
	processSetPiece(ScorerSimulation{r: rand.New(rand.NewSource(1))}, dbCtx, match, match.Home, common.SET_PIECE_PENALTY)
	if got := (*match.Events)[0].TeamOwnerPlayers[0]; got != 3 {
		t.Fatalf("penalty taken by %d, want the automatic pick 3", got)
	}
}

// Named free-kick taker substituted off: once he has left the pitch the blend picks the passer (4).
func TestProcessSetPiece_NamedFreeKickTakerSubstitutedOff(t *testing.T) {
	match, dbCtx := namedTestMatch()
	withHomeRoles(match, common.LineupRoles{FreeKickTakerId: id(2)})
	s := ScorerSimulation{r: rand.New(rand.NewSource(1))}
	processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_DIRECT_FREE_KICK)
	if got := (*match.Events)[0].TeamOwnerPlayers[0]; got != 2 {
		t.Fatalf("free kick taken by %d before the substitution, want the named taker 2", got)
	}
	removePlayerFromPosition(match.HomePlayersByPosition, common.DEFENSE, 2)
	processSetPiece(s, dbCtx, match, match.Home, common.SET_PIECE_DIRECT_FREE_KICK)
	if got := (*match.Events)[1].TeamOwnerPlayers[0]; got != 4 {
		t.Fatalf("free kick taken by %d after the substitution, want the automatic pick 4", got)
	}
}

// The other side's roles never pick this side's taker.
func TestProcessSetPiece_AwayUsesItsOwnRoles(t *testing.T) {
	match, dbCtx := namedTestMatch()
	withHomeRoles(match, common.LineupRoles{PenaltyTakerId: id(4)})
	match.AwayLineup = &common.Lineup{TeamId: match.Away, LineupRoles: common.LineupRoles{PenaltyTakerId: id(12)}}
	processSetPiece(ScorerSimulation{r: rand.New(rand.NewSource(1))}, dbCtx, match, match.Away, common.SET_PIECE_PENALTY)
	if got := (*match.Events)[0].TeamOwnerPlayers[0]; got != 12 {
		t.Fatalf("away penalty taken by %d, want its named taker 12", got)
	}
}

func cancelRate(roles common.LineupRoles, byPosition map[int][]int64, player int64) float64 {
	r := rand.New(rand.NewSource(7))
	const n = 20000
	cancelled := 0
	for i := 0; i < n; i++ {
		if captainCancelsCard(r, roles, &byPosition, player) {
			cancelled++
		}
	}
	return float64(cancelled) / n
}

// Captain on the pitch: a team-mate's card is cancelled about 10% of the time.
func TestCaptainCancelsCard_CaptainOnThePitch(t *testing.T) {
	pitch := map[int][]int64{common.GOALKEEPER: {1}, common.DEFENSE: {2}, common.MIDFIELD: {4}}
	got := cancelRate(common.LineupRoles{CaptainId: id(2)}, pitch, 4)
	if math.Abs(got-common.CaptainCardCancelProbability) > 0.01 {
		t.Errorf("cancel rate %.3f, want about %.2f", got, common.CaptainCardCancelProbability)
	}
}

// Captain booked: his own card is never cancelled.
func TestCaptainCancelsCard_CaptainBooked(t *testing.T) {
	pitch := map[int][]int64{common.GOALKEEPER: {1}, common.DEFENSE: {2}, common.MIDFIELD: {4}}
	if got := cancelRate(common.LineupRoles{CaptainId: id(2)}, pitch, 2); got != 0 {
		t.Errorf("captain's own card cancelled at rate %.3f, want never", got)
	}
}

// Captain sent off (no longer on the pitch), or no captain at all: nothing is cancelled and no
// random number is drawn, so a side without a captain is booked exactly as before.
func TestCaptainCancelsCard_CaptainSentOffOrNone(t *testing.T) {
	pitch := map[int][]int64{common.GOALKEEPER: {1}, common.MIDFIELD: {4}}
	for name, roles := range map[string]common.LineupRoles{"sent off": {CaptainId: id(2)}, "no captain": {}} {
		r := rand.New(rand.NewSource(7))
		before := r.Int63()
		r = rand.New(rand.NewSource(7))
		if captainCancelsCard(r, roles, &pitch, 4) {
			t.Errorf("%s: card cancelled", name)
		}
		if after := r.Int63(); after != before {
			t.Errorf("%s: a random number was drawn", name)
		}
	}
}

// A goalkeeper captain on the pitch steadies his team-mates too.
func TestCaptainCancelsCard_GoalkeeperCaptain(t *testing.T) {
	pitch := map[int][]int64{common.GOALKEEPER: {1}, common.MIDFIELD: {4}}
	if got := cancelRate(common.LineupRoles{CaptainId: id(1)}, pitch, 4); got == 0 {
		t.Error("a goalkeeper captain never cancelled a card")
	}
}
