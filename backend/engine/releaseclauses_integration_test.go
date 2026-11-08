package engine

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func (w *preContractWorld) strikerPlayer() common.Player {
	w.t.Helper()
	p, err := common.GetPlayerById(w.dbCtx, w.striker)
	if err != nil {
		w.t.Fatal(err)
	}
	return *p
}

// release-clauses "Setting a clause" ("Clause too low", "Renewal without a clause") and "Wage
// discount for a clause" ("Low clause").
func TestReleaseClauses_SetDiscountAndClear(t *testing.T) {
	w := newPreContractWorld(t, "ra")
	p := w.strikerPlayer()
	low := common.MinReleaseClause(p) - 1
	if _, err := common.MakeContractRenewalOfferWithClause(w.dbCtx, w.striker, w.home, w.skillWage(), 3, &low); !errors.Is(err, common.ErrReleaseClauseTooLow) {
		t.Fatalf("clause below the minimum: %v, want ErrReleaseClauseTooLow", err)
	}

	clause := 2 * common.ClauseBaseValue(p)
	discounted := common.DiscountedWage(p, w.skillWage(), &clause)
	offer, err := common.MakeContractRenewalOfferWithClause(w.dbCtx, w.striker, w.home, discounted, 3, &clause)
	if err != nil || !offer.Accepted {
		t.Fatalf("renewal with a 2x clause at %d: %+v, %v", discounted, offer, err)
	}
	if got := w.strikerPlayer(); got.ReleaseClause == nil || *got.ReleaseClause != clause || got.ReleaseClausePayableAt == nil {
		t.Fatalf("clause after the renewal %v, want %d with a payable date", got.ReleaseClause, clause)
	}

	if offer, err := common.MakeContractRenewalOffer(w.dbCtx, w.striker, w.home, w.skillWage()*2, 3); err != nil || !offer.Accepted {
		t.Fatalf("renewal without a clause: %+v, %v", offer, err)
	}
	if got := w.strikerPlayer(); got.ReleaseClause != nil {
		t.Errorf("a renewal without a clause kept %d", *got.ReleaseClause)
	}
}

// release-clauses "Paying a clause": "Too soon", then "Clause paid" by exactly one of two clubs
// paying at once; "Clause visibility and notifications": "Human loses a player".
func TestReleaseClauses_Paying(t *testing.T) {
	w := newPreContractWorld(t, "rb")
	p := w.strikerPlayer()
	clause := common.MinReleaseClause(p)
	w.exec(`UPDATE players SET contract_years_remaining = 3, release_clause = $2, release_clause_set_at = now() - interval '10 days' WHERE player_id = $1`, w.striker, clause)
	wage, years, err := common.PlayerTermsExpectation(w.dbCtx, w.strikerPlayer(), common.TermsTransfer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := common.PayReleaseClause(w.dbCtx, w.striker, w.buyer, wage, years, nil); !errors.Is(err, common.ErrReleaseClauseLockedIn) {
		t.Fatalf("paying a 10-day-old clause: %v, want ErrReleaseClauseLockedIn", err)
	}
	if r, err := common.PayReleaseClause(w.dbCtx, w.striker, w.buyer, wage-1, years, nil); err == nil && r.Accepted {
		t.Fatal("terms below his expectation were accepted")
	}
	w.exec(`UPDATE players SET release_clause_set_at = now() - interval '15 days' WHERE player_id = $1`, w.striker)
	homeBudget := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, w.home)

	var signed atomic.Int32
	buyers := []int64{w.buyer, w.rival}
	var next atomic.Int32
	runConcurrently(2, func() {
		buyer := buyers[next.Add(1)-1]
		if r, err := common.PayReleaseClause(w.dbCtx, w.striker, buyer, wage, years, nil); err == nil && r.Accepted {
			signed.Add(1)
		}
	})
	if signed.Load() != 1 {
		t.Fatalf("%d clubs paid the clause, want exactly 1", signed.Load())
	}
	if team := w.playerTeam(w.striker); team != w.buyer && team != w.rival {
		t.Fatalf("he plays for %d, want one of the payers", team)
	}
	if got := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, w.home); got != homeBudget+clause {
		t.Errorf("home budget %d, want %d (+ the clause)", got, homeBudget+clause)
	}
	if n := w.count(`SELECT count(*) FROM player_transfers WHERE player_id = $1 AND kind = $2 AND fee = $3`, w.striker, common.PLAYER_TRANSFER_RELEASE_CLAUSE, clause); n != 1 {
		t.Errorf("%d release-clause moves recorded, want 1", n)
	}
	if n := w.notified(w.homeOwner, "releaseClausePaid"); n != 1 {
		t.Errorf("home club told %d times, want 1", n)
	}
}

// cpu-transfer-market "Paying release clauses": "Rate limit" and "Cheap clause".
func TestReleaseClauses_CpuPaysWithinTheRateLimit(t *testing.T) {
	w := newPreContractWorld(t, "rc")
	director := w.team("rc", "RC Director", common.AiLevelDirector, nil, 50_000_000)
	w.squad("rc", director)
	clause := common.MinReleaseClause(w.strikerPlayer())
	w.exec(`UPDATE players SET contract_years_remaining = 3, release_clause = $2, release_clause_set_at = now() - interval '15 days' WHERE player_id = $1`, w.striker, clause)
	now := time.Now()

	w.exec(`INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, created_at) VALUES ($1, $2, $3, 1, $4, now() - interval '3 days')`,
		w.player("rc", director, common.MIDFIELD, 40, 25), w.home, director, common.PLAYER_TRANSFER_RELEASE_CLAUSE)
	if ok, err := common.CpuMayPayClauseOn(w.dbCtx, w.home, now); err != nil || ok {
		t.Fatalf("3 days after a CPU clause payment on the club: may pay=%v err=%v, want false", ok, err)
	}
	w.exec(`UPDATE player_transfers SET created_at = now() - interval '8 days' WHERE from_team_id = $1 AND kind = $2`, w.home, common.PLAYER_TRANSFER_RELEASE_CLAUSE)
	if ok, err := common.CpuMayPayClauseOn(w.dbCtx, w.home, now); err != nil || !ok {
		t.Fatalf("8 days after: may pay=%v err=%v, want true", ok, err)
	}

	clauses := payableClauses(w.dbCtx, []int64{w.striker}, now)
	if clauses[w.striker] != clause {
		t.Fatalf("payable clauses %v, want %d for the striker", clauses, clause)
	}
	target := scoredCandidate{transferCandidate: transferCandidate{Player: w.strikerPlayer(), Kind: candidateReleaseClause, SellerTeamId: w.home, AskingPrice: clause}}
	if !payAiReleaseClause(w.dbCtx, w.teamOf(director), target) {
		t.Fatal("the Director did not pay the clause")
	}
	if team := w.playerTeam(w.striker); team != director {
		t.Errorf("he plays for %d, want the Director %d", team, director)
	}
}

// cpu-team-management "Release clauses in CPU renewals": "Manager bench renewal".
func TestReleaseClauses_ManagerSetsOneOnABenchRenewal(t *testing.T) {
	w := newPreContractWorld(t, "rd")
	team := w.team("rd", "RD Manager", common.AiLevelManager, nil, 5_000_000)
	w.squad("rd", team)
	bench := w.player("rd", team, common.MIDFIELD, 30, 24)
	w.exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 1 WHERE player_id = $1`, bench)
	refreshAiTeamLineup(w.dbCtx, team, common.AiLevelManager)

	renewExpiringContracts(w.dbCtx, w.teamOf(team), false)

	p, err := common.GetPlayerById(w.dbCtx, bench)
	if err != nil {
		t.Fatal(err)
	}
	if p.ContractYearsRemaining <= 1 {
		t.Fatal("the young bench player was not renewed")
	}
	if p.ReleaseClause == nil || *p.ReleaseClause < common.MinReleaseClause(*p) {
		t.Fatalf("bench renewal clause %v, want one of at least the minimum", p.ReleaseClause)
	}
	skillWage := common.PlayerWageFromSkills(0, 0, 0, 30, 30, 0, 0)
	if p.Wage >= skillWage {
		t.Errorf("renewed at %d, want the discounted wage below %d", p.Wage, skillWage)
	}
}
