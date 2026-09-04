package engine

import "testing"

func TestSponsorshipForTier_HigherTiersPayMore(t *testing.T) {
	if sponsorshipForTier(1) <= sponsorshipForTier(2) {
		t.Fatal("expected tier 1 sponsorship to exceed tier 2")
	}
	if sponsorshipForTier(2) <= sponsorshipForTier(3) {
		t.Fatal("expected tier 2 sponsorship to exceed tier 3")
	}
	// An unrecognized tier falls back to the smallest payment rather than paying nothing.
	if sponsorshipForTier(99) != SponsorshipPerMatchTier3 {
		t.Fatal("expected an unknown tier to fall back to the tier-3 payment")
	}
}
