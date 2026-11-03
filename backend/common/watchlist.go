package common

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// The player watchlist (see openspec/specs/player-watchlist): a club follows up to
// MaxWatchlistSize players of other clubs (or free agents), sees their market status in one
// place and is told when one is listed, becomes a free agent or moves. Who follows a player is
// only ever visible to the follower: nothing but this file reads player_watchlist.

const MaxWatchlistSize = 30

var ErrWatchlistFull = errors.New("watchlist is full")
var ErrCannotWatchOwnPlayer = errors.New("cannot follow a player of your own squad")

// The four market events a follower is told about (see NotifyWatchers).
const (
	WatchEventTransferListed = iota
	WatchEventLoanListed
	WatchEventFreeAgent
	WatchEventMoved
)

// WatchlistEntry is one followed player with his current market status: his club (TeamName is
// empty for a free agent) and his open transfer or loan listing, if any. Player carries what any
// other club may see of him - full skills only while he is listed or a free agent, no contract
// terms while he belongs to a club.
type WatchlistEntry struct {
	Player          Player           `json:"player"`
	TeamName        string           `json:"teamName,omitempty"`
	FollowedAt      time.Time        `json:"followedAt"`
	TransferListing *TransferListing `json:"transferListing,omitempty"`
	LoanListing     *LoanListing     `json:"loanListing,omitempty"`
}

const (
	getPlayerTeamStmtSql       = "SELECT COALESCE(team_id, 0) FROM players WHERE player_id = $1"
	countWatchlistStmtSql      = "SELECT count(*) FROM player_watchlist WHERE team_id = $1"
	isWatchedStmtSql           = "SELECT EXISTS (SELECT 1 FROM player_watchlist WHERE team_id = $1 AND player_id = $2)"
	insertWatchlistStmtSql     = "INSERT INTO player_watchlist (team_id, player_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	deleteWatchlistStmtSql     = "DELETE FROM player_watchlist WHERE team_id = $1 AND player_id = $2"
	clearWatchlistStmtSql      = "DELETE FROM player_watchlist WHERE team_id = $1"
	getWatchersStmtSql         = "SELECT team_id FROM player_watchlist WHERE player_id = $1 ORDER BY team_id"
	exportWatchlistStmtSql     = "SELECT w.player_id, p.first_name, p.last_name, w.created_at FROM player_watchlist w JOIN players p ON p.player_id = w.player_id WHERE w.team_id = $1 ORDER BY w.created_at, w.player_id"
	getWatchlistEntriesStmtSql = "SELECT " + playerColumnsSql + ", w.created_at, COALESCE(t.name, ''), " +
		"tl.listing_id, tl.selling_team_id, tl.asking_price, tl.listed_at, " +
		"ll.listing_id, ll.lending_team_id, ll.duration_seasons, ll.loan_fee, ll.wage_contribution_percent, ll.buy_option_fee, ll.listed_at " +
		"FROM player_watchlist w JOIN players p ON p.player_id = w.player_id " +
		"LEFT JOIN teams t ON t.team_id = p.team_id " +
		"LEFT JOIN transfer_listings tl ON tl.player_id = p.player_id " +
		"LEFT JOIN loan_listings ll ON ll.player_id = p.player_id " +
		"WHERE w.team_id = $1 ORDER BY w.created_at DESC, w.player_id"
)

// DeletePlayerFromWatchlistStmtSql drops a player from one club's watchlist - run inside every
// transaction that makes him that club's player (movePlayerToTeam, SignFreeAgent), so a follower
// who signs him stops following him.
const DeletePlayerFromWatchlistStmtSql = deleteWatchlistStmtSql

// WatchPlayer adds playerId to teamId's watchlist. Following a player already followed succeeds
// without change; following one of the club's own players fails with ErrCannotWatchOwnPlayer, and
// a 31st player with ErrWatchlistFull. The count and the insert run under a per-club advisory lock,
// so concurrent follows never take the list past MaxWatchlistSize.
func WatchPlayer(dbCtx *DatabaseContext, teamId int64, playerId int64) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", NamespacedLockKey(LockNamespaceWatchlist, teamId)); err != nil {
		return err
	}
	var playerTeamId int64
	err = tx.QueryRow(getPlayerTeamStmtSql, playerId).Scan(&playerTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayerNotFound
	}
	if err != nil {
		return err
	}
	if playerTeamId == teamId {
		return ErrCannotWatchOwnPlayer
	}
	var watched bool
	if err := tx.QueryRow(isWatchedStmtSql, teamId, playerId).Scan(&watched); err != nil {
		return err
	}
	if watched {
		return nil
	}
	var count int
	if err := tx.QueryRow(countWatchlistStmtSql, teamId).Scan(&count); err != nil {
		return err
	}
	if count >= MaxWatchlistSize {
		return ErrWatchlistFull
	}
	if _, err := tx.Exec(insertWatchlistStmtSql, teamId, playerId); err != nil {
		return err
	}
	return tx.Commit()
}

// UnwatchPlayer removes playerId from teamId's watchlist; unfollowing a player who isn't followed
// succeeds without change.
func UnwatchPlayer(dbCtx *DatabaseContext, teamId int64, playerId int64) error {
	_, err := dbCtx.Db.Exec(deleteWatchlistStmtSql, teamId, playerId)
	return err
}

// ClearWatchlist empties teamId's watchlist inside ex (the club reset's transaction).
func ClearWatchlist(ex sqlExecutor, teamId int64) error {
	_, err := ex.Exec(clearWatchlistStmtSql, teamId)
	return err
}

// GetWatchlist returns teamId's followed players, most recently followed first, each with his
// club and open listing. A rostered player's skills are hidden unless he is transfer-listed, and
// his contract terms always, exactly as on any other club's squad page (see GetTeamDetailsById);
// his AverageSkill stays visible.
func GetWatchlist(dbCtx *DatabaseContext, teamId int64) ([]WatchlistEntry, error) {
	rows, err := dbCtx.Db.Query(getWatchlistEntriesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]WatchlistEntry, 0)
	for rows.Next() {
		var e WatchlistEntry
		var tlId, tlTeam, tlPrice sql.NullInt64
		var tlAt sql.NullTime
		var llId, llTeam, llDuration, llFee, llWage, llBuy sql.NullInt64
		var llAt sql.NullTime
		player, err := scanPlayer(rows, &e.FollowedAt, &e.TeamName, &tlId, &tlTeam, &tlPrice, &tlAt,
			&llId, &llTeam, &llDuration, &llFee, &llWage, &llBuy, &llAt)
		if err != nil {
			return nil, err
		}
		player.AverageSkill = AverageSkill(*player)
		if tlId.Valid {
			e.TransferListing = &TransferListing{ListingId: tlId.Int64, SellingTeamId: tlTeam.Int64, SellingTeamName: e.TeamName,
				AskingPrice: tlPrice.Int64, ListedAt: tlAt.Time}
		}
		if llId.Valid {
			e.LoanListing = &LoanListing{ListingId: llId.Int64, LendingTeamId: llTeam.Int64, LendingTeamName: e.TeamName,
				DurationSeasons: int(llDuration.Int64), LoanFee: llFee.Int64, WageContributionPercent: int(llWage.Int64),
				BuyOptionFee: llBuy.Int64, ListedAt: llAt.Time}
		}
		wage := player.Wage
		player.RedactFinancials()
		if player.TeamId != 0 {
			player.Wage = wage // like the squad page: a visitor sees the wage, not the contract
		}
		player.RedactSkills(e.TransferListing != nil)
		e.Player = *player
		if e.TransferListing != nil {
			e.TransferListing.Player = e.Player
		}
		if e.LoanListing != nil {
			e.LoanListing.Player = e.Player
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ExportedWatchedPlayer is one followed player in the data export.
type ExportedWatchedPlayer struct {
	PlayerId   int64     `json:"playerId"`
	Name       string    `json:"name"`
	FollowedAt time.Time `json:"followedAt"`
}

// exportWatchlist lists teamId's followed players for the data export, oldest first.
func exportWatchlist(dbCtx *DatabaseContext, teamId int64) ([]ExportedWatchedPlayer, error) {
	rows, err := dbCtx.Db.Query(exportWatchlistStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExportedWatchedPlayer, 0)
	for rows.Next() {
		var w ExportedWatchedPlayer
		var first, last string
		if err := rows.Scan(&w.PlayerId, &first, &last, &w.FollowedAt); err != nil {
			return nil, err
		}
		w.Name = first + " " + last
		out = append(out, w)
	}
	return out, rows.Err()
}

// NotifyWatchers tells every club following playerId about a market event (WatchEvent*), except
// the clubs that caused it (actingTeamIds: seller and buyer, lender and borrower, the releasing
// club) - one NOTIFICATION_WATCHLIST each, linking to the player. teamName and price fill the
// message where the event has them. Best-effort and called after the event's transaction commits:
// a failure here only loses notifications, never the event.
func NotifyWatchers(dbCtx *DatabaseContext, playerId int64, event int, teamName string, price int64, actingTeamIds ...int64) {
	rows, err := dbCtx.Db.Query(getWatchersStmtSql, playerId)
	if err != nil {
		log.Errorf("error reading the followers of player %d: %s", playerId, err)
		return
	}
	acting := make(map[int64]bool, len(actingTeamIds))
	for _, id := range actingTeamIds {
		acting[id] = true
	}
	watchers := make([]int64, 0)
	for rows.Next() {
		var teamId int64
		if err := rows.Scan(&teamId); err != nil {
			rows.Close()
			log.Errorf("error reading the followers of player %d: %s", playerId, err)
			return
		}
		if !acting[teamId] {
			watchers = append(watchers, teamId)
		}
	}
	rows.Close()
	if len(watchers) == 0 {
		return
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		log.Errorf("error reading player %d to notify his followers: %s", playerId, err)
		return
	}
	msg := watchlistMessage(player.FirstName+" "+player.LastName, playerId, event, teamName, price)
	link := "/player/" + strconv.FormatInt(playerId, 10)
	for _, teamId := range watchers {
		if err := NotifyTeamOwnerWithLink(dbCtx, teamId, NOTIFICATION_WATCHLIST, msg, link); err != nil {
			log.Errorf("error notifying team %d about followed player %d: %s", teamId, playerId, err)
		}
	}
}

func watchlistMessage(playerName string, playerId int64, event int, teamName string, price int64) NotificationMessage {
	params := map[string]any{"player": playerName, "playerId": playerId, "team": teamName}
	switch event {
	case WatchEventTransferListed:
		params["price"] = FormatFee(price)
		return NotificationMessage{Text: playerName + ", on your watchlist, has been transfer-listed by " + teamName + " for " + FormatFee(price),
			Key: "notifications.messages.watchlistListed", Params: params}
	case WatchEventLoanListed:
		return NotificationMessage{Text: playerName + ", on your watchlist, has been loan-listed by " + teamName,
			Key: "notifications.messages.watchlistLoanListed", Params: params}
	case WatchEventFreeAgent:
		return NotificationMessage{Text: playerName + ", on your watchlist, is a free agent and can be signed",
			Key: "notifications.messages.watchlistFreeAgent", Params: params}
	default:
		return NotificationMessage{Text: playerName + ", on your watchlist, has moved to " + teamName,
			Key: "notifications.messages.watchlistMoved", Params: params}
	}
}
