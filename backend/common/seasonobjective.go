package common

import "database/sql"

// A CPU club's season objectives (see openspec/specs/cpu-team-management "CPU season objective").
const (
	SeasonObjectiveOpen            = "open"
	SeasonObjectiveRelegationFight = "relegation_fight"
	SeasonObjectivePromotionPush   = "promotion_push"
	SeasonObjectiveNothingAtStake  = "nothing_at_stake"
)

// SeasonObjectiveStartShare is the share of its league matches a club must have played before it
// has an objective other than open; SeasonObjectiveMargin is the "within reach" margin in points.
const (
	SeasonObjectiveStartShare = 1.0 / 3.0
	SeasonObjectiveMargin     = 3
)

// SeasonSituation is what a club's objective is derived from: its league's points best to worst,
// its own place in that list, its league matches played and still to play, and the promotion and
// relegation places its league has (PromotionSlots is 1 - the title - in a top division;
// RelegationSlots is 0 without a tier below).
type SeasonSituation struct {
	Points          []int
	Index           int
	Played          int
	Remaining       int
	PromotionSlots  int
	RelegationSlots int
}

// SeasonObjectiveFor derives a club's objective from its situation. Pure.
func SeasonObjectiveFor(s SeasonSituation) string {
	n := len(s.Points)
	total := s.Played + s.Remaining
	if n == 0 || s.Index < 0 || s.Index >= n || total == 0 || float64(s.Played) < SeasonObjectiveStartShare*float64(total) {
		return SeasonObjectiveOpen
	}
	own := s.Points[s.Index]
	reach := 3 * s.Remaining

	hasZone := s.RelegationSlots > 0 && s.RelegationSlots < n
	zoneBest := 0
	if hasZone {
		zoneBest = s.Points[n-s.RelegationSlots]
		if own <= zoneBest+SeasonObjectiveMargin {
			return SeasonObjectiveRelegationFight
		}
	}
	hasPlace := s.PromotionSlots > 0 && s.PromotionSlots <= n
	place := 0
	if hasPlace {
		place = s.Points[s.PromotionSlots-1]
		if own >= place-SeasonObjectiveMargin {
			return SeasonObjectivePromotionPush
		}
	}
	cannotReach := !hasPlace || own+reach < place
	cannotBeCaught := !hasZone || zoneBest+reach < own
	if cannotReach && cannotBeCaught {
		return SeasonObjectiveNothingAtStake
	}
	return SeasonObjectiveOpen
}

// CountRemainingLeagueMatchesStmtSql counts a team's unplayed league matches in its current league
// this season.
const CountRemainingLeagueMatchesStmtSql = "SELECT count(*) FROM matches m JOIN teams t ON t.team_id = $1 " +
	"JOIN leagues l ON l.league_id = t.league_id JOIN countries c ON c.name = l.country " +
	"WHERE m.league_id = t.league_id AND m.season = c.current_season AND m.played = false " +
	"AND (m.home_team_id = $1 OR m.away_team_id = $1)"

const getTeamLeagueTierStmtSql = "SELECT l.league_id, l.tier, l.country, " +
	"(SELECT MAX(l2.tier) FROM leagues l2 WHERE l2.country = l.country AND l2.archived_season IS NULL) " +
	"FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.team_id = $1"

// GetTeamSeasonSituation reads teamId's SeasonSituation: its league's ranked table, its record
// and remaining league matches, and the promotion/relegation places its tier has.
func GetTeamSeasonSituation(dbCtx *DatabaseContext, teamId int64) (SeasonSituation, error) {
	var leagueId int64
	var tier, maxTier int
	var country string
	var maxTierNull sql.NullInt64
	if err := dbCtx.Db.QueryRow(getTeamLeagueTierStmtSql, teamId).Scan(&leagueId, &tier, &country, &maxTierNull); err != nil {
		return SeasonSituation{}, err
	}
	maxTier = int(maxTierNull.Int64)
	standings, err := GetRankedClassification(dbCtx, leagueId)
	if err != nil {
		return SeasonSituation{}, err
	}
	record, err := GetTeamSeasonRecord(dbCtx, teamId)
	if err != nil {
		return SeasonSituation{}, err
	}
	var remaining int
	if err := dbCtx.Db.QueryRow(CountRemainingLeagueMatchesStmtSql, teamId).Scan(&remaining); err != nil {
		return SeasonSituation{}, err
	}
	// The places at stake come from the country's pyramid (see PyramidZones): below tier 1 the
	// direct and playoff promotion places, above the bottom tier the direct and playoff relegation
	// places. In tier 1 the place worth chasing is the title.
	shape, err := GetCountryPyramidShape(dbCtx, country)
	if err != nil {
		return SeasonSituation{}, err
	}
	zones := PyramidZones(tier, maxTier, len(standings), !shape.FollowsRatioRule())
	s := SeasonSituation{Index: -1, Played: record.Matches, Remaining: remaining, PromotionSlots: 1}
	if tier > 1 {
		s.PromotionSlots = zones.PromotionPlaces()
	}
	if tier < maxTier {
		s.RelegationSlots = zones.RelegationPlaces()
	}
	s.Points = make([]int, len(standings))
	for i, st := range standings {
		s.Points[i] = st.Points
		if st.TeamId == teamId {
			s.Index = i
		}
	}
	return s, nil
}

// TeamSeasonObjective is teamId's objective, open when its situation can't be read.
func TeamSeasonObjective(dbCtx *DatabaseContext, teamId int64) string {
	s, err := GetTeamSeasonSituation(dbCtx, teamId)
	if err != nil {
		return SeasonObjectiveOpen
	}
	return SeasonObjectiveFor(s)
}

// AdjustProfileForObjective is profile as a club with objective shops (see
// openspec/specs/cpu-transfer-market "Market urgency"): a Manager or Director fighting relegation
// lowers its minimum score by 2 (never below 1) and halves its budget reserve; a Director in a
// promotion push lowers its minimum score by 1. Pure; profile is a value, so the caller's copy is
// untouched.
func AdjustProfileForObjective(profile AiTransferProfile, objective string) AiTransferProfile {
	if profile.Level < AiLevelManager {
		return profile
	}
	switch objective {
	case SeasonObjectiveRelegationFight:
		profile.MinScore -= 2
		if profile.MinScore < 1 {
			profile.MinScore = 1
		}
		profile.BudgetReserve /= 2
	case SeasonObjectivePromotionPush:
		if profile.Level >= AiLevelDirector {
			profile.MinScore--
		}
	}
	return profile
}
