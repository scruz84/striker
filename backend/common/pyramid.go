package common

import (
	"fmt"
	"sort"
	"strings"
)

// The league pyramid (see openspec/specs/league-pyramid): every tier has PyramidGroupRatio times the
// groups of the tier above it, so each pair of adjacent tiers balances exactly. An upper tier with
// G groups sends 2G teams down directly and 2G into the relegation playoff; the tier below, with
// 2G groups, sends 2G group winners up directly and 2G runners-up into the promotion playoff.
const (
	PyramidGroupRatio        = 2
	MinPyramidTeamsPerLeague = 8
	DirectUpPerGroup         = 1
	PlayoffUpPerGroup        = 1
	PlayoffDownPerGroup      = 2
	DirectDownPerGroup       = 2
)

// Legacy pyramids (a country whose active leagues break the ratio rule - in practice the season
// in progress when this rule shipped) keep the previous one-for-one swap: the bottom
// LegacyRelegationPerGroup of each upper group against the best LegacyCandidatesPerGroup winners
// of every group below - see engine.computeLegacyMoves.
const (
	LegacyCandidatesPerGroup = 1
	LegacyRelegationPerGroup = 2
)

// Zone names a standings position can fall in - see ZoneOf.
const (
	ZoneNone        = ""
	ZoneDirectUp    = "direct_up"
	ZonePlayoffUp   = "playoff_up"
	ZonePlayoffDown = "playoff_down"
	ZoneDirectDown  = "direct_down"
	// ZoneCandidate is a legacy pyramid's promotion candidate: a group winner that goes up only if
	// it is among the best winners of its tier.
	ZoneCandidate = "candidate"
)

// Zones is how many places of one group sit in each zone, counted from the top (DirectUp,
// PlayoffUp, Candidate) or the bottom (DirectDown, PlayoffDown) of its table.
type Zones struct {
	DirectUp    int  `json:"directUp"`
	PlayoffUp   int  `json:"playoffUp"`
	PlayoffDown int  `json:"playoffDown"`
	DirectDown  int  `json:"directDown"`
	Candidate   int  `json:"candidate"`
	Legacy      bool `json:"legacy"`
}

// PyramidZones returns the zones of a group of groupSize teams in tier (1 = top) of a country
// with tierCount active tiers. legacy is true when the country's active leagues break the ratio
// rule (see PyramidShape.FollowsRatioRule): then a group has no playoff places, only the old
// candidate and direct relegation places. Zones never cover more places than the group has.
// Pure.
func PyramidZones(tier, tierCount, groupSize int, legacy bool) Zones {
	z := Zones{Legacy: legacy}
	hasUpper := tier > 1
	hasLower := tier < tierCount
	if legacy {
		if hasUpper {
			z.Candidate = LegacyCandidatesPerGroup
		}
		if hasLower {
			z.DirectDown = LegacyRelegationPerGroup
		}
	} else {
		if hasUpper {
			z.DirectUp = DirectUpPerGroup
			z.PlayoffUp = PlayoffUpPerGroup
		}
		if hasLower {
			z.PlayoffDown = PlayoffDownPerGroup
			z.DirectDown = DirectDownPerGroup
		}
	}
	// A group too small for every zone keeps its promotion places first.
	top := z.DirectUp + z.PlayoffUp + z.Candidate
	if top > groupSize {
		top = groupSize
	}
	room := groupSize - top
	if z.DirectDown > room {
		z.DirectDown = room
	}
	room -= z.DirectDown
	if z.PlayoffDown > room {
		z.PlayoffDown = room
	}
	return z
}

// PromotionPlaces is how many places at the top of a group are in a promotion zone (direct,
// playoff or candidate).
func (z Zones) PromotionPlaces() int { return z.DirectUp + z.PlayoffUp + z.Candidate }

// RelegationPlaces is how many places at the bottom of a group are in a relegation zone (direct or
// playoff).
func (z Zones) RelegationPlaces() int { return z.DirectDown + z.PlayoffDown }

// ZoneOf returns the zone of 1-based position in a group of groupSize teams. Pure.
func ZoneOf(position int, z Zones, groupSize int) string {
	switch {
	case position < 1 || position > groupSize:
		return ZoneNone
	case position <= z.DirectUp:
		return ZoneDirectUp
	case position <= z.DirectUp+z.PlayoffUp:
		return ZonePlayoffUp
	case position <= z.DirectUp+z.PlayoffUp+z.Candidate:
		return ZoneCandidate
	case position > groupSize-z.DirectDown:
		return ZoneDirectDown
	case position > groupSize-z.DirectDown-z.PlayoffDown:
		return ZonePlayoffDown
	}
	return ZoneNone
}

// PyramidLeague is one active league of a country's pyramid.
type PyramidLeague struct {
	LeagueId    int64
	Tier        int
	GroupNumber int
	Name        string
	Teams       int
}

// PyramidShape is a country's active leagues, ordered by tier then group number.
type PyramidShape struct {
	Leagues []PyramidLeague
}

// TierCount is the deepest active tier (0 with no leagues).
func (s PyramidShape) TierCount() int {
	max := 0
	for _, l := range s.Leagues {
		if l.Tier > max {
			max = l.Tier
		}
	}
	return max
}

// GroupsByTier counts the active leagues of each tier.
func (s PyramidShape) GroupsByTier() map[int]int {
	groups := make(map[int]int)
	for _, l := range s.Leagues {
		groups[l.Tier]++
	}
	return groups
}

// LeaguesOfTier returns tier's active leagues in group order.
func (s PyramidShape) LeaguesOfTier(tier int) []PyramidLeague {
	out := make([]PyramidLeague, 0)
	for _, l := range s.Leagues {
		if l.Tier == tier {
			out = append(out, l)
		}
	}
	return out
}

// FollowsRatioRule reports whether the tiers are numbered 1..T with no gaps, tier 1 has one group
// and every other tier has exactly PyramidGroupRatio times the groups of the tier above it - the
// shape the playoff pairings need (see engine.ComputePlayoffPairings). A country with no leagues
// does not follow it.
func (s PyramidShape) FollowsRatioRule() bool {
	groups := s.GroupsByTier()
	count := s.TierCount()
	if count == 0 || len(groups) != count {
		return false
	}
	expected := 1
	for tier := 1; tier <= count; tier++ {
		if groups[tier] != expected {
			return false
		}
		expected *= PyramidGroupRatio
	}
	return true
}

// Zones returns the zones of league's group within this shape.
func (s PyramidShape) Zones(l PyramidLeague) Zones {
	return PyramidZones(l.Tier, s.TierCount(), l.Teams, !s.FollowsRatioRule())
}

// Kinds of ShapeDifference.
const (
	ShapeDiffExtra            = "extra"             // an active league whose tier and group the structure lacks
	ShapeDiffMissing          = "missing"           // a group of the structure with no active league
	ShapeDiffSize             = "size"              // a group whose team count is not the structure's
	ShapeDiffRatio            = "ratio"             // the active leagues break the doubling rule
	ShapeDiffInvalidStructure = "invalid_structure" // the structure itself breaks ValidateWorldConfig
	// ShapeDiffUnreadable is a restructure that could not read the country back to check it (see
	// engine.restructureIfNeeded); Differences never returns it.
	ShapeDiffUnreadable = "unreadable"
)

// ShapeDifference is one way a country's active leagues differ from its structure (see
// openspec/specs/league-pyramid "Admin sees why a country is pending").
type ShapeDifference struct {
	Kind     string `json:"kind"`
	Tier     int    `json:"tier,omitempty"`
	Group    int    `json:"group,omitempty"`
	League   string `json:"league,omitempty"`
	Teams    int    `json:"teams,omitempty"`
	Expected int    `json:"expected,omitempty"`
	// Message is the validation error of an invalid structure.
	Message string `json:"message,omitempty"`
}

// Differences lists every way the shape differs from target, in tier and group order: extra
// groups, missing groups, groups of the wrong size, then a broken doubling rule. A target that
// breaks ValidateWorldConfig is reported alone, as invalid_structure, because no restructure can
// reach it (see engine.restructureIfNeeded). Pure.
func (s PyramidShape) Differences(target []TierConfig) []ShapeDifference {
	if err := ValidateWorldConfig(target); err != nil {
		return []ShapeDifference{{Kind: ShapeDiffInvalidStructure, Message: err.Error()}}
	}
	byTier := make(map[int]TierConfig, len(target))
	for _, tc := range target {
		byTier[tc.Tier] = tc
	}
	type slot struct{ tier, group int }
	present := make(map[slot]bool, len(s.Leagues))
	out := make([]ShapeDifference, 0)
	for _, l := range s.Leagues {
		key := slot{l.Tier, l.GroupNumber}
		duplicate := present[key]
		present[key] = true
		tc, ok := byTier[l.Tier]
		if !ok || l.GroupNumber < 1 || l.GroupNumber > tc.Groups || duplicate {
			out = append(out, ShapeDifference{Kind: ShapeDiffExtra, Tier: l.Tier, Group: l.GroupNumber, League: l.Name, Teams: l.Teams})
			continue
		}
		if l.Teams != tc.TeamsPerLeague {
			out = append(out, ShapeDifference{Kind: ShapeDiffSize, Tier: l.Tier, Group: l.GroupNumber, League: l.Name, Teams: l.Teams, Expected: tc.TeamsPerLeague})
		}
	}
	tiers := append([]TierConfig(nil), target...)
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Tier < tiers[j].Tier })
	for _, tc := range tiers {
		for g := 1; g <= tc.Groups; g++ {
			if !present[slot{tc.Tier, g}] {
				out = append(out, ShapeDifference{Kind: ShapeDiffMissing, Tier: tc.Tier, Group: g, League: LeagueName(tc.Name, g), Expected: tc.TeamsPerLeague})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Group < out[j].Group
	})
	if len(s.Leagues) > 0 && !s.FollowsRatioRule() {
		out = append(out, ShapeDifference{Kind: ShapeDiffRatio})
	}
	return out
}

// DescribeShapeDifferences renders diffs on one line for the logs, e.g.
// "missing III.4 (expected 10 teams); extra III.7 (4 teams); doubling rule broken".
func DescribeShapeDifferences(diffs []ShapeDifference) string {
	parts := make([]string, 0, len(diffs))
	for _, d := range diffs {
		switch d.Kind {
		case ShapeDiffExtra:
			parts = append(parts, fmt.Sprintf("extra %s (tier %d group %d, %d teams)", d.League, d.Tier, d.Group, d.Teams))
		case ShapeDiffMissing:
			parts = append(parts, fmt.Sprintf("missing %s (expected %d teams)", d.League, d.Expected))
		case ShapeDiffSize:
			parts = append(parts, fmt.Sprintf("%s has %d teams, expected %d", d.League, d.Teams, d.Expected))
		case ShapeDiffRatio:
			parts = append(parts, "doubling rule broken")
		default:
			parts = append(parts, d.Kind+": "+d.Message)
		}
	}
	return strings.Join(parts, "; ")
}

// MatchesConfig reports whether the shape is exactly tiers: same tier count, groups per tier and
// teams in every group, with tiers a valid structure. A country that matches needs no restructure
// (see engine.restructureIfNeeded).
func (s PyramidShape) MatchesConfig(tiers []TierConfig) bool {
	return len(tiers) > 0 && len(s.Differences(tiers)) == 0
}

// GetCountryPyramidShapeStmtSql lists a country's active leagues with their team counts.
const GetCountryPyramidShapeStmtSql = "SELECT l.league_id, l.tier, l.group_number, l.name, count(t.team_id) " +
	"FROM leagues l LEFT JOIN teams t ON t.league_id = l.league_id " +
	"WHERE l.country = $1 AND l.archived_season IS NULL " +
	"GROUP BY l.league_id, l.tier, l.group_number, l.name ORDER BY l.tier, l.group_number, l.league_id"

// GetCountryPyramidShape reads country's active league pyramid.
func GetCountryPyramidShape(dbCtx *DatabaseContext, country string) (PyramidShape, error) {
	rows, err := dbCtx.Db.Query(GetCountryPyramidShapeStmtSql, country)
	if err != nil {
		return PyramidShape{}, err
	}
	defer rows.Close()
	shape := PyramidShape{Leagues: make([]PyramidLeague, 0)}
	for rows.Next() {
		var l PyramidLeague
		if err := rows.Scan(&l.LeagueId, &l.Tier, &l.GroupNumber, &l.Name, &l.Teams); err != nil {
			return PyramidShape{}, err
		}
		shape.Leagues = append(shape.Leagues, l)
	}
	sort.SliceStable(shape.Leagues, func(i, j int) bool {
		a, b := shape.Leagues[i], shape.Leagues[j]
		if a.Tier != b.Tier {
			return a.Tier < b.Tier
		}
		return a.GroupNumber < b.GroupNumber
	})
	return shape, rows.Err()
}

const archiveLeagueIfEmptyStmtSql = "UPDATE leagues SET archived_season = $2 WHERE league_id = $1 AND archived_season IS NULL " +
	"AND NOT EXISTS (SELECT 1 FROM teams WHERE league_id = $1)"

// ArchiveLeagueIfEmpty retires leagueId from the pyramid (see openspec/specs/league-pyramid
// "Restructure keeps league history") once it has no teams left, recording the last season it was
// played. A league that still has teams is left active.
func ArchiveLeagueIfEmpty(dbCtx *DatabaseContext, leagueId int64, season int) error {
	_, err := dbCtx.Db.Exec(archiveLeagueIfEmptyStmtSql, leagueId, season)
	return err
}

const getLeagueCountryStmtSql = "SELECT country FROM leagues WHERE league_id = $1"

// GetLeagueZones returns leagueId's zones within its country's active pyramid. An archived league
// has no zones (it is not part of the pyramid any more).
func GetLeagueZones(dbCtx *DatabaseContext, leagueId int64) (Zones, error) {
	var country string
	if err := dbCtx.Db.QueryRow(getLeagueCountryStmtSql, leagueId).Scan(&country); err != nil {
		return Zones{}, err
	}
	shape, err := GetCountryPyramidShape(dbCtx, country)
	if err != nil {
		return Zones{}, err
	}
	for _, l := range shape.Leagues {
		if l.LeagueId == leagueId {
			return shape.Zones(l), nil
		}
	}
	return Zones{}, nil
}
