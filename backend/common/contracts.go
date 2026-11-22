package common

import (
	"context"
	"database/sql"
	"errors"
	"math"
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

// shirt_number isn't selected here - a free agent has none (team_id IS NULL implies it), so
// Player.ShirtNumber's zero value is already the right answer.
const GetFreeAgentsStmtSql = "SELECT player_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"free_agent_seasons, COALESCE(last_team_id, 0), stamina, form " +
	"FROM players WHERE team_id IS NULL " +
	"ORDER BY (goal_keeper + defense + \"lateral\" + pass + midfield + dribbling + shot) DESC"

// GetBestFreeAgentsStmtSql is GetFreeAgentsStmtSql cut to the best $1. Its ORDER BY expression
// and WHERE are exactly those of the players_free_agents_by_skill index (see
// init120_query_performance_indexes.sql), so Postgres reads the first $1 entries of that index
// instead of sorting every free agent in the world.
const GetBestFreeAgentsStmtSql = GetFreeAgentsStmtSql + " LIMIT $1"

// GetPlayerOwnershipForUpdateStmtSql locks a player row and returns just enough to validate a
// contract action: who (if anyone) currently employs them and their wage (needed for
// ReleasePlayer's severance). team_id comes back NULL for a free agent.
const GetPlayerOwnershipForUpdateStmtSql = "SELECT team_id, wage, contract_years_remaining FROM players WHERE player_id = $1 " + ForUpdate

// free_agent_seasons resets to 0 on signing - see engine.ProgressFreeAgents, which increments it
// once per season a player stays unsigned, and AdvanceFreeAgencyClock's doc comment. shirt_number
// is assigned by the caller (see nextFreeShirtNumber) - a free agent has none to carry over.
const SignFreeAgentStmtSql = "UPDATE players SET team_id = $1, contract_years_remaining = $2, wage = $3, free_agent_seasons = 0, shirt_number = $4, " +
	"release_clause = $6, release_clause_set_at = CASE WHEN $6::bigint IS NULL THEN NULL ELSE now() END WHERE player_id = $5"

// ApplyRenewalOfferStmtSql sets a player's new contract: years, wage and its release clause ($4,
// NULL for none - a new contract without one clears the old one; see openspec/specs/
// release-clauses), stamped with now() so its lock-in starts.
const ApplyRenewalOfferStmtSql = "UPDATE players SET contract_years_remaining = $1, wage = $2, release_clause = $4, " +
	"release_clause_set_at = CASE WHEN $4::bigint IS NULL THEN NULL ELSE now() END WHERE player_id = $3"

// GetPlayerCurrentTeamStmtSql reads the team a player belongs to right now (0 for a free agent) -
// AdvancePlayerContract needs it before ReleaseFromTeamStmtSql clears team_id.
const GetPlayerCurrentTeamStmtSql = "SELECT COALESCE(team_id, 0) FROM players WHERE player_id = $1"

// ReleaseFromTeamStmtSql captures the releasing club into last_team_id before clearing team_id -
// see SignFreeAgent, which charges that club a transfer premium if this player is re-signed
// within their first season of free agency. shirt_number is cleared too - it belonged to the
// team the player is leaving, not to them (issue #42).
const ReleaseFromTeamStmtSql = "UPDATE players SET last_team_id = team_id, team_id = NULL, contract_years_remaining = 0, shirt_number = NULL, " +
	"release_clause = NULL, release_clause_set_at = NULL WHERE player_id = $1"

// GetFreeAgentListingsStmtSql mirrors GetFreeAgentsStmtSql but left-joins in the last club's name
// for display on the transfer market's free agents tab (see GetFreeAgentListings) - a free agent
// with no last_team_id (never released) gets a NULL/empty name, same as LastTeamId being 0.
const GetFreeAgentListingsStmtSql = "SELECT p.player_id, p.first_name, p.last_name, p.birth_day, p.goal_keeper, p.defense, p.\"lateral\", p.pass, p.midfield, " +
	"p.dribbling, p.shot, p.\"header\", p.quick, p.creative, p.technical, p.wage, p.age, p.avatar, p.contract_years_remaining, p.birth_country, " +
	"p.free_agent_seasons, COALESCE(p.last_team_id, 0), COALESCE(t.name, ''), p.stamina, p.form " +
	"FROM players p LEFT JOIN teams t ON t.team_id = p.last_team_id " +
	"WHERE p.team_id IS NULL " +
	"ORDER BY (p.goal_keeper + p.defense + p.\"lateral\" + p.pass + p.midfield + p.dribbling + p.shot) DESC"

// GetFreeAgentListings returns every unowned player, best-skilled first, same as GetFreeAgents,
// but with each one's last club's name resolved for the transfer market's free agents tab.
func GetFreeAgentListings(dbCtx *DatabaseContext) ([]*FreeAgentListing, error) {
	rows, err := dbCtx.Db.Query(GetFreeAgentListingsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings := make([]*FreeAgentListing, 0)
	for rows.Next() {
		var l FreeAgentListing
		var birthDay int64
		var avatar string
		var birthCountry sql.NullString
		if err := rows.Scan(&l.Id, &l.FirstName, &l.LastName, &birthDay,
			&l.GoalKeeper, &l.Defense, &l.Lateral, &l.Pass, &l.Midfield, &l.Dribbling, &l.Shot,
			&l.Header, &l.Quick, &l.Creative, &l.Technical, &l.Wage, &l.Age, &avatar, &l.ContractYearsRemaining, &birthCountry,
			&l.FreeAgentSeasons, &l.LastTeamId, &l.LastTeamName, &l.Stamina, &l.Form); err != nil {
			return nil, err
		}
		l.BirthDay = time.UnixMilli(birthDay)
		l.Avatar = GetAvatarFromString(avatar)
		l.BirthCountry = birthCountry.String
		listings = append(listings, &l)
	}

	playerIds := make([]int64, len(listings))
	for i, l := range listings {
		playerIds[i] = l.Id
	}
	caps, err := GetPlayerCapsBatch(dbCtx, playerIds)
	if err != nil {
		return nil, err
	}
	for _, l := range listings {
		l.Caps = caps[l.Id]
	}
	return listings, nil
}

// GetFreeAgents returns every unowned player, best-skilled first - both those released after
// their contract expired (see AdvancePlayerContract) and any player orphaned by
// common.AssignTeam reassigning a CPU team's squad to a new human owner.
func GetFreeAgents(dbCtx *DatabaseContext) ([]*Player, error) {
	return queryFreeAgents(dbCtx, GetFreeAgentsStmtSql)
}

// GetBestFreeAgents returns the limit best-skilled unowned players, in GetFreeAgents' order - the
// same as GetFreeAgents()[:limit], without loading the rest. For callers that only ever look at
// the top of the list (an AI-managed team picking candidates) and run once per team.
func GetBestFreeAgents(dbCtx *DatabaseContext, limit int) ([]*Player, error) {
	return queryFreeAgents(dbCtx, GetBestFreeAgentsStmtSql, limit)
}

func queryFreeAgents(dbCtx *DatabaseContext, query string, args ...any) ([]*Player, error) {
	rows, err := dbCtx.Db.Query(query, args...)
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
			&p.FreeAgentSeasons, &p.LastTeamId, &p.Stamina, &p.Form); err != nil {
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
	return SignFreeAgentWithClause(dbCtx, playerId, teamId, offeredWage, offeredYears, nil)
}

// SignFreeAgentWithClause is SignFreeAgent with an optional release clause in the new contract
// (nil for none), which lowers the wage he asks (see DiscountedWage).
func SignFreeAgentWithClause(dbCtx *DatabaseContext, playerId int64, teamId int64, offeredWage int, offeredYears int, clause *int64) (*FreeAgentSigningResult, error) {
	if offeredWage <= 0 || offeredYears < 1 || offeredYears > MaxContractYearsRemaining {
		return nil, ErrInvalidFreeAgentOffer
	}
	if !IsTransferWindowOpen(dbCtx) {
		return nil, ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, teamId) {
		return nil, ErrTeamInsolvent
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
	intelligence, err := GetPlayerIntelligence(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	trend, err := PlayerSkillTrend(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if err := ValidateReleaseClause(*player, clause); err != nil {
		return nil, err
	}
	expectedWage, maxYears := PersonalTermsExpectation(*player, intelligence, trend, TermsFreeAgent)
	expectedWage = DiscountedWage(*player, expectedWage, clause, intelligence)

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

	shirtNumber, err := nextFreeShirtNumber(tx, teamId)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(SignFreeAgentStmtSql, teamId, offeredYears, offeredWage, shirtNumber, playerId, clauseArg(clause)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(DeletePlayerFromWatchlistStmtSql, teamId, playerId); err != nil {
		return nil, err
	}
	if err := RecordPlayerClubHistory(tx, playerId, teamId); err != nil {
		return nil, err
	}
	// A premium is paid to the last club, so it's the origin; with none due the player comes
	// straight from free agency (no origin club).
	var premiumRecipient int64
	if premium > 0 {
		premiumRecipient = player.LastTeamId
	}
	if err := RecordPlayerTransfer(tx, playerId, premiumRecipient, teamId, premium, PLAYER_TRANSFER_FREE_AGENT); err != nil {
		return nil, err
	}

	playerName := player.FirstName + " " + player.LastName
	var flows LedgerFlows
	if premium > 0 {
		if err := insertLedgerRow(tx, &flows, teamId, -premium, TRANSACTION_TRANSFER_OUT,
			"Paid free agent transfer premium for "+playerName, playerId); err != nil {
			return nil, err
		}
		if err := insertLedgerRow(tx, &flows, player.LastTeamId, premium, TRANSACTION_TRANSFER_IN,
			"Received free agent transfer premium for "+playerName, playerId); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	flows.Commit(context.Background())

	FreeAgentSigningsCounter.Add(context.Background(), 1)

	result.Accepted = true
	result.ContractYears = offeredYears
	result.Wage = offeredWage
	result.Premium = premium

	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Signed free agent " + playerName + " to a " + strconv.Itoa(offeredYears) + "-season contract",
		Key:    "notifications.messages.freeAgentSigned",
		Params: map[string]any{"name": playerName, "years": offeredYears},
	})
	if premium > 0 {
		_ = NotifyTeamOwner(dbCtx, player.LastTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
			Text:   playerName + " was signed by another club - transfer premium received",
			Key:    "notifications.messages.freeAgentPremium",
			Params: map[string]any{"name": playerName},
		})
	}
	if signing, err := GetTeamById(teamId, dbCtx); err == nil {
		NotifyWatchers(dbCtx, playerId, WatchEventMoved, signing.TeamName, 0, teamId)
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
	intelligence, err := GetPlayerIntelligence(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	trend, err := PlayerSkillTrend(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	wage, maxYears := PersonalTermsExpectation(*player, intelligence, trend, TermsFreeAgent)
	years := RandomContractYears()
	if years > maxYears {
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

// DeclineSkillPointsPerExtraRenewalYear sets how much a declining player's willingness to commit
// grows: every this many points of combined skill lost (see PlayerSkillTrend) buys one extra
// season of acceptable term, at a Mastermind's full reaction (see IntelligenceReactionFactor) -
// see MaxAcceptableRenewalYearsWithIntelligence. 5 mirrors the scale of a single skill drifting a
// handful of points, since that's the typical size of one logged progression run (see
// engine.driftSkill) - a player would need a real, sustained decline, not one unlucky match, to
// meaningfully extend what they'll accept.
const DeclineSkillPointsPerExtraRenewalYear = 5

// MaxAcceptableRenewalYearsWithIntelligence extends MaxAcceptableRenewalYears for a player whose
// combined skills have been falling (trend < 0, see PlayerSkillTrend): trading job security for a
// cheaper deal is the flip side of ExpectedWageWithIntelligence's raise demand, so the same
// per-tier reaction factor applies here too. A rising or flat trend changes nothing - this never
// shrinks the age-based figure, only ever grows it, capped at MaxContractYearsRemaining the same
// as every other renewal offer. intelligence 0 (not yet backfilled) or trend >= 0 reproduces
// MaxAcceptableRenewalYears exactly.
func MaxAcceptableRenewalYearsWithIntelligence(age, intelligence, trend int) int {
	base := MaxAcceptableRenewalYears(age)
	if trend >= 0 {
		return base
	}
	decline := -trend
	extraYears := int(float64(decline) * IntelligenceReactionFactor(intelligence) / float64(DeclineSkillPointsPerExtraRenewalYear))
	years := base + extraYears
	if years > MaxContractYearsRemaining {
		return MaxContractYearsRemaining
	}
	return years
}

// ExpectedWageWithIntelligence extends PlayerWageFromSkills for a player whose combined skills
// have been rising (trend > 0, see PlayerSkillTrend): the raise they demand is exactly what a
// brand-new player with that much extra skill would command (reusing
// WagePerSkillPointNumerator/Denominator, the same rate PlayerWageFromSkills itself charges),
// scaled down by how strongly their own intelligence tier reacts to it (see
// IntelligenceReactionFactor) - a Naive player (factor 0) never notices their own improvement. A
// falling or flat trend changes nothing - this never lowers the skill-only figure, only ever
// raises it. intelligence 0 (not yet backfilled) or trend <= 0 reproduces PlayerWageFromSkills
// exactly.
func ExpectedWageWithIntelligence(goalKeeper, defense, lateral, pass, midfield, dribbling, shot, intelligence, trend int) int {
	base := PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
	if trend <= 0 {
		return base
	}
	raise := int(float64(trend) * IntelligenceReactionFactor(intelligence) * float64(WagePerSkillPointNumerator) / float64(WagePerSkillPointDenominator))
	return base + raise
}

// TermsKind is the situation a player judges contract terms in (see PersonalTermsExpectation):
// a renewal with his own club, a move to another club (a transfer offer's personal terms) or a
// fresh deal as a free agent.
type TermsKind int

const (
	TermsRenewal TermsKind = iota
	TermsTransfer
	TermsFreeAgent
)

// Wage floors (openspec/specs/team-finances "Contract renewal and release" and "Personal terms of
// a transfer offer"): the share of his current wage a player insists on keeping. While he has 2 or
// more seasons left he renews for no less than he earns (RenewalWageFloorShare); in his last
// season he accepts a cut that depends on his negotiation tier (LastSeasonFloorShare: 75% for a
// Naive player up to 95% for a Mastermind, LastSeasonWageFloorShare for an Aware one); he never
// moves to another club for less than he earns (TransferWageFloorShare). A free agent has no
// current wage to protect.
const (
	RenewalWageFloorShare    = 1.0
	LastSeasonWageFloorShare = 0.85
	TransferWageFloorShare   = 1.0
)

// wageFloorShare is the share of his current wage p, of negotiation tier intelligence, insists on
// in a negotiation of kind.
func wageFloorShare(p Player, kind TermsKind, intelligence int) float64 {
	switch kind {
	case TermsRenewal:
		if p.ContractYearsRemaining >= 2 {
			return RenewalWageFloorShare
		}
		return LastSeasonFloorShare(intelligence)
	case TermsTransfer:
		return TransferWageFloorShare
	default:
		return 0
	}
}

// PersonalTermsExpectation is the lowest wage and the longest term p accepts in a negotiation of
// kind: his market wage nudged by his intelligence and skill trend (ExpectedWageWithIntelligence),
// never below the wage floor of kind (see wageFloorShare), and his age's maximum term
// (MaxAcceptableRenewalYearsWithIntelligence). Pure - the one rule every renewal, transfer offer
// and free-agent signing judges personal terms by.
func PersonalTermsExpectation(p Player, intelligence, trend int, kind TermsKind) (wage int, years int) {
	wage = ExpectedWageWithIntelligence(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot, intelligence, trend)
	if share := wageFloorShare(p, kind, intelligence); share > 0 {
		if floor := int(math.Ceil(float64(p.Wage) * share)); floor > wage {
			wage = floor
		}
	}
	return wage, MaxAcceptableRenewalYearsWithIntelligence(p.Age, intelligence, trend)
}

// PlayerTermsExpectation reads p's intelligence and skill trend and returns
// PersonalTermsExpectation for kind - what a CPU club offers so the player accepts (see
// engine.renewContract and makeAiOffer).
func PlayerTermsExpectation(dbCtx *DatabaseContext, p Player, kind TermsKind) (wage int, years int, err error) {
	intelligence, err := GetPlayerIntelligence(dbCtx, p.Id)
	if err != nil {
		return 0, 0, err
	}
	trend, err := PlayerSkillTrend(dbCtx, p.Id)
	if err != nil {
		return 0, 0, err
	}
	wage, years = PersonalTermsExpectation(p, intelligence, trend, kind)
	return wage, years, nil
}

// MakeContractRenewalOffer proposes offeredWage/offeredYears to one of the team's own players.
// The player judges it against their own market wage (ExpectedWageWithIntelligence - what a
// similarly-skilled player earns, nudged up if they've recently been improving) and their
// age-appropriate maximum term (MaxAcceptableRenewalYearsWithIntelligence - nudged up, i.e. more
// seasons they'll accept, if they've recently been declining); there's no other condition, no
// fee, and no cost to the team beyond the wage itself once accepted. A declined offer changes
// nothing in the database and reports the player's real expectations, so the team can immediately
// retry with better terms. Rejects a player currently out on loan (ErrPlayerOnLoan) for the same
// reason as ReleasePlayer - team_id belongs to the borrower for the loan's duration, but the
// contract belongs to the parent club, not them.
func MakeContractRenewalOffer(dbCtx *DatabaseContext, playerId int64, teamId int64, offeredWage int, offeredYears int) (*ContractRenewalOffer, error) {
	return MakeContractRenewalOfferWithClause(dbCtx, playerId, teamId, offeredWage, offeredYears, nil)
}

// MakeContractRenewalOfferWithClause is MakeContractRenewalOffer with an optional release clause
// in the new contract (nil for none): the player then asks the discounted wage (see
// DiscountedWage), and the renewal replaces whatever clause the old contract carried.
func MakeContractRenewalOfferWithClause(dbCtx *DatabaseContext, playerId int64, teamId int64, offeredWage int, offeredYears int, clause *int64) (*ContractRenewalOffer, error) {
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
	if signed, err := IsPlayerPreContracted(tx, playerId); err != nil {
		return nil, err
	} else if signed {
		return nil, ErrPlayerPreContracted
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	intelligence, err := GetPlayerIntelligence(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	trend, err := PlayerSkillTrend(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	expectedWage, maxYears := PersonalTermsExpectation(*player, intelligence, trend, TermsRenewal)
	// While pre-contract offers are pending he stays for the best of them less the move premium
	// (openspec/specs/pre-contracts "Right to counter-offer").
	bar, err := PreContractCounterBar(tx, playerId)
	if err != nil {
		return nil, err
	}
	if bar > expectedWage {
		expectedWage = bar
	}
	if err := ValidateReleaseClause(*player, clause); err != nil {
		return nil, err
	}
	expectedWage = DiscountedWage(*player, expectedWage, clause, intelligence)

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

	if _, err := tx.Exec(ApplyRenewalOfferStmtSql, offeredYears, offeredWage, playerId, clauseArg(clause)); err != nil {
		return nil, err
	}
	preContractBuyers, err := cancelPendingPreContracts(tx, playerId)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	notifyPreContractsCancelled(dbCtx, playerId, preContractBuyers)

	result.Accepted = true
	result.NewContractYears = offeredYears
	result.NewWage = offeredWage

	playerName := player.FirstName + " " + player.LastName
	_ = NotifyTeamOwner(dbCtx, teamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   playerName + " accepted a " + strconv.Itoa(offeredYears) + "-season contract renewal",
		Key:    "notifications.messages.contractRenewed",
		Params: map[string]any{"name": playerName, "years": offeredYears},
	})

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
	if signed, err := IsPlayerPreContracted(tx, playerId); err != nil {
		return err
	} else if signed {
		return ErrPlayerPreContracted
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
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, teamId, -severance, TRANSACTION_CONTRACT_RELEASE,
		"Released "+player.FirstName+" "+player.LastName, playerId); err != nil {
		return err
	}
	if err := removePlayerFromTeamLineupsTx(tx, playerId, teamId); err != nil {
		return err
	}
	if err := clearPlayerMarketTx(tx, playerId, teamId); err != nil {
		return err
	}
	if _, err := tx.Exec(ReleaseFromTeamStmtSql, playerId); err != nil {
		return err
	}
	if err := RecordPlayerTransfer(tx, playerId, teamId, 0, 0, PLAYER_TRANSFER_RELEASE); err != nil {
		return err
	}
	preContractBuyers, err := cancelPendingPreContracts(tx, playerId)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	flows.Commit(context.Background())
	notifyPreContractsCancelled(dbCtx, playerId, preContractBuyers)
	NotifyWatchers(dbCtx, playerId, WatchEventFreeAgent, "", 0, teamId)
	return nil
}

// DecrementPlayerContractStmtSql decrements one player's remaining contract by a season,
// floored at zero, and returns the new value.
const DecrementPlayerContractStmtSql = "UPDATE players SET contract_years_remaining = GREATEST(contract_years_remaining - 1, 0) " +
	"WHERE player_id = $1 RETURNING contract_years_remaining"

// AdvancePlayerContract decrements a player's remaining contract by one season and, if it just
// expired, either moves him to the club he signed a pre-contract with (openspec/specs/
// pre-contracts "Pre-contracted player") or releases him to free agency (team_id set to NULL - see
// GetFreeAgents). Skipped entirely for a player currently out on loan (IsPlayerOnActiveLoan): the
// parent club's contract keeps ticking, but expiring it mid-loan would race the loan-return job
// over who gets to reassign team_id, so release is simply deferred until the loan ends. Meant to
// run once per season from engine.ProgressPlayers, alongside aging. Returns whether the player was
// released to free agency this call (a pre-contract move is not a release: both clubs and his
// followers are told here).
func AdvancePlayerContract(dbCtx *DatabaseContext, playerId int64) (bool, error) {
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return false, err
	}
	if onLoan {
		return false, nil
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var expiringTeamId int64
	if err := tx.QueryRow(GetPlayerTeamForUpdateStmtSql, playerId).Scan(&expiringTeamId); err != nil {
		return false, err
	}
	var remaining int
	if err := tx.QueryRow(DecrementPlayerContractStmtSql, playerId).Scan(&remaining); err != nil {
		return false, err
	}
	if remaining > 0 {
		return false, tx.Commit()
	}

	buyer, err := completePreContractTx(tx, playerId, expiringTeamId)
	if err != nil {
		return false, err
	}
	if buyer != 0 {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		notifyPreContractJoined(dbCtx, playerId, expiringTeamId, buyer)
		return false, nil
	}

	if err := removePlayerFromTeamLineupsTx(tx, playerId, expiringTeamId); err != nil {
		return false, err
	}
	if err := clearPlayerMarketTx(tx, playerId, expiringTeamId); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ReleaseFromTeamStmtSql, playerId); err != nil {
		return false, err
	}
	if err := RecordPlayerTransfer(tx, playerId, expiringTeamId, 0, 0, PLAYER_TRANSFER_RELEASE); err != nil {
		return false, err
	}
	// Any offer still pending on him lapses with his contract.
	preContractBuyers, err := cancelPendingPreContracts(tx, playerId)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	notifyPreContractsCancelled(dbCtx, playerId, preContractBuyers)
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
