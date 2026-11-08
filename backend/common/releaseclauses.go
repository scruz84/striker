package common

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"
)

// Release clauses (openspec/specs/release-clauses): an optional, public fee in a player's contract
// that any club may pay to sign him without his club's consent. A low clause buys a cheaper wage.
const (
	// MinReleaseClauseMultiple is the lowest clause allowed, as a multiple of ClauseBaseValue.
	MinReleaseClauseMultiple = 1.5
	// A clause of at most releaseClauseLowMultiple x ClauseBaseValue lowers the wage the player asks
	// by his tier's ClauseDiscountFor; one of at most releaseClauseMidMultiple x by half of it.
	releaseClauseLowMultiple = 2.0
	releaseClauseMidMultiple = 4.0
	// ReleaseClauseLockIn is how long after a contract was agreed its clause can first be paid.
	ReleaseClauseLockIn = 14 * 24 * time.Hour
	// AiReleaseClauseCooldown is how long after a CPU club paid a clause on a human club's player no
	// CPU club pays another on that club's players.
	AiReleaseClauseCooldown = 7 * 24 * time.Hour
)

var ErrReleaseClauseTooLow = errors.New("a release clause must be at least 1.5 times the player's market value")
var ErrNoReleaseClause = errors.New("player has no release clause")
var ErrReleaseClauseLockedIn = errors.New("the release clause can't be paid yet: the contract is too recent")

// ClauseBaseValue is the yardstick a clause is measured against: the player's market value on a
// fresh 3-season contract.
func ClauseBaseValue(p Player) int64 {
	fresh := p
	fresh.ContractYearsRemaining = MaxContractYearsRemaining
	return MarketValue(fresh)
}

// MinReleaseClause is the lowest clause p's contract may carry.
func MinReleaseClause(p Player) int64 {
	return int64(math.Ceil(float64(ClauseBaseValue(p)) * MinReleaseClauseMultiple))
}

// ReleaseClauseDiscount is the share of his expected wage a player of negotiation tier
// intelligence gives up for a clause of clause against a base value of base: ClauseDiscountFor up
// to 2x (10% for an Aware player), half of it up to 4x, none above or without a clause. Pure.
func ReleaseClauseDiscount(clause *int64, base int64, intelligence int) float64 {
	if clause == nil || base <= 0 {
		return 0
	}
	switch ratio := float64(*clause) / float64(base); {
	case ratio <= releaseClauseLowMultiple:
		return ClauseDiscountFor(intelligence)
	case ratio <= releaseClauseMidMultiple:
		return ClauseDiscountFor(intelligence) / 2
	default:
		return 0
	}
}

// ValidateReleaseClause checks a proposed clause for p's new contract: nil (no clause) or at least
// MinReleaseClause.
func ValidateReleaseClause(p Player, clause *int64) error {
	if clause != nil && *clause < MinReleaseClause(p) {
		return ErrReleaseClauseTooLow
	}
	return nil
}

// DiscountedWage is the wage p, of negotiation tier intelligence, accepts instead of expected when
// his new contract carries clause (see ReleaseClauseDiscount), rounded up. Pure.
func DiscountedWage(p Player, expected int, clause *int64, intelligence int) int {
	discount := ReleaseClauseDiscount(clause, ClauseBaseValue(p), intelligence)
	if discount == 0 {
		return expected
	}
	return int(math.Ceil(float64(expected) * (1 - discount)))
}

// clauseArg turns an optional clause into its SQL argument (NULL without one).
func clauseArg(clause *int64) any {
	if clause == nil {
		return nil
	}
	return *clause
}

// setReleaseClause fills p's clause fields from the nullable columns of playerColumnsSql.
func (p *Player) setReleaseClause(clause sql.NullInt64, setAt sql.NullTime) {
	if !clause.Valid {
		return
	}
	amount := clause.Int64
	p.ReleaseClause = &amount
	payable := setAt.Time.Add(ReleaseClauseLockIn)
	if !setAt.Valid {
		payable = time.Time{}
	}
	p.ReleaseClausePayableAt = &payable
}

// ReleaseClauseResult mirrors TransferOfferResult: the player judges the personal terms at once,
// so a refusal changes nothing and reports what he wants.
type ReleaseClauseResult struct {
	Accepted           bool   `json:"accepted"`
	Reason             string `json:"reason,omitempty"`
	ExpectedWage       int    `json:"expectedWage"`
	MaxAcceptableYears int    `json:"maxAcceptableYears"`
	Fee                int64  `json:"fee,omitempty"`
}

const lockPlayerClauseStmtSql = "SELECT COALESCE(team_id, 0), release_clause, release_clause_set_at FROM players WHERE player_id = $1 " + ForUpdate

// PayReleaseClause has buyerTeamId pay playerId's release clause in full and sign him at once on
// wage/years (and newClause for his new contract), without his club's consent (openspec/specs/
// release-clauses "Paying a clause"). Everything is validated under the player's and the buyer's
// row locks, so two clubs paying the same clause at once can't both succeed: the second finds
// the player already moved, on a fresh contract still inside its lock-in.
func PayReleaseClause(dbCtx *DatabaseContext, playerId int64, buyerTeamId int64, wage int, years int, newClause *int64) (*ReleaseClauseResult, error) {
	if wage <= 0 || years < 1 || years > MaxContractYearsRemaining {
		return nil, ErrInvalidPersonalTerms
	}
	if !IsTransferWindowOpen(dbCtx) {
		return nil, ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, buyerTeamId) {
		return nil, ErrTeamInsolvent
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var sellerTeamId int64
	var clause sql.NullInt64
	var setAt sql.NullTime
	if err := tx.QueryRow(lockPlayerClauseStmtSql, playerId).Scan(&sellerTeamId, &clause, &setAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}
	if sellerTeamId == 0 {
		return nil, ErrPlayerIsFreeAgent
	}
	if sellerTeamId == buyerTeamId {
		return nil, ErrCannotOfferOnOwnPlayer
	}
	if !clause.Valid {
		return nil, ErrNoReleaseClause
	}
	if setAt.Valid && time.Now().Before(setAt.Time.Add(ReleaseClauseLockIn)) {
		return nil, ErrReleaseClauseLockedIn
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

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if err := ValidateReleaseClause(*player, newClause); err != nil {
		return nil, err
	}
	expected, maxYears, err := PlayerTermsExpectation(dbCtx, *player, TermsTransfer)
	if err != nil {
		return nil, err
	}
	expected = DiscountedWage(*player, expected, newClause, negotiationTier(tx, playerId))
	result := &ReleaseClauseResult{ExpectedWage: expected, MaxAcceptableYears: maxYears}
	var reasons []string
	if wage < expected {
		reasons = append(reasons, "wage below what they expect")
	}
	if years > maxYears {
		reasons = append(reasons, "too many seasons for their age")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
		return result, nil
	}

	fee := clause.Int64
	var buyerBudget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, buyerTeamId).Scan(&buyerBudget); err != nil {
		return nil, err
	}
	if buyerBudget < fee {
		return nil, ErrInsufficientBudget
	}
	buyerTeam, err := GetTeamById(buyerTeamId, dbCtx)
	if err != nil {
		return nil, err
	}
	sellerTeam, err := GetTeamById(sellerTeamId, dbCtx)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -fee, buyerTeamId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, fee, sellerTeamId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return nil, err
	}
	if err := movePlayerToTeam(tx, buyerTeamId, playerId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ApplyRenewalOfferStmtSql, years, wage, playerId, clauseArg(newClause)); err != nil {
		return nil, err
	}
	if err := RecordPlayerClubHistory(tx, playerId, buyerTeamId); err != nil {
		return nil, err
	}
	if err := RecordPlayerTransfer(tx, playerId, sellerTeamId, buyerTeamId, fee, PLAYER_TRANSFER_RELEASE_CLAUSE); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(DeleteOffersForPlayerStmtSql, playerId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(DeleteTransferListingByPlayerAndTeamStmtSql, playerId, sellerTeamId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(DeleteLoanListingByPlayerAndTeamStmtSql, playerId, sellerTeamId); err != nil {
		return nil, err
	}
	preContractBuyers, err := cancelPendingPreContracts(tx, playerId)
	if err != nil {
		return nil, err
	}

	playerName := player.FirstName + " " + player.LastName
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, buyerTeamId, -fee, TRANSACTION_TRANSFER_OUT,
		"Paid the release clause of "+playerName+" to "+sellerTeam.TeamName, playerId); err != nil {
		return nil, err
	}
	if err := insertLedgerRow(tx, &flows, sellerTeamId, fee, TRANSACTION_TRANSFER_IN,
		"Release clause of "+playerName+" paid by "+buyerTeam.TeamName, playerId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	flows.Commit(context.Background())
	RecordTransferCompleted(context.Background(), fee)
	notifyPreContractsCancelled(dbCtx, playerId, preContractBuyers)

	_ = NotifyTeamOwner(dbCtx, sellerTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   buyerTeam.TeamName + " paid the " + FormatFee(fee) + " release clause of " + playerName + ": he has left",
		Key:    "notifications.messages.releaseClausePaid",
		Params: map[string]any{"team": buyerTeam.TeamName, "player": playerName, "fee": FormatFee(fee)},
	})
	_ = NotifyTeamOwner(dbCtx, buyerTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Signed " + playerName + " from " + sellerTeam.TeamName + " by paying his " + FormatFee(fee) + " release clause",
		Key:    "notifications.messages.releaseClauseSigned",
		Params: map[string]any{"team": sellerTeam.TeamName, "player": playerName, "fee": FormatFee(fee)},
	})
	NotifyWatchers(dbCtx, playerId, WatchEventMoved, buyerTeam.TeamName, 0, sellerTeamId, buyerTeamId)

	result.Accepted = true
	result.Fee = fee
	return result, nil
}

const lastCpuClausePaymentOnStmtSql = "SELECT MAX(pt.created_at) FROM player_transfers pt JOIN teams t ON t.team_id = pt.to_team_id " +
	"WHERE pt.from_team_id = $1 AND pt.kind = $2 AND t.owner_id IS NULL"

// CpuMayPayClauseOn reports whether a CPU club may pay a release clause on sellerTeamId's player
// now: always for a CPU seller, and for a human one only when no CPU club paid one on it in the
// last AiReleaseClauseCooldown (openspec/specs/cpu-transfer-market "Paying release clauses").
func CpuMayPayClauseOn(dbCtx *DatabaseContext, sellerTeamId int64, now time.Time) (bool, error) {
	owner, err := GetTeamOwner(dbCtx, sellerTeamId)
	if err != nil {
		return false, err
	}
	if owner <= 0 {
		return true, nil
	}
	var last sql.NullTime
	if err := dbCtx.Db.QueryRow(lastCpuClausePaymentOnStmtSql, sellerTeamId, PLAYER_TRANSFER_RELEASE_CLAUSE).Scan(&last); err != nil {
		return false, err
	}
	return !last.Valid || now.Sub(last.Time) >= AiReleaseClauseCooldown, nil
}
