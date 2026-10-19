package common

import (
	"database/sql"
	"errors"
	"math"
)

// DivisionWageFactorTier1/2/3 scale what a club actually pays in wages by the division it plays
// in. Income grows ~5x from third to first division (sponsorship, prize money, merchandising)
// while skills - and so stored wages - are identical in every division, which let first-division
// clubs print money. The factor is applied when wages are PAID (GetTeamWageBill,
// GetTeamLoanWageContribution, the coach's wage in engine.deductTrainerWage) and nowhere else:
// the stored wage stays the base wage that offers, renewals and the CPU's valuation work with,
// and promotion/relegation raises/lowers both income and payroll at once without a migration.
const DivisionWageFactorTier1 = 4.0
const DivisionWageFactorTier2 = 2.1
const DivisionWageFactorTier3 = 1.1

// DivisionWageFactor returns the wage multiplier for a division tier (1 = top flight). Any tier
// below the third pays the third-division factor; a tier below 1 (no division) pays no factor.
func DivisionWageFactor(tier int) float64 {
	switch {
	case tier <= 0:
		return 1.0
	case tier == 1:
		return DivisionWageFactorTier1
	case tier == 2:
		return DivisionWageFactorTier2
	default:
		return DivisionWageFactorTier3
	}
}

// ApplyWageFactor scales a stored (base) wage by a division factor, rounded to the nearest unit.
func ApplyWageFactor(baseWage int64, factor float64) int64 {
	return int64(math.Round(float64(baseWage) * factor))
}

// GetTeamWageFactor returns the wage multiplier of the division teamId currently plays in. A team
// with no league (a national team) pays no factor.
func GetTeamWageFactor(dbCtx *DatabaseContext, teamId int64) (float64, error) {
	tier, err := GetTeamTier(dbCtx, teamId)
	if errors.Is(err, sql.ErrNoRows) {
		return 1.0, nil
	}
	if err != nil {
		return 1.0, err
	}
	return DivisionWageFactor(tier), nil
}
