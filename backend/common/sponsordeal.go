package common

import "errors"

// SponsorDeal is a team's chosen tradeoff between guaranteed per-match sponsorship income and a
// bonus tied to that match's result - see engine.sponsorshipAmount, which applies the actual
// multipliers. Locked for the season it's chosen in (see SetSponsorDeal) - a genuine season-long
// commitment, not something to flip before every match chasing whichever formula suits the next
// result.
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
var ErrSponsorDealLocked = errors.New("sponsor deal is locked for the rest of this season")

// IsValidSponsorDeal reports whether deal is one of the three SponsorDeal* constants above.
func IsValidSponsorDeal(deal int) bool {
	return deal >= int(SponsorDealSteady) && deal <= int(SponsorDealPerformance)
}

const GetSponsorDealStmtSql = "SELECT sponsor_deal FROM teams WHERE team_id = $1"
const GetSponsorDealSeasonStmtSql = "SELECT sponsor_deal_season FROM teams WHERE team_id = $1"
const SetSponsorDealStmtSql = "UPDATE teams SET sponsor_deal = $1, sponsor_deal_season = $2 WHERE team_id = $3"

// GetSponsorDeal returns teamId's currently chosen deal.
func GetSponsorDeal(dbCtx *DatabaseContext, teamId int64) (SponsorDeal, error) {
	var deal int
	err := dbCtx.Db.QueryRow(GetSponsorDealStmtSql, teamId).Scan(&deal)
	if err != nil {
		return SponsorDealBalanced, err
	}
	return SponsorDeal(deal), nil
}

// SetSponsorDeal lets a team's owner switch to a different sponsor deal - but only once per
// season (see sponsor_deal_season): once a choice has been made for the current season,
// ErrSponsorDealLocked blocks any further change until the season rolls over.
func SetSponsorDeal(dbCtx *DatabaseContext, teamId int64, deal SponsorDeal) error {
	if !IsValidSponsorDeal(int(deal)) {
		return ErrInvalidSponsorDeal
	}

	var lockedSeason int
	if err := dbCtx.Db.QueryRow(GetSponsorDealSeasonStmtSql, teamId).Scan(&lockedSeason); err != nil {
		return err
	}
	currentSeason, err := GetCurrentSeasonForTeam(dbCtx, teamId)
	if err != nil {
		return err
	}
	if lockedSeason == currentSeason {
		return ErrSponsorDealLocked
	}

	_, err = dbCtx.Db.Exec(SetSponsorDealStmtSql, int(deal), currentSeason, teamId)
	return err
}

// IsSponsorDealLocked reports whether teamId has already chosen a sponsor deal for the current
// season - see SetSponsorDeal. Used to let the finance page disable the selector rather than
// waiting for a rejected save to reveal the lock.
func IsSponsorDealLocked(dbCtx *DatabaseContext, teamId int64) (bool, error) {
	var lockedSeason int
	if err := dbCtx.Db.QueryRow(GetSponsorDealSeasonStmtSql, teamId).Scan(&lockedSeason); err != nil {
		return false, err
	}
	currentSeason, err := GetCurrentSeasonForTeam(dbCtx, teamId)
	if err != nil {
		return false, err
	}
	return lockedSeason == currentSeason, nil
}
