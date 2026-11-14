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
	groups := 1
	for i := range tooMany {
		tooMany[i] = TierConfig{Tier: i + 1, Name: "X", Groups: groups, TeamsPerLeague: 8}
		groups *= 2
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
	// 1+2+4+8+16+32 = 63 groups of 24 = 1512 teams.
	tiers := make([]TierConfig, 6)
	groups := 1
	for i := range tiers {
		tiers[i] = TierConfig{Tier: i + 1, Name: "X", Groups: groups, TeamsPerLeague: MaxTeamsPerLeague}
		groups *= 2
	}
	if err := ValidateWorldConfig(tiers); err == nil {
		t.Fatal("expected an error when the total team count exceeds MaxTeamsPerCountryTotal")
	}
}

func TestValidateWorldConfig_DefaultIsTheFiveTierPyramid(t *testing.T) {
	if len(DefaultWorldTiers) != 5 || WorldConfigPlaces(DefaultWorldTiers) != 310 {
		t.Fatalf("default should be 5 tiers / 310 teams, got %d tiers / %d", len(DefaultWorldTiers), WorldConfigPlaces(DefaultWorldTiers))
	}
}

func pyramid(teams int, groups ...int) []TierConfig {
	tiers := make([]TierConfig, len(groups))
	for i, g := range groups {
		tiers[i] = TierConfig{Tier: i + 1, Name: "T", Groups: g, TeamsPerLeague: teams}
	}
	return tiers
}

func TestValidateWorldConfig_PyramidRule(t *testing.T) {
	cases := []struct {
		name  string
		tiers []TierConfig
		want  string
	}{
		{"doubling", pyramid(10, 1, 4, 16), "tier 2 must have exactly 2 groups (twice tier 1)"},
		{"top tier", pyramid(10, 2, 4), "tier 1 must have exactly 1 group"},
		{"gap", []TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10}, {Tier: 3, Name: "III", Groups: 2, TeamsPerLeague: 10}}, "tiers must be numbered 1 to 2 with no gaps"},
		{"uneven teams", []TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10}, {Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 12}}, "every tier must have the same number of teams per group"},
	}
	for _, c := range cases {
		err := ValidateWorldConfig(c.tiers)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
	if err := ValidateWorldConfig(pyramid(6, 1, 2)); err == nil {
		t.Error("6 teams per group should be rejected: no room for every zone")
	}
	if err := ValidateWorldConfig(pyramid(10, 1, 2, 4, 8, 16)); err != nil {
		t.Errorf("1/2/4/8/16 x 10 should be valid, got %v", err)
	}
}

func TestValidateWorldConfigFitsTeams(t *testing.T) {
	err := ValidateWorldConfigFitsTeams(pyramid(10, 1, 2, 4, 8), map[string]int{"es": 310, "it": 100})
	if err == nil || err.Error() != "country es has 310 teams but this structure only has 150 places" {
		t.Fatalf("got %v", err)
	}
	if err := ValidateWorldConfigFitsTeams(pyramid(10, 1, 2, 4, 8, 16), map[string]int{"es": 210}); err != nil {
		t.Fatalf("210 teams fit in 310 places, got %v", err)
	}
}
