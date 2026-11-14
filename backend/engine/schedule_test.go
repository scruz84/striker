package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func TestNextScheduledOccurrence_LandsOnSelectedWeekdayAtConfiguredTime(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC) // a Monday
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(from, schedule)

	if occurrence.Weekday() != time.Saturday {
		t.Fatalf("expected a Saturday, got %s", occurrence.Weekday())
	}
	if occurrence.Hour() != 14 || occurrence.Minute() != 0 {
		t.Fatalf("expected 14:00, got %02d:%02d", occurrence.Hour(), occurrence.Minute())
	}
	if occurrence.Before(from) {
		t.Fatalf("occurrence %v must not be before from %v", occurrence, from)
	}
}

func TestNextScheduledOccurrence_UsesTodayIfTimeHasNotPassedYet(t *testing.T) {
	saturdayMorning := time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC) // a Saturday, before 14:00
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(saturdayMorning, schedule)

	expected := time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC)
	if !occurrence.Equal(expected) {
		t.Fatalf("expected today at 14:00 (%v), got %v", expected, occurrence)
	}
}

func TestNextScheduledOccurrence_SkipsToNextWeekIfTimeAlreadyPassedToday(t *testing.T) {
	saturdayEvening := time.Date(2026, 3, 7, 20, 0, 0, 0, time.UTC) // a Saturday, after 14:00
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(saturdayEvening, schedule)

	expected := time.Date(2026, 3, 14, 14, 0, 0, 0, time.UTC) // the following Saturday
	if !occurrence.Equal(expected) {
		t.Fatalf("expected the following Saturday (%v), got %v", expected, occurrence)
	}
}

func TestComputeFixtureDates_SingleWeekdayGivesOneRoundPerWeek(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC) // a Monday
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 4)

	for i, d := range dates {
		if d.Weekday() != time.Saturday {
			t.Fatalf("round %d: expected a Saturday, got %s", i, d.Weekday())
		}
		if i > 0 && dates[i].Sub(dates[i-1]) != 7*24*time.Hour {
			t.Fatalf("round %d: expected exactly 7 days after the previous round, got %v", i, dates[i].Sub(dates[i-1]))
		}
	}
}

func TestComputeFixtureDates_AllWeekdaysGivesOneRoundPerDay(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	schedule := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 5)

	for i := 1; i < len(dates); i++ {
		if dates[i].Sub(dates[i-1]) != 24*time.Hour {
			t.Fatalf("round %d: expected exactly 1 day after the previous round, got %v", i, dates[i].Sub(dates[i-1]))
		}
	}
}

func TestComputeFixtureDates_MultipleWeekdaysOnlyLandOnSelectedDays(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdayTuesday | common.WeekdayFriday, Hour: 20, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 6)

	for i, d := range dates {
		if d.Weekday() != time.Tuesday && d.Weekday() != time.Friday {
			t.Fatalf("round %d: expected Tuesday or Friday, got %s", i, d.Weekday())
		}
		if i > 0 && !dates[i].After(dates[i-1]) {
			t.Fatalf("round %d: expected strictly increasing dates", i)
		}
	}
}

func TestComputeFixtureDates_ReturnsRequestedCount(t *testing.T) {
	schedule := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}

	dates := ComputeFixtureDates(time.Now(), schedule, 0)
	if len(dates) != 0 {
		t.Fatalf("expected 0 dates, got %d", len(dates))
	}

	dates = ComputeFixtureDates(time.Now(), schedule, 3)
	if len(dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(dates))
	}
}

// A season closes at 15:50 while its last round (dated 16:00, simulated 15 minutes ahead) is still
// in the future; with a daily 16:00 schedule the next season's round 1 used to land on that very
// minute.
func TestFixtureStart_NextSeasonStartsAfterLastRoundStillInTheFuture(t *testing.T) {
	schedule := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}
	now := time.Date(2026, 3, 2, 15, 50, 0, 0, time.UTC)
	lastRound := time.Date(2026, 3, 2, 16, 0, 0, 0, time.UTC)

	naive := ComputeFixtureDates(now, schedule, 1)[0]
	if !naive.Equal(lastRound) {
		t.Fatalf("test premise: searching from now should hit the last round's slot, got %v", naive)
	}

	dates := ComputeFixtureDates(FixtureStart(now, &lastRound), schedule, 2)
	want := time.Date(2026, 3, 3, 16, 0, 0, 0, time.UTC)
	if !dates[0].Equal(want) {
		t.Fatalf("round 1: got %v, want %v", dates[0], want)
	}
	if !dates[0].After(lastRound) {
		t.Fatalf("round 1 %v must be strictly after the last round %v", dates[0], lastRound)
	}
}

func TestFixtureStart_IgnoresPastOrMissingMatches(t *testing.T) {
	now := time.Date(2026, 3, 2, 15, 50, 0, 0, time.UTC)
	if got := FixtureStart(now, nil); !got.Equal(now) {
		t.Fatalf("no previous matches: got %v, want now", got)
	}
	past := now.Add(-48 * time.Hour)
	if got := FixtureStart(now, &past); !got.Equal(now) {
		t.Fatalf("long-finished previous season: got %v, want now", got)
	}
	future := now.Add(10 * time.Minute)
	if got := FixtureStart(now, &future); !got.Equal(future.Add(time.Minute)) {
		t.Fatalf("future last match: got %v, want one minute after it", got)
	}
}

// Spec scenario "Staggered tiers": a 5-tier Spanish country with the default 14:00 kickoff dates
// tier 1 at 14:00, tier 2 at 14:15 ... tier 5 at 15:00, Madrid time.
func TestEffectiveLeagueSchedule_StaggeredTiers(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skipf("no tz data: %s", err)
	}
	def := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}
	from := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	for tier, want := range map[int]string{1: "14:00", 2: "14:15", 3: "14:30", 4: "14:45", 5: "15:00"} {
		schedule := common.EffectiveLeagueSchedule(nil, def, "es", tier, common.DefaultLeagueKickoffMinute)
		got := NextScheduledOccurrence(from, schedule).In(loc).Format("15:04")
		if got != want {
			t.Errorf("tier %d kicks off at %s Madrid time, want %s", tier, got, want)
		}
	}
}

// Spec scenario "Daylight saving": tier 1 keeps 14:00 on the Madrid clock across the end of
// summer time (25 Oct 2026).
func TestComputeFixtureDates_LocalKickoffHoldsAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skipf("no tz data: %s", err)
	}
	def := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask}
	schedule := common.EffectiveLeagueSchedule(nil, def, "es", 1, 14*60)
	dates := ComputeFixtureDates(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC), schedule, 4)
	for _, d := range dates {
		if got := d.In(loc).Format("15:04"); got != "14:00" {
			t.Errorf("%s: %s on the Madrid clock, want 14:00", d.In(loc).Format("Mon 2 Jan"), got)
		}
	}
	if dates[0].UTC().Hour() == dates[3].UTC().Hour() {
		t.Errorf("the UTC kickoff should move by an hour across the change: %v", dates)
	}
}

// Spec scenario "Override wins": an admin-set override is used as-is, on the server clock.
func TestEffectiveLeagueSchedule_OverrideWins(t *testing.T) {
	override := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 18, Minute: 0}
	got := common.EffectiveLeagueSchedule(&override, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask}, "es", 3, 14*60)
	if got.Hour != 18 || got.Minute != 0 || got.WeekdaysMask != common.WeekdaySaturday || got.Location != nil {
		t.Errorf("override not kept: %+v", got)
	}
	derived := common.EffectiveLeagueSchedule(nil, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask}, "es", 3, 14*60)
	if derived.Hour != 14 || derived.Minute != 30 {
		t.Errorf("tier 3 without an override should kick off at 14:30, got %02d:%02d", derived.Hour, derived.Minute)
	}
}
