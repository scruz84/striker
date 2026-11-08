package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// preContractWorld is a loanWorld with two human clubs: home owns a striker in his last season,
// buyer and rival want him.
type preContractWorld struct {
	*loanWorld
	home, buyer, rival int64
	homeOwner          int64
	buyerOwner         int64
	striker            int64
}

func newPreContractWorld(t *testing.T, country string) *preContractWorld {
	w := &preContractWorld{loanWorld: newLoanWorld(t, country)}
	w.homeOwner = w.user("pc-home")
	w.buyerOwner = w.user("pc-buyer")
	w.home = w.team(country, "PC Home", 0, w.homeOwner, 1_000_000)
	w.buyer = w.team(country, "PC Buyer", 0, w.buyerOwner, 1_000_000)
	w.rival = w.team(country, "PC Rival", 0, w.user("pc-rival"), 1_000_000)
	w.striker = w.player(country, w.home, common.STRIKER, 60, 25)
	w.exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 1, wage = $2 WHERE player_id = $1`, w.striker, w.skillWage())
	w.exec(`DELETE FROM pre_contract_offers WHERE player_id = $1`, w.striker)
	return w
}

func (w *preContractWorld) skillWage() int {
	return common.PlayerWageFromSkills(0, 0, 0, 60, 0, 60, 60)
}

// halfSeason records 15 league matches played by team, with none left: its league is past half.
func (w *preContractWorld) halfSeason(team int64) {
	w.exec(`INSERT INTO classification (league_id, team_id, season, matches, points) VALUES ($1, $2, 1, 15, 20)`, w.league, team)
}

func (w *preContractWorld) notified(userId int64, key string) int {
	return w.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND message_key = $2`, userId, "notifications.messages."+key)
}

// pre-contracts "Making a pre-contract offer" ("Too early", "Offer made") and "Player demands"
// ("Below the premium").
func TestPreContracts_OfferRules(t *testing.T) {
	w := newPreContractWorld(t, "pa")
	minimum := common.PreContractMinimumWage(w.skillWage(), common.IntelligenceNaive)

	if _, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, minimum, 2); !errors.Is(err, common.ErrPreContractTooEarly) {
		t.Fatalf("offer before half the season: %v, want ErrPreContractTooEarly", err)
	}
	w.halfSeason(w.home)

	result, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, minimum-1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted || result.ExpectedWage != minimum {
		t.Fatalf("offer below the premium: accepted=%v expected=%d, want refused and %d", result.Accepted, result.ExpectedWage, minimum)
	}

	w.exec(`UPDATE players SET contract_years_remaining = 2 WHERE player_id = $1`, w.striker)
	if _, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, minimum, 2); !errors.Is(err, common.ErrPreContractNotLastSeason) {
		t.Fatalf("offer with 2 seasons left: %v, want ErrPreContractNotLastSeason", err)
	}
	w.exec(`UPDATE players SET contract_years_remaining = 1 WHERE player_id = $1`, w.striker)

	result, err = common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, minimum, 2)
	if err != nil || !result.Accepted {
		t.Fatalf("offer at the premium: %+v, %v", result, err)
	}
	if n := w.notified(w.homeOwner, "preContractOffered"); n != 1 {
		t.Errorf("home club got %d offer notifications, want 1", n)
	}
	if _, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, minimum, 2); !errors.Is(err, common.ErrPreContractAlreadyOffered) {
		t.Errorf("second offer from the same club: %v, want ErrPreContractAlreadyOffered", err)
	}

	received, err := common.GetPreContractOffersReceived(w.dbCtx, w.home)
	if err != nil || len(received) != 1 {
		t.Fatalf("received offers: %v, %v", received, err)
	}
	if received[0].DecidesAt == nil || received[0].KeepWage != common.PreContractKeepWage(minimum, common.IntelligenceNaive) {
		t.Errorf("received offer %+v, want a decision time and keep wage %d", received[0], common.PreContractKeepWage(minimum, common.IntelligenceNaive))
	}
}

// pre-contracts "Right to counter-offer": "Matching a rival" and "Renewed in time".
func TestPreContracts_CounterOfferByRenewal(t *testing.T) {
	w := newPreContractWorld(t, "pb")
	w.halfSeason(w.home)
	best := common.PreContractMinimumWage(w.skillWage(), common.IntelligenceNaive) + 1100
	if r, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, best, 2); err != nil || !r.Accepted {
		t.Fatalf("offer: %+v, %v", r, err)
	}
	keep := common.PreContractKeepWage(best, common.IntelligenceNaive)

	offer, err := common.MakeContractRenewalOffer(w.dbCtx, w.striker, w.home, keep-1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if offer.Accepted || offer.ExpectedWage != keep {
		t.Fatalf("renewal below the bar: accepted=%v expected=%d, want refused and %d", offer.Accepted, offer.ExpectedWage, keep)
	}
	offer, err = common.MakeContractRenewalOffer(w.dbCtx, w.striker, w.home, keep, 3)
	if err != nil || !offer.Accepted {
		t.Fatalf("renewal at the bar: %+v, %v", offer, err)
	}
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1`, w.striker); n != 0 {
		t.Errorf("%d pre-contract offers left after the renewal, want 0", n)
	}
	if n := w.notified(w.buyerOwner, "preContractCancelled"); n != 1 {
		t.Errorf("buyer got %d cancellation notices, want 1", n)
	}
}

// pre-contracts "Signing a pre-contract" ("Two suitors", decided once by concurrent replicas),
// "Pre-contracted player" ("Listing refused", "Rollover") and the reminder, sent once.
func TestPreContracts_DecisionAndRollover(t *testing.T) {
	w := newPreContractWorld(t, "pc")
	w.halfSeason(w.home)
	wage := common.PreContractMinimumWage(w.skillWage(), common.IntelligenceNaive) + 100
	for _, buyer := range []struct {
		team  int64
		years int
	}{{w.buyer, 3}, {w.rival, 2}} {
		if r, err := common.MakePreContractOffer(w.dbCtx, w.striker, buyer.team, wage, buyer.years); err != nil || !r.Accepted {
			t.Fatalf("offer from %d: %+v, %v", buyer.team, r, err)
		}
	}

	// 49 hours in: the home club is reminded once, however many replicas run the sweep.
	w.exec(`UPDATE pre_contract_offers SET created_at = now() - interval '49 hours' WHERE player_id = $1`, w.striker)
	runConcurrently(2, func() { common.DecidePreContracts(w.dbCtx, time.Now()) })
	if n := w.notified(w.homeOwner, "preContractDecisionSoon"); n != 1 {
		t.Errorf("home club reminded %d times, want 1", n)
	}
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NOT NULL`, w.striker); n != 0 {
		t.Fatalf("a pre-contract was signed before 72 hours")
	}

	w.exec(`UPDATE pre_contract_offers SET created_at = now() - interval '73 hours' WHERE player_id = $1`, w.striker)
	runConcurrently(2, func() { common.DecidePreContracts(w.dbCtx, time.Now()) })
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NOT NULL AND from_team_id = $2`, w.striker, w.buyer); n != 1 {
		t.Fatalf("the 3-season offer was not the one signed")
	}
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1`, w.striker); n != 1 {
		t.Errorf("%d pre-contract rows left, want only the signed one", n)
	}
	if n := w.notified(w.buyerOwner, "preContractSigned"); n != 1 {
		t.Errorf("buyer told %d times it signed, want 1", n)
	}
	if n := w.notified(w.homeOwner, "preContractSignedAway"); n != 1 {
		t.Errorf("home club told %d times he signed away, want 1", n)
	}

	if err := common.CreateTransferListing(w.dbCtx, w.striker, w.home, 1000); !errors.Is(err, common.ErrPlayerPreContracted) {
		t.Errorf("listing a pre-contracted player: %v, want ErrPlayerPreContracted", err)
	}
	if _, err := common.MakeContractRenewalOffer(w.dbCtx, w.striker, w.home, wage*2, 3); !errors.Is(err, common.ErrPlayerPreContracted) {
		t.Errorf("renewing a pre-contracted player: %v, want ErrPlayerPreContracted", err)
	}

	released, err := common.AdvancePlayerContract(w.dbCtx, w.striker)
	if err != nil || released {
		t.Fatalf("rollover: released=%v err=%v, want a move", released, err)
	}
	if team := w.playerTeam(w.striker); team != w.buyer {
		t.Fatalf("after the rollover he plays for %d, want the buyer %d", team, w.buyer)
	}
	if got := w.count(`SELECT wage FROM players WHERE player_id = $1`, w.striker); got != wage {
		t.Errorf("wage %d, want the agreed %d", got, wage)
	}
	if got := w.count(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, w.striker); got != 3 {
		t.Errorf("contract %d seasons, want the agreed 3", got)
	}
	if n := w.count(`SELECT count(*) FROM player_transfers WHERE player_id = $1 AND kind = $2 AND fee = 0`, w.striker, common.PLAYER_TRANSFER_PRE_CONTRACT); n != 1 {
		t.Errorf("%d pre-contract moves in his history, want 1", n)
	}
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1`, w.striker); n != 0 {
		t.Errorf("the pre-contract row survived the move")
	}
}

// pre-contracts "Pre-contracted player": "Retirement".
func TestPreContracts_RetirementLapses(t *testing.T) {
	w := newPreContractWorld(t, "pd")
	w.halfSeason(w.home)
	if r, err := common.MakePreContractOffer(w.dbCtx, w.striker, w.buyer, common.PreContractMinimumWage(w.skillWage(), common.IntelligenceNaive), 1); err != nil || !r.Accepted {
		t.Fatalf("offer: %+v, %v", r, err)
	}
	common.LapsePreContractsOnRetirement(w.dbCtx, w.striker)
	if err := common.RetirePlayer(w.dbCtx, w.striker); err != nil {
		t.Fatal(err)
	}
	if n := w.notified(w.buyerOwner, "preContractRetired"); n != 1 {
		t.Errorf("buyer told %d times, want 1", n)
	}
}

// cpu-team-management "Answering pre-contract offers": "Manager keeps its starter" and "Caretaker
// bench player".
func TestPreContracts_CpuSellerAnswers(t *testing.T) {
	w := newPreContractWorld(t, "pe")
	for _, level := range []int{common.AiLevelManager, common.AiLevelCaretaker} {
		team := w.team("pe", "PC CPU", level, nil, 5_000_000)
		w.squad("pe", team)
		star := w.player("pe", team, common.STRIKER, 70, 25)
		w.exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 1 WHERE player_id = $1`, star)
		w.halfSeason(team)
		refreshAiTeamLineup(w.dbCtx, team, level)
		if level == common.AiLevelCaretaker {
			// A Caretaker renews its starters in its own daily pass, not in answer to an offer:
			// bench the striker so only the answer could renew him.
			w.exec(`DELETE FROM default_lineup_positions WHERE player_id = $1`, star)
		}
		skill := common.PlayerWageFromSkills(0, 0, 0, 70, 0, 70, 70)
		if r, err := common.MakePreContractOffer(w.dbCtx, star, w.buyer, common.PreContractMinimumWage(skill, common.IntelligenceNaive)*12/10, 2); err != nil || !r.Accepted {
			t.Fatalf("offer: %+v, %v", r, err)
		}

		answerPreContractOffers(w.dbCtx, w.teamOf(team))

		years := w.count(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, star)
		if level == common.AiLevelManager && years <= 1 {
			t.Errorf("the Manager did not counter by renewing its starter")
		}
		if level == common.AiLevelCaretaker && years != 1 {
			t.Errorf("the Caretaker reacted to the offer (%d seasons left)", years)
		}
	}
}

// cpu-transfer-market "Pre-contract targets": "Director plans next season".
func TestPreContracts_DirectorOffersOne(t *testing.T) {
	w := newPreContractWorld(t, "pf")
	director := w.team("pf", "PC Director", common.AiLevelDirector, nil, 50_000_000)
	w.squad("pf", director)
	target := w.player("pf", w.home, common.STRIKER, 97, 26)
	w.exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 1 WHERE player_id = $1`, target)
	w.halfSeason(w.home)

	team := w.teamOf(director)
	state, err := loadTransferState(w.dbCtx, team)
	if err != nil {
		t.Fatal(err)
	}
	profile := aiProfile(team)
	var found *transferCandidate
	for _, c := range preContractCandidates(w.dbCtx, team, profile, state, map[int64]bool{}) {
		if c.Player.Id == target {
			c := c
			found = &c
		}
	}
	if found == nil {
		t.Fatal("the last-season striker is not a pre-contract candidate")
	}
	p, err := common.GetPlayerById(w.dbCtx, target)
	if err != nil {
		t.Fatal(err)
	}
	expected, _, err := common.PlayerTermsExpectation(w.dbCtx, *p, common.TermsTransfer)
	if err != nil {
		t.Fatal(err)
	}
	if want := common.PreContractMinimumWage(expected, common.IntelligenceNaive); found.Player.Wage != want {
		t.Errorf("candidate wage %d, want %d (1.10 x expectation)", found.Player.Wage, want)
	}
	if !makeAiPreContractOffer(w.dbCtx, team, scoredCandidate{transferCandidate: *found}) {
		t.Fatal("the Director's pre-contract offer was not made")
	}
	if n := w.count(`SELECT count(*) FROM pre_contract_offers WHERE player_id = $1 AND from_team_id = $2`, target, director); n != 1 {
		t.Errorf("%d offers recorded, want 1", n)
	}
	if n := w.notified(w.homeOwner, "preContractOffered"); n != 1 {
		t.Errorf("the human owner got %d notices, want 1", n)
	}

	// "Pile-up on a human's player": with three CPU offers pending, a fourth CPU club stays out.
	for _, name := range []string{"PC CPU Two", "PC CPU Three"} {
		other := w.team("pf", name, common.AiLevelManager, nil, 1_000_000)
		w.exec(`INSERT INTO pre_contract_offers (player_id, from_team_id, wage, years) VALUES ($1, $2, $3, 1)`, target, other, found.Player.Wage)
	}
	fourth := w.teamOf(w.team("pf", "PC CPU Four", common.AiLevelDirector, nil, 50_000_000))
	if makeAiPreContractOffer(w.dbCtx, fourth, scoredCandidate{transferCandidate: *found}) {
		t.Error("a fourth CPU club offered a pre-contract to a human club's player")
	}
}
