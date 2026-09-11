-- Admin-configurable weekday restriction on when accepted friendly challenges may kick off -
-- e.g. weekends only (see common.GetFriendlyScheduleWeekdays, common.nextAllowedFriendlyDate).
-- Unlike the league match_schedule (init18_match_schedule.sql), this has no time-of-day
-- component: a friendly should still kick off as soon as the schedule allows (the existing
-- ~5-minute post-acceptance delay), just possibly pushed forward to the next allowed day.
--
-- Bitmask, bit i set means time.Weekday(i) is allowed (bit 0 = Sunday ... bit 6 = Saturday,
-- same convention as match_schedule_weekdays); default 127 allows every day.
ALTER TABLE season ADD COLUMN IF NOT EXISTS friendly_schedule_weekdays smallint NOT NULL DEFAULT 127;
