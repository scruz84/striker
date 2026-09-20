package engine

import "testing"

func TestInternationalCupName(t *testing.T) {
	cases := map[string]string{
		"europe":   "European Cup",
		"americas": "Copa America",
		"asia":     "Asian Cup",
		"africa":   "African Cup",
	}
	for region, want := range cases {
		if got := internationalCupName(region); got != want {
			t.Errorf("internationalCupName(%q) = %q, want %q", region, got, want)
		}
	}
}

func TestInternationalCupNameFallsBackForUnknownRegion(t *testing.T) {
	if got := internationalCupName("antarctica"); got != "International Cup" {
		t.Errorf("internationalCupName(unknown) = %q, want %q", got, "International Cup")
	}
}
