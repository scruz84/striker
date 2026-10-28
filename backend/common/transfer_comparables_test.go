package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func comparablesWithFees(fees ...int64) *TransferComparablesResult {
	r := &TransferComparablesResult{}
	for _, fee := range fees {
		r.Comparables = append(r.Comparables, TransferComparable{Fee: fee})
	}
	r.SummarizeComparables()
	return r
}

func TestSummarizeComparables(t *testing.T) {
	even := comparablesWithFees(400000, 100000, 150000, 120000)
	if even.Count != 4 || *even.MedianFee != 135000 || *even.MinFee != 100000 || *even.MaxFee != 400000 {
		t.Errorf("even count: got count %d median %d min %d max %d, want 4 / 135000 / 100000 / 400000", even.Count, *even.MedianFee, *even.MinFee, *even.MaxFee)
	}

	odd := comparablesWithFees(300, 100, 200)
	if *odd.MedianFee != 200 || *odd.MinFee != 100 || *odd.MaxFee != 300 {
		t.Errorf("odd count: got median %d min %d max %d, want 200 / 100 / 300", *odd.MedianFee, *odd.MinFee, *odd.MaxFee)
	}

	// An even count rounds the mean of the two middle fees down.
	if rounded := comparablesWithFees(101, 100); *rounded.MedianFee != 100 {
		t.Errorf("rounding: median = %d, want 100", *rounded.MedianFee)
	}

	empty := comparablesWithFees()
	if empty.Count != 0 || empty.MedianFee != nil || empty.MinFee != nil || empty.MaxFee != nil {
		t.Errorf("empty: got %+v, want zero count and nil fee figures", empty)
	}
}

// The statement spells the partial-index predicate as literals (Postgres can't match a partial index
// through bind parameters), so pin them to the constants they stand for.
func TestTransferComparablesStatementLiterals(t *testing.T) {
	kinds := fmt.Sprintf("kind IN (%d, %d, %d)", PLAYER_TRANSFER_LISTING, PLAYER_TRANSFER_OFFER, PLAYER_TRANSFER_BUY_OPTION)
	window := fmt.Sprintf("interval '%d days'", ComparablesWindowDays)
	for _, want := range []string{kinds, "fee > 0", window} {
		if !strings.Contains(getTransferComparablesStmtSql, want) {
			t.Errorf("getTransferComparablesStmtSql does not contain %q", want)
		}
	}
}

// A comparable may carry only public figures: no rating, trait, stamina, form, wage or contract.
func TestTransferComparablesResultJSONHasNoPrivateField(t *testing.T) {
	result := comparablesWithFees(100000)
	result.Comparables[0] = TransferComparable{PlayerName: "A B", CreatedAt: time.Now()}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	var collect func(v any)
	collect = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				keys[strings.ToLower(k)] = true
				collect(child)
			}
		case []any:
			for _, child := range x {
				collect(child)
			}
		}
	}
	collect(decoded)
	for _, private := range []string{"goalkeeper", "defense", "lateral", "pass", "midfield", "dribbling", "shot", "header", "quick",
		"creative", "technical", "stamina", "form", "wage", "contractyearsremaining", "contract"} {
		if keys[private] {
			t.Errorf("result JSON exposes private field %q", private)
		}
	}
}
