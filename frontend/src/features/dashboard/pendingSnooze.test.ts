import { afterEach, describe, expect, it, vi } from 'vitest'
import { isSnoozed, nextLocalMidnight, readSnoozed, snoozeItem, writeSnoozed, SNOOZE_STORAGE_KEY } from './pendingSnooze'

const offers1 = { id: 'offers', contentKey: 'offers:1' }
const offers2 = { id: 'offers', contentKey: 'offers:2' }
const afternoon = new Date(2026, 10, 2, 15, 30).getTime()

describe('pending snooze', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    window.localStorage.clear()
  })

  it('hides an item until the next local midnight', () => {
    const state = snoozeItem({}, offers1, afternoon)

    expect(nextLocalMidnight(afternoon)).toBe(new Date(2026, 10, 3).getTime())
    expect(isSnoozed(state, offers1, afternoon)).toBe(true)
    expect(isSnoozed(state, offers1, new Date(2026, 10, 3).getTime() - 1)).toBe(true)
    expect(isSnoozed(state, offers1, new Date(2026, 10, 3).getTime())).toBe(false)
  })

  it('brings a snoozed item back when its content changes the same day', () => {
    const state = snoozeItem({}, offers1, afternoon)

    expect(isSnoozed(state, offers2, afternoon + 60_000)).toBe(false)
  })

  it('round-trips through localStorage', () => {
    writeSnoozed(snoozeItem({}, offers1, afternoon))

    expect(readSnoozed()).toEqual({ offers: { contentKey: 'offers:1', until: new Date(2026, 10, 3).getTime() } })
  })

  it('treats unreadable stored data as nothing snoozed', () => {
    window.localStorage.setItem(SNOOZE_STORAGE_KEY, '{not json')
    expect(readSnoozed()).toEqual({})
    window.localStorage.setItem(SNOOZE_STORAGE_KEY, '[1,2]')
    expect(readSnoozed()).toEqual({})
  })

  it('survives a localStorage that throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })

    expect(readSnoozed()).toEqual({})
    expect(() => writeSnoozed(snoozeItem({}, offers1, afternoon))).not.toThrow()
  })

  it('drops expired entries when snoozing another item', () => {
    const yesterday = snoozeItem({}, offers1, afternoon)
    const later = new Date(2026, 10, 3, 9).getTime()

    const state = snoozeItem(yesterday, { id: 'noTrainer', contentKey: 'noTrainer' }, later)

    expect(Object.keys(state)).toEqual(['noTrainer'])
  })
})
