package common

import "errors"

// SponsorDeal is a team's chosen tradeoff between guaranteed per-match sponsorship income and a
// bonus tied to that match's result - see engine.sponsorshipAmount, which applies the actual
// multipliers. Nothing locks a team into one deal for a season; it can be changed any time, the
// same way ticket price can.
type SponsorDeal int

const (
	// SponsorDealSteady pays the highest guaranteed amount with no result-based bonus - the
	// safest choice for a team that would rather not gamble on results.
	SponsorDealSteady SponsorDeal = 0
	// SponsorDealBalanced pays a middling guaranteed amount plus a modest bonus for a win or
	// draw - the default for a team that hasn't chosen otherwise.
	SponsorDealBalanced SponsorDeal = 1
	// SponsorDealPerformance pays the lowest guaranteed amount but the biggest win/draw bonus -
	// the highest-variance choice: better than Steady for a team on a good run, worse for one
	// that mostly loses.
	SponsorDealPerformance SponsorDeal = 2
)

var ErrInvalidSponsorDeal = errors.New("invalid sponsor deal")

// IsValidSponsorDeal reports whether deal is one of the three SponsorDeal* constants above.
func IsValidSponsorDeal(deal int) bool {
	return deal >= int(SponsorDealSteady) && deal <= int(SponsorDealPerformance)
}

const GetSponsorDealStmtSql = "SELECT sponsor_deal FROM teams WHERE team_id = $1"
const SetSponsorDealStmtSql = "UPDATE teams SET sponsor_deal = $1 WHERE team_id = $2"

// GetSponsorDeal returns teamId's currently chosen deal.
func GetSponsorDeal(dbCtx *DatabaseContext, teamId int64) (SponsorDeal, error) {
	var deal int
	err := dbCtx.Db.QueryRow(GetSponsorDealStmtSql, teamId).Scan(&deal)
	if err != nil {
		return SponsorDealBalanced, err
	}
	return SponsorDeal(deal), nil
}

// SetSponsorDeal lets a team's owner switch to a different sponsor deal.
func SetSponsorDeal(dbCtx *DatabaseContext, teamId int64, deal SponsorDeal) error {
	if !IsValidSponsorDeal(int(deal)) {
		return ErrInvalidSponsorDeal
	}
	_, err := dbCtx.Db.Exec(SetSponsorDealStmtSql, int(deal), teamId)
	return err
}
