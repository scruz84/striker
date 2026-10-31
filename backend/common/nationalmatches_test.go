package common

import (
	"errors"
	"testing"
	"time"
)

func TestApplyNationalMatchStatus(t *testing.T) {
	now := time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC)
	cutoff := now.Add(-LiveBroadcastWindow)
	liveAt := func(minute, home, away int, complete bool) liveScoreLookup {
		return func(string, int64, time.Time) (int, int, int, bool, error) { return minute, home, away, complete, nil }
	}
	noLookup := func(string, int64, time.Time) (int, int, int, bool, error) {
		return 0, 0, 0, false, errors.New("must not look up a live score")
	}

	t.Run("not simulated yet", func(t *testing.T) {
		m := NationalMatch{Kickoff: now.Add(time.Hour)}
		if err := applyNationalMatchStatus(&m, false, 0, 0, cutoff, noLookup); err != nil {
			t.Fatal(err)
		}
		if m.Status != NationalMatchScheduled || m.HomeGoals != nil || m.Minute != nil {
			t.Errorf("got %+v, want scheduled with no score", m)
		}
	})
	t.Run("simulated before kickoff", func(t *testing.T) {
		m := NationalMatch{Kickoff: now.Add(5 * time.Minute)}
		if err := applyNationalMatchStatus(&m, true, 2, 3, cutoff, liveAt(0, 0, 0, false)); err != nil {
			t.Fatal(err)
		}
		if m.Status != NationalMatchLive || *m.Minute != 0 || *m.HomeGoals != 0 || *m.AwayGoals != 0 {
			t.Errorf("got %+v, want live at minute 0, 0-0", m)
		}
	})
	t.Run("live mid-match shows the score as of now, not the final one", func(t *testing.T) {
		m := NationalMatch{Kickoff: now.Add(-40 * time.Minute)}
		if err := applyNationalMatchStatus(&m, true, 2, 3, cutoff, liveAt(40, 1, 0, false)); err != nil {
			t.Fatal(err)
		}
		if m.Status != NationalMatchLive || *m.Minute != 40 || *m.HomeGoals != 1 || *m.AwayGoals != 0 {
			t.Errorf("got status %s minute %v score %v-%v, want live, 40, 1-0", m.Status, *m.Minute, *m.HomeGoals, *m.AwayGoals)
		}
	})
	t.Run("broadcast already complete", func(t *testing.T) {
		m := NationalMatch{Kickoff: now.Add(-110 * time.Minute)}
		if err := applyNationalMatchStatus(&m, true, 2, 3, cutoff, liveAt(90, 2, 3, true)); err != nil {
			t.Fatal(err)
		}
		if m.Status != NationalMatchFinished || m.Minute != nil || *m.HomeGoals != 2 || *m.AwayGoals != 3 {
			t.Errorf("got %+v, want finished 2-3", m)
		}
	})
	t.Run("past the broadcast window", func(t *testing.T) {
		m := NationalMatch{Kickoff: now.Add(-3 * time.Hour)}
		if err := applyNationalMatchStatus(&m, true, 2, 3, cutoff, noLookup); err != nil {
			t.Fatal(err)
		}
		if m.Status != NationalMatchFinished || *m.HomeGoals != 2 || *m.AwayGoals != 3 {
			t.Errorf("got %+v, want finished 2-3", m)
		}
	})
	t.Run("knockout ties use the cup-tie broadcast", func(t *testing.T) {
		var gotKind string
		lookup := func(kind string, _ int64, _ time.Time) (int, int, int, bool, error) {
			gotKind = kind
			return 10, 0, 0, false, nil
		}
		m := NationalMatch{Kind: NationalMatchKindWorldCupKnockout, Kickoff: now.Add(-10 * time.Minute)}
		if err := applyNationalMatchStatus(&m, true, 1, 0, cutoff, lookup); err != nil {
			t.Fatal(err)
		}
		if gotKind != MatchKindCup {
			t.Errorf("looked up kind %q, want %q", gotKind, MatchKindCup)
		}
	})
}
