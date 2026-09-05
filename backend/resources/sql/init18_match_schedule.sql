-- Admin-configurable recurring schedule for when newly generated league fixtures are dated -
-- e.g. "Saturdays at 14:00" or "every day at 16:00" (see common.MatchSchedule,
-- engine.ComputeFixtureDates). Replaces the previous hardcoded demo cadence of one match round
-- every 15 minutes in engine.randomWeekMatches2.
--
-- match_schedule_weekdays is a bitmask, bit i set means time.Weekday(i) is a match day
-- (bit 0 = Sunday ... bit 6 = Saturday); default 127 selects every day. match_schedule_minute_of_day
-- is minutes since midnight (0-1439); default 960 is 16:00.
ALTER TABLE season ADD COLUMN IF NOT EXISTS match_schedule_weekdays smallint NOT NULL DEFAULT 127;
ALTER TABLE season ADD COLUMN IF NOT EXISTS match_schedule_minute_of_day smallint NOT NULL DEFAULT 960;
