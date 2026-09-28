import { useState } from 'react'

export interface MinuteScrub {
  minute: number
  atNow: boolean
  goToMinute: (value: number) => void
}

// useMinuteScrub tracks a "viewing minute" for a live or played match's minute-by-minute panels
// (zone comparison, player ratings, lineup ratings) - null internally means "pinned to now", so a
// still-broadcasting match's minute keeps climbing without this needing an effect to follow it;
// once a viewer scrubs to an earlier minute, it sticks there until they jump back
// (goToMinute(currentMinute)) or currentMinute itself catches up to whatever they picked.
//
// Extracted so multiple panels on the same live page (e.g. the lineup pitch and the zone
// comparison) can share one scrub position instead of each owning an independent one.
export function useMinuteScrub(currentMinute: number): MinuteScrub {
  const [scrubbedMinute, setScrubbedMinute] = useState<number | null>(null)
  const minute = scrubbedMinute ?? currentMinute
  const atNow = minute >= currentMinute

  const goToMinute = (value: number) => {
    const clamped = Math.max(0, Math.min(currentMinute, value))
    setScrubbedMinute(clamped >= currentMinute ? null : clamped)
  }

  return { minute, atNow, goToMinute }
}
