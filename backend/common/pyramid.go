package common

import "sort"

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

// MatchesConfig reports whether the shape is exactly tiers: same tier count, groups per tier and
// teams in every group. A country that matches needs no restructure (see
// engine.restructureIfNeeded).
func (s PyramidShape) MatchesConfig(tiers []TierConfig) bool {
	if len(tiers) == 0 || s.TierCount() != len(tiers) {
		return false
	}
	byTier := make(map[int]TierConfig, len(tiers))
	for _, tc := range tiers {
		byTier[tc.Tier] = tc
	}
	groups := s.GroupsByTier()
	for tier := 1; tier <= len(tiers); tier++ {
		tc, ok := byTier[tier]
		if !ok || groups[tier] != tc.Groups {
			return false
		}
	}
	for _, l := range s.Leagues {
		if l.Teams != byTier[l.Tier].TeamsPerLeague {
			return false
		}
	}
	return true
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
