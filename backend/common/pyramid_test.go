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
