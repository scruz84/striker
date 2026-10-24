import { describe, expect, it } from 'vitest'
import { formatDateTime } from './formatDate'
import { matchSubtitle } from './matchSubtitle'

const DATE = '2026-10-21T20:00:00Z'

describe('matchSubtitle', () => {
  it('joins label, date and weather emoji', () => {
    expect(matchSubtitle('Jornada 3 de 38', DATE, 'sunny')).toBe(`Jornada 3 de 38 · ${formatDateTime(DATE)} · ☀️`)
  })

  it('drops the weather when none is stored', () => {
    expect(matchSubtitle('Amistoso', DATE, undefined)).toBe(`Amistoso · ${formatDateTime(DATE)}`)
  })

  it('drops an unknown label and an unknown weather code', () => {
    expect(matchSubtitle(undefined, DATE, 'fog')).toBe(formatDateTime(DATE))
  })

  it('is empty when nothing is known', () => {
    expect(matchSubtitle(undefined, undefined, undefined)).toBe('')
  })
})
