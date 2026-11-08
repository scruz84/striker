package common

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

// PlayerCareerStats is a player's all-time totals, computed from match_events - see
// engine/scorer_simulation.go and engine/discipline_simulation.go for how those events are
// written. team_players is always a single-element JSON array holding the acting player's id
// (never a real list), so an exact string match on "[<id>]" finds their events without
// decoding JSON per row.
//
// There's deliberately no "appearances" count here: counting one would need an index over every
// lineup_positions/cup_tie_lineup_positions/friendly_lineup_positions row by player_id, and nothing
// here needs that shape of lookup yet (see engine.persistResolvedLineupSnapshot and its cup-tie/
// friendly equivalents for how those per-match rows end up populated).
type PlayerCareerStats struct {
	Goals       int `json:"goals"`
	YellowCards int `json:"yellowCards"`
	RedCards    int `json:"redCards"`
	// GoalsForTeam is goals scored while contracted to whichever team currently employs this
	// player (0 for a free agent, who currently plays for none) - narrower than Goals, which
	// counts every team a player has ever scored for across their whole career.
	GoalsForTeam int `json:"goalsForTeam"`
	// Caps is how many played national-team friendlies this player was selected to start (see
	// common.SetNationalTeamLineup/ScheduleNationalTeamFriendlies) - counted from
	// friendly_lineup_positions rather than match_events, since an appearance alone (not a
	// goal/card action) is what a cap actually is.
	Caps int `json:"caps"`
}

// match_date <= $3 excludes an event from a match still inside its live-broadcast window (see
// LiveBroadcastWindow) - otherwise a player's career goal/card tally here would jump the instant
// their match is simulated, well before that match's own page stops showing it as "LIVE" (issue:
// match results review).
const countPlayerEventStmtSql = "SELECT count(*) FROM match_events me JOIN matches m ON m.match_id = me.match_id " +
	"WHERE me.event_type = $1 AND me.team_players = $2 AND m.match_date <= $3"
const countPlayerEventForTeamStmtSql = "SELECT count(*) FROM match_events me JOIN matches m ON m.match_id = me.match_id " +
	"WHERE me.event_type = $1 AND me.team_players = $2 AND me.team_owner = $3 AND m.match_date <= $4"

// countPlayerCapsStmtSql counts played national-team appearances playerId was selected to
// start - a weekly friendly (see common.SetNationalTeamLineup, friendly_lineup_positions) plus a
// World Cup knockout tie (see common.SetNationalTeamCupTieLineup, cup_tie_lineup_positions); both
// are the only writers of their respective lineup-positions table for a team flagged
// is_national_team, so this covers every way a cap can happen.
const countPlayerCapsStmtSql = "SELECT " +
	"(SELECT count(*) FROM friendly_lineup_positions flp " +
	"JOIN friendly_lineup fl ON fl.lineup_id = flp.lineup_id " +
	"JOIN friendly_matches fm ON fm.match_id = fl.match_id " +
	"JOIN teams t ON t.team_id = fl.team_id " +
	"WHERE flp.player_id = $1 AND t.is_national_team = true AND fm.played = true) + " +
	"(SELECT count(*) FROM cup_tie_lineup_positions ctlp " +
	"JOIN cup_tie_lineup ctl ON ctl.lineup_id = ctlp.lineup_id " +
	"JOIN cup_ties ct ON ct.tie_id = ctl.tie_id " +
	"JOIN teams t2 ON t2.team_id = ctl.team_id " +
	"WHERE ctlp.player_id = $1 AND t2.is_national_team = true AND ct.played = true)"

// getPlayerCapsBatchStmtSql is countPlayerCapsStmtSql's batched counterpart - one query for a
// whole set of players (player_id = ANY($1)) instead of one round trip per player, grouping the
// same two union'd sources (played friendly lineups, played cup-tie lineups) by player_id. See
// GetPlayerCapsBatch.
const getPlayerCapsBatchStmtSql = "SELECT player_id, count(*) FROM (" +
	"SELECT flp.player_id FROM friendly_lineup_positions flp " +
	"JOIN friendly_lineup fl ON fl.lineup_id = flp.lineup_id " +
	"JOIN friendly_matches fm ON fm.match_id = fl.match_id " +
	"JOIN teams t ON t.team_id = fl.team_id " +
	"WHERE flp.player_id = ANY($1) AND t.is_national_team = true AND fm.played = true " +
	"UNION ALL " +
	"SELECT ctlp.player_id FROM cup_tie_lineup_positions ctlp " +
	"JOIN cup_tie_lineup ctl ON ctl.lineup_id = ctlp.lineup_id " +
	"JOIN cup_ties ct ON ct.tie_id = ctl.tie_id " +
	"JOIN teams t2 ON t2.team_id = ctl.team_id " +
	"WHERE ctlp.player_id = ANY($1) AND t2.is_national_team = true AND ct.played = true" +
	") caps GROUP BY player_id"

// PlayerCapsBatchKey prefixes the ObjectCache entries GetPlayerCapsBatch keeps, one per distinct
// set of player ids asked for.
const PlayerCapsBatchKey = "player_caps_batch:"

// playerCapsBatchCacheKey identifies a set of player ids regardless of the order they were asked
// for in, so the transfer, loan and free-agent tabs each hit their own entry on every reload.
func playerCapsBatchCacheKey(playerIds []int64) string {
	sorted := slices.Clone(playerIds)
	slices.Sort(sorted)
	sum := sha1.Sum([]byte(fmt.Sprint(sorted)))
	return PlayerCapsBatchKey + hex.EncodeToString(sum[:])
}

// GetPlayerCapsBatch batch-fetches international caps (see countPlayerCapsStmtSql) for a set of
// players, keyed by player_id - used to show caps on every row of a market listing (the transfer,
// loan and free-agent tabs) without a query per listed player. A player id with no caps simply
// has no entry in the returned map; callers treat that as zero.
//
// The query joins every national-team lineup row ever played, so it's the slowest part of loading
// a market tab; the result is kept in dbCtx.ObjectCache for its short default TTL (caps only
// change when a national-team match is played, and a minute-stale count on a market row is
// harmless). A missing or failing cache just falls through to the query.
func GetPlayerCapsBatch(dbCtx *DatabaseContext, playerIds []int64) (map[int64]int, error) {
	caps := make(map[int64]int, len(playerIds))
	if len(playerIds) == 0 {
		return caps, nil
	}

	cacheKey := ""
	if dbCtx.ObjectCache != nil {
		cacheKey = playerCapsBatchCacheKey(playerIds)
		if cached, err := dbCtx.ObjectCache.GetValue(cacheKey); err == nil && cached != "" {
			if err := json.Unmarshal([]byte(cached), &caps); err == nil {
				return caps, nil
			}
			caps = make(map[int64]int, len(playerIds))
		}
	}

	rows, err := dbCtx.Db.Query(getPlayerCapsBatchStmtSql, playerIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var playerId int64
		var count int
		if err := rows.Scan(&playerId, &count); err != nil {
			return nil, err
		}
		caps[playerId] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if cacheKey != "" {
		if encoded, err := json.Marshal(caps); err == nil {
			dbCtx.ObjectCache.SetValue(cacheKey, string(encoded))
		}
	}
	return caps, nil
}

// GetPlayerCareerStats totals goals and cards across every match the player has ever taken part
// in, regardless of which team they were on at the time, plus GoalsForTeam scoped to teamId (the
// player's current team - see PlayerDetails.TeamId).
func GetPlayerCareerStats(dbCtx *DatabaseContext, playerId int64, teamId int64) (PlayerCareerStats, error) {
	var stats PlayerCareerStats
	marker := fmt.Sprintf("[%d]", playerId)
	cutoff := LiveBroadcastCutoff()
	if err := dbCtx.Db.QueryRow(countPlayerEventStmtSql, GOAL, marker, cutoff).Scan(&stats.Goals); err != nil {
		return stats, err
	}
	if err := dbCtx.Db.QueryRow(countPlayerEventStmtSql, YELLOW_CARD, marker, cutoff).Scan(&stats.YellowCards); err != nil {
		return stats, err
	}
	if err := dbCtx.Db.QueryRow(countPlayerEventStmtSql, RED_CARD, marker, cutoff).Scan(&stats.RedCards); err != nil {
		return stats, err
	}
	if err := dbCtx.Db.QueryRow(countPlayerEventForTeamStmtSql, GOAL, marker, teamId, cutoff).Scan(&stats.GoalsForTeam); err != nil {
		return stats, err
	}
	if err := dbCtx.Db.QueryRow(countPlayerCapsStmtSql, playerId).Scan(&stats.Caps); err != nil {
		return stats, err
	}
	return stats, nil
}

// PlayerStatus is a player's current situation, as opposed to their history - on loan, listed
// for transfer or loan, or unavailable through injury/suspension. Each pointer is nil when that
// status doesn't apply, so the frontend only renders the sections that are actually relevant.
type PlayerStatus struct {
	UnavailableReason           *int    `json:"unavailableReason,omitempty"`
	UnavailableMatchesRemaining *int    `json:"unavailableMatchesRemaining,omitempty"`
	OnLoanFromTeamId            *int64  `json:"onLoanFromTeamId,omitempty"`
	OnLoanFromTeamName          *string `json:"onLoanFromTeamName,omitempty"`
	OnLoanReturnSeason          *int    `json:"onLoanReturnSeason,omitempty"`
	// OnLoanWageContributionPercent/OnLoanBuyOptionFee are only set when the loan actually carries
	// that term (see common.CreateLoanListing) - nil, not zero, when it doesn't, so the frontend
	// can tell "no buy option" apart from "buy option fee of zero".
	OnLoanWageContributionPercent *int   `json:"onLoanWageContributionPercent,omitempty"`
	OnLoanBuyOptionFee            *int64 `json:"onLoanBuyOptionFee,omitempty"`
	TransferListingAskingPrice    *int64 `json:"transferListingAskingPrice,omitempty"`
	LoanListingDurationSeasons    *int   `json:"loanListingDurationSeasons,omitempty"`
}

const getPlayerUnavailabilityStmtSql = "SELECT reason, matches_remaining FROM player_unavailability WHERE player_id = $1"
const getPlayerActiveLoanStmtSql = "SELECT pl.from_team_id, t.name, pl.return_season, pl.wage_contribution_percent, pl.buy_option_fee " +
	"FROM player_loans pl JOIN teams t ON t.team_id = pl.from_team_id WHERE pl.player_id = $1"
const getPlayerTransferListingPriceStmtSql = "SELECT asking_price FROM transfer_listings WHERE player_id = $1"
const getPlayerLoanListingDurationStmtSql = "SELECT duration_seasons FROM loan_listings WHERE player_id = $1"

// GetPlayerStatus reports whatever of "unavailable / on loan / listed for transfer / listed for
// loan" currently applies to the player - any, all, or none of these can be true at once (e.g.
// injured AND listed for transfer).
func GetPlayerStatus(dbCtx *DatabaseContext, playerId int64) (PlayerStatus, error) {
	var status PlayerStatus

	var reason, matchesRemaining int
	err := dbCtx.Db.QueryRow(getPlayerUnavailabilityStmtSql, playerId).Scan(&reason, &matchesRemaining)
	if err == nil {
		status.UnavailableReason = &reason
		status.UnavailableMatchesRemaining = &matchesRemaining
	} else if err != sql.ErrNoRows {
		return status, err
	}

	var fromTeamId int64
	var fromTeamName string
	var returnSeason, wageContributionPercent int
	var buyOptionFee int64
	err = dbCtx.Db.QueryRow(getPlayerActiveLoanStmtSql, playerId).Scan(&fromTeamId, &fromTeamName, &returnSeason, &wageContributionPercent, &buyOptionFee)
	if err == nil {
		status.OnLoanFromTeamId = &fromTeamId
		status.OnLoanFromTeamName = &fromTeamName
		status.OnLoanReturnSeason = &returnSeason
		if wageContributionPercent > 0 {
			status.OnLoanWageContributionPercent = &wageContributionPercent
		}
		if buyOptionFee > 0 {
			status.OnLoanBuyOptionFee = &buyOptionFee
		}
	} else if err != sql.ErrNoRows {
		return status, err
	}

	var askingPrice int64
	err = dbCtx.Db.QueryRow(getPlayerTransferListingPriceStmtSql, playerId).Scan(&askingPrice)
	if err == nil {
		status.TransferListingAskingPrice = &askingPrice
	} else if err != sql.ErrNoRows {
		return status, err
	}

	var loanDuration int
	err = dbCtx.Db.QueryRow(getPlayerLoanListingDurationStmtSql, playerId).Scan(&loanDuration)
	if err == nil {
		status.LoanListingDurationSeasons = &loanDuration
	} else if err != sql.ErrNoRows {
		return status, err
	}

	return status, nil
}

const getPlayerTransactionsStmtSql = "SELECT transaction_id, team_id, amount, reason, description, created_at, player_id " +
	"FROM team_transactions WHERE player_id = $1 ORDER BY created_at DESC LIMIT $2"

// GetPlayerTransactions returns this player's transfer/release history - see
// resources/sql/init.sql's team_transactions.player_id column. Contract renewals and loan moves aren't logged here
// (no money changes hands for either, so nothing ever wrote a team_transactions row for them);
// GetPlayerStatus covers "currently on loan" as live state instead.
func GetPlayerTransactions(dbCtx *DatabaseContext, playerId int64, limit int) ([]TeamTransaction, error) {
	rows, err := dbCtx.Db.Query(getPlayerTransactionsStmtSql, playerId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]TeamTransaction, 0)
	for rows.Next() {
		var t TeamTransaction
		var description sql.NullString
		var scannedPlayerId sql.NullInt64
		if err := rows.Scan(&t.Id, &t.TeamId, &t.Amount, &t.Reason, &description, &t.CreatedAt, &scannedPlayerId); err != nil {
			return nil, err
		}
		t.Description = description.String
		if scannedPlayerId.Valid {
			t.PlayerId = &scannedPlayerId.Int64
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

// PlayerDetails is the full profile shown on a player's page - their base attributes, current
// team, career stats, current status, and transfer history. See handlers.PlayerDetailsHandler.
type PlayerDetails struct {
	Player
	// NegotiationStyle is the player's negotiation tier (IntelligenceNaive..IntelligenceMastermind),
	// set only for a viewer whose club has negotiated with him (see NegotiationStyleFor).
	NegotiationStyle *int   `json:"negotiationStyle,omitempty"`
	TeamName         string `json:"teamName"`
	DivisionId       int64  `json:"divisionId"`
	DivisionName     string `json:"divisionName"`
	Country          string `json:"country"`
	// BestPosition is one of GOALKEEPER/DEFENSE/MIDFIELD/STRIKER - whichever PositionValue rates
	// highest for this player - see BestCorePosition. Left/right lateral positions are
	// deliberately excluded: their value formulas don't depend on side, so they'd always tie
	// with each other and couldn't be picked between meaningfully.
	BestPosition int `json:"bestPosition"`
	// AverageSkill is this player's seven core skills reduced to a single 0-99 figure (see
	// common.AverageSkill) - unlike the individual skills/traits, it stays visible on someone
	// else's rostered, unlisted player too, giving a visitor a rough scouting read without
	// exposing the full breakdown. See handlers.PlayerDetailsHandler, which computes it before
	// RedactSkills zeroes the underlying fields.
	AverageSkill int `json:"averageSkill"`
	PlayerCareerStats
	PlayerStatus
	// SeasonAwards is how many times this player has been their league's Golden Boot / Player of
	// the Season - see GetPlayerSeasonAwards. Never redacted: the winners are announced publicly.
	SeasonAwards PlayerSeasonAwards `json:"seasonAwards"`
	History      []TeamTransaction  `json:"history"`
	// Transfers is this player's structured transfer history (origin, destination, fee, kind),
	// newest first - see GetPlayerTransfers. Unlike History it is market information, so it is
	// never redacted.
	Transfers []PlayerTransfer `json:"transfers"`
	// ClubHistory is every club this player has permanently belonged to, most recent first - see
	// GetPlayerClubHistory. Unlike History (financial transactions), this is never redacted: which
	// clubs a player has played for is public the same way their transfer-market listing is.
	ClubHistory []PlayerClubHistoryEntry `json:"clubHistory"`
	// RecentMatches is this player's current team's most recent played league matches, with the
	// player's own goals/cards and (when available) match rating - see GetPlayerRecentMatches.
	// Empty for a free agent. Unlike History/skills, this is never redacted: a team's results and
	// which matches a player featured in are public the same way ClubHistory is.
	RecentMatches []PlayerMatchSummary `json:"recentMatches"`
	// RecentCallUps is this player's most recent national-team call-ups, most recent first - see
	// GetRecentNationalTeamCallUps. Empty for a player never selected. Never redacted, same
	// reasoning as RecentMatches/ClubHistory: a call-up is public the moment it happens (see
	// NotifyTeamOwnerWithLink in SetNationalTeamLineup/SetNationalTeamCupTieLineup).
	RecentCallUps []NationalTeamCallUp `json:"recentCallUps"`
	// LatestEvolution is this player's most recent match-triggered progression run (see
	// GetLatestPlayerEvolution and engine.updatePlayerSkillsAfterMatch) - nil if they've never
	// been through one, or (like skills generally, see RedactSkills) if the viewer isn't the
	// owning team and this player isn't a free agent or transfer-listed.
	LatestEvolution *PlayerEvolution `json:"latestEvolution,omitempty"`
}

// BestCorePosition reports which of the four non-lateral tactical positions
// (GOALKEEPER/DEFENSE/MIDFIELD/STRIKER) PositionValue rates this player highest for - used to
// show a player's natural position on their profile page. Ties favor the earlier position in
// that list.
func BestCorePosition(p Player) int {
	best := GOALKEEPER
	bestValue := PositionValue(GOALKEEPER, p)
	for _, position := range []int{DEFENSE, MIDFIELD, STRIKER} {
		if value := PositionValue(position, p); value > bestValue {
			best = position
			bestValue = value
		}
	}
	return best
}
