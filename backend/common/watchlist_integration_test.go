package common

import (
	"database/sql"
	"strconv"
	"sync"
	"testing"
)

// The player watchlist end to end (openspec add-player-watchlist; needs STRIKER_TEST_DB_DSN).

type watchRig struct {
	t     *testing.T
	db    *sql.DB
	dbCtx *DatabaseContext
}

func newWatchRig(t *testing.T, country string) *watchRig {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, country)
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatal(err)
	}
	return &watchRig{t: t, db: db, dbCtx: dbCtx}
}

// ownedTeam seeds a club with a human owner, so its notifications are written.
func (r *watchRig) ownedTeam(name string, country string) (teamId int64, userId int64) {
	r.t.Helper()
	teamId = seedTransfersTeam(r.t, r.db, name, country)
	if err := r.db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ($1, 'dev', $2) RETURNING user_id`,
		name, name+"-"+strconv.FormatInt(teamId, 10)+"@example.test").Scan(&userId); err != nil {
		r.t.Fatal(err)
	}
	if _, err := r.db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, userId, teamId); err != nil {
		r.t.Fatal(err)
	}
	return teamId, userId
}

// watchNotifications lists the message keys of userId's watchlist notifications.
func (r *watchRig) watchNotifications(userId int64) []string {
	r.t.Helper()
	rows, err := r.db.Query(`SELECT message_key, link FROM notifications WHERE user_id = $1 AND notification_type = $2 ORDER BY notification_id`, userId, NOTIFICATION_WATCHLIST)
	if err != nil {
		r.t.Fatal(err)
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key, link sql.NullString
		if err := rows.Scan(&key, &link); err != nil {
			r.t.Fatal(err)
		}
		keys = append(keys, key.String+" "+link.String)
	}
	return keys
}

func watchedIds(t *testing.T, dbCtx *DatabaseContext, teamId int64) map[int64]bool {
	t.Helper()
	entries, err := GetWatchlist(dbCtx, teamId)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[int64]bool{}
	for _, e := range entries {
		ids[e.Player.Id] = true
	}
	return ids
}

func TestWatchlist_FollowLimitAndOwnPlayer(t *testing.T) {
	r := newWatchRig(t, "zx")
	follower := seedTransfersTeam(t, r.db, "Watch Follower", "zx")
	other := seedTransfersTeam(t, r.db, "Watch Other", "zx")
	own := seedTransfersPlayer(t, r.db, follower, "zx")

	// Follow: a striker of another club appears in the watchlist, with his club; following him
	// again changes nothing, and so does unfollowing a player nobody follows.
	striker := seedTransfersPlayer(t, r.db, other, "zx")
	if err := WatchPlayer(r.dbCtx, follower, striker); err != nil {
		t.Fatal(err)
	}
	if err := WatchPlayer(r.dbCtx, follower, striker); err != nil {
		t.Fatalf("following twice: %s", err)
	}
	entries, err := GetWatchlist(r.dbCtx, follower)
	if err != nil || len(entries) != 1 || entries[0].Player.Id != striker || entries[0].TeamName != "Watch Other" {
		t.Fatalf("watchlist = %+v (%v), want the striker at Watch Other", entries, err)
	}
	// A rostered, unlisted player shows no skills and no contract, only his average.
	if p := entries[0].Player; p.Shot != 0 || p.ContractYearsRemaining != 0 || p.AverageSkill == 0 {
		t.Errorf("followed player leaks private data: %+v", p)
	}
	if err := UnwatchPlayer(r.dbCtx, follower, own); err != nil {
		t.Fatal(err)
	}

	// Own player.
	if err := WatchPlayer(r.dbCtx, follower, own); err != ErrCannotWatchOwnPlayer {
		t.Errorf("following an own player = %v, want ErrCannotWatchOwnPlayer", err)
	}
	if err := WatchPlayer(r.dbCtx, follower, 999999999); err != ErrPlayerNotFound {
		t.Errorf("following an unknown player = %v, want ErrPlayerNotFound", err)
	}

	// Limit: up to 29 one by one, then two follows racing for the 30th place - exactly one wins.
	for len(watchedIds(t, r.dbCtx, follower)) < MaxWatchlistSize-1 {
		if err := WatchPlayer(r.dbCtx, follower, seedTransfersPlayer(t, r.db, other, "zx")); err != nil {
			t.Fatal(err)
		}
	}
	a, b := seedTransfersPlayer(t, r.db, other, "zx"), seedTransfersPlayer(t, r.db, other, "zx")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []int64{a, b} {
		wg.Add(1)
		go func(i int, id int64) {
			defer wg.Done()
			errs[i] = WatchPlayer(r.dbCtx, follower, id)
		}(i, id)
	}
	wg.Wait()
	if n := len(watchedIds(t, r.dbCtx, follower)); n != MaxWatchlistSize {
		t.Fatalf("two follows at 29 left %d players, want exactly %d", n, MaxWatchlistSize)
	}
	if (errs[0] == nil) == (errs[1] == nil) || (errs[0] != ErrWatchlistFull && errs[1] != ErrWatchlistFull) {
		t.Errorf("racing follows = %v / %v, want one success and one ErrWatchlistFull", errs[0], errs[1])
	}
	if err := WatchPlayer(r.dbCtx, follower, seedTransfersPlayer(t, r.db, other, "zx")); err != ErrWatchlistFull {
		t.Errorf("a 31st follow = %v, want ErrWatchlistFull", err)
	}
	// Following one already followed still succeeds at the limit.
	if err := WatchPlayer(r.dbCtx, follower, striker); err != nil {
		t.Errorf("re-following at the limit = %v", err)
	}
}

func TestWatchlist_MarketEventsNotifyFollowers(t *testing.T) {
	r := newWatchRig(t, "zy")
	seller, sellerUser := r.ownedTeam("Watch Seller", "zy")
	buyer, buyerUser := r.ownedTeam("Watch Buyer", "zy")
	fan, fanUser := r.ownedTeam("Watch Fan", "zy")
	player := seedTransfersPlayer(t, r.db, seller, "zy")
	link := " /player/" + strconv.FormatInt(player, 10)

	for _, team := range []int64{buyer, fan} {
		if err := WatchPlayer(r.dbCtx, team, player); err != nil {
			t.Fatal(err)
		}
	}

	// Player listed: both followers are told, with a link to him; the selling club isn't.
	if err := CreateTransferListing(r.dbCtx, player, seller, 500000); err != nil {
		t.Fatal(err)
	}
	for name, user := range map[string]int64{"buyer": buyerUser, "fan": fanUser} {
		if got := r.watchNotifications(user); len(got) != 1 || got[0] != "notifications.messages.watchlistListed"+link {
			t.Errorf("%s after the listing = %v", name, got)
		}
	}
	if got := r.watchNotifications(sellerUser); len(got) != 0 {
		t.Errorf("the selling club got %v", got)
	}
	// Listed player: the watchlist shows the listing and his skills (a buyer must see them).
	entries, err := GetWatchlist(r.dbCtx, fan)
	if err != nil || len(entries) != 1 || entries[0].TransferListing == nil || entries[0].TransferListing.AskingPrice != 500000 || entries[0].Player.Shot == 0 {
		t.Fatalf("fan's watchlist after the listing = %+v (%v)", entries, err)
	}

	// Own club's sale / Signed by the follower: the buyer buys him - he leaves the buyer's
	// watchlist and the buyer gets no watchlist notification for it; the fan is told he moved.
	if err := ExecuteTransfer(r.dbCtx, entries[0].TransferListing.ListingId, buyer); err != nil {
		t.Fatal(err)
	}
	if watchedIds(t, r.dbCtx, buyer)[player] {
		t.Error("the buyer still follows the player he bought")
	}
	if got := r.watchNotifications(buyerUser); len(got) != 1 {
		t.Errorf("buyer notifications after his own purchase = %v, want only the listing one", got)
	}
	if got := r.watchNotifications(fanUser); len(got) != 2 || got[1] != "notifications.messages.watchlistMoved"+link {
		t.Errorf("fan notifications after the sale = %v", got)
	}

	// Player released: the fan is told he can be signed; the releasing club is not.
	if err := ReleasePlayer(r.dbCtx, player, buyer); err != nil {
		t.Fatal(err)
	}
	if got := r.watchNotifications(fanUser); len(got) != 3 || got[2] != "notifications.messages.watchlistFreeAgent"+link {
		t.Errorf("fan notifications after the release = %v", got)
	}
	if entries, err := GetWatchlist(r.dbCtx, fan); err != nil || len(entries) != 1 || entries[0].Player.TeamId != 0 || entries[0].TeamName != "" {
		t.Errorf("fan's watchlist after the release = %+v (%v), want a free agent", entries, err)
	}

	// Loan-listed: a player of the seller's loaned out; the fan is told.
	loanee := seedTransfersPlayer(t, r.db, seller, "zy")
	if err := WatchPlayer(r.dbCtx, fan, loanee); err != nil {
		t.Fatal(err)
	}
	if err := CreateLoanListing(r.dbCtx, loanee, seller, 1, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if got := r.watchNotifications(fanUser); len(got) != 4 || got[3] != "notifications.messages.watchlistLoanListed /player/"+strconv.FormatInt(loanee, 10) {
		t.Errorf("fan notifications after the loan listing = %v", got)
	}
}

func TestWatchlist_ClubResetAndExport(t *testing.T) {
	r := newWatchRig(t, "zv")
	club, userId := r.ownedTeam("Watch Reset", "zv")
	other := seedTransfersTeam(t, r.db, "Watch Elsewhere", "zv")
	for i := 0; i < 3; i++ {
		if err := WatchPlayer(r.dbCtx, club, seedTransfersPlayer(t, r.db, other, "zv")); err != nil {
			t.Fatal(err)
		}
	}

	// Watchlist in the export.
	export, err := ExportUserData(r.dbCtx, userId)
	if err != nil {
		t.Fatal(err)
	}
	if len(export.Watchlist) != 3 || export.Watchlist[0].Name != "Test Player" {
		t.Errorf("exported watchlist = %+v, want the 3 followed players", export.Watchlist)
	}

	// Watchlist: the club reset empties it.
	if _, err := r.db.Exec(`UPDATE teams SET owner_id = NULL, reset_pending_since = now() WHERE team_id = $1`, club); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResetReleasedTeam(r.dbCtx, club); err != nil {
		t.Fatal(err)
	}
	if n := len(watchedIds(t, r.dbCtx, club)); n != 0 {
		t.Errorf("watchlist after the reset has %d players, want none", n)
	}
}
