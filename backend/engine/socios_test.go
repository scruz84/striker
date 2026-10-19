package engine

import "testing"

func TestSociosTarget_BigClubOutgrowsDefaultGroundSmallDoesNot(t *testing.T) {
	big := sociosTarget(80, 5, 10)
	small := sociosTarget(30, 5, 10)
	if big < 1200 || big > 1800 {
		t.Errorf("a fanbase-80 mid-table club should aim for ~1,400-1,700 socios, got %.0f", big)
	}
	if small < 300 || small > 600 {
		t.Errorf("a fanbase-30 mid-table club should aim for ~360-500 socios, got %.0f", small)
	}
}

func TestSociosTarget_PositionSwingsTenPercent(t *testing.T) {
	champion := sociosTarget(60, 1, 10)
	last := sociosTarget(60, 10, 10)
	if champion <= last {
		t.Fatalf("the champion (%.0f) should aim higher than the last team (%.0f)", champion, last)
	}
	mid := (champion + last) / 2
	if champion/mid < 1.09 || champion/mid > 1.11 {
		t.Errorf("champion should sit ~10%% above the midpoint, got %.3f", champion/mid)
	}
}

func TestDriftSocios_MovesThirtyPercentOfTheWay(t *testing.T) {
	target := sociosTarget(60, 5, 10)
	got := driftSocios(500, 60, 5, 10)
	want := 500 + 0.3*(target-500)
	if diff := float64(got) - want; diff < -1 || diff > 1 {
		t.Fatalf("driftSocios = %d, want ~%.0f", got, want)
	}
}

func TestDriftSocios_ConvergesAndNeverDropsBelowOne(t *testing.T) {
	current := 500
	target := sociosTarget(40, 5, 10)
	for i := 0; i < 40; i++ {
		current = driftSocios(current, 40, 5, 10)
	}
	if diff := float64(current) - target; diff < -2 || diff > 2 {
		t.Errorf("socios should converge to the target %.0f, got %d", target, current)
	}
	if got := driftSocios(1, 0, 10, 10); got < 1 {
		t.Errorf("socios must stay at least 1, got %d", got)
	}
}
