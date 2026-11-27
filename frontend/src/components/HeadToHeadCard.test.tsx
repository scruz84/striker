import { cleanup, render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { HeadToHead, HeadToHeadMeeting } from '../api/types'
import i18n from '../i18n'
import { formatDate } from '../utils/formatDate'
import { HeadToHeadCard } from './HeadToHeadCard'

const mocks = vi.hoisted(() => ({ data: undefined as HeadToHead | undefined }))
vi.mock('../api/hooks', () => ({
  useHeadToHead: () => ({ data: mocks.data, isLoading: false }),
}))

const ALPHA = 1
const BETA = 2

function meeting(overrides: Partial<HeadToHeadMeeting>): HeadToHeadMeeting {
  return {
    matchId: 10,
    season: 1,
    matchDate: new Date(2026, 10, 20, 18, 0).toISOString(),
    homeTeamId: ALPHA,
    awayTeamId: BETA,
    homeGoals: 0,
    awayGoals: 0,
    ...overrides,
  }
}

const renderCard = () =>
  render(
    <MemoryRouter>
      <HeadToHeadCard teamAId={ALPHA} teamAName="Alpha United" teamBId={BETA} teamBName="Beta City" />
    </MemoryRouter>,
  )

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(() => {
  cleanup()
  mocks.data = undefined
})

describe('HeadToHeadCard', () => {
  it('names each team once and summarizes wins, goals and draws by side', () => {
    mocks.data = {
      teamAId: ALPHA,
      teamBId: BETA,
      totalMatches: 2,
      teamAWins: 1,
      teamBWins: 0,
      draws: 1,
      teamAGoals: 3,
      teamBGoals: 1,
      meetings: [
        meeting({ matchId: 11, homeGoals: 2, awayGoals: 0 }),
        meeting({ matchId: 12, homeTeamId: BETA, awayTeamId: ALPHA, homeGoals: 1, awayGoals: 1 }),
      ],
    }
    renderCard()
    expect(screen.getAllByText('Alpha United')).toHaveLength(1)
    expect(screen.getAllByText('Beta City')).toHaveLength(1)
    expect(screen.getByText('Wins')).toBeTruthy()
    expect(screen.getByText('Goals')).toBeTruthy()
    expect(screen.getByText('1 draw')).toBeTruthy()
  })

  it('keeps the home side on the left, links to the match and bolds only the winner', () => {
    const date = new Date(2026, 10, 20, 18, 0).toISOString()
    mocks.data = {
      teamAId: ALPHA,
      teamBId: BETA,
      totalMatches: 2,
      teamAWins: 1,
      teamBWins: 0,
      draws: 1,
      teamAGoals: 5,
      teamBGoals: 3,
      meetings: [
        // Team A away: the row must still read home (Beta) first.
        meeting({ matchId: 21, matchDate: date, homeTeamId: BETA, awayTeamId: ALPHA, homeGoals: 2, awayGoals: 4 }),
        meeting({ matchId: 22, matchDate: date, homeGoals: 1, awayGoals: 1 }),
      ],
    }
    renderCard()

    const awayWin = screen.getByRole('link', { name: `Beta City 2 - 4 Alpha United, ${formatDate(date)}` })
    expect(awayWin.getAttribute('href')).toBe('/match/21')
    const [homeGoals, , awayGoals] = within(awayWin).getAllByText(/^\d+$|^–$/)
    expect(homeGoals.textContent).toBe('2')
    expect(awayGoals.textContent).toBe('4')
    expect(getComputedStyle(awayGoals).fontWeight).toBe('800')
    expect(getComputedStyle(homeGoals).fontWeight).toBe('400')

    const draw = screen.getByRole('link', { name: `Alpha United 1 - 1 Beta City, ${formatDate(date)}` })
    expect(draw.getAttribute('href')).toBe('/match/22')
    for (const goals of within(draw).getAllByText('1')) {
      expect(getComputedStyle(goals).fontWeight).toBe('400')
    }
  })

  it('says so when the teams have never met', () => {
    mocks.data = { teamAId: ALPHA, teamBId: BETA, totalMatches: 0, teamAWins: 0, teamBWins: 0, draws: 0, teamAGoals: 0, teamBGoals: 0, meetings: [] }
    renderCard()
    expect(screen.getByText("These teams haven't played each other yet.")).toBeTruthy()
  })
})
