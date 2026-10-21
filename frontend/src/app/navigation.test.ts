import { describe, expect, it } from 'vitest'
import { findDetailRoute, isClubPage, isNavSelected, parseDivisionName } from './navigation'

describe('parseDivisionName', () => {
  it('reads roman tier and group', () => {
    expect(parseDivisionName('I.1')).toEqual({ tier: 1, group: 1 })
    expect(parseDivisionName('III.16')).toEqual({ tier: 3, group: 16 })
    expect(parseDivisionName('IV')).toEqual({ tier: 4, group: undefined })
  })
  it('leaves custom names alone', () => {
    expect(parseDivisionName('Premier')).toBeUndefined()
  })
})

describe('findDetailRoute', () => {
  it('treats other clubs as drill-downs but not the own club', () => {
    expect(findDetailRoute('/team/7', 3)?.pattern).toBe('/team/:id')
    expect(findDetailRoute('/team/3', 3)).toBeUndefined()
    expect(findDetailRoute('/team/3/matches', 3)).toBeUndefined()
    expect(findDetailRoute('/match/9')?.fallback).toBe('/dashboard')
    expect(findDetailRoute('/dashboard')).toBeUndefined()
  })
})

describe('isClubPage', () => {
  it('only matches pages about the own club', () => {
    expect(isClubPage('/dashboard', 3)).toBe(true)
    expect(isClubPage('/team/3', 3)).toBe(true)
    expect(isClubPage('/team/4', 3)).toBe(false)
    expect(isClubPage('/cup', 3)).toBe(false)
  })
})

describe('isNavSelected', () => {
  it('keeps a section selected on its sub-routes', () => {
    expect(isNavSelected('/cup', '/cup/ties/4', ['/cup/ties'])).toBe(true)
    expect(isNavSelected('/cup', '/friendlies')).toBe(false)
  })
})
