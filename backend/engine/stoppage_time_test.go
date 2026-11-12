package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func stoppageEvents(minute int, eventType int, count int) []common.Event {
	events := make([]common.Event, 0, count)
	for i := 0; i < count; i++ {
		events = append(events, common.Event{EventType: eventType, Minute: minute})
	}
	return events
}

// "Time lost per half" and "Added minutes per half" (openspec/specs/stoppage-time).
func TestTimeLostPerHalf(t *testing.T) {
	t.Run("quiet first half", func(t *testing.T) {
		if got := timeLost(nil, 1, 45, StoppageBaseFirstHalf); got != 30*time.Second {
			t.Fatalf("got %s, want 30s", got)
		}
	})

	t.Run("busy second half", func(t *testing.T) {
		// 2 goals, 5 substitutions, 1 yellow and 1 injury replaced by a bench player (a sixth substitution).
		events := stoppageEvents(60, common.GOAL, 2)
		events = append(events, stoppageEvents(70, common.SUBSTITUTION, 5)...)
		events = append(events, common.Event{EventType: common.YELLOW_CARD, Minute: 75})
		events = append(events, common.Event{EventType: common.INJURY, Minute: 80}, common.Event{EventType: common.SUBSTITUTION, Minute: 80})
		if got := timeLost(events, 46, 90, StoppageBaseSecondHalf); got != 395*time.Second {
			t.Fatalf("got %s, want 395s", got)
		}
	})

	t.Run("penalty scored", func(t *testing.T) {
		events := []common.Event{{EventType: common.GOAL, Minute: 30, SetPiece: common.SET_PIECE_PENALTY}}
		if got := timeLost(events, 1, 45, 0); got != 85*time.Second {
			t.Fatalf("got %s, want 45s for the penalty and 40s for the goal", got)
		}
	})

	t.Run("penalty missed", func(t *testing.T) {
		events := []common.Event{{EventType: common.NO_GOAL, Minute: 30, SetPiece: common.SET_PIECE_PENALTY}}
		if got := timeLost(events, 1, 45, 0); got != 45*time.Second {
			t.Fatalf("got %s, want 45s", got)
		}
	})

	t.Run("second yellow counts as one red", func(t *testing.T) {
		events := []common.Event{{EventType: common.RED_CARD, Minute: 30}}
		if got := timeLost(events, 1, 45, 0); got != 40*time.Second {
			t.Fatalf("got %s, want 40s", got)
		}
	})

	t.Run("other half and added time ignored", func(t *testing.T) {
		events := []common.Event{
			{EventType: common.GOAL, Minute: 50},                 // second half
			{EventType: common.GOAL, Minute: 45, AddedMinute: 1}, // first-half added time
			{EventType: common.MENTALITY_CHANGE, Minute: 20},     // costs nothing
			{EventType: common.ADDED_TIME, Minute: 45},           // the board itself
		}
		if got := timeLost(events, 1, 45, StoppageBaseFirstHalf); got != 30*time.Second {
			t.Fatalf("got %s, want only the 30s base", got)
		}
	})
}

func TestAddedMinutesPerHalf(t *testing.T) {
	cases := []struct {
		name     string
		lostSecs int
		max      int
		want     int
	}{
		{"rounding 250s", 250, MaxAddedSecondHalf, 4},
		{"minimum 30s", 30, MaxAddedFirstHalf, 1},
		{"nothing lost", 0, MaxAddedFirstHalf, 1},
		{"first-half clamp 330s", 330, MaxAddedFirstHalf, 5},
		{"second-half clamp 395s", 395, MaxAddedSecondHalf, 5},
		{"half up 150s", 150, MaxAddedSecondHalf, 3},
		{"just under half 149s", 149, MaxAddedSecondHalf, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := addedMinutesFor(nil, 1, 45, time.Duration(c.lostSecs)*time.Second, c.max); got != c.want {
				t.Fatalf("%ds lost: got %d added minutes, want %d", c.lostSecs, got, c.want)
			}
		})
	}
	if MaxAddedFirstHalf != 5 || MaxAddedSecondHalf != 5 {
		t.Fatalf("each half is capped at 5 added minutes, got %d and %d", MaxAddedFirstHalf, MaxAddedSecondHalf)
	}
}

// scriptedPipeline stands in for the real stages: every minute it records which minute ran,
// appends that minute's instant (as InitialSimulation does) and lets the test inject events.
type scriptedPipeline struct {
	minutes *[][2]int
	script  func(match *common.MatchSimulation)
}

func (p scriptedPipeline) Simulate(_ *common.DatabaseContext, match *common.MatchSimulation) {
	*p.minutes = append(*p.minutes, [2]int{match.Minute, match.AddedMinute})
	*match.MatchInstants = append(*match.MatchInstants, common.MatchInstant{Minute: match.Minute, AddedMinute: match.AddedMinute})
	if p.script != nil {
		p.script(match)
	}
}

// fullSides returns a match between two complete XIs (ids 1-11 and 101-111), with no stages.
func fullSides() *common.MatchSimulation {
	side := func(first int64) *map[int][]int64 {
		positions := map[int][]int64{common.GOALKEEPER: {first}, common.DEFENSE: {}, common.MIDFIELD: {}, common.STRIKER: {}}
		for i := int64(1); i <= 10; i++ {
			line := []int{common.DEFENSE, common.MIDFIELD, common.STRIKER}[i%3]
			positions[line] = append(positions[line], first+i)
		}
		return &positions
	}
	events := make([]common.Event, 0)
	instants := make([]common.MatchInstant, 0)
	return &common.MatchSimulation{
		Home: 1, Away: 2,
		HomePlayersByPosition: side(1), AwayPlayersByPosition: side(101),
		HomeMinutesByPosition: map[int64]map[int]int{}, AwayMinutesByPosition: map[int64]map[int]int{},
		Events: &events, MatchInstants: &instants,
	}
}

func addEvents(match *common.MatchSimulation, eventType int, count int) {
	for i := 0; i < count; i++ {
		*match.Events = append(*match.Events, common.Event{EventType: eventType, Minute: match.Minute, TeamOwner: match.Home, OtherTeam: match.Away})
	}
}

// boards returns the ADDED_TIME events of a match, as (minute, announced minutes, index in events).
func boards(match *common.MatchSimulation) [][3]int {
	out := make([][3]int, 0)
	for i, e := range *match.Events {
		if e.EventType == common.ADDED_TIME {
			out = append(out, [3]int{e.Minute, *e.AddedTime, i})
		}
	}
	return out
}

// "Added minutes play like any other" (match-simulation) and "Added-time board event"
// (stoppage-time): a first half losing 90 s gets 2 added minutes and a second half losing 210 s
// gets 4, so 96 minutes run in the order 1..45, 45+1, 45+2, 46..90, 90+1..90+4.
func TestPlayMatchMinutes_OrderAndBoards(t *testing.T) {
	match := fullSides()
	minutes := make([][2]int, 0)
	pipeline := scriptedPipeline{minutes: &minutes, script: func(m *common.MatchSimulation) {
		switch {
		case m.Minute == 10:
			addEvents(m, common.INJURY, 1) // 30 + 60 = 90 s
		case m.Minute == 70:
			addEvents(m, common.SUBSTITUTION, 5) // 60 + 150 = 210 s
		case m.Minute == 45 && m.AddedMinute == 0:
			addEvents(m, common.GOAL, 1) // +40 s: 130 s rounds to 2, still
			m.HomeGoals++
		case m.Minute == 90 && m.AddedMinute == 1:
			addEvents(m, common.GOAL, 3) // added time: must not lengthen the half
			m.HomeGoals += 3
		}
	}}

	playMatchMinutes(nil, match, pipeline)

	want := make([][2]int, 0, 96)
	for m := 1; m <= 45; m++ {
		want = append(want, [2]int{m, 0})
	}
	want = append(want, [2]int{45, 1}, [2]int{45, 2})
	for m := 46; m <= 90; m++ {
		want = append(want, [2]int{m, 0})
	}
	want = append(want, [2]int{90, 1}, [2]int{90, 2}, [2]int{90, 3}, [2]int{90, 4})
	if len(minutes) != len(want) {
		t.Fatalf("played %d minutes, want %d: %v", len(minutes), len(want), minutes)
	}
	for i := range want {
		if minutes[i] != want[i] {
			t.Fatalf("minute %d is %d+%d, want %d+%d", i, minutes[i][0], minutes[i][1], want[i][0], want[i][1])
		}
		if inst := (*match.MatchInstants)[i]; inst.Minute != want[i][0] || inst.AddedMinute != want[i][1] {
			t.Fatalf("instant %d is %d+%d, want %d+%d", i, inst.Minute, inst.AddedMinute, want[i][0], want[i][1])
		}
	}
	if match.FirstHalfAddedTime != 2 || match.SecondHalfAddedTime != 4 {
		t.Errorf("added time %d and %d, want 2 and 4", match.FirstHalfAddedTime, match.SecondHalfAddedTime)
	}
	if match.PlayedMinutes != 96 {
		t.Errorf("PlayedMinutes = %d, want 96", match.PlayedMinutes)
	}

	b := boards(match)
	if len(b) != 2 || b[0][0] != 45 || b[0][1] != 2 || b[1][0] != 90 || b[1][1] != 4 {
		t.Fatalf("boards %v, want +2 at 45 and +4 at 90", b)
	}
	events := *match.Events
	// The first board comes right after minute 45's goal, carries the score and names both sides.
	if prev := events[b[0][2]-1]; prev.EventType != common.GOAL || prev.Minute != 45 || prev.AddedMinute != 0 {
		t.Errorf("event before the first board: %+v, want the goal of minute 45", prev)
	}
	if board := events[b[0][2]]; board.HomeGoals != 1 || board.AddedMinute != 0 || board.TeamOwner != 1 || board.OtherTeam != 2 {
		t.Errorf("first board %+v, want 1-0, added minute 0, home and away named", board)
	}
	for _, e := range events[b[1][2]+1:] {
		if e.EventType != common.GOAL || e.Minute != 90 || e.AddedMinute != 1 {
			t.Errorf("event after the second board: %+v, want the goals of 90+1", e)
		}
	}
	// Every player on the pitch throughout is credited every minute, added minutes included.
	if got := minutesPlayedSoFar(match.HomeMinutesByPosition[1]); got != 96 {
		t.Errorf("home goalkeeper played %d minutes, want 96", got)
	}
}

// "Forfeit before the board" (stoppage-time) and the forfeit scenario of "Stored match record for
// every competition" (match-simulation): a quiet first half gets 1 added minute, and a side
// dropping to 7 players in minute 61 ends the match there with no second-half board.
func TestPlayMatchMinutes_ForfeitInTheSecondHalf(t *testing.T) {
	match := fullSides()
	minutes := make([][2]int, 0)
	pipeline := scriptedPipeline{minutes: &minutes, script: func(m *common.MatchSimulation) {
		if m.Minute == 61 {
			positions := *m.HomePlayersByPosition
			positions[common.DEFENSE] = nil
			positions[common.STRIKER] = positions[common.STRIKER][:1]
		}
	}}

	playMatchMinutes(nil, match, pipeline)

	if last := minutes[len(minutes)-1]; last != [2]int{61, 0} {
		t.Fatalf("last minute played %v, want 61", last)
	}
	if len(*match.MatchInstants) != 45+1+16 {
		t.Errorf("%d instants, want 62 (1-45, 45+1, 46-61)", len(*match.MatchInstants))
	}
	if match.FirstHalfAddedTime != 1 || match.SecondHalfAddedTime != 0 {
		t.Errorf("added time %d and %d, want 1 and 0", match.FirstHalfAddedTime, match.SecondHalfAddedTime)
	}
	if b := boards(match); len(b) != 1 || b[0][0] != 45 {
		t.Errorf("boards %v, want only the first half's", b)
	}
	events := *match.Events
	if abandoned := events[len(events)-1]; abandoned.EventType != common.MATCH_ABANDONED || abandoned.Minute != 61 {
		t.Errorf("last event %+v, want MATCH_ABANDONED at 61", abandoned)
	}
}

// A forfeit in the first half's added time stops the match before the second half.
func TestPlayMatchMinutes_ForfeitInAddedTime(t *testing.T) {
	match := fullSides()
	minutes := make([][2]int, 0)
	pipeline := scriptedPipeline{minutes: &minutes, script: func(m *common.MatchSimulation) {
		if m.Minute == 45 && m.AddedMinute == 1 {
			positions := *m.AwayPlayersByPosition
			positions[common.DEFENSE] = nil
			positions[common.STRIKER] = positions[common.STRIKER][:1]
		}
	}}

	playMatchMinutes(nil, match, pipeline)

	if len(minutes) != 46 || minutes[45] != [2]int{45, 1} {
		t.Fatalf("minutes %v, want 1-45 then 45+1", minutes)
	}
	events := *match.Events
	if abandoned := events[len(events)-1]; abandoned.EventType != common.MATCH_ABANDONED || abandoned.Minute != 45 || abandoned.AddedMinute != 1 {
		t.Errorf("last event %+v, want MATCH_ABANDONED at 45+1", abandoned)
	}
}
