package common

import "testing"

func TestGetStrikerValue_WeighsShotTwiceAsHeavilyAsPassOrDribbling(t *testing.T) {
	// A pure finisher: strong Shot/Dribbling but weak Pass - the profile that used to get
	// mislabeled as a non-striker under a flat three-way average (see BestCorePosition).
	finisher := Player{Shot: 64, Dribbling: 60, Pass: 34}

	got := GetStrikerValue(finisher)
	want := 56 // (64*2 + 60 + 34 + 2) / 4, rounded to nearest rather than truncated

	if got != want {
		t.Fatalf("GetStrikerValue(%+v) = %d, want %d", finisher, got, want)
	}
}

func TestGetStrikerValue_MatchesShotAloneWhenPassAndDribblingAreEqual(t *testing.T) {
	// With Pass and Dribbling pinned equal to each other, the weighted average is an exact
	// midpoint between Shot and that shared value - a direct check that Shot's weight is now
	// double theirs, not just "higher" by some unverified amount.
	p := Player{Shot: 80, Dribbling: 50, Pass: 50}

	got := GetStrikerValue(p)
	want := 65 // (80 + 50) / 2

	if got != want {
		t.Fatalf("GetStrikerValue(%+v) = %d, want %d", p, got, want)
	}
}
