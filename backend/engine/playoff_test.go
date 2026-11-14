package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func madrid(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skipf("no tz data: %s", err)
	}
	return loc
}

// Spec scenario "Last matchday broadcast ends": a 14:00 country whose last matchday is on Monday
// plays its playoff on Tuesday at 14:00 (I-II), 14:15 (II-III), 14:30 (III-IV) and 14:45 (IV-V).
func TestPlayoffKickoff_DayAfterAtTheUpperTierTime(t *testing.T) {
	loc := madrid(t)
	lastMatchday := time.Date(2026, 11, 16, 15, 0, 0, 0, loc) // Monday, tier 5's kickoff
	now := time.Date(2026, 11, 16, 16, 50, 0, 0, loc)         // tier 5's broadcast just ended
	for tier, want := range map[int]string{1: "14:00", 2: "14:15", 3: "14:30", 4: "14:45"} {
		got := playoffKickoff(loc, 14*60, tier, lastMatchday, now)
		if got.Weekday() != time.Tuesday || got.Format("15:04") != want {
			t.Errorf("tier %d: kickoff %s, want Tuesday %s", tier, got.Format(time.RFC1123), want)
		}
	}
}

// Spec scenario "Late matchday ending after midnight": the day after the matchday's own date.
func TestPlayoffKickoff_LateMatchdayEndingAfterMidnight(t *testing.T) {
	loc := madrid(t)
	lastMatchday := time.Date(2026, 11, 16, 23, 0, 0, 0, loc) // Monday 23:00
	now := time.Date(2026, 11, 17, 1, 0, 0, 0, loc)           // Tuesday 01:00
	got := playoffKickoff(loc, 14*60, 1, lastMatchday, now)
	if got.Day() != 17 || got.Format("15:04") != "14:00" {
		t.Errorf("kickoff %s, want Tuesday 17 at 14:00", got.Format(time.RFC1123))
	}
}

func TestPlayoffKickoff_MovesADayWhenUnderAnHourAway(t *testing.T) {
	loc := madrid(t)
	lastMatchday := time.Date(2026, 11, 16, 14, 0, 0, 0, loc)
	now := time.Date(2026, 11, 17, 13, 30, 0, 0, loc) // 30 min before Tuesday's 14:00
	got := playoffKickoff(loc, 14*60, 1, lastMatchday, now)
	if got.Day() != 18 || got.Format("15:04") != "14:00" {
		t.Errorf("kickoff %s, want Wednesday 18 at 14:00", got.Format(time.RFC1123))
	}
}

func TestPlayoffKickoff_HoldsTheLocalClockAcrossDST(t *testing.T) {
	loc := madrid(t)
	// Summer time ends in Spain on Sunday 25 Oct 2026 at 03:00.
	lastMatchday := time.Date(2026, 10, 24, 14, 0, 0, 0, loc)
	now := time.Date(2026, 10, 24, 16, 0, 0, 0, loc)
	got := playoffKickoff(loc, 14*60, 1, lastMatchday, now)
	if got.Day() != 25 || got.In(loc).Format("15:04") != "14:00" || got.UTC().Hour() != 13 {
		t.Errorf("kickoff %s (UTC %s), want 25 Oct 14:00 Madrid = 13:00 UTC", got, got.UTC())
	}
}

func TestValidateCountryLeagueKickoff(t *testing.T) {
	if err := common.ValidateCountryLeagueKickoff(14*60, 5); err != nil {
		t.Errorf("14:00 with 5 tiers should be valid: %v", err)
	}
	if err := common.ValidateCountryLeagueKickoff(14*60+10, 5); err == nil {
		t.Error("14:10 is not a quarter-hour")
	}
	// Spec scenario "Too late for the deepest tier": 23:30 with 5 tiers puts tier 5 at 00:30.
	if err := common.ValidateCountryLeagueKickoff(23*60+30, 5); err == nil {
		t.Error("23:30 with 5 tiers should be rejected")
	}
	if got := common.TierKickoffMinute(13*60, 5); got != 14*60 {
		t.Errorf("Portugal at 13:00: tier 5 at minute %d, want 14:00", got)
	}
}
