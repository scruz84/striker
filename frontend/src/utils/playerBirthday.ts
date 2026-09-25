/** Mirrors the backend's engine.BirthdayCycleDays: how many in-game days pass between a
 * player's birthdays. player.birthDay is the anchor for that cycle - it's rolled forward by
 * this many days server-side every time a birthday fires (see engine.ProcessBirthdays) - so the
 * next birthday is always exactly this many days after it. */
const BIRTHDAY_CYCLE_DAYS = 20
const MS_PER_DAY = 24 * 60 * 60 * 1000

/** Days remaining until this player's next birthday, computed straight from their birth_day
 * anchor rather than any value the backend precomputed - so it stays accurate between the daily
 * server-side job's runs. 0 means today. Never negative: a birthday that's already due but not
 * yet processed by the backend job just reads as 0 until the job catches up. */
export function daysUntilNextBirthday(birthDay: string, now: Date = new Date()): number {
  const anchor = new Date(birthDay).getTime()
  const elapsedDays = Math.max(0, Math.floor((now.getTime() - anchor) / MS_PER_DAY))
  const daysIntoCycle = elapsedDays % BIRTHDAY_CYCLE_DAYS
  return daysIntoCycle === 0 ? 0 : BIRTHDAY_CYCLE_DAYS - daysIntoCycle
}
