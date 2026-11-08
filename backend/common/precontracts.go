package common

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// sqlQuerier is satisfied by both *sql.DB and *sql.Tx, so a read can run inside a caller's
// transaction or on its own.
type sqlQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

// Pre-contracts (openspec/specs/pre-contracts): a club agrees a free move for next season with
// another club's player in the last season of his contract.
const (
	// PreContractSeasonShare is the share of his club's league season that must be played before a
	// player in his last season can be offered a pre-contract.
	PreContractSeasonShare = 0.5
	// PreContractMovePremium is how much more than his usual expected wage an Aware player asks to
	// leave his club (the premium of each tier is PreContractPremium): a pre-contract offer must
	// reach it, and while offers are pending his own club keeps him for the best pending wage
	// divided by it (see PreContractCounterBar).
	PreContractMovePremium = 1.10
	// PreContractDecisionDelay is how long after a player's first pending offer he decides.
	PreContractDecisionDelay = 72 * time.Hour
	// PreContractReminderLead is how long before the decision his club is reminded.
	PreContractReminderLead = 24 * time.Hour
	// MaxSignedPreContracts is how many signed pre-contracts a club may hold at once;
	// AiMaxSignedPreContracts the CPU's own, lower limit.
	MaxSignedPreContracts   = 3
	AiMaxSignedPreContracts = 2
	// AiMaxPreContractOffersPerHumanPlayer is how many CPU pre-contract offers may be pending on
	// one human club's player (one on a CPU club's player), as for transfer offers.
	AiMaxPreContractOffersPerHumanPlayer = AiMaxOpenOffersPerHumanPlayer
)

var ErrPreContractNotLastSeason = errors.New("only a player in the last season of his contract can be offered a pre-contract")
var ErrPreContractTooEarly = errors.New("pre-contracts open once half of the player's league season has been played")
var ErrPlayerPreContracted = errors.New("player has signed a pre-contract with another club")
var ErrPreContractLimit = errors.New("your club already holds the maximum number of signed pre-contracts")
var ErrPreContractAlreadyOffered = errors.New("you already have a pending pre-contract offer for this player")
var ErrPreContractNotFound = errors.New("pre-contract offer not found")
var ErrInvalidPreContractTerms = errors.New("pre-contract terms must be a positive wage for 1 to " + strconv.Itoa(MaxContractYearsRemaining) + " seasons")

const insertPreContractOfferStmtSql = "INSERT INTO pre_contract_offers (player_id, from_team_id, wage, years, release_clause) VALUES ($1, $2, $3, $4, $5) RETURNING offer_id"
const isPlayerPreContractedStmtSql = "SELECT EXISTS (SELECT 1 FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NOT NULL)"
const countSignedPreContractsStmtSql = "SELECT count(*) FROM pre_contract_offers WHERE from_team_id = $1 AND signed_at IS NOT NULL"
const bestPendingPreContractStmtSql = "SELECT COALESCE(MAX(wage), 0), MIN(created_at) FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NULL"
const deletePendingPreContractsStmtSql = "DELETE FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NULL RETURNING from_team_id"
const withdrawPreContractOfferStmtSql = "DELETE FROM pre_contract_offers WHERE offer_id = $1 AND from_team_id = $2 AND signed_at IS NULL"
const lockPendingPreContractsStmtSql = "SELECT offer_id, from_team_id, wage, years, created_at, release_clause FROM pre_contract_offers " +
	"WHERE player_id = $1 AND signed_at IS NULL ORDER BY wage DESC, years DESC, created_at, offer_id " + ForUpdate
const signPreContractStmtSql = "UPDATE pre_contract_offers SET signed_at = now() WHERE offer_id = $1"
const deletePreContractOfferStmtSql = "DELETE FROM pre_contract_offers WHERE offer_id = $1"
const signedPreContractStmtSql = "SELECT offer_id, from_team_id, wage, years, release_clause FROM pre_contract_offers WHERE player_id = $1 AND signed_at IS NOT NULL"
const playersDueForPreContractDecisionStmtSql = "SELECT player_id FROM pre_contract_offers WHERE signed_at IS NULL " +
	"GROUP BY player_id HAVING MIN(created_at) <= $1 ORDER BY player_id"

// claimPreContractRemindersStmtSql marks, in one statement, the first pending offer of every player
// whose decision is PreContractReminderLead or less away and whose club has not been reminded yet,
// returning those players: two replicas running it at once remind each club once.
const claimPreContractRemindersStmtSql = "UPDATE pre_contract_offers SET reminded_at = now() WHERE offer_id IN (" +
	"SELECT DISTINCT ON (player_id) offer_id FROM pre_contract_offers WHERE signed_at IS NULL ORDER BY player_id, created_at, offer_id" +
	") AND reminded_at IS NULL AND created_at <= $1 RETURNING player_id"

const countAiPreContractOffersOnPlayerStmtSql = "SELECT count(*) FROM pre_contract_offers o JOIN teams t ON t.team_id = o.from_team_id " +
	"WHERE o.player_id = $1 AND o.signed_at IS NULL AND t.owner_id IS NULL"

const getPreContractOffersStmtSql = "SELECT o.offer_id, o.player_id, o.from_team_id, tf.name, p.team_id, tt.name, o.wage, o.years, o.created_at, o.signed_at, " +
	"MIN(o.created_at) FILTER (WHERE o.signed_at IS NULL) OVER (PARTITION BY o.player_id), " +
	"COALESCE(MAX(o.wage) FILTER (WHERE o.signed_at IS NULL) OVER (PARTITION BY o.player_id), 0) " +
	"FROM pre_contract_offers o JOIN players p ON p.player_id = o.player_id " +
	"JOIN teams tf ON tf.team_id = o.from_team_id JOIN teams tt ON tt.team_id = p.team_id "

const getPreContractOffersReceivedStmtSql = getPreContractOffersStmtSql + "WHERE p.team_id = $1 ORDER BY o.player_id, o.wage DESC, o.created_at"
const getPreContractOffersSentStmtSql = getPreContractOffersStmtSql + "WHERE o.from_team_id = $1 ORDER BY o.created_at DESC"

// PreContractOffer is one pre-contract offer, pending or signed.
type PreContractOffer struct {
	OfferId      int64     `json:"offerId"`
	Player       Player    `json:"player"`
	FromTeamId   int64     `json:"fromTeamId"`
	FromTeamName string    `json:"fromTeamName"`
	ToTeamId     int64     `json:"toTeamId"`
	ToTeamName   string    `json:"toTeamName"`
	Wage         int       `json:"wage"`
	Years        int       `json:"years"`
	CreatedAt    time.Time `json:"createdAt"`
	// SignedAt is set once the player signed it; he then joins FromTeamId at the next rollover.
	SignedAt *time.Time `json:"signedAt,omitempty"`
	// DecidesAt is when the player chooses among the pending offers (his first offer +
	// PreContractDecisionDelay); absent on a signed one.
	DecidesAt *time.Time `json:"decidesAt,omitempty"`
	// KeepWage is the wage his own club must offer to renew him while offers are pending (see
	// PreContractCounterBar) - only reported to that club.
	KeepWage int `json:"keepWage,omitempty"`
}

// PreContractOfferResult mirrors TransferOfferResult: the player judges the terms at once, and a
// refusal reports what he would accept (ExpectedWage already includes PreContractMovePremium).
type PreContractOfferResult struct {
	Accepted           bool   `json:"accepted"`
	Reason             string `json:"reason,omitempty"`
	ExpectedWage       int    `json:"expectedWage"`
	MaxAcceptableYears int    `json:"maxAcceptableYears"`
	OfferId            int64  `json:"offerId,omitempty"`
}

// PreContractMinimumWage is the least a pre-contract offer must pay a player of negotiation tier
// intelligence whose usual expectation is expected: expected × PreContractPremium(intelligence),
// rounded up. Pure.
func PreContractMinimumWage(expected int, intelligence int) int {
	return int(math.Ceil(float64(expected)*PreContractPremium(intelligence) - 1e-9))
}

// PreContractKeepWage is the wage the club of a player of negotiation tier intelligence must offer
// to keep him while best is the best pending pre-contract wage: best ÷ PreContractPremium,
// rounded up; 0 without offers. Pure.
func PreContractKeepWage(best int, intelligence int) int {
	if best <= 0 {
		return 0
	}
	return int(math.Ceil(float64(best)/PreContractPremium(intelligence) - 1e-9))
}

// PreContractSeasonOpen reports whether a league season with played matches played and remaining
// to play is far enough along for pre-contracts. Pure.
func PreContractSeasonOpen(played, remaining int) bool {
	total := played + remaining
	return total > 0 && float64(played) >= PreContractSeasonShare*float64(total)
}

// PreContractDecisionTime is when a player whose first pending offer arrived at first decides.
func PreContractDecisionTime(first time.Time) time.Time {
	return first.Add(PreContractDecisionDelay)
}

// IsPlayerPreContracted reports whether playerId has signed a pre-contract.
func IsPlayerPreContracted(q sqlQuerier, playerId int64) (bool, error) {
	var signed bool
	err := q.QueryRow(isPlayerPreContractedStmtSql, playerId).Scan(&signed)
	return signed, err
}

// PreContractCounterBar is the renewal wage that keeps playerId while pre-contract offers on him
// are pending (see PreContractKeepWage), 0 when none is.
func PreContractCounterBar(q sqlQuerier, playerId int64) (int, error) {
	var best int
	var first sql.NullTime
	if err := q.QueryRow(bestPendingPreContractStmtSql, playerId).Scan(&best, &first); err != nil {
		return 0, err
	}
	return PreContractKeepWage(best, negotiationTier(q, playerId)), nil
}

// negotiationTierStmtSql reads a player's negotiation tier.
const negotiationTierStmtSql = "SELECT intelligence FROM players WHERE player_id = $1"

// negotiationTier is playerId's negotiation tier (see IntelligenceNaive..IntelligenceMastermind),
// 0 - read as Aware - when it can't be read.
func negotiationTier(q sqlQuerier, playerId int64) int {
	var tier int
	if err := q.QueryRow(negotiationTierStmtSql, playerId).Scan(&tier); err != nil {
		return 0
	}
	return tier
}

// CountAiPreContractOffersOnPlayer is how many CPU clubs have a pending pre-contract offer on him.
func CountAiPreContractOffersOnPlayer(dbCtx *DatabaseContext, playerId int64) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow(countAiPreContractOffersOnPlayerStmtSql, playerId).Scan(&n)
	return n, err
}

// CountSignedPreContracts is how many signed pre-contracts teamId holds.
func CountSignedPreContracts(q sqlQuerier, teamId int64) (int, error) {
	var n int
	err := q.QueryRow(countSignedPreContractsStmtSql, teamId).Scan(&n)
	return n, err
}

// MakePreContractOffer offers playerId a contract with fromTeamId for next season (openspec/
// specs/pre-contracts "Making a pre-contract offer" and "Player demands"). The player judges it at
// once: below PreContractMinimumWage of his transfer expectation, or longer than his age allows,
// nothing is created and the result reports what he would accept. His club is told the buyer,
// the terms, when he decides and the wage that keeps him.
func MakePreContractOffer(dbCtx *DatabaseContext, playerId int64, fromTeamId int64, wage int, years int) (*PreContractOfferResult, error) {
	return MakePreContractOfferWithClause(dbCtx, playerId, fromTeamId, wage, years, nil)
}

// MakePreContractOfferWithClause is MakePreContractOffer whose contract also carries an optional
// release clause (nil for none), which lowers the wage the player asks (see DiscountedWage).
func MakePreContractOfferWithClause(dbCtx *DatabaseContext, playerId int64, fromTeamId int64, wage int, years int, clause *int64) (*PreContractOfferResult, error) {
	if wage <= 0 || years < 1 || years > MaxContractYearsRemaining {
		return nil, ErrInvalidPreContractTerms
	}
	if !IsTransferWindowOpen(dbCtx) {
		return nil, ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, fromTeamId) {
		return nil, ErrTeamInsolvent
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ownerTeamId, _, contractYears, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ownerTeamId.Valid {
		return nil, ErrPlayerIsFreeAgent
	}
	if ownerTeamId.Int64 == fromTeamId {
		return nil, ErrCannotOfferOnOwnPlayer
	}
	if contractYears != 1 {
		return nil, ErrPreContractNotLastSeason
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if onLoan {
		return nil, ErrPlayerOnLoan
	}
	if signed, err := IsPlayerPreContracted(tx, playerId); err != nil {
		return nil, err
	} else if signed {
		return nil, ErrPlayerPreContracted
	}
	situation, err := GetTeamSeasonSituation(dbCtx, ownerTeamId.Int64)
	if err != nil {
		return nil, err
	}
	if !PreContractSeasonOpen(situation.Played, situation.Remaining) {
		return nil, ErrPreContractTooEarly
	}
	if n, err := CountSignedPreContracts(tx, fromTeamId); err != nil {
		return nil, err
	} else if n >= MaxSignedPreContracts {
		return nil, ErrPreContractLimit
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if err := ValidateReleaseClause(*player, clause); err != nil {
		return nil, err
	}
	expected, maxYears, err := PlayerTermsExpectation(dbCtx, *player, TermsTransfer)
	if err != nil {
		return nil, err
	}
	tier := negotiationTier(tx, playerId)
	expected = DiscountedWage(*player, expected, clause, tier)
	result := &PreContractOfferResult{ExpectedWage: PreContractMinimumWage(expected, tier), MaxAcceptableYears: maxYears}
	var reasons []string
	if wage < result.ExpectedWage {
		reasons = append(reasons, "wage below what they want to leave their club")
	}
	if years > maxYears {
		reasons = append(reasons, "too many seasons for their age")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
		return result, nil
	}

	var offerId int64
	if err := tx.QueryRow(insertPreContractOfferStmtSql, playerId, fromTeamId, wage, years, clauseArg(clause)).Scan(&offerId); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPreContractAlreadyOffered
		}
		return nil, err
	}
	var best int
	var first sql.NullTime
	if err := tx.QueryRow(bestPendingPreContractStmtSql, playerId).Scan(&best, &first); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result.Accepted = true
	result.OfferId = offerId

	buyerName := teamNameOrEmpty(dbCtx, fromTeamId)
	playerName := player.FirstName + " " + player.LastName
	hours := int(math.Ceil(time.Until(PreContractDecisionTime(first.Time)).Hours()))
	if hours < 0 {
		hours = 0
	}
	_ = NotifyTeamOwnerWithLink(dbCtx, ownerTeamId.Int64, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
		Text: buyerName + " offered " + playerName + " a pre-contract: " + FormatFee(int64(wage)) + " for " + strconv.Itoa(years) +
			" season(s). He decides in " + strconv.Itoa(hours) + " hours; renew him at " + FormatFee(int64(PreContractKeepWage(best, tier))) + " to keep him",
		Key: "notifications.messages.preContractOffered",
		Params: map[string]any{"team": buyerName, "player": playerName, "wage": FormatFee(int64(wage)), "years": years,
			"hours": hours, "keepWage": FormatFee(int64(PreContractKeepWage(best, tier)))},
	}, "/transfers?tab=offers")
	return result, nil
}

// WithdrawPreContractOffer withdraws one of fromTeamId's pending pre-contract offers.
func WithdrawPreContractOffer(dbCtx *DatabaseContext, offerId int64, fromTeamId int64) error {
	res, err := dbCtx.Db.Exec(withdrawPreContractOfferStmtSql, offerId, fromTeamId)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPreContractNotFound
	}
	return nil
}

// cancelPendingPreContracts deletes, inside tx, every pending pre-contract offer on playerId and
// returns the clubs that made them, to be told after the commit (see notifyPreContractsCancelled)
// - his club renewed, sold or loaned him (openspec/specs/pre-contracts "Right to counter-offer").
func cancelPendingPreContracts(tx *sql.Tx, playerId int64) ([]int64, error) {
	rows, err := tx.Query(deletePendingPreContractsStmtSql, playerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buyers := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		buyers = append(buyers, id)
	}
	return buyers, rows.Err()
}

// notifyPreContractsCancelled tells every club in buyers its pre-contract offer on playerId is off.
func notifyPreContractsCancelled(dbCtx *DatabaseContext, playerId int64, buyers []int64) {
	if len(buyers) == 0 {
		return
	}
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return
	}
	playerName := player.FirstName + " " + player.LastName
	for _, buyer := range buyers {
		_ = NotifyTeamOwnerWithLink(dbCtx, buyer, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   "Your pre-contract offer for " + playerName + " is off: he renewed his contract or is no longer available",
			Key:    "notifications.messages.preContractCancelled",
			Params: map[string]any{"player": playerName},
		}, "/transfers?tab=offers")
	}
}

func teamNameOrEmpty(dbCtx *DatabaseContext, teamId int64) string {
	if teamId == 0 {
		return ""
	}
	team, err := GetTeamById(teamId, dbCtx)
	if err != nil {
		return ""
	}
	return team.TeamName
}

// GetPreContractOffersReceived returns every pre-contract offer, pending or signed, on teamId's
// players, each with the decision time and the wage that keeps the player.
func GetPreContractOffersReceived(dbCtx *DatabaseContext, teamId int64) ([]PreContractOffer, error) {
	offers, err := queryPreContractOffers(dbCtx, getPreContractOffersReceivedStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	for i := range offers {
		if offers[i].SignedAt != nil {
			offers[i].KeepWage = 0
		}
	}
	return offers, nil
}

// GetPreContractOffersSent returns every pre-contract offer teamId made, pending or signed. The
// players belong to other clubs, so their financials and skills are redacted as anywhere else, and
// the wage that keeps them is not reported.
func GetPreContractOffersSent(dbCtx *DatabaseContext, teamId int64) ([]PreContractOffer, error) {
	offers, err := queryPreContractOffers(dbCtx, getPreContractOffersSentStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	for i := range offers {
		offers[i].KeepWage = 0
		wage := offers[i].Player.Wage
		offers[i].Player.RedactFinancials()
		offers[i].Player.Wage = wage
	}
	return offers, nil
}

func queryPreContractOffers(dbCtx *DatabaseContext, query string, teamId int64) ([]PreContractOffer, error) {
	rows, err := dbCtx.Db.Query(query, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offers := make([]PreContractOffer, 0)
	playerIds := make([]int64, 0)
	for rows.Next() {
		var o PreContractOffer
		var playerId int64
		var signedAt, first sql.NullTime
		var best int
		if err := rows.Scan(&o.OfferId, &playerId, &o.FromTeamId, &o.FromTeamName, &o.ToTeamId, &o.ToTeamName,
			&o.Wage, &o.Years, &o.CreatedAt, &signedAt, &first, &best); err != nil {
			return nil, err
		}
		if signedAt.Valid {
			t := signedAt.Time
			o.SignedAt = &t
		} else if first.Valid {
			t := PreContractDecisionTime(first.Time)
			o.DecidesAt = &t
			o.KeepWage = PreContractKeepWage(best, negotiationTier(dbCtx.Db, playerId))
		}
		o.Player.Id = playerId
		offers = append(offers, o)
		playerIds = append(playerIds, playerId)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range offers {
		p, err := GetPlayerById(dbCtx, playerIds[i])
		if err != nil {
			return nil, err
		}
		offers[i].Player = *p
	}
	return offers, nil
}

type pendingPreContract struct {
	offerId    int64
	fromTeamId int64
	wage       int
	years      int
	clause     *int64
}

// DecidePreContracts runs the pre-contract clock (openspec/specs/pre-contracts "Signing a
// pre-contract" and "Pre-contract notifications"): it first reminds every owning club whose
// player decides within PreContractReminderLead, then lets every player whose first pending offer
// is PreContractDecisionDelay old sign the best one. Each player is decided under his row lock, so
// a decision never races a renewal, and a second replica finds nothing left to decide.
func DecidePreContracts(dbCtx *DatabaseContext, now time.Time) {
	remindPreContractDecisions(dbCtx, now)

	rows, err := dbCtx.Db.Query(playersDueForPreContractDecisionStmtSql, now.Add(-PreContractDecisionDelay))
	if err != nil {
		log.Errorf("error listing players due a pre-contract decision: %s", err)
		return
	}
	due := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			due = append(due, id)
		}
	}
	rows.Close()
	for _, playerId := range due {
		if err := decidePreContract(dbCtx, playerId); err != nil {
			log.Errorf("error deciding the pre-contract offers on player %d: %s", playerId, err)
		}
	}
}

func remindPreContractDecisions(dbCtx *DatabaseContext, now time.Time) {
	rows, err := dbCtx.Db.Query(claimPreContractRemindersStmtSql, now.Add(-(PreContractDecisionDelay - PreContractReminderLead)))
	if err != nil {
		log.Errorf("error claiming pre-contract reminders: %s", err)
		return
	}
	players := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			players = append(players, id)
		}
	}
	rows.Close()
	for _, playerId := range players {
		player, err := GetPlayerById(dbCtx, playerId)
		if err != nil || player.TeamId == 0 {
			continue
		}
		bar, err := PreContractCounterBar(dbCtx.Db, playerId)
		if err != nil {
			continue
		}
		playerName := player.FirstName + " " + player.LastName
		_ = NotifyTeamOwnerWithLink(dbCtx, player.TeamId, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " decides on his pre-contract offers within 24 hours; renew him at " + FormatFee(int64(bar)) + " to keep him",
			Key:    "notifications.messages.preContractDecisionSoon",
			Params: map[string]any{"player": playerName, "keepWage": FormatFee(int64(bar))},
		}, "/transfers?tab=offers")
	}
}

func decidePreContract(dbCtx *DatabaseContext, playerId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ownerTeamId, _, contractYears, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	offers, err := lockPendingPreContracts(tx, playerId)
	if err != nil {
		return err
	}
	if len(offers) == 0 {
		return nil
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return err
	}
	signedAlready, err := IsPlayerPreContracted(tx, playerId)
	if err != nil {
		return err
	}

	var chosen *pendingPreContract
	if ownerTeamId.Valid && contractYears == 1 && !onLoan && !signedAlready {
		player, err := GetPlayerById(dbCtx, playerId)
		if err != nil {
			return err
		}
		expected, maxYears, err := PlayerTermsExpectation(dbCtx, *player, TermsTransfer)
		if err != nil {
			return err
		}
		tier := negotiationTier(tx, playerId)
		for i := range offers {
			o := offers[i]
			// A clause in the offered contract lowers what he asks (see DiscountedWage).
			if o.wage < PreContractMinimumWage(DiscountedWage(*player, expected, o.clause, tier), tier) || o.years > maxYears {
				continue
			}
			signed, err := CountSignedPreContracts(tx, o.fromTeamId)
			if err != nil {
				return err
			}
			if signed >= MaxSignedPreContracts {
				continue
			}
			chosen = &o
			break
		}
	}

	losers := make([]int64, 0, len(offers))
	for _, o := range offers {
		if chosen != nil && o.offerId == chosen.offerId {
			continue
		}
		if _, err := tx.Exec(deletePreContractOfferStmtSql, o.offerId); err != nil {
			return err
		}
		losers = append(losers, o.fromTeamId)
	}
	if chosen != nil {
		if _, err := tx.Exec(signPreContractStmtSql, chosen.offerId); err != nil {
			return err
		}
		// He is leaving: nothing else can be agreed for him with his club now.
		if _, err := tx.Exec(DeleteOffersForPlayerStmtSql, playerId); err != nil {
			return err
		}
		if _, err := tx.Exec(DeleteTransferListingByPlayerAndTeamStmtSql, playerId, ownerTeamId.Int64); err != nil {
			return err
		}
		if _, err := tx.Exec(DeleteLoanListingByPlayerAndTeamStmtSql, playerId, ownerTeamId.Int64); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil
	}
	playerName := player.FirstName + " " + player.LastName
	clubName := teamNameOrEmpty(dbCtx, player.TeamId)
	if chosen != nil {
		buyerName := teamNameOrEmpty(dbCtx, chosen.fromTeamId)
		_ = NotifyTeamOwnerWithLink(dbCtx, chosen.fromTeamId, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " signed your pre-contract: he joins at the end of the season",
			Key:    "notifications.messages.preContractSigned",
			Params: map[string]any{"player": playerName, "team": clubName},
		}, "/transfers?tab=offers")
		_ = NotifyTeamOwnerWithLink(dbCtx, player.TeamId, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " signed a pre-contract with " + buyerName + ": he leaves at the end of the season",
			Key:    "notifications.messages.preContractSignedAway",
			Params: map[string]any{"player": playerName, "team": buyerName},
		}, "/transfers?tab=offers")
	}
	for _, loser := range losers {
		_ = NotifyTeamOwnerWithLink(dbCtx, loser, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " turned down your pre-contract offer",
			Key:    "notifications.messages.preContractRejected",
			Params: map[string]any{"player": playerName},
		}, "/transfers?tab=offers")
	}
	return nil
}

func lockPendingPreContracts(tx *sql.Tx, playerId int64) ([]pendingPreContract, error) {
	rows, err := tx.Query(lockPendingPreContractsStmtSql, playerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offers := make([]pendingPreContract, 0)
	for rows.Next() {
		var o pendingPreContract
		var created time.Time
		var clause sql.NullInt64
		if err := rows.Scan(&o.offerId, &o.fromTeamId, &o.wage, &o.years, &created, &clause); err != nil {
			return nil, err
		}
		if clause.Valid {
			amount := clause.Int64
			o.clause = &amount
		}
		offers = append(offers, o)
	}
	return offers, rows.Err()
}

// completePreContractTx moves playerId, whose contract has just run out, to the club he signed a
// pre-contract with, on the agreed terms, inside tx (see AdvancePlayerContract). It reports the
// buying club, 0 when he signed none.
func completePreContractTx(tx *sql.Tx, playerId int64, fromTeamId int64) (int64, error) {
	var offerId, buyer int64
	var wage, years int
	var clause sql.NullInt64
	err := tx.QueryRow(signedPreContractStmtSql, playerId).Scan(&offerId, &buyer, &wage, &years, &clause)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return 0, err
	}
	if err := movePlayerToTeam(tx, buyer, playerId); err != nil {
		return 0, err
	}
	var clauseValue any
	if clause.Valid {
		clauseValue = clause.Int64
	}
	if _, err := tx.Exec(ApplyRenewalOfferStmtSql, years, wage, playerId, clauseValue); err != nil {
		return 0, err
	}
	if err := RecordPlayerClubHistory(tx, playerId, buyer); err != nil {
		return 0, err
	}
	if err := RecordPlayerTransfer(tx, playerId, fromTeamId, buyer, 0, PLAYER_TRANSFER_PRE_CONTRACT); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(deletePreContractOfferStmtSql, offerId); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(deletePendingPreContractsStmtSql, playerId); err != nil {
		return 0, err
	}
	return buyer, nil
}

// notifyPreContractJoined tells both clubs and the player's followers that he moved at the rollover.
func notifyPreContractJoined(dbCtx *DatabaseContext, playerId int64, fromTeamId int64, buyer int64) {
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return
	}
	playerName := player.FirstName + " " + player.LastName
	buyerName := teamNameOrEmpty(dbCtx, buyer)
	fromName := teamNameOrEmpty(dbCtx, fromTeamId)
	for _, team := range []int64{buyer, fromTeamId} {
		_ = NotifyTeamOwner(dbCtx, team, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " joined " + buyerName + " from " + fromName + " on a pre-contract",
			Key:    "notifications.messages.preContractJoined",
			Params: map[string]any{"player": playerName, "team": buyerName, "from": fromName},
		})
	}
	NotifyWatchers(dbCtx, playerId, WatchEventMoved, buyerName, 0, fromTeamId)
}

// LapsePreContractsOnRetirement tells every club with a pre-contract offer on playerId, pending or
// signed, that it lapsed because he is retiring; the rows themselves go with the player's.
func LapsePreContractsOnRetirement(dbCtx *DatabaseContext, playerId int64) {
	rows, err := dbCtx.Db.Query("SELECT from_team_id FROM pre_contract_offers WHERE player_id = $1", playerId)
	if err != nil {
		return
	}
	buyers := make([]int64, 0)
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			buyers = append(buyers, id)
		}
	}
	rows.Close()
	if len(buyers) == 0 {
		return
	}
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return
	}
	playerName := player.FirstName + " " + player.LastName
	for _, buyer := range buyers {
		_ = NotifyTeamOwner(dbCtx, buyer, NOTIFICATION_PRE_CONTRACT, NotificationMessage{
			Text:   playerName + " retired: your pre-contract with him lapsed",
			Key:    "notifications.messages.preContractRetired",
			Params: map[string]any{"player": playerName},
		})
	}
}

// deleteTeamPreContractOffersStmtSql drops every pre-contract offer a club made - its reset (see
// ResetReleasedTeam).
const deleteTeamPreContractOffersStmtSql = "DELETE FROM pre_contract_offers WHERE from_team_id = $1"

const getPlayersWithPreContractOfferFromTeamStmtSql = "SELECT player_id FROM pre_contract_offers WHERE from_team_id = $1"

// GetPlayersWithPreContractOfferFromTeam is the set of players teamId has a pre-contract offer on,
// pending or signed.
func GetPlayersWithPreContractOfferFromTeam(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(getPlayersWithPreContractOfferFromTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

const getTeamPlayersWithPendingPreContractsStmtSql = "SELECT DISTINCT o.player_id FROM pre_contract_offers o " +
	"JOIN players p ON p.player_id = o.player_id WHERE p.team_id = $1 AND o.signed_at IS NULL"

// GetTeamPlayersWithPendingPreContracts is the set of teamId's players with a pending pre-contract
// offer - those a CPU club answers on its offer sweep.
func GetTeamPlayersWithPendingPreContracts(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(getTeamPlayersWithPendingPreContractsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// PlayerRenewalExpectation is what playerId expects from his own club for a renewal right now:
// PlayerTermsExpectation for a renewal, raised to PreContractCounterBar while pre-contract offers on
// him are pending (contested is then true) - what a CPU club offers to keep him.
func PlayerRenewalExpectation(dbCtx *DatabaseContext, p Player) (wage int, years int, contested bool, err error) {
	wage, years, err = PlayerTermsExpectation(dbCtx, p, TermsRenewal)
	if err != nil {
		return 0, 0, false, err
	}
	bar, err := PreContractCounterBar(dbCtx.Db, p.Id)
	if err != nil {
		return 0, 0, false, err
	}
	if bar > wage {
		wage = bar
	}
	return wage, years, bar > 0, nil
}

// PreContractMarketSize is how many last-season players the pre-contract market lists.
const PreContractMarketSize = 100

// PreContractMarketEntry is one player in the pre-contract market (openspec/specs/pre-contracts
// "Pre-contract market"): another club's player in the last season of his contract, seen as on
// any other club's squad page (skills and contract terms hidden, wage, average skill and clause
// public), with his best position, whether his league is far enough along for an offer and
// whether the viewer already made him one.
type PreContractMarketEntry struct {
	Player       Player `json:"player"`
	TeamName     string `json:"teamName"`
	BestPosition int    `json:"bestPosition"`
	Open         bool   `json:"open"`
	Offered      bool   `json:"offered"`
}

const getPreContractMarketStmtSql = "SELECT " + playerColumnsSql + ", t.name, " +
	"COALESCE((SELECT cl.matches FROM classification cl WHERE cl.team_id = t.team_id LIMIT 1), 0), " +
	"(SELECT count(*) FROM matches m WHERE m.league_id = t.league_id AND m.season = co.current_season AND m.played = false " +
	"AND (m.home_team_id = t.team_id OR m.away_team_id = t.team_id)), " +
	"EXISTS (SELECT 1 FROM pre_contract_offers o WHERE o.player_id = p.player_id AND o.from_team_id = $1) " +
	"FROM players p JOIN teams t ON t.team_id = p.team_id JOIN leagues l ON l.league_id = t.league_id " +
	"JOIN countries co ON co.name = l.country " +
	"WHERE p.contract_years_remaining = 1 AND p.team_id <> $1 AND p.in_academy = false AND t.is_national_team = false " +
	"AND NOT EXISTS (SELECT 1 FROM player_loans pl WHERE pl.player_id = p.player_id) " +
	"AND NOT EXISTS (SELECT 1 FROM pre_contract_offers s WHERE s.player_id = p.player_id AND s.signed_at IS NOT NULL) " +
	"ORDER BY (p.goal_keeper + p.defense + p.\"lateral\" + p.pass + p.midfield + p.dribbling + p.shot) DESC, p.player_id LIMIT $2"

// GetPreContractMarket lists the best PreContractMarketSize players of other clubs than
// viewerTeamId in the last season of their contract, not on loan and not already signed by anyone,
// best first.
func GetPreContractMarket(dbCtx *DatabaseContext, viewerTeamId int64) ([]PreContractMarketEntry, error) {
	rows, err := dbCtx.Db.Query(getPreContractMarketStmtSql, viewerTeamId, PreContractMarketSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]PreContractMarketEntry, 0)
	for rows.Next() {
		var e PreContractMarketEntry
		var played, remaining int
		player, err := scanPlayer(rows, &e.TeamName, &played, &remaining, &e.Offered)
		if err != nil {
			return nil, err
		}
		e.BestPosition = BestCorePosition(*player)
		e.Open = PreContractSeasonOpen(played, remaining)
		player.AverageSkill = AverageSkill(*player)
		wage := player.Wage
		player.RedactFinancials()
		player.Wage = wage
		player.RedactSkills(false)
		e.Player = *player
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
