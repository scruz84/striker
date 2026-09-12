package common

import "errors"

// Defaults match the previous hardcoded constants (engine.RelegationSlotsPerGroup=2,
// PromotionCandidatesPerGroup=1) exactly, so an existing world sees no behavior change until an
// admin actually touches this setting.
const (
	DefaultPromotionSlotsPerGroup  = 1
	DefaultRelegationSlotsPerGroup = 2
)

// MaxPromotionRelegationSlotsPerGroup bounds how many teams can move per group each season - a
// league only ever has a handful of teams (see common.TierConfig.TeamsPerLeague), so anything
// much larger would promote/relegate an entire division and isn't a meaningful setting.
const MaxPromotionRelegationSlotsPerGroup = 10

// PromotionRelegationConfig is the admin-configured number of teams that move between adjacent
// tiers at season rollover - see engine.ComputePromotionRelegation.
type PromotionRelegationConfig struct {
	PromotionSlotsPerGroup  int `json:"promotionSlotsPerGroup"`
	RelegationSlotsPerGroup int `json:"relegationSlotsPerGroup"`
}

var ErrInvalidPromotionRelegationConfig = errors.New("promotion and relegation slots must each be between 0 and 10")

// ValidatePromotionRelegationConfig checks a config is usable before it's persisted - zero is
// allowed for either (turns that direction of movement off entirely), same as
// SetMaxFriendliesPerWeek's zero-disables convention.
func ValidatePromotionRelegationConfig(c PromotionRelegationConfig) error {
	if c.PromotionSlotsPerGroup < 0 || c.PromotionSlotsPerGroup > MaxPromotionRelegationSlotsPerGroup {
		return ErrInvalidPromotionRelegationConfig
	}
	if c.RelegationSlotsPerGroup < 0 || c.RelegationSlotsPerGroup > MaxPromotionRelegationSlotsPerGroup {
		return ErrInvalidPromotionRelegationConfig
	}
	return nil
}

const GetPromotionRelegationConfigStmtSql = "SELECT promotion_slots_per_group, relegation_slots_per_group FROM season"
const SetPromotionRelegationConfigStmtSql = "UPDATE season SET promotion_slots_per_group = $1, relegation_slots_per_group = $2"

// GetPromotionRelegationConfig returns the current admin-configured slot counts, falling back to
// the historical defaults on a lookup error - season rollover must never be blocked by a
// settings-row hiccup.
func GetPromotionRelegationConfig(dbCtx *DatabaseContext) PromotionRelegationConfig {
	var promotion, relegation int
	if err := dbCtx.Db.QueryRow(GetPromotionRelegationConfigStmtSql).Scan(&promotion, &relegation); err != nil {
		return PromotionRelegationConfig{
			PromotionSlotsPerGroup:  DefaultPromotionSlotsPerGroup,
			RelegationSlotsPerGroup: DefaultRelegationSlotsPerGroup,
		}
	}
	return PromotionRelegationConfig{PromotionSlotsPerGroup: promotion, RelegationSlotsPerGroup: relegation}
}

// SetPromotionRelegationConfig persists new slot counts. Only affects the next season rollover
// that hasn't happened yet - a promotion/relegation pass already applied doesn't get undone.
func SetPromotionRelegationConfig(dbCtx *DatabaseContext, c PromotionRelegationConfig) error {
	if err := ValidatePromotionRelegationConfig(c); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(SetPromotionRelegationConfigStmtSql, c.PromotionSlotsPerGroup, c.RelegationSlotsPerGroup)
	return err
}
