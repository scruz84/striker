package handlers

import "testing"

func TestParseLeagueWeek(t *testing.T) {
	cases := []struct {
		name       string
		league     string
		week       string
		wantLeague int64
		wantWeek   int
		wantErr    bool
	}{
		{"week zero", "7", "0", 0, 0, true},
		{"negative week", "7", "-1", 0, 0, true},
		{"non-numeric week", "7", "abc", 0, 0, true},
		{"non-numeric league", "abc", "3", 0, 0, true},
		{"first week", "7", "1", 7, 1, false},
		{"past the season", "7", "30", 7, 30, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			league, week, err := parseLeagueWeek(c.league, c.week)
			if (err != nil) != c.wantErr {
				t.Fatalf("parseLeagueWeek(%q, %q) error = %v, wantErr %v", c.league, c.week, err, c.wantErr)
			}
			if !c.wantErr && (league != c.wantLeague || week != c.wantWeek) {
				t.Errorf("parseLeagueWeek(%q, %q) = %d, %d; want %d, %d", c.league, c.week, league, week, c.wantLeague, c.wantWeek)
			}
		})
	}
}
