package engine

import (
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

// team-finances "Contract renewal and release": "No free wage cut" and "Last-season veteran".
func TestContractTerms_NoFreeWageCut(t *testing.T) {
	const country = "zt"
	w := newLoanWorld(t, country)
	owner := w.user("wage-floor")
	team := w.team(country, "Wage Floor FC", 0, owner, 1_000_000)
	player := w.player(country, team, common.MIDFIELD, 50, 31)
	skillWage := common.PlayerWageFromSkills(0, 0, 0, 50, 50, 0, 0)
	current := skillWage * 2
	w.exec(`UPDATE players SET wage = $1, contract_years_remaining = 3, intelligence = 1 WHERE player_id = $2`, current, player)

	offer, err := common.MakeContractRenewalOffer(w.dbCtx, player, team, skillWage, 2)
	if err != nil {
		t.Fatal(err)
	}
	if offer.Accepted || offer.ExpectedWage != current {
		t.Fatalf("renewal at the market wage with 3 seasons left: accepted=%v expected=%d, want declined and %d", offer.Accepted, offer.ExpectedWage, current)
	}

	w.exec(`UPDATE players SET contract_years_remaining = 1 WHERE player_id = $1`, player)
	floor := int(math.Ceil(float64(current) * common.LastSeasonWageFloorShare))
	offer, err = common.MakeContractRenewalOffer(w.dbCtx, player, team, floor, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !offer.Accepted || offer.ExpectedWage != floor {
		t.Errorf("last-season renewal at 85%%: accepted=%v expected=%d, want accepted at %d", offer.Accepted, offer.ExpectedWage, floor)
	}
	if wage := w.count(`SELECT wage FROM players WHERE player_id = $1`, player); wage != floor {
		t.Errorf("renewed wage %d, want %d", wage, floor)
	}
}

// team-finances "Personal terms of a transfer offer": "Pay cut refused"; cpu-transfer-market
// "CPU personal terms": "Overpaid target".
func TestContractTerms_PayCutRefused(t *testing.T) {
	const country = "zu"
	w := newLoanWorld(t, country)
	seller := w.team(country, "Overpayers", common.AiLevelManager, nil, 1_000_000)
	buyer := w.team(country, "Bargain Hunters", common.AiLevelDirector, nil, 10_000_000)
	player := w.player(country, seller, common.STRIKER, 50, 25)
	skillWage := common.PlayerWageFromSkills(0, 0, 0, 50, 0, 50, 50)
	current := skillWage + 700
	w.exec(`UPDATE players SET wage = $1, contract_years_remaining = 3, intelligence = 1 WHERE player_id = $2`, current, player)

	result, err := common.MakeTransferOffer(w.dbCtx, player, buyer, 100_000, skillWage, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted || result.ExpectedWage != current {
		t.Fatalf("offer at the market wage: accepted=%v expected=%d, want refused and %d", result.Accepted, result.ExpectedWage, current)
	}
	if n := w.count(`SELECT COUNT(*) FROM transfer_offers WHERE player_id = $1`, player); n != 0 {
		t.Errorf("%d offers created, want none", n)
	}

	p, err := common.GetPlayerById(w.dbCtx, player)
	if err != nil {
		t.Fatal(err)
	}
	wage, years, err := common.PlayerTermsExpectation(w.dbCtx, *p, common.TermsTransfer)
	if err != nil {
		t.Fatal(err)
	}
	if wage != current {
		t.Errorf("CPU terms for an overpaid target: wage %d, want his current %d", wage, current)
	}
	result, err = common.MakeTransferOffer(w.dbCtx, player, buyer, 100_000, wage, years)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted {
		t.Errorf("offer on the player's own terms refused: %s", result.Reason)
	}
}

// cpu-team-management "Selling contracts it will not renew": "Bench veteran with one season left"
// and "Caretaker".
func TestContractTerms_ManagerListsTheVeteranItDoesNotRenew(t *testing.T) {
	const country = "zv"
	w := newLoanWorld(t, country)
	for _, level := range []int{common.AiLevelManager, common.AiLevelCaretaker} {
		team := w.team(country, "Expiring "+string(rune('A'+level)), level, nil, 5_000_000)
		w.squad(country, team)
		veteran := w.player(country, team, common.MIDFIELD, 30, 31)
		w.exec(`UPDATE players SET contract_years_remaining = 1 WHERE player_id = $1`, veteran)
		refreshAiTeamLineup(w.dbCtx, team, level)

		RunAITeamManagement(w.dbCtx, team)

		if years := w.count(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, veteran); years != 1 {
			t.Fatalf("level %d renewed the bench veteran (%d seasons left)", level, years)
		}
		listed := w.count(`SELECT COUNT(*) FROM transfer_listings WHERE player_id = $1`, veteran)
		if level == common.AiLevelCaretaker {
			if listed != 0 {
				t.Errorf("a Caretaker listed its unrenewed veteran")
			}
			continue
		}
		if listed != 1 {
			t.Fatalf("the Manager did not list its unrenewed veteran")
		}
		p, err := common.GetPlayerById(w.dbCtx, veteran)
		if err != nil {
			t.Fatal(err)
		}
		fresh := *p
		fresh.ContractYearsRemaining = 3
		want := int64(math.Round(float64(common.MarketValue(fresh)) * common.LastSeasonContractValueFactor))
		if price := int64(w.count(`SELECT asking_price FROM transfer_listings WHERE player_id = $1`, veteran)); price != want {
			t.Errorf("asking price %d, want %d (contract factor 0.6)", price, want)
		}
	}
}
