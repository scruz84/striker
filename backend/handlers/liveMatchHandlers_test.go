package handlers

import (
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/scruz84/striker/common"
)

// redisReachableForTest skips the test when no Redis is reachable at REDIS_HOST:REDIS_PORT -
// serveLiveMatch's stale-broadcast handling goes through a real LiveMatchHub (Redis pub/sub and
// history), same as common's own LiveMatchHub tests (see common/livehub_test.go), so there's no
// meaningful way to exercise it against a fake.
func redisReachableForTest(t *testing.T) {
	t.Helper()
	addr := os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT")
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Skipf("redis not reachable at %s, skipping: %s", addr, err)
	}
	_ = conn.Close()
}

// newLiveMatchTestServer wires serveLiveMatch straight up behind a single "/live" route, with a
// fixed channel/played/score baked in - the surrounding id lookup that LiveMatchHandler and
// siblings do (see WeekMatchById etc.) is irrelevant to what's under test here.
func newLiveMatchTestServer(t *testing.T, channel string, played bool, homeGoals, awayGoals int) (*httptest.Server, *common.DatabaseContext) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dbCtx := &common.DatabaseContext{LiveHub: common.NewLiveMatchHub()}

	router := gin.New()
	router.GET("/live", func(c *gin.Context) {
		serveLiveMatch(c, dbCtx, channel, played, homeGoals, awayGoals)
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, dbCtx
}

func dialLiveMatch(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/live"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial live match websocket: %s", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readLiveMatchMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) common.LiveMatchMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var msg common.LiveMatchMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read live match message: %s", err)
	}
	return msg
}

// TestServeLiveMatch_StaleBroadcastReportsFinalResult reproduces the reported bug: a live viewer
// stuck showing a frozen minute forever. engine.BroadcastMatchLive runs as a bare goroutine with
// no persisted "still alive" state (see liveMatchStaleTimeout's own comment), so a backend
// crash/redeploy mid-broadcast leaves serveLiveMatch's connection open with nothing left to ever
// publish a "complete" for it. Once liveMatchStaleTimeout elapses with no further message,
// serveLiveMatch must give up waiting and report the (already known, already correct) final score
// instead of hanging until liveMatchMaxConnectionLifetime.
func TestServeLiveMatch_StaleBroadcastReportsFinalResult(t *testing.T) {
	redisReachableForTest(t)

	old := liveMatchStaleTimeout
	liveMatchStaleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { liveMatchStaleTimeout = old })

	channel := common.LeagueMatchChannel(time.Now().UnixNano())
	server, dbCtx := newLiveMatchTestServer(t, channel, true, 2, 1)

	// Simulates BroadcastMatchLive having started, then dying partway through: one "instant"
	// published, nothing more ever comes.
	dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
		Type: "instant", HomeGoals: 1, AwayGoals: 0,
		Instant: &common.MatchInstant{Minute: 12},
	})

	conn := dialLiveMatch(t, server)

	first := readLiveMatchMessage(t, conn, 2*time.Second)
	if first.Type != "instant" {
		t.Fatalf("expected the already-published instant first, got: %+v", first)
	}

	stale := readLiveMatchMessage(t, conn, 2*time.Second)
	if stale.Type != "complete" {
		t.Fatalf("expected serveLiveMatch to give up and report complete, got: %+v", stale)
	}
	if stale.HomeGoals != 2 || stale.AwayGoals != 1 {
		t.Fatalf("expected the true final score (2-1), got %d-%d", stale.HomeGoals, stale.AwayGoals)
	}
}

// TestServeLiveMatch_QuietBeforeKickoffIsNotStale makes sure the stale-broadcast timeout doesn't
// fire just because a viewer opened a match page long before kickoff and the broadcast simply
// hasn't started yet - see broadcastStarted in serveLiveMatch.
func TestServeLiveMatch_QuietBeforeKickoffIsNotStale(t *testing.T) {
	redisReachableForTest(t)

	old := liveMatchStaleTimeout
	liveMatchStaleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { liveMatchStaleTimeout = old })

	channel := common.LeagueMatchChannel(time.Now().UnixNano())
	server, _ := newLiveMatchTestServer(t, channel, false, 0, 0)

	conn := dialLiveMatch(t, server)

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg common.LiveMatchMessage
	if err := conn.ReadJSON(&msg); err == nil {
		t.Fatalf("expected no message before the broadcast has started, got: %+v", msg)
	}
}
