export interface Countdown {
  days: number
  hours: number
  minutes: number
}

/** Time left until `targetIso`, floored to whole minutes; undefined once the moment has passed or
 * if the date is unparseable. */
export function countdownUntil(targetIso: string, now: number = Date.now()): Countdown | undefined {
  const target = new Date(targetIso).getTime()
  if (Number.isNaN(target) || target <= now) {
    return undefined
  }
  const totalMinutes = Math.floor((target - now) / 60000)
  return {
    days: Math.floor(totalMinutes / (60 * 24)),
    hours: Math.floor((totalMinutes % (60 * 24)) / 60),
    minutes: totalMinutes % 60,
  }
}

/** Which i18n key + params describe a countdown most compactly: "2 d 3 h", "3 h 20 min" or "12 min". */
export function countdownParts(countdown: Countdown): { key: 'days' | 'hours' | 'minutes'; params: Record<string, number> } {
  if (countdown.days > 0) {
    return { key: 'days', params: { days: countdown.days, hours: countdown.hours } }
  }
  if (countdown.hours > 0) {
    return { key: 'hours', params: { hours: countdown.hours, minutes: countdown.minutes } }
  }
  return { key: 'minutes', params: { minutes: Math.max(countdown.minutes, 1) } }
}
