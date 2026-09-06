package common

import "testing"

func TestValidateWorldConfig_AcceptsDefault(t *testing.T) {
	if err := ValidateWorldConfig(DefaultWorldTiers); err != nil {
		t.Fatalf("expected the default tier structure to be valid, got %v", err)
	}
}

func TestValidateWorldConfig_RejectsTierCountOutOfRange(t *testing.T) {
	if err := ValidateWorldConfig(nil); err == nil {
		t.Fatal("expected an error for zero tiers")
	}
	tooMany := make([]TierConfig, MaxWorldTiers+1)
	for i := range tooMany {
		tooMany[i] = TierConfig{Tier: i + 1, Name: "X", Groups: 1, TeamsPerLeague: 10}
	}
	if err := ValidateWorldConfig(tooMany); err == nil {
		t.Fatal("expected an error for more than MaxWorldTiers tiers")
	}
}

func TestValidateWorldConfig_RejectsDuplicateOrNonPositiveTierNumbers(t *testing.T) {
	cases := [][]TierConfig{
		{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10}, {Tier: 1, Name: "II", Groups: 1, TeamsPerLeague: 10}},
		{{Tier: 0, Name: "I", Groups: 1, TeamsPerLeague: 10}},
		{{Tier: -1, Name: "I", Groups: 1, TeamsPerLeague: 10}},
	}
	for _, tiers := range cases {
		if err := ValidateWorldConfig(tiers); err == nil {
			t.Fatalf("expected an error for %+v", tiers)
		}
	}
}

func TestValidateWorldConfig_RejectsMissingName(t *testing.T) {
	tiers := []TierConfig{{Tier: 1, Name: "", Groups: 1, TeamsPerLeague: 10}}
	if err := ValidateWorldConfig(tiers); err == nil {
		t.Fatal("expected an error for a blank tier name")
	}
}

func TestValidateWorldConfig_RejectsGroupsOutOfRange(t *testing.T) {
	cases := []TierConfig{
		{Tier: 1, Name: "I", Groups: 0, TeamsPerLeague: 10},
		{Tier: 1, Name: "I", Groups: MaxGroupsPerTier + 1, TeamsPerLeague: 10},
	}
	for _, tc := range cases {
		if err := ValidateWorldConfig([]TierConfig{tc}); err == nil {
			t.Fatalf("expected an error for %+v", tc)
		}
	}
}

func TestValidateWorldConfig_RejectsBadTeamsPerLeague(t *testing.T) {
	cases := []TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: MinTeamsPerLeague - 1},
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: MaxTeamsPerLeague + 1},
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 11}, // odd - round-robin needs an even count
	}
	for _, tc := range cases {
		if err := ValidateWorldConfig([]TierConfig{tc}); err == nil {
			t.Fatalf("expected an error for %+v", tc)
		}
	}
}

func TestValidateWorldConfig_RejectsExcessiveTotalTeamCount(t *testing.T) {
	tiers := []TierConfig{
		{Tier: 1, Name: "I", Groups: MaxGroupsPerTier, TeamsPerLeague: MaxTeamsPerLeague},
		{Tier: 2, Name: "II", Groups: MaxGroupsPerTier, TeamsPerLeague: MaxTeamsPerLeague},
	}
	if err := ValidateWorldConfig(tiers); err == nil {
		t.Fatal("expected an error when the total team count exceeds MaxTeamsPerCountryTotal")
	}
}
