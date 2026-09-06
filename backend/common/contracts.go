package common

import (
	"database/sql"
	"errors"
	"math/rand"
	"strconv"
	"time"
)

// InitialContractMinYears/MaxYears bound the randomly assigned length of a freshly created
// player's first contract (see doCreatePlayers) and a free agent's fresh deal on signing (see
// SignFreeAgent).
const InitialContractMinYears = 2
const InitialContractMaxYears = 4

// RenewalContractYears is how many seasons a contract renewal adds (see RenewPlayerContract).
const RenewalContractYears = 2

// MaxContractYearsRemaining caps how far a contract can be extended by renewal, so a team can't
// stack years of renewals in advance of ever needing them.
const MaxContractYearsRemaining = 5

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
var ErrContractAtMax = errors.New("contract is already at its maximum length")

const GetFreeAgentsStmtSql = "SELECT player_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining " +
	"FROM players WHERE team_id IS NULL " +
	"ORDER BY (goal_keeper + defense + \"lateral\" + pass + midfield + dribbling + shot) DESC"

// GetPlayerOwnershipForUpdateStmtSql locks a player row and returns just enough to validate and
// carry out a contract action: who (if anyone) currently employs them, their wage (needed for
// ReleasePlayer's severance), and how much contract they have left (needed for
// RenewPlayerContract's cap). team_id comes back NULL for a free agent.
const GetPlayerOwnershipForUpdateStmtSql = "SELECT team_id, wage, contract_years_remaining FROM players WHERE player_id = $1 " + ForUpdate

const SignFreeAgentStmtSql = "UPDATE players SET team_id = $1, contract_years_remaining = $2, wage = $3 WHERE player_id = $4"
const RenewContractStmtSql = "UPDATE players SET contract_years_remaining = $1 WHERE player_id = $2"
const ReleaseFromTeamStmtSql = "UPDATE players SET team_id = NULL, contract_years_remaining = 0 WHERE player_id = $1"

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
		if err := rows.Scan(&p.Id, &p.FirstName, &p.LastName, &birthDay,
			&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot,
			&p.Header, &p.Quick, &p.Creative, &p.Technical, &p.Wage, &p.Age, &avatar, &p.ContractYearsRemaining); err != nil {
			return nil, err
		}
		p.BirthDay = time.UnixMilli(birthDay)
		p.Avatar = GetAvatarFromString(avatar)
		players = append(players, &p)
	}
	return players, nil
}

func lockPlayerOwnership(tx *sql.Tx, playerId int64) (teamId sql.NullInt64, wage int, contractYearsRemaining int, err error) {
	err = tx.QueryRow(GetPlayerOwnershipForUpdateStmtSql, playerId).Scan(&teamId, &wage, &contractYearsRemaining)
	return
}

// SignFreeAgent signs an unowned player to teamId on a fresh contract, at a freshly negotiated
// market wage (see PlayerWageFromSkills) - there's no signing fee, the same way player loans
// already move for free; only the wage a team takes on is the cost. The player row is locked
// for the duration so two teams can't both sign the same free agent.
func SignFreeAgent(dbCtx *DatabaseContext, playerId int64, teamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentTeamId, _, _, err := lockPlayerOwnership(tx, playerId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayerNotFound
	}
	if err != nil {
		return err
	}
	if currentTeamId.Valid {
		return ErrPlayerNotFree
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	wage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	contractYears := RandomContractYears()

	if _, err := tx.Exec(SignFreeAgentStmtSql, teamId, contractYears, wage, playerId); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	playerName := player.FirstName + " " + player.LastName
	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRANSFER,
		"Signed free agent "+playerName+" to a "+strconv.Itoa(contractYears)+"-season contract")
	return nil
}

// RenewPlayerContract extends playerId's contract by RenewalContractYears, capped at
// MaxContractYearsRemaining. No money changes hands - renewing just means agreeing to keep
// paying the existing wage for longer, unlike signing a free agent or buying a transfer.
// Rejects a player currently out on loan (ErrPlayerOnLoan): team_id points at the borrowing
// team for the loan's duration, but the contract belongs to the parent club, not them - same
// rule as CreateTransferListing/MakeTransferOffer for the same reason.
func RenewPlayerContract(dbCtx *DatabaseContext, playerId int64, teamId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentTeamId, _, remaining, err := lockPlayerOwnership(tx, playerId)
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
	if remaining >= MaxContractYearsRemaining {
		return ErrContractAtMax
	}

	newRemaining := remaining + RenewalContractYears
	if newRemaining > MaxContractYearsRemaining {
		newRemaining = MaxContractYearsRemaining
	}
	if _, err := tx.Exec(RenewContractStmtSql, newRemaining, playerId); err != nil {
		return err
	}
	return tx.Commit()
}

// ReleasePlayer terminates playerId's contract with teamId before it expires, paying a
// severance lump sum (EarlyReleaseWageMultiplier x their wage) and releasing them to free
// agency immediately - the same team_id/lineup mechanics AdvancePlayerContract uses for a
// contract that expires naturally at season end, just paid for and team-initiated rather than
// automatic. Rejects a player currently out on loan (ErrPlayerOnLoan) for the same reason as
// RenewPlayerContract - team_id belongs to the borrower for the loan's duration, not the club
// that actually holds the contract.
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
		"Released "+player.FirstName+" "+player.LastName); err != nil {
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
