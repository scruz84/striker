package common

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// InitialContractMinYears/MaxYears bound the randomly assigned length of a freshly created
// player's first contract (see doCreatePlayers) and a free agent's fresh deal on signing (see
// SignFreeAgent).
const InitialContractMinYears = 2
const InitialContractMaxYears = 3

// MaxContractYearsRemaining is the longest a contract can ever run, initial signing or renewal
// alike - a team can't lock a player up for more than three seasons at a time.
const MaxContractYearsRemaining = 3

// renewalMaxYearsRetirementAge mirrors engine.RetirementEligibleAge - kept as its own constant
// rather than importing the engine package from here (which already imports common) to avoid a
// cycle. A player this age or older won't agree to a renewal offer longer than one season; see
// MaxAcceptableRenewalYears.
const renewalMaxYearsRetirementAge = 33

// EarlyReleaseWageMultiplier sets the severance cost of releasing a player before their
// contract expires (see ReleasePlayer): a flat multiple of their current wage figure, standing
// in for a real buyout-clause calculation.
const EarlyReleaseWageMultiplier = 10

// RandomContractYears picks a random initial contract length in [InitialContractMinYears,
// InitialContractMaxYears].
func RandomContractYears() int {
	return InitialContractMinYears + rand.Intn(InitialContractMaxYears-InitialContractMinYears+1)
}

var ErrPlayerNotFound = errors.New("player not found")
var ErrPlayerNotFree = errors.New("player is not a free agent")
var ErrPlayerNotOwnedByTeam = errors.New("player does not belong to your team")
var ErrInvalidRenewalOffer = errors.New("renewal offer must be a positive wage for 1 to " + strconv.Itoa(MaxContractYearsRemaining) + " seasons")

const GetFreeAgentsStmtSql = "SELECT player_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"free_agent_seasons, COALESCE(last_team_id, 0) " +
	"FROM players WHERE team_id IS NULL " +
	"ORDER BY (goal_keeper + defense + \"lateral\" + pass + midfield + dribbling + shot) DESC"

// GetPlayerOwnershipForUpdateStmtSql locks a player row and returns just enough to validate a
// contract action: who (if anyone) currently employs them and their wage (needed for
// ReleasePlayer's severance). team_id comes back NULL for a free agent.
const GetPlayerOwnershipForUpdateStmtSql = "SELECT team_id, wage, contract_years_remaining FROM players WHERE player_id = $1 " + ForUpdate

// free_agent_seasons resets to 0 on signing - see engine.ProgressFreeAgents, which increments it
// once per season a player stays unsigned, and AdvanceFreeAgencyClock's doc comment.
const SignFreeAgentStmtSql = "UPDATE players SET team_id = $1, contract_years_remaining = $2, wage = $3, free_agent_seasons = 0 WHERE player_id = $4"
const ApplyRenewalOfferStmtSql = "UPDATE players SET contract_years_remaining = $1, wage = $2 WHERE player_id = $3"

// ReleaseFromTeamStmtSql captures the releasing club into last_team_id before clearing team_id -
// see SignFreeAgent, which charges that club a transfer premium if this player is re-signed
// within their first season of free agency.
const ReleaseFromTeamStmtSql = "UPDATE players SET last_team_id = team_id, team_id = NULL, contract_years_remaining = 0 WHERE player_id = $1"

// GetFreeAgents returns every unowned player, best-skilled first - both those released after
// their contract expired (see AdvancePlayerContract) and any player orphaned by
// common.AssignTeam reassigning a CPU team's squad to a new human owner.
func GetFreeAgents(dbCtx *DatabaseContext) ([]*Player, error) {
	rows, err := dbCtx.Db.Query(GetFreeAgentsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]*Player, 0)
	for rows.Next() {
		var p Player
		var birthDay int64
		var avatar string
		var birthCountry sql.NullString
		if err := rows.Scan(&p.Id, &p.FirstName, &p.LastName, &birthDay,
			&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot,
			&p.Header, &p.Quick, &p.Creative, &p.Technical, &p.Wage, &p.Age, &avatar, &p.ContractYearsRemaining, &birthCountry,
			&p.FreeAgentSeasons, &p.LastTeamId); err != nil {
			return nil, err
		}
		p.BirthDay = time.UnixMilli(birthDay)
		p.Avatar = GetAvatarFromString(avatar)
		p.BirthCountry = birthCountry.String
		players = append(players, &p)
	}
	return players, nil
}

func lockPlayerOwnership(tx *sql.Tx, playerId int64) (teamId sql.NullInt64, wage int, contractYearsRemaining int, err error) {
	err = tx.QueryRow(GetPlayerOwnershipForUpdateStmtSql, playerId).Scan(&teamId, &wage, &contractYearsRemaining)
	return
}

// FreeAgentPremiumWageMultiplier sets the transfer premium a team pays a free agent's most
// recent club (see players.last_team_id) when signing them within their first season of free
// agency (free_agent_seasons == 0) - a flat multiple of the newly agreed wage, smaller than
// EarlyReleaseWageMultiplier's full severance since this only compensates that club for losing
// the player to someone else, not for tearing up a contract early. No premium is owed once a
// player has gone a full season unsigned (nobody "poached" them from anywhere current), or if
// they have no recorded last club at all (see Player.LastTeamId).
const FreeAgentPremiumWageMultiplier = 4

var ErrInvalidFreeAgentOffer = errors.New("free agent offer must be a positive wage for 1 to " + strconv.Itoa(MaxContractYearsRemaining) + " seasons")

// FreeAgentSigningResult mirrors ContractRenewalOffer's shape - the player judges the offered
// terms on the spot (same expected-wage/age-based-max-years logic), so the response always
// carries a final verdict plus their real expectations, accepted or not, so the team can
// immediately retry with better terms.
type FreeAgentSigningResult struct {
	Accepted           bool   `json:"accepted"`
	Reason             string `json:"reason,omitempty"`
	ExpectedWage       int    `json:"expectedWage"`
	MaxAcceptableYears int    `json:"maxAcceptableYears"`
	// Only meaningful when Accepted is true.
	ContractYears int   `json:"contractYearsRemaining,omitempty"`
	Wage          int   `json:"wage,omitempty"`
	Premium       int64 `json:"premium,omitempty"`
}

// SignFreeAgent signs an unowned player to teamId on offeredWage/offeredYears, judged against the
// player's own expectations exactly like MakeContractRenewalOffer - if they wouldn't accept,
// nothing is charged or changed. If they would, and they're still within their first season of
// free agency (free_agent_seasons == 0) with a recorded last club, that club is paid a transfer
// premium (see FreeAgentPremiumWageMultiplier) - the signing team must be able to afford it, on
// top of the wage they're now taking on. The player row is locked for the duration so two teams
// can't both sign the same free agent.
func SignFreeAgent(dbCtx *DatabaseContext, playerId int64, teamId int64, offeredWage int, offeredYears int) (*FreeAgentSigningResult, error) {
	if offeredWage <= 0 || offeredYears < 1 || offeredYears > MaxContractYearsRemaining {
		return nil, ErrInvalidFreeAgentOffer
	}
	if !IsTransferWindowOpen(dbCtx) {
		return nil, ErrTransferWindowClosed
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	currentTeamId, _, _, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	if currentTeamId.Valid {
		return nil, ErrPlayerNotFree
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	expectedWage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	maxYears := MaxAcceptableRenewalYears(player.Age)

	result := &FreeAgentSigningResult{ExpectedWage: expectedWage, MaxAcceptableYears: maxYears}

	var reasons []string
	if offeredWage < expectedWage {
		reasons = append(reasons, "wage below what they expect")
	}
	if offeredYears > maxYears {
		reasons = append(reasons, "too many seasons for their age")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
		return result, nil
	}

	var premium int64
	if player.FreeAgentSeasons == 0 && player.LastTeamId != 0 {
		premium = int64(offeredWage) * FreeAgentPremiumWageMultiplier

		var buyerBudget int64
		if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, teamId).Scan(&buyerBudget); err != nil {
			return nil, err
		}
		if buyerBudget < premium {
			return nil, ErrInsufficientBudget
		}
		if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -premium, teamId); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(UpdateTeamBudgetStmtSql, premium, player.LastTeamId); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(SignFreeAgentStmtSql, teamId, offeredYears, offeredWage, playerId); err != nil {
		return nil, err
	}

	playerName := player.FirstName + " " + player.LastName
	if premium > 0 {
		if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, -premium, TRANSACTION_TRANSFER_OUT,
			"Paid free agent transfer premium for "+playerName, playerId); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(InsertTeamTransactionStmtSql, player.LastTeamId, premium, TRANSACTION_TRANSFER_IN,
			"Received free agent transfer premium for "+playerName, playerId); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	FreeAgentSigningsCounter.Add(context.Background(), 1)

	result.Accepted = true
	result.ContractYears = offeredYears
	result.Wage = offeredWage
	result.Premium = premium

	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRANSFER,
		"Signed free agent "+playerName+" to a "+strconv.Itoa(offeredYears)+"-season contract")
	if premium > 0 {
		_ = NotifyTeamOwner(dbCtx, player.LastTeamId, NOTIFICATION_TRANSFER,
			playerName+" was signed by another club - transfer premium received")
	}
	return result, nil
}

// SignFreeAgentAtMarketRate offers a free agent exactly their expected market wage (see
// PlayerWageFromSkills) for the longest contract their age allows - the terms an AI-managed team
// always offers (see engine.RunAITeamManagement), guaranteed to be accepted on personal terms
// since they match what SignFreeAgent itself computes as "expected". Still subject to the same
// transfer premium and budget checks as any human-negotiated signing.
func SignFreeAgentAtMarketRate(dbCtx *DatabaseContext, playerId int64, teamId int64) (*FreeAgentSigningResult, error) {
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	wage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	years := RandomContractYears()
	if maxYears := MaxAcceptableRenewalYears(player.Age); years > maxYears {
		years = maxYears
	}
	return SignFreeAgent(dbCtx, playerId, teamId, wage, years)
}

// ContractRenewalOffer is the outcome of a team proposing new terms to one of its own players
// (see MakeContractRenewalOffer). The player decides on the spot rather than an offer sitting
// pending for a human to answer, so the response always carries a final verdict - plus, either
// way, the player's actual expectations (ExpectedWage/MaxAcceptableYears) so the team can
// immediately try again with better terms if declined. That retry loop is the "negotiation" -
// there's no separate counter-offer step modeled server-side.
type ContractRenewalOffer struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
	// What this player actually wants - present on every response, accepted or not, so the
	// frontend can show it even on a first, successful offer.
	ExpectedWage       int `json:"expectedWage"`
	MaxAcceptableYears int `json:"maxAcceptableYears"`
	// Only meaningful when Accepted is true.
	NewContractYears int `json:"contractYearsRemaining,omitempty"`
	NewWage          int `json:"wage,omitempty"`
}

// MaxAcceptableRenewalYears is the longest contract a player of this age will agree to in a
// renewal offer - the closer to retirement (see renewalMaxYearsRetirementAge, mirroring
// engine.RetirementEligibleAge), the less willing they are to commit years in advance.
func MaxAcceptableRenewalYears(age int) int {
	switch {
	case age < 30:
		return MaxContractYearsRemaining
	case age < renewalMaxYearsRetirementAge:
		return 2
	default:
		return 1
	}
}

// MakeContractRenewalOffer proposes offeredWage/offeredYears to one of the team's own players.
// The player judges it against their own market wage (PlayerWageFromSkills - what a
// similarly-skilled player earns) and their age-appropriate maximum term
// (MaxAcceptableRenewalYears); there's no other condition, no fee, and no cost to the team
// beyond the wage itself once accepted. A declined offer changes nothing in the database and
// reports the player's real expectations, so the team can immediately retry with better terms.
// Rejects a player currently out on loan (ErrPlayerOnLoan) for the same reason as
// ReleasePlayer - team_id belongs to the borrower for the loan's duration, but the contract
// belongs to the parent club, not them.
func MakeContractRenewalOffer(dbCtx *DatabaseContext, playerId int64, teamId int64, offeredWage int, offeredYears int) (*ContractRenewalOffer, error) {
	if offeredWage <= 0 || offeredYears < 1 || offeredYears > MaxContractYearsRemaining {
		return nil, ErrInvalidRenewalOffer
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	currentTeamId, _, _, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	if !currentTeamId.Valid || currentTeamId.Int64 != teamId {
		return nil, ErrPlayerNotOwnedByTeam
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if onLoan {
		return nil, ErrPlayerOnLoan
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	expectedWage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	maxYears := MaxAcceptableRenewalYears(player.Age)

	result := &ContractRenewalOffer{ExpectedWage: expectedWage, MaxAcceptableYears: maxYears}

	var reasons []string
	if offeredWage < expectedWage {
		reasons = append(reasons, "wage below what they expect")
	}
	if offeredYears > maxYears {
		reasons = append(reasons, "too many seasons for their age")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
		return result, nil
	}

	if _, err := tx.Exec(ApplyRenewalOfferStmtSql, offeredYears, offeredWage, playerId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	result.Accepted = true
	result.NewContractYears = offeredYears
	result.NewWage = offeredWage

	playerName := player.FirstName + " " + player.LastName
	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRANSFER,
		playerName+" accepted a "+strconv.Itoa(offeredYears)+"-season contract renewal")

	return result, nil
}

// ReleasePlayer terminates playerId's contract with teamId before it expires, paying a
// severance lump sum (EarlyReleaseWageMultiplier x their wage) and releasing them to free
// agency immediately - the same team_id/lineup mechanics AdvancePlayerContract uses for a
// contract that expires naturally at season end, just paid for and team-initiated rather than
// automatic. Rejects a player currently out on loan (ErrPlayerOnLoan) for the same reason as
// MakeContractRenewalOffer - team_id belongs to the borrower for the loan's duration, not the
// club that actually holds the contract.
func ReleasePlayer(dbCtx *DatabaseContext, playerId int64, teamId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentTeamId, wage, _, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayerNotFound
	}
	if err != nil {
		return err
	}
	if !currentTeamId.Valid || currentTeamId.Int64 != teamId {
		return ErrPlayerNotOwnedByTeam
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return err
	}
	if onLoan {
		return ErrPlayerOnLoan
	}

	severance := int64(wage * EarlyReleaseWageMultiplier)

	var budget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, teamId).Scan(&budget); err != nil {
		return err
	}
	if budget < severance {
		return ErrInsufficientBudget
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -severance, teamId); err != nil {
		return err
	}
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, -severance, TRANSACTION_CONTRACT_RELEASE,
		"Released "+player.FirstName+" "+player.LastName, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(ReleaseFromTeamStmtSql, playerId); err != nil {
		return err
	}

	return tx.Commit()
}

// DecrementPlayerContractStmtSql decrements one player's remaining contract by a season,
// floored at zero, and returns the new value.
const DecrementPlayerContractStmtSql = "UPDATE players SET contract_years_remaining = GREATEST(contract_years_remaining - 1, 0) " +
	"WHERE player_id = $1 RETURNING contract_years_remaining"

// AdvancePlayerContract decrements a player's remaining contract by one season and, if it just
// expired, releases them to free agency (team_id set to NULL - see GetFreeAgents). Skipped
// entirely for a player currently out on loan (IsPlayerOnActiveLoan): the parent club's contract
// keeps ticking, but expiring it mid-loan would race the loan-return job over who gets to
// reassign team_id, so release is simply deferred until the loan ends. Meant to run once per
// season from engine.ProgressPlayers, alongside aging. Returns whether the player was released
// this call.
func AdvancePlayerContract(dbCtx *DatabaseContext, playerId int64) (bool, error) {
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return false, err
	}
	if onLoan {
		return false, nil
	}

	var remaining int
	if err := dbCtx.Db.QueryRow(DecrementPlayerContractStmtSql, playerId).Scan(&remaining); err != nil {
		return false, err
	}
	if remaining > 0 {
		return false, nil
	}

	if _, err := dbCtx.Db.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return false, err
	}
	if _, err := dbCtx.Db.Exec(ReleaseFromTeamStmtSql, playerId); err != nil {
		return false, err
	}
	return true, nil
}

// AdvanceFreeAgencyClockStmtSql ages a free agent by one season and extends their unbroken spell
// of free agency by one, returning the new total so the caller can decide whether it's crossed
// engine.FreeAgencyRetirementSeasons.
const AdvanceFreeAgencyClockStmtSql = "UPDATE players SET age = $1, free_agent_seasons = free_agent_seasons + 1 " +
	"WHERE player_id = $2 RETURNING free_agent_seasons"

// AdvanceFreeAgencyClock ages a free agent by one season and increments how many consecutive
// seasons they've now gone unsigned, returning that new count. Only ever called for a player
// who is actually still a free agent this season (see engine.ProgressFreeAgents, the only
// caller) - signing resets the counter to 0 (see SignFreeAgentStmtSql), so it only ever measures
// time since their most recent release to free agency.
func AdvanceFreeAgencyClock(dbCtx *DatabaseContext, playerId int64, newAge int) (seasonsUnsigned int, err error) {
	err = dbCtx.Db.QueryRow(AdvanceFreeAgencyClockStmtSql, newAge, playerId).Scan(&seasonsUnsigned)
	return seasonsUnsigned, err
}
