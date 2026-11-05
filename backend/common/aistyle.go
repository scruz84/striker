package common

import "database/sql"

// CPU club styles (see openspec/specs/cpu-team-management "Club style").
const (
	AiStyleBalanced  = "balanced"
	AiStyleYouth     = "youth"
	AiStyleFrugal    = "frugal"
	AiStyleAmbitious = "ambitious"
)

// aiStyleCodes is how each style is stored in teams.ai_style (NULL reads as balanced).
var aiStyleCodes = map[string]int{AiStyleBalanced: 0, AiStyleYouth: 1, AiStyleFrugal: 2, AiStyleAmbitious: 3}

// aiStyleCycle maps teamId mod 5 to the style a club is given: 2 in 5 clubs balanced, 1 each
// youth, frugal and ambitious (ids are sequential, so the shares hold within a league).
var aiStyleCycle = [5]string{AiStyleBalanced, AiStyleBalanced, AiStyleYouth, AiStyleFrugal, AiStyleAmbitious}

// AiClubStyleFor is the style a club with teamId is given when it is assigned one.
func AiClubStyleFor(teamId int64) string {
	i := teamId % int64(len(aiStyleCycle))
	if i < 0 {
		i = -i
	}
	return aiStyleCycle[i]
}

// AiStyleCode is style's teams.ai_style value; AiStyleFromCode reads one back (a NULL or unknown
// code is balanced).
func AiStyleCode(style string) int { return aiStyleCodes[style] }

func AiStyleFromCode(code sql.NullInt64) string {
	if code.Valid {
		for style, c := range aiStyleCodes {
			if int64(c) == code.Int64 {
				return style
			}
		}
	}
	return AiStyleBalanced
}

const SetTeamAiStyleStmtSql = "UPDATE teams SET ai_style = $1 WHERE team_id = $2"
const getTeamsWithoutAiStyleStmtSql = "SELECT team_id FROM teams WHERE ai_style IS NULL AND is_national_team = false"

// AssignAiStyle stores teamId's style (see AiClubStyleFor) - called when a club is created.
func AssignAiStyle(dbCtx *DatabaseContext, teamId int64) error {
	_, err := dbCtx.Db.Exec(SetTeamAiStyleStmtSql, AiStyleCode(AiClubStyleFor(teamId)), teamId)
	return err
}

// BackfillAiStyles gives every club that has no style yet its AiClubStyleFor style - called once at
// service startup (see service.StartSvc), like BackfillAiLevels, so a world created before styles
// existed gets them. A style is never changed once stored.
func BackfillAiStyles(dbCtx *DatabaseContext) error {
	ids, err := queryIds(dbCtx, getTeamsWithoutAiStyleStmtSql)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := AssignAiStyle(dbCtx, id); err != nil {
			return err
		}
	}
	return nil
}

// Style adjustments of a Manager's or Director's transfer profile.
const (
	frugalReserveBonus      = 0.15
	frugalKeyPlayerDiscount = 0.5
	ambitiousReserveCut     = 0.10
	ambitiousMinReserve     = 0.05
	ambitiousKeyPlayerBonus = 0.5
	ambitiousMinScoreCut    = 1
	styleMinScoreFloor      = 1
	frugalMinKeyMultiplier  = 1.5
)

// ApplyClubStyle adjusts profile for a club of style: a youth club signs projects, reads potential
// and lends youngsters; a frugal one keeps more of its budget and protects key players less; an
// ambitious one keeps less, settles for smaller gains and protects key players more. A Caretaker's
// profile is returned unchanged. Pure.
func ApplyClubStyle(profile AiTransferProfile, style string) AiTransferProfile {
	if profile.Level < AiLevelManager {
		return profile
	}
	switch style {
	case AiStyleYouth:
		profile.SignsProjects = true
		profile.EstimatesPotential = true
		profile.LoansOutYouth = true
	case AiStyleFrugal:
		profile.BudgetReserve += frugalReserveBonus
		profile.KeyPlayerPriceMultiplier -= frugalKeyPlayerDiscount
		if profile.KeyPlayerPriceMultiplier < frugalMinKeyMultiplier {
			profile.KeyPlayerPriceMultiplier = frugalMinKeyMultiplier
		}
	case AiStyleAmbitious:
		profile.BudgetReserve -= ambitiousReserveCut
		if profile.BudgetReserve < ambitiousMinReserve {
			profile.BudgetReserve = ambitiousMinReserve
		}
		profile.MinScore -= ambitiousMinScoreCut
		if profile.MinScore < styleMinScoreFloor {
			profile.MinScore = styleMinScoreFloor
		}
		profile.KeyPlayerPriceMultiplier += ambitiousKeyPlayerBonus
	}
	return profile
}

// AiTransferProfileForTeam is the transfer profile of a CPU club: its level's profile adjusted by
// its style.
func AiTransferProfileForTeam(team *Team) AiTransferProfile {
	return ApplyClubStyle(AiTransferProfileFor(team.AiLevel), team.AiStyle)
}
