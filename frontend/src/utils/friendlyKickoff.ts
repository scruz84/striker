import type { FriendlySchedule } from '../api/types'

// The minimum lead time backend.AcceptFriendlyChallenge enforces (friendlyScheduleDelay) -
// mirrored here only to seed a sensible default kickoff value and a client-side `min` on the
// picker; the backend re-validates regardless.
export const MIN_KICKOFF_LEAD_MINUTES = 5

export function toDateTimeLocalValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// Friendlies kick off only on the hour, at quarter past, half past or quarter to - mirrored from
// common.IsFriendlyKickoffSlot, which rejects anything else. Also the pickers' `step` (seconds).
export const KICKOFF_STEP_MINUTES = 15
export const KICKOFF_STEP_SECONDS = KICKOFF_STEP_MINUTES * 60

// The first quarter-hour slot at least minutesFromNow ahead. Used both as a default value and as
// the pickers' `min`, which is also where the browser starts counting `step` from - so it has to
// sit on a slot itself.
export function defaultKickoffValue(minutesFromNow: number): string {
  const stepMs = KICKOFF_STEP_MINUTES * 60 * 1000
  const earliest = Date.now() + minutesFromNow * 60 * 1000
  return toDateTimeLocalValue(new Date(Math.ceil(earliest / stepMs) * stepMs))
}

// Whether a datetime-local value falls on :00, :15, :30 or :45 (an empty value is left to the
// caller). Every timezone offset is a multiple of 15 minutes, so a local slot is a UTC slot too.
export function isQuarterHour(value: string): boolean {
  if (!value) return true
  const date = new Date(value)
  return date.getMinutes() % KICKOFF_STEP_MINUTES === 0 && date.getSeconds() === 0
}

// Shared by every friendly kickoff picker (the challenge form's optional proposed-kickoff field,
// the accept dialog's kickoff field, and the team-page challenge dialog) - all need to reject a
// datetime-local value that falls on a day common.GetFriendlyScheduleWeekdays doesn't allow,
// before the backend gets a chance to.
export function isWeekdayAllowed(schedule: FriendlySchedule | undefined, value: string): boolean {
  if (!schedule || !value) return true
  const day = new Date(value).getDay()
  return (schedule.weekdaysMask & (1 << day)) !== 0
}

// The i18n key of what is wrong with a picked kickoff, or null when it is fine (or empty): a day
// the friendly schedule doesn't open, or a time off the quarter-hour grid.
export function kickoffErrorKey(schedule: FriendlySchedule | undefined, value: string): string | null {
  if (!isWeekdayAllowed(schedule, value)) return 'friendlies.received.kickoffNotAllowed'
  if (!isQuarterHour(value)) return 'friendlies.received.kickoffNotQuarterHour'
  return null
}
