package common

import "errors"

// Team mentality (see openspec/specs/match-simulation "Team mentality"): how a side plays, carried
// by every lineup (Lineup.Mentality). Balanced is the default and plays exactly as before
// mentality existed.
const (
	MENTALITY_DEFENSIVE = -1
	MENTALITY_BALANCED  = 0
	MENTALITY_ATTACKING = 1
)

// Mentality effect sizes. An Attacking side's three attack ratings are multiplied by
// 1 + MentalityAttackFactor and its three defense ratings by 1 - MentalityDefenseFactor (a
// Defensive side the other way round); each Attacking side adds MentalityChanceShift to the
// per-minute chance probability and each Defensive side removes it; and the side's maximum
// fatigue penalty moves by MentalityFatigueShift per step (FatigueMaxPenalty ± 0.03).
const (
	MentalityAttackFactor  = 0.08
	MentalityDefenseFactor = 0.08
	MentalityChanceShift   = 0.01
	MentalityFatigueShift  = 0.03
)

var ErrInvalidMentality = errors.New("mentality must be -1 (defensive), 0 (balanced) or 1 (attacking)")

// ValidateMentality rejects anything but the three mentalities.
func ValidateMentality(mentality int) error {
	if mentality < MENTALITY_DEFENSIVE || mentality > MENTALITY_ATTACKING {
		return ErrInvalidMentality
	}
	return nil
}

// FatigueMaxPenaltyFor is the maximum in-match fatigue penalty of a side playing with mentality:
// 0.30 Balanced, 0.33 Attacking, 0.27 Defensive.
func FatigueMaxPenaltyFor(mentality int) float64 {
	return FatigueMaxPenalty + MentalityFatigueShift*float64(mentality)
}

const GetTeamMentalityStmtSql = "SELECT mentality FROM teams WHERE team_id = $1"
const SetTeamMentalityStmtSql = "UPDATE teams SET mentality = $1 WHERE team_id = $2"
const GetTeamFriendlyMentalityStmtSql = "SELECT friendly_mentality FROM teams WHERE team_id = $1"
const SetTeamFriendlyMentalityStmtSql = "UPDATE teams SET friendly_mentality = $1 WHERE team_id = $2"

// GetTeamMentality is the mentality of a team's standing league/cup default lineup (see
// GetDefaultLineup), stored on teams next to its formation.
func GetTeamMentality(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var mentality int
	err := dbCtx.Db.QueryRow(GetTeamMentalityStmtSql, teamId).Scan(&mentality)
	return mentality, err
}

func SetTeamMentality(dbCtx *DatabaseContext, teamId int64, mentality int) error {
	if err := ValidateMentality(mentality); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(SetTeamMentalityStmtSql, mentality, teamId)
	return err
}

// GetTeamFriendlyMentality is the mentality of a team's standing friendly default lineup (see
// GetDefaultFriendlyLineup), independent of the league/cup one.
func GetTeamFriendlyMentality(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var mentality int
	err := dbCtx.Db.QueryRow(GetTeamFriendlyMentalityStmtSql, teamId).Scan(&mentality)
	return mentality, err
}

func SetTeamFriendlyMentality(dbCtx *DatabaseContext, teamId int64, mentality int) error {
	if err := ValidateMentality(mentality); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(SetTeamFriendlyMentalityStmtSql, mentality, teamId)
	return err
}

// CPU mentality thresholds (see openspec/specs/cpu-team-management "CPU mentality"): R is the CPU
// side's kickoff XI strength over the opponent's.
const (
	AiManagerAttackingRatio  = 1.15
	AiManagerDefensiveRatio  = 0.87
	AiDirectorAttackingRatio = 1.10
	AiDirectorDefensiveRatio = 0.90
	// AiDirectorVenueFactor is how much a Director values playing at home (R × it) or away
	// (R / it) before applying its thresholds.
	AiDirectorVenueFactor = 1.07
)

// PickAiMentality is a CPU team's mentality at kickoff: Balanced for a Caretaker and in every
// friendly; a Manager and a Director go Attacking when clearly stronger and Defensive when clearly
// weaker, a Director also weighing the venue. Pure, so it is tested without a database.
func PickAiMentality(aiLevel int, ownStrength int, opponentStrength int, isHome bool, isFriendly bool) int {
	if isFriendly || aiLevel < AiLevelManager || ownStrength <= 0 || opponentStrength <= 0 {
		return MENTALITY_BALANCED
	}
	ratio := float64(ownStrength) / float64(opponentStrength)
	attacking, defensive := AiManagerAttackingRatio, AiManagerDefensiveRatio
	if aiLevel >= AiLevelDirector {
		if isHome {
			ratio *= AiDirectorVenueFactor
		} else {
			ratio /= AiDirectorVenueFactor
		}
		attacking, defensive = AiDirectorAttackingRatio, AiDirectorDefensiveRatio
	}
	switch {
	case ratio >= attacking:
		return MENTALITY_ATTACKING
	case ratio <= defensive:
		return MENTALITY_DEFENSIVE
	}
	return MENTALITY_BALANCED
}

// LineupStrength is the strength of a lineup's XI as the CPU judges it: the sum of each player's
// value at his slot (see PositionValue). Players not in playersById count nothing.
func LineupStrength(lineup *Lineup, playersById map[int64]*Player) int {
	if lineup == nil {
		return 0
	}
	total := 0
	for _, p := range lineup.Positions {
		if player, ok := playersById[p.PlayerId]; ok {
			total += PositionValue(p.Position, *player)
		}
	}
	return total
}
