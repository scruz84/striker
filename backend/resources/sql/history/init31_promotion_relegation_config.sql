-- Admin-configurable promotion/relegation slot counts, replacing the previous hardcoded
-- constants (engine.RelegationSlotsPerGroup=2, PromotionCandidatesPerGroup=1). Defaults match
-- those old constants exactly, so existing worlds see no behavior change until an admin touches
-- this. See common.GetPromotionRelegationConfig/SetPromotionRelegationConfig and
-- engine.ComputePromotionRelegation.
ALTER TABLE season ADD COLUMN IF NOT EXISTS promotion_slots_per_group smallint NOT NULL DEFAULT 1;
ALTER TABLE season ADD COLUMN IF NOT EXISTS relegation_slots_per_group smallint NOT NULL DEFAULT 2;
