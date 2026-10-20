import { describe, expect, it } from 'vitest'
import { ADMIN_TABS, DEFAULT_ADMIN_TAB, adminTabFromParam } from './adminTabs'

describe('adminTabFromParam', () => {
  it('accepts every known tab', () => {
    for (const tab of ADMIN_TABS) {
      expect(adminTabFromParam(tab)).toBe(tab)
    }
  })

  it('falls back to the default tab for a missing or unknown value', () => {
    expect(adminTabFromParam(null)).toBe(DEFAULT_ADMIN_TAB)
    expect(adminTabFromParam('')).toBe(DEFAULT_ADMIN_TAB)
    expect(adminTabFromParam('nope')).toBe(DEFAULT_ADMIN_TAB)
  })
})
