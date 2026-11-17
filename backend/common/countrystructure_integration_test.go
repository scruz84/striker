package common

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestEffectiveStructure covers league-pyramid "Per-country structure": a country without its own
// structure follows the global one, one with its own uses it, and removing it puts the country
// back on the global one. Saving for an unknown country fails.
func TestEffectiveStructure(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	stamp := time.Now().UnixNano() % 1000
	own, plain := fmt.Sprintf("o%03d", stamp), fmt.Sprintf("p%03d", stamp)
	seedTransfersCountry(t, db, own)
	seedTransfersCountry(t, db, plain)
	global := GetWorldConfig(dbCtx)
	small := []TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10}, {Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 10}}

	if err := SetCountryStructure(dbCtx, own, small); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SetCountryStructure(dbCtx, own, nil) })
	if got := EffectiveStructure(dbCtx, own); len(got) != 2 || WorldConfigPlaces(got) != 30 {
		t.Errorf("own structure: got %+v, want 1/2 x 10", got)
	}
	if got := EffectiveStructure(dbCtx, plain); WorldConfigPlaces(got) != WorldConfigPlaces(global) {
		t.Errorf("a country without its own structure got %+v, want the global one", got)
	}
	all, err := GetCountryStructures(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if all[own] == nil || all[plain] != nil {
		t.Errorf("GetCountryStructures = %v, want %s only", all, own)
	}
	if err := SetCountryStructure(dbCtx, own, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := GetCountryStructure(dbCtx, own); got != nil {
		t.Errorf("after removing it, %s still has %+v", own, got)
	}
	if got := EffectiveStructure(dbCtx, own); WorldConfigPlaces(got) != WorldConfigPlaces(global) {
		t.Errorf("after removing it, %s follows %+v, want the global structure", own, got)
	}
	if err := SetCountryStructure(dbCtx, "zz9", small); !errors.Is(err, ErrUnknownCountry) {
		t.Errorf("saving for an unknown country: %v, want ErrUnknownCountry", err)
	}
}
