package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

func TestAsTeamIdPtr(t *testing.T) {
	if got := asTeamIdPtr(ByeTeamId); got != nil {
		t.Fatalf("expected nil for a bye slot, got %v", *got)
	}
	if got := asTeamIdPtr(42); got == nil || *got != 42 {
		t.Fatalf("expected a pointer to 42, got %v", got)
	}
}

// state builds one country's standing: entered is the last season it sent to the cup (0 = never),
// champion its latest champion season (0 = none yet), unused how many champions it has since.
func state(country string, entered, champion, unused int) common.RegionalCupCountryState {
	s := common.RegionalCupCountryState{Country: country, LastEnteredSeason: entered, ChampionSeason: champion, UnusedChampions: unused}
	if champion > 0 {
		s.ChampionTeamId = int64(len(country)*1000 + champion)
	}
	return s
}

func entrantCountries(entrants []common.RegionalCupCountryState) []string {
	countries := make([]string, len(entrants))
	for i, e := range entrants {
		countries[i] = e.Country
	}
	return countries
}

func TestRegionalCupEntrants(t *testing.T) {
	cases := []struct {
		name   string
		states []common.RegionalCupCountryState
		want   []string // nil: no edition due
	}{
		{
			name:   "every country has a new champion",
			states: []common.RegionalCupCountryState{state("es", 3, 4, 1), state("fr", 3, 4, 1), state("it", 2, 3, 1)},
			want:   []string{"es", "fr", "it"},
		},
		{
			name:   "waiting for the last country",
			states: []common.RegionalCupCountryState{state("es", 3, 4, 1), state("fr", 3, 4, 1), state("it", 3, 3, 0)},
		},
		{
			name:   "a slow country does not hold the region back",
			states: []common.RegionalCupCountryState{state("es", 3, 5, 2), state("fr", 3, 4, 1), state("it", 3, 3, 0)},
			want:   []string{"es", "fr"},
		},
		{
			name:   "a country that never finished a season does not hold the region back for ever",
			states: []common.RegionalCupCountryState{state("es", 0, 2, 2), state("fr", 0, 1, 1), state("pt", 0, 0, 0)},
			want:   []string{"es", "fr"},
		},
		{
			name:   "first edition waits for a brand-new country while nobody is overdue",
			states: []common.RegionalCupCountryState{state("es", 0, 1, 1), state("fr", 0, 1, 1), state("pt", 0, 0, 0)},
		},
		{
			name:   "one country only, however many seasons",
			states: []common.RegionalCupCountryState{state("es", 3, 7, 4), state("fr", 4, 4, 0)},
		},
		{
			name:   "a region of a single country",
			states: []common.RegionalCupCountryState{state("nz", 0, 3, 3)},
		},
		{
			name:   "champion already played",
			states: []common.RegionalCupCountryState{state("es", 4, 5, 1), state("fr", 4, 4, 0), state("it", 4, 5, 1)},
		},
		{
			name:   "two countries, both fresh",
			states: []common.RegionalCupCountryState{state("es", 1, 2, 1), state("fr", 1, 2, 1)},
			want:   []string{"es", "fr"},
		},
		{
			name: "no states",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := regionalCupEntrants(c.states)
			if c.want == nil {
				if got != nil {
					t.Fatalf("expected no edition, got entrants %v", entrantCountries(got))
				}
				return
			}
			countries := entrantCountries(got)
			if len(countries) != len(c.want) {
				t.Fatalf("entrants %v, want %v", countries, c.want)
			}
			for i := range c.want {
				if countries[i] != c.want[i] {
					t.Fatalf("entrants %v, want %v", countries, c.want)
				}
			}
		})
	}
}
