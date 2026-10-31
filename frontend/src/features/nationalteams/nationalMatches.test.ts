import { beforeEach, describe, expect, it } from 'vitest'
import type { NationalMatch } from '../../api/types'
import { nationalRoundPollMs } from '../../api/hooks'
import i18n from '../../i18n'
import { countriesMatching, EMPTY_FILTER, formStrip, matchesFilter, nationalMatchLabel, nationalMatchLink, splitRound } from './nationalMatches'

function match(overrides: Partial<NationalMatch> = {}): NationalMatch {
  return {
    kind: 'friendly',
    id: 1,
    homeTeamId: 10,
    homeTeamName: 'Spain',
    homeCountry: 'es',
    awayTeamId: 20,
    awayTeamName: 'Portugal',
    awayCountry: 'pt',
    kickoff: '2026-11-07T20:00:00Z',
    status: 'scheduled',
    ...overrides,
  }
}

const t = (key: string, options?: Record<string, unknown>) => i18n.t(key, options)

beforeEach(async () => {
  await i18n.changeLanguage('es')
})

describe('splitRound', () => {
  it('puts live matches first and groups the rest by kickoff', () => {
    const sections = splitRound([
      match({ id: 1, kickoff: '2026-11-07T21:00:00Z' }),
      match({ id: 2, kickoff: '2026-11-07T20:00:00Z' }),
      match({ id: 3, kickoff: '2026-11-07T21:00:00Z' }),
      match({ id: 4, status: 'live', minute: 30, homeGoals: 1, awayGoals: 0, kickoff: '2026-10-31T21:00:00Z' }),
      match({ id: 5, status: 'finished', homeGoals: 0, awayGoals: 0, kickoff: '2026-10-31T18:00:00Z' }),
      match({ id: 6, status: 'finished', homeGoals: 2, awayGoals: 0, kickoff: '2026-10-31T19:00:00Z' }),
    ])
    expect(sections.live.map((m) => m.id)).toEqual([4])
    expect(sections.upcoming.map((g) => g.matches.map((m) => m.id))).toEqual([[2], [1, 3]])
    expect(sections.finished.map((g) => g.matches.map((m) => m.id))).toEqual([[6], [5]])
  })

  it('does not cap the round - 104 Saturday fixtures all come back', () => {
    const many = Array.from({ length: 104 }, (_, i) => match({ id: i + 1 }))
    expect(splitRound(many).upcoming[0].matches).toHaveLength(104)
  })
})

describe('matchesFilter', () => {
  it('finds a country by its localized name, ignoring case and accents', () => {
    expect(matchesFilter(t, match(), { ...EMPTY_FILTER, query: 'Esp' })).toBe(true)
    expect(matchesFilter(t, match(), { ...EMPTY_FILTER, query: 'espana' })).toBe(true)
    expect(matchesFilter(t, match(), { ...EMPTY_FILTER, query: 'Nigeria' })).toBe(false)
  })

  it('keeps a match when either side is in the region', () => {
    const intercontinental = match({ awayCountry: 'ng', awayTeamName: 'Nigeria' })
    expect(matchesFilter(t, intercontinental, { ...EMPTY_FILTER, region: 'africa' })).toBe(true)
    expect(matchesFilter(t, match(), { ...EMPTY_FILTER, region: 'africa' })).toBe(false)
  })

  it('keeps only matches with the viewer\'s players when asked', () => {
    expect(matchesFilter(t, match({ myPlayers: 2 }), { ...EMPTY_FILTER, onlyMine: true })).toBe(true)
    expect(matchesFilter(t, match(), { ...EMPTY_FILTER, onlyMine: true })).toBe(false)
  })
})

describe('countriesMatching', () => {
  it('prefers an exact name so one country can be sent to the server', () => {
    expect(countriesMatching(t, 'irlanda')).toEqual(['ie'])
    expect(countriesMatching(t, 'irlan').length).toBeGreaterThan(1)
    expect(countriesMatching(t, '')).toEqual([])
  })
})

describe('labels and links', () => {
  it('labels each competition', () => {
    expect(nationalMatchLabel(t, match())).toBe('Amistoso')
    expect(nationalMatchLabel(t, match({ kind: 'worldCupGroup', groupName: 'C', round: 2 }))).toBe('Mundial · Grupo C · J2')
    expect(nationalMatchLabel(t, match({ kind: 'worldCupKnockout', round: 4, totalRounds: 4 }))).toBe(`Mundial · ${t('cup.final')}`)
  })

  it('links knockout ties to the cup tie page and the rest to the friendly page', () => {
    expect(nationalMatchLink(match({ id: 7 }))).toBe('/friendlies/7')
    expect(nationalMatchLink(match({ id: 7, kind: 'worldCupKnockout' }))).toBe('/cup/ties/7')
  })
})

describe('formStrip', () => {
  it('reads results from the team\'s side, newest first, at most five', () => {
    const results = [
      match({ status: 'finished', homeGoals: 2, awayGoals: 0 }),
      match({ status: 'finished', homeTeamId: 20, awayTeamId: 10, homeGoals: 3, awayGoals: 1 }),
      match({ status: 'finished', homeGoals: 1, awayGoals: 1 }),
      match({ status: 'finished', homeTeamId: 20, awayTeamId: 10, homeGoals: 0, awayGoals: 4 }),
      match({ status: 'finished', homeGoals: 0, awayGoals: 1 }),
      match({ status: 'finished', homeGoals: 5, awayGoals: 0 }),
    ]
    expect(formStrip(results, 10)).toEqual(['W', 'L', 'D', 'W', 'L'])
  })
})

describe('nationalRoundPollMs', () => {
  const now = new Date('2026-11-07T19:50:00Z').getTime()
  it('polls fast while a match is live or about to start, slowly otherwise', () => {
    expect(nationalRoundPollMs([match({ status: 'live' })], now)).toBe(30_000)
    expect(nationalRoundPollMs([match({ kickoff: '2026-11-07T20:00:00Z' })], now)).toBe(30_000)
    expect(nationalRoundPollMs([match({ kickoff: '2026-11-08T20:00:00Z' })], now)).toBe(300_000)
    expect(nationalRoundPollMs(undefined, now)).toBe(300_000)
  })
})
