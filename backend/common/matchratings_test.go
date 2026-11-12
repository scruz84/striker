package common

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

const (
	ratingsHome = int64(1000)
	ratingsAway = int64(2000)
)

// contributionMinute is one instant listing the given players with the same contribution.
type contributionMinute struct {
	home map[int64]PlayerContribution
	away map[int64]PlayerContribution
}

// instantsWith builds minutes 1..90, each listing every player whose [from, to] range covers it -
// a starter is [1, 90], a substitute who came on at 60 is [61, 90] (the minute-60 instant is
// recorded before the substitution happens that minute, see engine.runMatchSimulation).
type onPitch struct {
	playerId   int64
	home       bool
	position   int
	from, to   int
	own, rival float32
}

func instantsWith(players ...onPitch) []MatchInstant {
	instants := make([]MatchInstant, 0, 90)
	for minute := 1; minute <= 90; minute++ {
		c := &InstantContributions{Home: map[int64]PlayerContribution{}, Away: map[int64]PlayerContribution{}}
		for _, p := range players {
			if minute < p.from || minute > p.to {
				continue
			}
			side := c.Away
			if p.home {
				side = c.Home
			}
			side[p.playerId] = PlayerContribution{Position: p.position, Own: p.own, Rival: p.rival}
		}
		instants = append(instants, MatchInstant{Minute: minute, PlayerContributions: c})
	}
	return instants
}

func ratingOf(t *testing.T, ratings []PlayerRating, playerId int64) PlayerRating {
	t.Helper()
	for _, r := range ratings {
		if r.PlayerId == playerId {
			return r
		}
	}
	t.Fatalf("player %d has no rating in %+v", playerId, ratings)
	return PlayerRating{}
}

func goal(minute int, teamId int64, playerId int64, homeGoals int, awayGoals int) Event {
	return Event{EventType: GOAL, Minute: minute, TeamOwner: teamId, TeamOwnerPlayers: []int64{playerId}, HomeGoals: homeGoals, AwayGoals: awayGoals}
}

func substitution(minute int, teamId int64, out int64, in int64) Event {
	return Event{EventType: SUBSTITUTION, Minute: minute, TeamOwner: teamId, TeamOwnerPlayers: []int64{out, in}}
}

func TestComputePlayerRatings_SubstitutionKeepsEachPlayersOwnRecord(t *testing.T) {
	instants := instantsWith(
		onPitch{playerId: 7, home: true, position: STRIKER, from: 1, to: 60, own: 50, rival: 50},
		onPitch{playerId: 17, home: true, position: STRIKER, from: 61, to: 90, own: 50, rival: 50},
	)
	events := []Event{goal(20, ratingsHome, 7, 1, 0), substitution(60, ratingsHome, 7, 17)}
	ratings := ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 90)

	starter := ratingOf(t, ratings, 7)
	if starter.Minutes != 60 || starter.Goals != 1.4 || starter.WentOff == nil || *starter.WentOff != 60 || starter.CameOn != nil {
		t.Fatalf("starter: %+v", starter)
	}
	// 6.0 + 1.4 goal + 0.3 * 60/90 = 0.2 result.
	if starter.Rating != 7.6 || !starter.Rated {
		t.Fatalf("starter rating: got %v (rated %v), want 7.6", starter.Rating, starter.Rated)
	}
	sub := ratingOf(t, ratings, 17)
	if sub.Minutes != 30 || sub.Goals != 0 || sub.CameOn == nil || *sub.CameOn != 60 || sub.Result != 0.1 || sub.Rating != 6.1 {
		t.Fatalf("substitute: %+v", sub)
	}
}

func TestComputePlayerRatings_SentOffKeepsHisMinutesAndCard(t *testing.T) {
	instants := instantsWith(onPitch{playerId: 4, home: false, position: DEFENSE, from: 1, to: 35, own: 50, rival: 50})
	events := []Event{{EventType: RED_CARD, Minute: 35, TeamOwner: ratingsAway, TeamOwnerPlayers: []int64{4}}}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 90), 4)
	if r.Minutes != 35 || r.Cards != -1.5 || r.WentOff == nil || *r.WentOff != 35 || r.Rating != 4.5 {
		t.Fatalf("sent off: %+v", r)
	}
}

func TestComputePlayerRatings_StrikerWithTwoGoalsInAWin(t *testing.T) {
	// 60 vs 49.1 gives a mean edge of 10.9/109.1 ~ 0.0999, so a zone component of 3.0 * 0.0999 ~ +0.3.
	instants := instantsWith(onPitch{playerId: 9, home: true, position: STRIKER, from: 1, to: 90, own: 60, rival: 49.1})
	events := []Event{goal(10, ratingsHome, 9, 1, 0), goal(70, ratingsHome, 9, 2, 0)}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 90), 9)
	if r.Zone != 0.3 || r.Goals != 2.8 || r.Result != 0.3 || r.Rating != 9.4 {
		t.Fatalf("striker: %+v", r)
	}
	if r.Own == nil || *r.Own != 60 || r.Rival == nil || *r.Rival != 49.1 {
		t.Fatalf("striker contribution: own %v rival %v", r.Own, r.Rival)
	}
}

func TestComputePlayerRatings_LateSubstituteInADefeat(t *testing.T) {
	// Edge (45 - 50) / 95 = -0.0526 -> zone -0.158 -> -0.2; result -0.3 * 30/90 = -0.1.
	instants := instantsWith(onPitch{playerId: 8, home: true, position: MIDFIELD, from: 61, to: 90, own: 45, rival: 50})
	events := []Event{substitution(60, ratingsHome, 3, 8), goal(75, ratingsAway, 30, 0, 1)}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 90), 8)
	if r.Zone != -0.2 || r.Result != -0.1 || r.Rating != 5.7 || !r.Rated {
		t.Fatalf("late sub: %+v", r)
	}
}

func TestComputePlayerRatings_ClampedToTen(t *testing.T) {
	instants := instantsWith(onPitch{playerId: 9, home: true, position: STRIKER, from: 1, to: 90, own: 90, rival: 10})
	events := []Event{goal(10, ratingsHome, 9, 1, 0), goal(20, ratingsHome, 9, 2, 0), goal(30, ratingsHome, 9, 3, 0)}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 90), 9)
	if r.Rating != 10 {
		t.Fatalf("got %v, want 10 (6 + 1.5 + 4.2 + 0.3 clamped)", r.Rating)
	}
}

func TestComputePlayerRatings_ZoneComponent(t *testing.T) {
	// Edge 0.15 every minute: own 57.5, rival 42.5 -> 15/100.
	instants := instantsWith(onPitch{playerId: 5, home: true, position: DEFENSE, from: 1, to: 90, own: 57.5, rival: 42.5})
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, nil, 90), 5)
	if r.Zone != 0.5 { // 0.45 rounded half away from zero to one decimal
		t.Fatalf("zone: got %v, want 0.5", r.Zone)
	}
	// Mean edge 0.6 (80 vs 20) is capped at +1.5.
	instants = instantsWith(onPitch{playerId: 5, home: true, position: DEFENSE, from: 1, to: 90, own: 80, rival: 20})
	r = ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, nil, 90), 5)
	if r.Zone != 1.5 {
		t.Fatalf("capped zone: got %v, want 1.5", r.Zone)
	}
}

func TestComputePlayerRatings_UnderTenMinutesIsUnrated(t *testing.T) {
	instants := instantsWith(
		onPitch{playerId: 1, home: true, position: MIDFIELD, from: 1, to: 90, own: 50, rival: 50},
		onPitch{playerId: 21, home: true, position: MIDFIELD, from: 85, to: 90, own: 50, rival: 50},
		onPitch{playerId: 22, home: true, position: MIDFIELD, from: 81, to: 90, own: 50, rival: 50},
	)
	ratings := ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, nil, 90)
	if r := ratingOf(t, ratings, 21); r.Rated || r.Minutes != 6 {
		t.Fatalf("6-minute sub should be s.c.: %+v", r)
	}
	if r := ratingOf(t, ratings, 22); !r.Rated || r.Minutes != 10 {
		t.Fatalf("10-minute sub should be rated: %+v", r)
	}
	byId := *RatingsById(ratings)
	if _, ok := byId[21]; ok {
		t.Fatal("an unrated player must not appear in the id -> rating map")
	}
	// As of minute 5, a starter's 5 minutes are all that has elapsed: he is rated.
	if r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, nil, 5), 1); !r.Rated || r.Minutes != 5 {
		t.Fatalf("starter at minute 5: %+v", r)
	}
}

func TestComputePlayerRatings_AsOfAnEarlierMinute(t *testing.T) {
	instants := instantsWith(onPitch{playerId: 1, home: true, position: MIDFIELD, from: 1, to: 90, own: 50, rival: 50})
	events := []Event{goal(30, ratingsAway, 30, 0, 1), goal(60, ratingsHome, 1, 1, 1), goal(80, ratingsHome, 1, 2, 1)}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, 45), 1)
	// At 45' it's 0-1 and the player has no goal yet: -0.3 * 45/90 = -0.15 -> -0.2.
	if r.Minutes != 45 || r.Goals != 0 || r.Result != -0.2 || r.Rating != 5.8 {
		t.Fatalf("as of 45': %+v", r)
	}
}

func TestComputePlayerRatings_LegacyMatchFallsBackToLineupsAndEvents(t *testing.T) {
	home := &Lineup{TeamId: ratingsHome, Positions: []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 7, Position: STRIKER}}}
	away := &Lineup{TeamId: ratingsAway, Positions: []LineupPosition{{PlayerId: 101, Position: GOALKEEPER}}}
	events := []Event{goal(20, ratingsHome, 7, 1, 0), substitution(60, ratingsHome, 7, 17)}
	ratings := ComputePlayerRatings(ratingsHome, ratingsAway, home, away, nil, events, 90)

	starter := ratingOf(t, ratings, 7)
	if starter.Minutes != 60 || starter.Goals != 1.4 || starter.Zone != 0 || starter.Own != nil {
		t.Fatalf("legacy starter: %+v", starter)
	}
	sub := ratingOf(t, ratings, 17)
	if sub.Minutes != 30 || sub.Result != 0.1 || sub.Position != STRIKER || sub.TeamId != ratingsHome {
		t.Fatalf("legacy substitute: %+v", sub)
	}
	if keeper := ratingOf(t, ratings, 101); keeper.Result != -0.3 || keeper.Minutes != 90 {
		t.Fatalf("legacy away keeper: %+v", keeper)
	}
}

func TestComputePlayerRatings_LegacyMatchWithoutLineupsHasNoRatings(t *testing.T) {
	if got := ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, nil, []Event{goal(5, ratingsHome, 7, 1, 0)}, 90); got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

func TestComputePlayerRatings_MostPlayedPosition(t *testing.T) {
	instants := instantsWith(
		onPitch{playerId: 6, home: true, position: MIDFIELD, from: 1, to: 30, own: 50, rival: 50},
		onPitch{playerId: 6, home: true, position: LATERAL_MIDFIELD_LEFT, from: 31, to: 90, own: 50, rival: 50},
	)
	if r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, nil, 90), 6); r.Position != LATERAL_MIDFIELD_LEFT {
		t.Fatalf("position: got %d, want %d", r.Position, LATERAL_MIDFIELD_LEFT)
	}
}

func TestRoundTenth_HalfAwayFromZero(t *testing.T) {
	for in, want := range map[float64]float64{0.45: 0.5, -0.45: -0.5, 0.15: 0.2, -0.15: -0.2, 2.8000000000000003: 2.8} {
		if got := roundTenth(in); math.Abs(got-want) > 1e-12 {
			t.Errorf("roundTenth(%v) = %v, want %v", in, got, want)
		}
	}
}

// ratingsParityFixture is frontend/src/utils/testdata/match_ratings_parity.json: matches (one with per-minute
// contributions, one simulated before them) and the ratings expected at several minutes.
// frontend/src/utils/liveMatchRatings.test.ts asserts the very same file, so the Go and TypeScript
// formulas can't drift apart. Regenerate it with STRIKER_WRITE_RATINGS_FIXTURE=1 after a deliberate
// formula change (and check the diff).
type ratingsParityCase struct {
	Name       string         `json:"name"`
	HomeTeamId int64          `json:"homeTeamId"`
	AwayTeamId int64          `json:"awayTeamId"`
	HomeLineup *Lineup        `json:"homeLineup"`
	AwayLineup *Lineup        `json:"awayLineup"`
	Instants   []MatchInstant `json:"instants"`
	Events     []Event        `json:"events"`
	Expected   []struct {
		Minute  int            `json:"minute"`
		Ratings []PlayerRating `json:"ratings"`
	} `json:"expected"`
}

// ratingsParityFixturePath lives on the frontend side: its Docker build only sees frontend/, and its
// type-check compiles liveMatchRatings.test.ts (which imports the fixture), while this test only
// ever runs with the whole repository checked out.
const ratingsParityFixturePath = "../../frontend/src/utils/testdata/match_ratings_parity.json"

// parityCases builds the fixture's matches: a 3-a-side contribution match with a substitution, a
// red card, an injury, goals on both sides, a position change and one-decimal contributions that
// drift every minute, plus the same events on a match without contributions.
func parityCases() []ratingsParityCase {
	home, away := int64(11), int64(22)
	contribution := func(seed int, minute int) float32 {
		return float32(math.Round((30+float64(seed)*2.3+float64((seed*37+minute*13)%200)/10)*10) / 10)
	}
	instants := make([]MatchInstant, 0, 90)
	for minute := 1; minute <= 90; minute++ {
		c := &InstantContributions{Home: map[int64]PlayerContribution{}, Away: map[int64]PlayerContribution{}}
		c.Home[1] = PlayerContribution{Position: GOALKEEPER, Own: contribution(1, minute), Rival: contribution(2, minute)}
		if minute <= 60 {
			c.Home[2] = PlayerContribution{Position: STRIKER, Own: contribution(3, minute), Rival: contribution(4, minute)}
		} else {
			c.Home[12] = PlayerContribution{Position: STRIKER, Own: contribution(5, minute), Rival: contribution(6, minute)}
		}
		position := MIDFIELD
		if minute > 45 {
			position = LATERAL_MIDFIELD_RIGHT
		}
		c.Home[3] = PlayerContribution{Position: position, Own: contribution(7, minute), Rival: contribution(8, minute)}
		c.Away[101] = PlayerContribution{Position: GOALKEEPER, Own: contribution(9, minute), Rival: contribution(10, minute)}
		if minute <= 35 {
			c.Away[102] = PlayerContribution{Position: DEFENSE, Own: contribution(11, minute), Rival: contribution(12, minute)}
		}
		if minute <= 70 {
			c.Away[103] = PlayerContribution{Position: STRIKER, Own: contribution(13, minute), Rival: 0}
		}
		if minute >= 85 {
			c.Away[113] = PlayerContribution{Position: STRIKER, Own: contribution(14, minute), Rival: contribution(15, minute)}
		}
		instants = append(instants, MatchInstant{Minute: minute, PlayerContributions: c})
	}
	events := []Event{
		goal(12, away, 103, 0, 1),
		{EventType: YELLOW_CARD, Minute: 20, TeamOwner: home, TeamOwnerPlayers: []int64{3}},
		{EventType: RED_CARD, Minute: 35, TeamOwner: away, TeamOwnerPlayers: []int64{102}},
		goal(50, home, 2, 1, 1),
		substitution(60, home, 2, 12),
		{EventType: INJURY, Minute: 70, TeamOwner: away, TeamOwnerPlayers: []int64{103}},
		{EventType: NO_GOAL, Minute: 75, TeamOwner: home, TeamOwnerPlayers: []int64{12}},
		goal(88, home, 12, 2, 1),
		substitution(84, away, 103, 113),
	}
	homeLineup := &Lineup{TeamId: home, Positions: []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 2, Position: STRIKER}, {PlayerId: 3, Position: MIDFIELD}}}
	awayLineup := &Lineup{TeamId: away, Positions: []LineupPosition{{PlayerId: 101, Position: GOALKEEPER}, {PlayerId: 102, Position: DEFENSE}, {PlayerId: 103, Position: STRIKER}}}
	minutes := []int{0, 5, 12, 45, 60, 85, 90}
	cases := []ratingsParityCase{
		{Name: "with contributions", HomeTeamId: home, AwayTeamId: away, HomeLineup: homeLineup, AwayLineup: awayLineup, Instants: instants, Events: events},
		{Name: "without contributions", HomeTeamId: home, AwayTeamId: away, HomeLineup: homeLineup, AwayLineup: awayLineup, Events: events},
	}
	for i := range cases {
		for _, minute := range minutes {
			cases[i].Expected = append(cases[i].Expected, struct {
				Minute  int            `json:"minute"`
				Ratings []PlayerRating `json:"ratings"`
			}{minute, ComputePlayerRatings(home, away, homeLineup, awayLineup, cases[i].Instants, events, minute)})
		}
	}
	return cases
}

func TestComputePlayerRatings_ParityFixture(t *testing.T) {
	if os.Getenv("STRIKER_WRITE_RATINGS_FIXTURE") == "1" {
		raw, err := json.Marshal(map[string]any{"cases": parityCases()})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ratingsParityFixturePath, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(ratingsParityFixturePath)
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}
	var fixture struct {
		Cases []ratingsParityCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parsing fixture: %s", err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}
	for _, c := range fixture.Cases {
		for _, expected := range c.Expected {
			got := ComputePlayerRatings(c.HomeTeamId, c.AwayTeamId, c.HomeLineup, c.AwayLineup, c.Instants, c.Events, expected.Minute)
			gotJson, _ := json.Marshal(got)
			wantJson, _ := json.Marshal(expected.Ratings)
			if string(gotJson) != string(wantJson) {
				t.Errorf("%s, minute %d:\n got  %s\n want %s", c.Name, expected.Minute, gotJson, wantJson)
			}
		}
	}
}

// withAddedTime appends first- and second-half added minutes (45+1..45+first after minute 45 and
// 90+1..90+second at the end) to instantsWith's 90 regulation instants. A player is on the pitch in
// an added minute when his [from, to] covers the clock minute (45 or 90) and addedFrom <= it, so a
// starter is listed throughout and a player with addedFrom 1 at 90 comes on at 90+1.
func withAddedTime(first int, second int, players ...onPitch) []MatchInstant {
	regulation := instantsWith(players...)
	added := func(clock int, count int) []MatchInstant {
		out := make([]MatchInstant, 0, count)
		for a := 1; a <= count; a++ {
			c := &InstantContributions{Home: map[int64]PlayerContribution{}, Away: map[int64]PlayerContribution{}}
			for _, p := range players {
				onAtClock := clock >= p.from && clock <= p.to
				cameOnInAddedTime := p.from > 90 && clock == 90 && a >= p.from-90
				if !onAtClock && !cameOnInAddedTime {
					continue
				}
				side := c.Away
				if p.home {
					side = c.Home
				}
				side[p.playerId] = PlayerContribution{Position: p.position, Own: p.own, Rival: p.rival}
			}
			out = append(out, MatchInstant{Minute: clock, AddedMinute: a, PlayerContributions: c})
		}
		return out
	}
	instants := append([]MatchInstant{}, regulation[:45]...)
	instants = append(instants, added(45, first)...)
	instants = append(instants, regulation[45:]...)
	return append(instants, added(90, second)...)
}

// "Full match with added time" (match-ratings "Added minutes count as minutes played"): a defender
// who plays the whole of a 2-1 win with 2 and 5 added minutes has 97 minutes and the full +0.3.
func TestComputePlayerRatings_FullMatchWithAddedTime(t *testing.T) {
	instants := withAddedTime(2, 5, onPitch{playerId: 1, home: true, position: DEFENSE, from: 1, to: 90, own: 50, rival: 50})
	events := []Event{goal(20, ratingsHome, 9, 1, 0), goal(50, ratingsAway, 30, 1, 1), {EventType: GOAL, Minute: 90, AddedMinute: 3,
		TeamOwner: ratingsHome, TeamOwnerPlayers: []int64{9}, HomeGoals: 2, AwayGoals: 1}}
	r := ratingOf(t, ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, MatchFinalMinute(instants, events)), 1)
	if r.Minutes != 97 || r.Result != 0.3 || !r.Rated {
		t.Fatalf("full match with added time: %+v, want 97 minutes and +0.3", r)
	}
}

// "Substitute in added time" (match-ratings): a striker who comes on through a substitution order
// at 90+1 of a match with 5 added minutes has 5 minutes and is unrated ("s.c.").
func TestComputePlayerRatings_SubstituteInAddedTimeIsUnrated(t *testing.T) {
	instants := withAddedTime(1, 5,
		onPitch{playerId: 1, home: true, position: STRIKER, from: 1, to: 90, own: 50, rival: 50},
		onPitch{playerId: 21, home: true, position: STRIKER, from: 91, to: 95, own: 50, rival: 50},
	)
	events := []Event{{EventType: SUBSTITUTION, Minute: 90, AddedMinute: 1, TeamOwner: ratingsHome, TeamOwnerPlayers: []int64{1, 21}}}
	ratings := ComputePlayerRatings(ratingsHome, ratingsAway, nil, nil, instants, events, MatchFinalMinute(instants, events))
	if r := ratingOf(t, ratings, 21); r.Minutes != 5 || r.Rated {
		t.Fatalf("substitute at 90+1: %+v, want 5 minutes and s.c.", r)
	}
	if final := MatchFinalMinute(instants, events); final != 90 {
		t.Fatalf("MatchFinalMinute = %d, want 90: the clock minute every second-half added minute hangs off", final)
	}
}
