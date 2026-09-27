package engine

import (
	"math"
	"testing"
)

func TestBlendLine_EvenLineIsUnchanged(t *testing.T) {
	left, center, right := blendLine(50, 50, 50)
	for _, got := range []float32{left, center, right} {
		if math.Abs(float64(got-50)) > 0.001 {
			t.Fatalf("expected an already-even line to stay at 50, got %v", got)
		}
	}
}

func TestBlendLine_FlanksPickUpAShareOfTheCenter(t *testing.T) {
	left, center, right := blendLine(0, 100, 0)
	if left != right {
		t.Fatalf("expected a symmetric line to blend both flanks equally, got left=%v right=%v", left, right)
	}
	if left <= 0 {
		t.Fatalf("expected a strong center to lend some strength to an empty flank, got %v", left)
	}
	if center >= 100 {
		t.Fatalf("expected the center to give up some of its own strength to the flanks, got %v", center)
	}
}

func TestBlendLine_CenterPicksUpAShareOfBothFlanks(t *testing.T) {
	_, center, _ := blendLine(100, 0, 100)
	if center <= 0 {
		t.Fatalf("expected strong flanks to lend some strength to an empty center, got %v", center)
	}
}

func TestBlendLine_TotalStrengthIsConserved(t *testing.T) {
	left, center, right := blendLine(30, 60, 90)
	total := left + center + right
	if math.Abs(float64(total-180)) > 0.01 {
		t.Fatalf("expected blending to conserve the line's total strength (180), got %v", total)
	}
}
