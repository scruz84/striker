import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_SUBSTITUTION,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_STRIKER,
  type Lineup,
  type MatchEvent,
  type MatchInstant,
  type Player,
  type PlayerRating,
} from '../../api/types'
import i18n from '../../i18n'
import { MatchMinuteRatings } from './MatchMinuteRatings'
import { RatingBreakdown, TeamRatingsColumn } from './TeamRatingsColumn'

function player(id: number, lastName: string): Player {
  return { id, firstName: 'P', lastName, birthCountry: 'es' } as Player
}

function rating(overrides: Partial<PlayerRating> & { playerId: number }): PlayerRating {
  return {
    teamId: 1,
    rated: true,
    rating: 6,
    minutes: 90,
    position: POSITION_STRIKER,
    zone: 0,
    goals: 0,
    cards: 0,
    injury: 0,
    result: 0,
    ...overrides,
  }
}

const lineup: Lineup = {
  matchId: 1,
  teamId: 1,
  formation: 0,
  positions: [
    { playerId: 1, position: POSITION_GOALKEEPER },
    { playerId: 6, position: POSITION_LATERAL_MIDFIELD_LEFT },
  ],
  substitutions: [],
}

const players: Record<string, Player> = {
  '1': player(1, 'Keeper'),
  '6': player(6, 'Winger'),
  '16': player(16, 'Sub'),
  '26': player(26, 'Late'),
}

function renderColumn(ratings: PlayerRating[]) {
  render(
    <MemoryRouter>
      <TeamRatingsColumn teamId={1} title="Home" startingLineup={lineup} players={players} ratings={ratings} />
    </MemoryRouter>,
  )
}

describe('TeamRatingsColumn', () => {
  it('lists a replaced starter and his substitute separately, starters first', () => {
    renderColumn([
      rating({ playerId: 16, rating: 6.4, minutes: 30, cameOn: 60, position: POSITION_LATERAL_MIDFIELD_LEFT }),
      rating({ playerId: 6, rating: 7.1, minutes: 60, wentOff: 60, position: POSITION_LATERAL_MIDFIELD_LEFT, own: 58, rival: 45 }),
      rating({ playerId: 1, rating: 6.2, position: POSITION_GOALKEEPER }),
    ])
    const names = screen.getAllByRole('link').map((link) => link.textContent)
    expect(names.filter((name) => name?.startsWith('P '))).toEqual(['P Keeper', 'P Winger', 'P Sub'])
    expect(screen.getByText(/↓ 60'/)).toBeInTheDocument()
    expect(screen.getByText(/↑ 60'/)).toBeInTheDocument()
    expect(screen.getByText('7.1')).toBeInTheDocument()
    expect(screen.getByText('6.4')).toBeInTheDocument()
    // The winger's zone contribution against the rival's.
    expect(screen.getByText('58')).toBeInTheDocument()
    expect(screen.getByText(/vs 45/)).toBeInTheDocument()
  })

  it('shows an appearance under 10 minutes as unrated', () => {
    renderColumn([rating({ playerId: 26, rated: false, rating: 6, minutes: 6, cameOn: 85 })])
    expect(screen.getByText(i18n.t('match.ratings.unrated'))).toBeInTheDocument()
    expect(screen.getByText(/↑ 85'/)).toBeInTheDocument()
    expect(screen.queryByText('6.0')).not.toBeInTheDocument()
  })
})

describe('RatingBreakdown', () => {
  it('lists every non-zero component to one decimal', () => {
    render(<RatingBreakdown rating={rating({ playerId: 9, rating: 9.6, zone: 0.5, goals: 2.8, result: 0.3 })} />)
    const t = i18n.t.bind(i18n)
    expect(
      screen.getByText(
        `${t('match.ratings.breakdown.base')} 6.0 · ${t('match.ratings.breakdown.zone')} +0.5 · ${t('match.ratings.breakdown.goals')} +2.8 · ${t('match.ratings.breakdown.result')} +0.3`,
      ),
    ).toBeInTheDocument()
  })
})

describe('MatchMinuteRatings', () => {
  const awayLineup: Lineup = { matchId: 1, teamId: 2, formation: 0, positions: [{ playerId: 101, position: POSITION_GOALKEEPER }], substitutions: [] }
  const instants: MatchInstant[] = Array.from({ length: 90 }, (_, i) => ({
    minute: i + 1,
    playerContributions: { h: { '6': { p: POSITION_STRIKER, o: 50, r: 50 } }, a: { '101': { p: POSITION_GOALKEEPER, o: 50, r: 50 } } },
  })) as unknown as MatchInstant[]
  const events: MatchEvent[] = [
    {
      eventType: EVENT_TYPE_GOAL,
      minute: 70,
      teamOwner: 1,
      teamOwnerPlayers: [6],
      teamOwnerPlayerNames: ['Winger'],
      otherTeam: 2,
      otherTeamPlayers: null,
      otherTeamPlayerNames: null,
      homeGoals: 1,
      awayGoals: 0,
    },
  ]
  // A stored figure no recomputation would give, so the test can tell which one is shown.
  const finalRatings = [rating({ playerId: 6, rating: 7.9, goals: 1.4, result: 0.3 }), rating({ playerId: 101, teamId: 2, rating: 5.7, result: -0.3 })]

  function renderRatings() {
    render(
      <MemoryRouter>
        <MatchMinuteRatings
          homeId={1}
          homeName="Home"
          awayId={2}
          awayName="Away"
          homeLineup={{ ...lineup, positions: [{ playerId: 6, position: POSITION_STRIKER }] }}
          awayLineup={awayLineup}
          homePlayers={players}
          awayPlayers={{ '101': player(101, 'Rival') }}
          events={events}
          instants={instants}
          currentMinute={90}
          finalRatings={finalRatings}
        />
      </MemoryRouter>,
    )
  }

  it('shows the stored ratings at the final minute and rebuilds them for an earlier one', () => {
    renderRatings()
    expect(screen.getByText('7.9')).toBeInTheDocument()

    // One minute back the rebuilt figure shows instead: 6 + 1.4 goal + 0.3 * 89/90 result = 7.7.
    fireEvent.click(screen.getByRole('button', { name: i18n.t('match.ratings.previousMinute') }))
    expect(screen.queryByText('7.9')).not.toBeInTheDocument()
    expect(screen.getByText('7.7')).toBeInTheDocument()
  })

  it('reads the starting XI back from a substituted lineup when there are no contributions', () => {
    const withSub: MatchEvent[] = [
      ...events,
      { ...events[0], eventType: EVENT_TYPE_SUBSTITUTION, minute: 80, teamOwnerPlayers: [6, 16], teamOwnerPlayerNames: ['Winger', 'Sub'] },
    ]
    render(
      <MemoryRouter>
        <MatchMinuteRatings
          homeId={1}
          homeName="Home"
          awayId={2}
          awayName="Away"
          homeLineup={{ ...lineup, positions: [{ playerId: 16, position: POSITION_STRIKER }] }}
          awayLineup={awayLineup}
          homePlayers={players}
          awayPlayers={{ '101': player(101, 'Rival') }}
          events={withSub}
          instants={[]}
          currentMinute={90}
        />
      </MemoryRouter>,
    )
    // No contributions: the fallback reads the starting XI back from the substituted lineup.
    expect(screen.getByText('P Winger')).toBeInTheDocument()
    expect(screen.getByText('P Sub')).toBeInTheDocument()
  })
})
