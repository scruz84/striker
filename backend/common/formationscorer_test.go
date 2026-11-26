package common

import (
	"math/rand"
	"testing"
)

func withFormationScorer(t *testing.T, scorer FormationScorer) {
	t.Helper()
	previous := formationScorer
	SetFormationScorer(scorer)
	t.Cleanup(func() { SetFormationScorer(previous) })
}

func scorerTestSquad() []*Player {
	rng := rand.New(rand.NewSource(4))
	squad := make([]*Player, 0, 18)
	r := func() int { return 30 + rng.Intn(31) }
	for id := int64(1); id <= 18; id++ {
		squad = append(squad, &Player{Id: id, GoalKeeper: r(), Defense: r(), Lateral: r(), Pass: r(), Midfield: r(), Dribbling: r(), Shot: r()})
	}
	return squad
}

// openspec cpu-team-management "Manager's shape": with a scorer registered, a Manager plays the
// formation the scorer rates highest against the squad's own 4-4-2 XI, and is handed that 4-4-2 XI
// as the reference.
func TestSelectBestFormationAndLineup_RanksByTheRegisteredScore(t *testing.T) {
	squad := scorerTestSquad()
	wantReference := SelectBestLineupForFormation(squad, FormationBalanced)
	withFormationScorer(t, func(s []*Player, lineup *Lineup, reference *Lineup) float64 {
		if reference.Formation != FormationBalanced || len(reference.Positions) != 11 {
			t.Errorf("reference should be the squad's best 4-4-2 XI, got %+v", reference)
		}
		for _, p := range reference.Positions {
			found := false
			for _, id := range wantReference[p.Position] {
				found = found || id == p.PlayerId
			}
			if !found {
				t.Errorf("reference player %d at %d is not in the best 4-4-2 XI", p.PlayerId, p.Position)
			}
		}
		if lineup.Formation == FormationAttacking {
			return 0.3
		}
		return 0
	})
	formation, _ := SelectBestFormationAndLineup(squad, AiLevelManager, rand.New(rand.NewSource(1)))
	if formation != FormationAttacking {
		t.Errorf("Manager plays %d, want 3-4-3, the best-scored shape", formation)
	}
}

// openspec cpu-team-management "Ranking ties": equal scores go to the stronger XI by summed slot
// value, and with no scorer registered the ranking is by summed slot value alone.
func TestSelectBestFormationAndLineup_TiesAndNoScorer(t *testing.T) {
	squad := scorerTestSquad()
	byId := map[int64]*Player{}
	for _, p := range squad {
		byId[p.Id] = p
	}
	strongest, best := -1, -1
	for _, f := range AllFormations {
		if s := lineupStrength(SelectBestLineupForFormation(squad, f), byId); s > best {
			strongest, best = f, s
		}
	}

	withFormationScorer(t, func([]*Player, *Lineup, *Lineup) float64 { return 0.1 })
	if formation, _ := SelectBestFormationAndLineup(squad, AiLevelManager, rand.New(rand.NewSource(1))); formation != strongest {
		t.Errorf("with every score tied, Manager plays %d, want the strongest XI's %d", formation, strongest)
	}

	SetFormationScorer(nil)
	if formation, _ := SelectBestFormationAndLineup(squad, AiLevelManager, rand.New(rand.NewSource(1))); formation != strongest {
		t.Errorf("with no scorer, Manager plays %d, want the strongest XI's %d", formation, strongest)
	}
}
