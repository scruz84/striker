package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// contributionsTestMatch builds one minute of a 4-4-2 (home, ids 1-11) against a 4-3-3 (away, ids
// 101-111), with every skill at 40 unless overridden - the figures the match-ratings spec's "Zone
// contribution per minute" scenarios use.
func contributionsTestMatch() *common.MatchSimulation {
	home := map[int][]int64{
		common.GOALKEEPER:             {1},
		common.LATERAL_DEFENSE_LEFT:   {2},
		common.DEFENSE:                {3, 4},
		common.LATERAL_DEFENSE_RIGHT:  {5},
		common.LATERAL_MIDFIELD_LEFT:  {6},
		common.MIDFIELD:               {7, 8},
		common.LATERAL_MIDFIELD_RIGHT: {9},
		common.STRIKER:                {10, 11},
	}
	away := map[int][]int64{
		common.GOALKEEPER:            {101},
		common.LATERAL_DEFENSE_LEFT:  {102},
		common.DEFENSE:               {103, 104},
		common.LATERAL_DEFENSE_RIGHT: {105},
		common.MIDFIELD:              {106, 107, 108},
		common.LATERAL_STRIKER_LEFT:  {109},
		common.STRIKER:               {110},
		common.LATERAL_STRIKER_RIGHT: {111},
	}
	players := func(byPos map[int][]int64) *map[int64]common.Player {
		out := make(map[int64]common.Player)
		for _, ids := range byPos {
			for _, id := range ids {
				out[id] = common.Player{Id: id, GoalKeeper: 40, Defense: 40, Lateral: 40, Pass: 40, Midfield: 40, Dribbling: 40, Shot: 40}
			}
		}
		return &out
	}
	homePlayers, awayPlayers := players(home), players(away)
	set := func(m *map[int64]common.Player, id int64, f func(p *common.Player)) {
		p := (*m)[id]
		f(&p)
		(*m)[id] = p
	}
	set(homePlayers, 1, func(p *common.Player) { p.GoalKeeper = 62 })
	set(homePlayers, 2, func(p *common.Player) { p.Lateral, p.Defense, p.Midfield = 50, 50, 58 })
	set(homePlayers, 10, func(p *common.Player) { p.Shot = 40 })
	set(homePlayers, 11, func(p *common.Player) { p.Shot = 50 })
	set(awayPlayers, 103, func(p *common.Player) { p.Defense = 58 })
	set(awayPlayers, 104, func(p *common.Player) { p.Defense = 58 })
	set(awayPlayers, 105, func(p *common.Player) { p.Lateral, p.Midfield = 48, 48 })
	set(awayPlayers, 109, func(p *common.Player) { p.Lateral, p.Shot = 50, 50 })
	set(awayPlayers, 110, func(p *common.Player) { p.Shot = 70 })
	set(awayPlayers, 111, func(p *common.Player) { p.Lateral, p.Shot = 40, 40 })

	return &common.MatchSimulation{
		HomePlayersByPosition: &home,
		AwayPlayersByPosition: &away,
		EffectiveHomePlayers:  homePlayers,
		EffectiveAwayPlayers:  awayPlayers,
	}
}

func assertContribution(t *testing.T, got *common.InstantContributions, id int64, want common.PlayerContribution) {
	t.Helper()
	side := got.Home
	if id > 100 {
		side = got.Away
	}
	c, ok := side[id]
	if !ok {
		t.Fatalf("player %d missing from contributions", id)
	}
	if c != want {
		t.Fatalf("player %d: got %+v, want %+v", id, c, want)
	}
}

func TestPlayerContributions_CentralDefenderAgainstPerStrikerShot(t *testing.T) {
	got := playerContributions(contributionsTestMatch())
	// Away central defender (Defense 58) against the home strikers' Shot 40 + 50 = 90 over two.
	assertContribution(t, got, 103, common.PlayerContribution{Position: common.DEFENSE, Own: 58, Rival: 45})
	// Home striker (Shot 50) against the away central defenders' 58 + 58 over two.
	assertContribution(t, got, 11, common.PlayerContribution{Position: common.STRIKER, Own: 50, Rival: 58})
}

func TestPlayerContributions_FullBackFeedsTwoMirroredDuels(t *testing.T) {
	got := playerContributions(contributionsTestMatch())
	// Home left back: flank defense (50+50)/2 = 50 against the away RIGHT wide forward's
	// (40+40)/2 = 40, flank midfield (50+58)/2 = 54 against the away right back's (48+48)/2 = 48.
	assertContribution(t, got, 2, common.PlayerContribution{Position: common.LATERAL_DEFENSE_LEFT, Own: 52, Rival: 44})
}

func TestPlayerContributions_GoalkeeperAgainstMeanAttacker(t *testing.T) {
	got := playerContributions(contributionsTestMatch())
	// Away attackers: central Shot 50, 70, 40 plus flank values 50 (left) and 40 (right) -> 50.
	assertContribution(t, got, 1, common.PlayerContribution{Position: common.GOALKEEPER, Own: 62, Rival: 50})
}

func TestPlayerContributions_LateralStrikerMeansFlankAndCentralDuels(t *testing.T) {
	got := playerContributions(contributionsTestMatch())
	// Away left wide forward: flank attack 50 against the home RIGHT back's (40+40)/2 = 40, and
	// central Shot 50 against the home central defenders' 40 -> 50 vs 40.
	assertContribution(t, got, 109, common.PlayerContribution{Position: common.LATERAL_STRIKER_LEFT, Own: 50, Rival: 40})
}

func TestPlayerContributions_EmptyRivalFlankReadsZero(t *testing.T) {
	match := contributionsTestMatch()
	// The away right wide forward is sent off: nobody attacks the home left back's flank.
	delete(*match.AwayPlayersByPosition, common.LATERAL_STRIKER_RIGHT)
	got := playerContributions(match)
	assertContribution(t, got, 2, common.PlayerContribution{Position: common.LATERAL_DEFENSE_LEFT, Own: 52, Rival: 24})
	if _, ok := got.Away[111]; ok {
		t.Fatal("a player off the pitch must not be listed")
	}
}

func TestPlayerContributions_ListsEveryPlayerOnThePitch(t *testing.T) {
	got := playerContributions(contributionsTestMatch())
	if len(got.Home) != 11 || len(got.Away) != 11 {
		t.Fatalf("got %d home and %d away contributions, want 11 each", len(got.Home), len(got.Away))
	}
	// A lateral midfielder in a 4-4-2 feeds his flank's attack, against the away RIGHT back's
	// flank defense (48+40)/2 = 44.
	assertContribution(t, got, 6, common.PlayerContribution{Position: common.LATERAL_MIDFIELD_LEFT, Own: 40, Rival: 44})
}
