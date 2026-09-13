// A MatchSchedule's hour/minute is a wire-format time-of-day the backend interprets in its own
// server timezone (see backend/engine/schedule.go's NextScheduledOccurrence, which builds each
// fixture's actual instant from time.Now().Location() - UTC on every deployment this project
// documents) - while every other timestamp in this app displays in the viewer's own local
// timezone (see formatDateTime, which never overrides Intl.DateTimeFormat's default timeZone).
// Without converting here, the admin schedule editor would show/accept a different hour than the
// one real matches actually land on for any viewer not in the server's own timezone - exactly the
// mismatch this app's other date displays are built to avoid.
//
// This only rescales the hour/minute-of-day; it doesn't shift which weekdays are selected. A
// schedule's WeekdaysMask stays in server-time weekday terms, so a viewer whose UTC offset is
// large enough to push a near-midnight hour across a day boundary will see a schedule that reads
// as one weekday off from the day their local clock would put it on - a known, narrow edge case
// judged not worth the complexity of also re-deriving the weekday mask through the same shift.
function convertHourMinute(hour: number, minute: number, toLocal: boolean): { hour: number; minute: number } {
  // Today's date, not a fixed placeholder - the viewer's UTC offset can depend on the time of
  // year (daylight saving), so anchoring to some arbitrary fixed date (e.g. one in January) would
  // silently apply the wrong offset for anyone viewing this in a different DST period, which is
  // exactly the kind of mismatch this conversion exists to avoid. This still can't perfectly
  // represent a recurring weekly rule across a DST transition - it reflects the offset in effect
  // right now, the same practical tradeoff most scheduling UIs make.
  const now = new Date()
  const reference = toLocal
    ? new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate(), hour, minute))
    : new Date(now.getFullYear(), now.getMonth(), now.getDate(), hour, minute)
  return toLocal
    ? { hour: reference.getHours(), minute: reference.getMinutes() }
    : { hour: reference.getUTCHours(), minute: reference.getUTCMinutes() }
}

/** Converts a schedule's wire-format (server-timezone) hour/minute to the viewer's local time -
 * use when displaying or pre-filling an editor from a MatchSchedule the API returned. */
export function scheduleHourMinuteToLocal(hour: number, minute: number): { hour: number; minute: number } {
  return convertHourMinute(hour, minute, true)
}

/** Converts an hour/minute the viewer picked in their own local time back to the wire format
 * (server timezone) a MatchSchedule is stored/sent as - use right before saving. */
export function localHourMinuteToSchedule(hour: number, minute: number): { hour: number; minute: number } {
  return convertHourMinute(hour, minute, false)
}
