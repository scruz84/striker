import { beforeEach, describe, expect, it } from 'vitest'
import i18n from '../i18n'
import { liveScoreLabel } from './liveScoreLabel'

const t = i18n.t.bind(i18n)

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

describe('liveScoreLabel', () => {
  const base = { minute: 0, matchDate: new Date(2026, 10, 5, 18, 30).toISOString() }

  it('reads the minute, pulsing in the error colour, while live', () => {
    expect(liveScoreLabel({ ...base, status: 'live', minute: 63 }, t)).toEqual({ text: "63'", color: 'error', pulsing: true })
  })

  it('reads Half-time without pulsing', () => {
    expect(liveScoreLabel({ ...base, status: 'halftime', minute: 45 }, t)).toEqual({ text: 'Half-time', color: 'default', pulsing: false })
  })

  it('reads FT once the broadcast is over, Final in Spanish', async () => {
    expect(liveScoreLabel({ ...base, status: 'finished', minute: 90 }, t).text).toBe('FT')
    await i18n.changeLanguage('es')
    expect(liveScoreLabel({ ...base, status: 'finished', minute: 90 }, t).text).toBe('Final')
  })

  it('shows the kickoff time while pending', () => {
    expect(liveScoreLabel({ ...base, status: 'pending' }, t)).toEqual({ text: '18:30', color: 'default', pulsing: false })
  })
})
