package common

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestActivityTracker_ThrottlesWrites(t *testing.T) {
	tracker := NewActivityTracker(time.Hour)
	var writes atomic.Int64
	touch := func(int64) (bool, error) { writes.Add(1); return true, nil }

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if exists, _ := tracker.Seen(7, touch); !exists {
				t.Error("an existing user must be seen")
			}
		}()
	}
	wg.Wait()
	// Concurrent first requests may race to the first write, but nowhere near 50.
	if got := writes.Load(); got < 1 || got > 5 {
		t.Fatalf("50 requests caused %d writes, want about 1", got)
	}

	before := writes.Load()
	for i := 0; i < 50; i++ {
		tracker.Seen(7, touch)
	}
	if writes.Load() != before {
		t.Errorf("requests inside the interval must not write again")
	}

	tracker.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	tracker.Seen(7, touch)
	if writes.Load() != before+1 {
		t.Errorf("a request after the interval must write once more, got %d writes", writes.Load()-before)
	}
}

func TestActivityTracker_DeletedUserIsRefusedWithoutQuery(t *testing.T) {
	tracker := NewActivityTracker(time.Hour)
	queried := false
	touch := func(int64) (bool, error) { queried = true; return false, nil }

	exists, _ := tracker.Seen(9, touch) // the row is gone: the touch reports 0 rows
	if exists || !queried {
		t.Fatalf("a user with no row must be refused after one query (exists=%v queried=%v)", exists, queried)
	}
	queried = false
	if exists, _ := tracker.Seen(9, touch); exists || queried {
		t.Errorf("a known-deleted user must be refused without a query (exists=%v queried=%v)", exists, queried)
	}

	tracker.MarkDeleted(11)
	if exists, _ := tracker.Seen(11, touch); exists {
		t.Errorf("MarkDeleted users must be refused")
	}
}

func TestActivityTracker_DatabaseErrorFailsOpen(t *testing.T) {
	tracker := NewActivityTracker(time.Hour)
	exists, err := tracker.Seen(5, func(int64) (bool, error) { return false, errTest })
	if !exists || err == nil {
		t.Errorf("a database error must not log users out (exists=%v err=%v)", exists, err)
	}
}

var errTest = &testError{}

type testError struct{}

func (*testError) Error() string { return "boom" }

func TestDeleteConfirmationMatches(t *testing.T) {
	tests := []struct {
		typed, team string
		want        bool
	}{
		{"Real Betis", "Real Betis", true},
		{"  Real Betis  ", "Real Betis", true},
		{"real betis", "Real Betis", false}, // case-sensitive for the club name
		{"DELETE", "Real Betis", false},     // the word is only for users without a club
		{"", "Real Betis", false},
		{"DELETE", "", true},
		{"delete", "", true},
		{"BORRAR", "", true},
		{"borrar", "", true},
		{"yes", "", false},
	}
	for _, tt := range tests {
		if got := DeleteConfirmationMatches(tt.typed, tt.team); got != tt.want {
			t.Errorf("DeleteConfirmationMatches(%q, %q) = %v, want %v", tt.typed, tt.team, got, tt.want)
		}
	}
}

func TestAcceptTerms_ValidatesBeforeTouchingTheDatabase(t *testing.T) {
	if err := AcceptTerms(nil, 1, CurrentTermsVersion+1, true); err != ErrInvalidTermsVersion {
		t.Errorf("a non-current version = %v, want ErrInvalidTermsVersion", err)
	}
	if err := AcceptTerms(nil, 1, CurrentTermsVersion, false); err != ErrAgeNotConfirmed {
		t.Errorf("age not confirmed = %v, want ErrAgeNotConfirmed", err)
	}
}

func TestTeamJSONHasNoOwnerName(t *testing.T) {
	raw, err := json.Marshal(Team{Id: 1, OwnerId: 2, TeamName: "Club"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "ownername") {
		t.Errorf("the public team response must not expose the owner's name: %s", raw)
	}
}
