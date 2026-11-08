package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "Selling contracts it will not renew": the most valuable player in his last
// season, never one whose sale thins his position or a protected one.
func TestPickUnrenewedToList(t *testing.T) {
	squad := squadOf(2, 6, 6, 5) // ids 1-2 GK, 3-8 D, 9-14 M, 15-19 S
	for _, p := range squad {
		p.Age, p.ContractYearsRemaining = 25, 3
	}
	squad[0].ContractYearsRemaining = 1 // a keeper: only 2 keepers, his sale would thin the position
	squad[9].ContractYearsRemaining = 1 // midfielder id 10
	squad[9].Age = 31
	squad[12].ContractYearsRemaining = 1 // midfielder id 13, weaker than id 10
	squad[12].Age = 31

	got := pickUnrenewedToList(squad, none(), none(), none())
	if got == nil || got.Id != 10 {
		t.Fatalf("listed %v, want the more valuable unrenewed midfielder 10", got)
	}
	if got := pickUnrenewedToList(squad, map[int64]bool{10: true}, none(), none()); got == nil || got.Id != 13 {
		t.Errorf("with 10 listed already, listed %v, want 13", got)
	}
	if got := pickUnrenewedToList(squad, none(), none(), map[int64]bool{10: true, 13: true}); got != nil {
		t.Errorf("protected players were listed: %v", got)
	}
	for _, p := range squad {
		p.ContractYearsRemaining = 2
	}
	if got := pickUnrenewedToList(squad, none(), none(), none()); got != nil {
		t.Errorf("nobody is in his last season, listed %v", got)
	}
}

// cpu-team-management "Director early renewals": "Rising star" and "Bench player with two
// seasons".
func TestContractsToRenewEarly(t *testing.T) {
	squad := squadOf(2, 6, 6, 5)
	for _, p := range squad {
		p.Age, p.ContractYearsRemaining = 26, 3
	}
	rising := squad[9] // a midfielder in the XI
	rising.Age, rising.ContractYearsRemaining = 21, 2
	bench := squad[13] // the weakest midfielder, not in the XI
	bench.ContractYearsRemaining = 2
	starters := map[int64]bool{rising.Id: true}

	got := contractsToRenewEarly(squad, starters)
	ids := map[int64]bool{}
	for _, p := range got {
		ids[p.Id] = true
	}
	if !ids[rising.Id] {
		t.Errorf("the 21-year-old starter with 2 seasons left was not renewed early: %v", ids)
	}
	if ids[bench.Id] {
		t.Errorf("a bench player with 2 seasons left was renewed early")
	}
	rising.ContractYearsRemaining = 1
	if got := contractsToRenewEarly(squad, starters); len(got) != 0 {
		t.Errorf("a player in his last season is the usual renewal's, not an early one: %v", got)
	}
}

// cpu-team-management "Answering pre-contract offers": "Manager keeps its starter", "Rival
// outbids the ceiling" and "Director key player".
func TestAcceptsCounterBar(t *testing.T) {
	cases := []struct {
		level        int
		starter, key bool
		expected     int
		want         bool
		scenario     string
	}{
		{common.AiLevelManager, true, false, 2800, true, "Manager starter at 1.4x"},
		{common.AiLevelManager, true, false, 3200, false, "Manager starter at 1.6x"},
		{common.AiLevelDirector, true, true, 3800, true, "Director key player at 1.9x"},
		{common.AiLevelManager, true, true, 3800, false, "a Manager's key player only gets the starter ceiling"},
		{common.AiLevelDirector, false, false, 2600, false, "bench player at 1.3x"},
		{common.AiLevelDirector, false, false, 2500, true, "bench player at 1.25x"},
	}
	for _, c := range cases {
		if got := acceptsCounterBar(c.level, c.starter, c.key, 2000, c.expected); got != c.want {
			t.Errorf("%s: got %v, want %v", c.scenario, got, c.want)
		}
	}
}

// cpu-transfer-market "Pre-contract targets": "Two signed already" and one offer per pass.
func TestPickNextTarget_PreContractLimits(t *testing.T) {
	state := &transferState{Squad: squadOf(2, 5, 6, 5), Listed: none(), LoanListed: none()}
	target := scoredCandidate{transferCandidate: transferCandidate{Kind: candidatePreContract}}
	if got := pickNextTarget([]scoredCandidate{target}, state, 0, 1); got == nil {
		t.Fatal("a pre-contract was not picked")
	}
	if got := pickNextTarget([]scoredCandidate{target}, state, 0, 0); got != nil {
		t.Error("a second pre-contract offer in the same pass was picked")
	}
	state.SignedPreContracts = common.AiMaxSignedPreContracts
	if got := pickNextTarget([]scoredCandidate{target}, state, 0, 1); got != nil {
		t.Error("a team with 2 signed pre-contracts made another offer")
	}
}

// cpu-team-management "Release clauses in CPU renewals": "Manager bench renewal" sets twice its
// valuation, never below the minimum.
func TestRenewalClauseAmount(t *testing.T) {
	p := common.Player{Id: 9, Age: 27, GoalKeeper: 60, Defense: 60, Lateral: 60, Pass: 60, Midfield: 60, Dribbling: 60, Shot: 60, ContractYearsRemaining: 1}
	profile := common.AiTransferProfileFor(common.AiLevelManager)
	clause := renewalClauseAmount(profile, p, nil, aiManagerClauseMultiple)
	if clause == nil || *clause < common.MinReleaseClause(p) {
		t.Fatalf("clause %v below the minimum %d", clause, common.MinReleaseClause(p))
	}
	base := common.ClauseBaseValue(p)
	if ratio := float64(*clause) / float64(base); ratio < 1.5 || ratio > 2.4 {
		t.Errorf("Manager clause %.2fx the base value, want about 2x (15%% noise)", ratio)
	}
	if again := renewalClauseAmount(profile, p, nil, aiManagerClauseMultiple); *again != *clause {
		t.Errorf("the same club set %d then %d for the same player", *clause, *again)
	}
}
