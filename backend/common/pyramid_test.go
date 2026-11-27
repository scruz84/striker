package common

import "testing"

func zonesOfGroup(z Zones, n int) []string {
	out := make([]string, n)
	for i := 1; i <= n; i++ {
		out[i-1] = ZoneOf(i, z, n)
	}
	return out
}

func assertZones(t *testing.T, got []string, want map[int]string) {
	t.Helper()
	for i, zone := range got {
		if zone != want[i+1] {
			t.Errorf("position %d: zone %q, want %q (all: %v)", i+1, zone, want[i+1], got)
		}
	}
}

func TestPyramidZones_TopDivision(t *testing.T) {
	z := PyramidZones(1, 5, 10, false)
	assertZones(t, zonesOfGroup(z, 10), map[int]string{7: ZonePlayoffDown, 8: ZonePlayoffDown, 9: ZoneDirectDown, 10: ZoneDirectDown})
}

func TestPyramidZones_MiddleTier(t *testing.T) {
	z := PyramidZones(2, 5, 10, false)
	assertZones(t, zonesOfGroup(z, 10), map[int]string{
		1: ZoneDirectUp, 2: ZonePlayoffUp, 7: ZonePlayoffDown, 8: ZonePlayoffDown, 9: ZoneDirectDown, 10: ZoneDirectDown,
	})
}

func TestPyramidZones_BottomTier(t *testing.T) {
	z := PyramidZones(5, 5, 10, false)
	assertZones(t, zonesOfGroup(z, 10), map[int]string{1: ZoneDirectUp, 2: ZonePlayoffUp})
}

func TestPyramidZones_SingleTierHasNoZones(t *testing.T) {
	z := PyramidZones(1, 1, 10, false)
	assertZones(t, zonesOfGroup(z, 10), map[int]string{})
}

func TestPyramidZones_Legacy(t *testing.T) {
	top := PyramidZones(1, 3, 10, true)
	assertZones(t, zonesOfGroup(top, 10), map[int]string{9: ZoneDirectDown, 10: ZoneDirectDown})
	middle := PyramidZones(2, 3, 10, true)
	assertZones(t, zonesOfGroup(middle, 10), map[int]string{1: ZoneCandidate, 9: ZoneDirectDown, 10: ZoneDirectDown})
}

func TestPyramidZones_TinyGroupNeverOverlaps(t *testing.T) {
	z := PyramidZones(2, 3, 4, false)
	if z.PromotionPlaces()+z.RelegationPlaces() > 4 {
		t.Fatalf("zones cover more than the group: %+v", z)
	}
}

func shapeOf(groupsPerTier ...int) PyramidShape {
	s := PyramidShape{}
	id := int64(1)
	for i, groups := range groupsPerTier {
		for g := 1; g <= groups; g++ {
			s.Leagues = append(s.Leagues, PyramidLeague{LeagueId: id, Tier: i + 1, GroupNumber: g, Teams: 10})
			id++
		}
	}
	return s
}

func TestPyramidShape_FollowsRatioRule(t *testing.T) {
	if !shapeOf(1, 2, 4, 8, 16).FollowsRatioRule() {
		t.Error("1/2/4/8/16 should follow the rule")
	}
	if shapeOf(1, 4, 16).FollowsRatioRule() {
		t.Error("1/4/16 should not follow the rule")
	}
	if !shapeOf(1).FollowsRatioRule() {
		t.Error("a single tier of 1 group should follow the rule")
	}
	if (PyramidShape{}).FollowsRatioRule() {
		t.Error("an empty shape should not follow the rule")
	}
}

func TestPyramidShape_MatchesConfig(t *testing.T) {
	if !shapeOf(1, 2, 4, 8, 16).MatchesConfig(DefaultWorldTiers) {
		t.Error("the default shape should match the default config")
	}
	if shapeOf(1, 4, 16).MatchesConfig(DefaultWorldTiers) {
		t.Error("1/4/16 should not match the default config")
	}
	s := shapeOf(1, 2, 4, 8, 16)
	s.Leagues[3].Teams = 9
	if s.MatchesConfig(DefaultWorldTiers) {
		t.Error("a group with the wrong team count should not match")
	}
}

func diffKinds(diffs []ShapeDifference) []string {
	out := make([]string, len(diffs))
	for i, d := range diffs {
		out[i] = d.Kind
		if d.League != "" {
			out[i] += ":" + d.League
		}
	}
	return out
}

func assertDiffs(t *testing.T, got []ShapeDifference, want ...string) {
	t.Helper()
	kinds := diffKinds(got)
	if len(kinds) != len(want) {
		t.Fatalf("differences %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("differences %v, want %v", kinds, want)
		}
	}
}

func namedShapeOf(groupsPerTier ...int) PyramidShape {
	s := shapeOf(groupsPerTier...)
	names := []string{"I", "II", "III", "IV", "V", "VI"}
	for i := range s.Leagues {
		s.Leagues[i].Name = LeagueName(names[s.Leagues[i].Tier-1], s.Leagues[i].GroupNumber)
	}
	return s
}

var tiers124 = []TierConfig{
	{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10},
	{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 10},
	{Tier: 3, Name: "III", Groups: 4, TeamsPerLeague: 10},
}

func TestPyramidShape_Differences(t *testing.T) {
	t.Run("in shape", func(t *testing.T) {
		assertDiffs(t, namedShapeOf(1, 2, 4).Differences(tiers124))
	})
	t.Run("1/2/3 against 1/2/4", func(t *testing.T) {
		diffs := namedShapeOf(1, 2, 3).Differences(tiers124)
		assertDiffs(t, diffs, "missing:III.4", "ratio")
		if diffs[0].Tier != 3 || diffs[0].Group != 4 || diffs[0].Expected != 10 {
			t.Errorf("missing group %+v", diffs[0])
		}
	})
	t.Run("leftover lower tier", func(t *testing.T) {
		s := namedShapeOf(1, 2, 4)
		s.Leagues = append(s.Leagues, PyramidLeague{LeagueId: 99, Tier: 4, GroupNumber: 2, Name: "IV.2", Teams: 3})
		diffs := s.Differences(tiers124)
		assertDiffs(t, diffs, "extra:IV.2", "ratio")
		if diffs[0].Teams != 3 {
			t.Errorf("extra group %+v", diffs[0])
		}
	})
	t.Run("leftover group in a tier", func(t *testing.T) {
		s := namedShapeOf(1, 2, 4)
		s.Leagues = append(s.Leagues, PyramidLeague{LeagueId: 99, Tier: 3, GroupNumber: 7, Name: "III.7", Teams: 4})
		assertDiffs(t, s.Differences(tiers124), "extra:III.7", "ratio")
	})
	t.Run("two leagues in one group", func(t *testing.T) {
		s := namedShapeOf(1, 2, 4)
		s.Leagues = append(s.Leagues, PyramidLeague{LeagueId: 99, Tier: 2, GroupNumber: 1, Name: "II.1", Teams: 2})
		assertDiffs(t, s.Differences(tiers124), "extra:II.1", "ratio")
	})
	t.Run("wrong size", func(t *testing.T) {
		s := namedShapeOf(1, 2, 4)
		s.Leagues[4].Teams = 11
		diffs := s.Differences(tiers124)
		assertDiffs(t, diffs, "size:III.2")
		if diffs[0].Teams != 11 || diffs[0].Expected != 10 {
			t.Errorf("size difference %+v", diffs[0])
		}
	})
	t.Run("invalid structure is reported alone", func(t *testing.T) {
		invalid := []TierConfig{
			{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10},
			{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 10},
			{Tier: 3, Name: "III", Groups: 3, TeamsPerLeague: 10},
		}
		diffs := namedShapeOf(1, 2, 3).Differences(invalid)
		assertDiffs(t, diffs, "invalid_structure")
		if diffs[0].Message == "" {
			t.Error("an invalid structure should say why")
		}
		if namedShapeOf(1, 2, 3).MatchesConfig(invalid) {
			t.Error("no shape matches an invalid structure")
		}
	})
}
