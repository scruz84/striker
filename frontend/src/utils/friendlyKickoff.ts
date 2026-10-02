import type { FriendlySchedule } from '../api/types'

// The minimum lead time backend.AcceptFriendlyChallenge enforces (friendlyScheduleDelay) -
// mirrored here only to seed a sensible default kickoff value and a client-side `min` on the
// picker; the backend re-validates regardless.
export const MIN_KICKOFF_LEAD_MINUTES = 5

export function toDateTimeLocalValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function defaultKickoffValue(minutesFromNow: number): string {
  return toDateTimeLocalValue(new Date(Date.now() + minutesFromNow * 60 * 1000))
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
