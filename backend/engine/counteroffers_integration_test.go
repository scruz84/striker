package engine

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// CPU clubs in a counter-offer round (openspec/specs/transfer-negotiation, cpu-team-management and
// cpu-transfer-market), through the same production functions a human uses. Needs
// STRIKER_TEST_DB_DSN.

// cpuCounterOffer seeds an offer of amount from buyer to seller on playerId.
func (w *loanWorld) cpuCounterOffer(playerId, buyer, seller int64, amount int64) int64 {
	return w.scan(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at)
		VALUES ($1, $2, $3, $4, 2000, 2, now() - interval '1 day') RETURNING offer_id`, playerId, buyer, seller, amount)
}

func (w *loanWorld) sentOffer(teamId, offerId int64) common.TransferOffer {
	w.t.Helper()
	sent, err := common.GetOffersSent(w.dbCtx, teamId)
	if err != nil {
		w.t.Fatal(err)
	}
	for _, o := range sent {
		if o.OfferId == offerId {
			return o
		}
	}
	w.t.Fatalf("offer %d not among team %d's sent offers", offerId, teamId)
	return common.TransferOffer{}
}

// A Director answers a human's near-miss offer with its threshold rounded up to the next 1,000,
// and the human is told.
func TestCpuSeller_CountersANearMissFromAHuman(t *testing.T) {
	const country = "zk"
	w := newLoanWorld(t, country)
	humanUser := w.user("counter-human")
	human := w.team(country, "Counter Human", 0, humanUser, 50000000)
	w.squad(country, human)
	director := w.team(country, "Counter Director", common.AiLevelDirector, nil, 50000000)
	w.squad(country, director)
	target := w.player(country, director, common.STRIKER, 45, 26)

	offerId := w.cpuCounterOffer(target, human, director, 1)
	// The Director's threshold, exactly as respondToReceivedOffers will form it (seeded by the
	// offer id); the offer is then set to 85% of it.
	squad, err := ownedSquad(w.dbCtx, director)
	if err != nil {
		t.Fatal(err)
	}
	player, err := common.GetPlayerById(w.dbCtx, target)
	if err != nil {
		t.Fatal(err)
	}
	international, nationalCaps := nationalStanding(w.dbCtx, target)
	decision := common.EvaluateIncomingOffer(common.AiTransferProfileFor(common.AiLevelDirector), common.OfferContext{
		Squad: squad, Player: *player, Caps: skillCapsFor(w.dbCtx, target), Amount: 1,
		International: international, InternationalCaps: nationalCaps,
	}, rand.New(rand.NewSource(offerId)))
	if decision.Reason == common.OfferReasonThinPosition || decision.Reason == common.OfferReasonSquadTooSmall || decision.Threshold < 2000 {
		t.Fatalf("fixture: decision %+v, want a sellable player with a real threshold", decision)
	}
	amount := int64(math.Round(0.85 * float64(decision.Threshold)))
	w.exec(`UPDATE transfer_offers SET amount = $1 WHERE offer_id = $2`, amount, offerId)

	respondToReceivedOffers(w.dbCtx, director)

	want := (decision.Threshold + 999) / 1000 * 1000
	if got := w.scan(`SELECT COALESCE(counter_amount, 0) FROM transfer_offers WHERE offer_id = $1`, offerId); got != want {
		t.Fatalf("counter = %d, want %d (threshold %d rounded up)", got, want, decision.Threshold)
	}
	if n := w.count(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND notification_type = $2`, humanUser, common.NOTIFICATION_COUNTER_OFFER); n != 1 {
		t.Errorf("human counter notifications = %d, want 1", n)
	}
	// A second sweep leaves the countered offer alone.
	respondToReceivedOffers(w.dbCtx, director)
	if got := w.scan(`SELECT COALESCE(counter_amount, 0) FROM transfer_offers WHERE offer_id = $1`, offerId); got != want {
		t.Errorf("after a second sweep counter = %d, want it unchanged at %d", got, want)
	}
}

// directorBuyerWorld is a Director that bid for a human's player, with the human's counter still to
// answer; value is the Director's valuation of him as answerCountersAsBuyer will form it.
type directorBuyerWorld struct {
	*loanWorld
	director, human, target, offerId int64
	humanUser                        int64
	value                            int64
}

func newDirectorBuyerWorld(t *testing.T, country string) *directorBuyerWorld {
	w := &directorBuyerWorld{loanWorld: newLoanWorld(t, country)}
	w.humanUser = w.user("counter-seller")
	w.human = w.team(country, "Counter Seller", 0, w.humanUser, 50000000)
	w.squad(country, w.human)
	w.director = w.team(country, "Counter Buyer", common.AiLevelDirector, nil, 500000000)
	w.squad(country, w.director)
	w.target = w.player(country, w.human, common.STRIKER, 60, 25)
	w.offerId = w.cpuCounterOffer(w.target, w.director, w.human, 1)
	w.exec(`UPDATE transfer_offers SET counter_amount = 2, countered_at = now() WHERE offer_id = $1`, w.offerId)

	team := w.teamOf(w.director)
	state, err := loadTransferState(w.dbCtx, team)
	if err != nil {
		t.Fatal(err)
	}
	radar, _ := common.GetNationalRadar(w.dbCtx)
	score := counterScore(common.AiTransferProfileFor(common.AiLevelDirector), state, w.sentOffer(w.director, w.offerId),
		skillCapsFor(w.dbCtx, w.target), radar)
	if score.Value < 2000 {
		t.Fatalf("fixture: the Director values the target at only %d", score.Value)
	}
	w.value = score.Value
	return w
}

// setCounter prices the bid at 1.1 times the valuation and the counter at multiple times it.
func (w *directorBuyerWorld) setCounter(multiple float64) (bid, counter int64) {
	bid = int64(1.1 * float64(w.value))
	counter = int64(multiple * float64(w.value))
	w.exec(`UPDATE transfer_offers SET amount = $1, counter_amount = $2 WHERE offer_id = $3`, bid, counter, w.offerId)
	return bid, counter
}

func TestCpuBuyer_AcceptsAnAcceptableCounter(t *testing.T) {
	w := newDirectorBuyerWorld(t, "zj")
	_, counter := w.setCounter(1.3)
	before := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, w.director)

	answerCountersAsBuyer(w.dbCtx, w.teamOf(w.director))

	if got := w.playerTeam(w.target); got != w.director {
		t.Fatalf("player is with team %d, want the Director (%d)", got, w.director)
	}
	if got := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, w.director); got != before-counter {
		t.Errorf("Director budget = %d, want %d (paid the counter)", got, before-counter)
	}
	if n := w.count(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND message_key = 'notifications.messages.transferSold'`, w.humanUser); n != 1 {
		t.Errorf("seller sold notifications = %d, want 1", n)
	}
}

func TestCpuBuyer_DeclinesACounterTooHigh(t *testing.T) {
	w := newDirectorBuyerWorld(t, "zi")
	bid, _ := w.setCounter(1.6)

	answerCountersAsBuyer(w.dbCtx, w.teamOf(w.director))

	if got := w.playerTeam(w.target); got != w.human {
		t.Fatalf("player is with team %d, want him still at the seller (%d)", got, w.human)
	}
	if n := w.count(`SELECT COUNT(*) FROM transfer_offers WHERE offer_id = $1`, w.offerId); n != 0 {
		t.Error("the declined offer is still there")
	}
	if n := w.count(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND message_key = 'notifications.messages.counterDeclined'`, w.humanUser); n != 1 {
		t.Errorf("seller declined notifications = %d, want 1", n)
	}
	rejections, err := common.GetRecentOfferRejections(w.dbCtx, w.director, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	rejection, ok := rejections[w.target]
	if !ok || rejection.Amount != bid {
		t.Fatalf("rejection = %+v (%v), want the original bid %d", rejection, ok, bid)
	}
	if allowed, _ := common.OfferRetryAllowed(&rejection, time.Now()); allowed {
		t.Error("the Director may bid again during the cooldown")
	}
	if _, minBid := common.OfferRetryAllowed(&rejection, time.Now().Add(common.AiOfferCooldown+time.Minute)); minBid != int64(float64(bid)*common.AiRebidMultiple) {
		t.Errorf("rebid floor = %d, want %d", minBid, int64(float64(bid)*common.AiRebidMultiple))
	}
}
