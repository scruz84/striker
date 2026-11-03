import { render as baseRender, screen } from '@testing-library/react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { FORMATION_BALANCED, POSITION_DEFENSE, POSITION_GOALKEEPER, type Lineup, type Player, type PlayerRating } from '../../api/types'
import i18n from '../../i18n'
import { MatchHeadToHeadPitch } from './MatchHeadToHeadPitch'
import { TeamRatingsColumn } from './TeamRatingsColumn'

// The captain's armband in the match viewer (openspec match-viewer "Captain's armband").

const testTheme = createTheme({ palette: { rating: ['#ccc', '#bbb', '#aaa', '#999', '#888'] } })
const render = (ui: ReactElement) => baseRender(<ThemeProvider theme={testTheme}>{ui}</ThemeProvider>)

function side(teamId: number, firstId: number, captainId?: number): { lineup: Lineup; players: Record<string, Player> } {
  const lineup: Lineup = {
    matchId: 1,
    teamId,
    formation: FORMATION_BALANCED,
    positions: [
      { playerId: firstId, position: POSITION_GOALKEEPER },
      { playerId: firstId + 1, position: POSITION_DEFENSE },
    ],
    substitutions: [],
    captainId,
  }
  const players: Record<string, Player> = {}
  for (const id of [firstId, firstId + 1]) {
    players[String(id)] = { id, firstName: 'P', lastName: `L${id}`, age: 25, birthCountry: 'es' } as Player
  }
  return { lineup, players }
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

describe('captain armband', () => {
  it('marks the home captain on a finished match pitch, and nobody on a side without one', () => {
    const home = side(1, 10, 11)
    const away = side(2, 20)
    render(
      <MemoryRouter>
        <MatchHeadToHeadPitch homeLineup={home.lineup} homePlayers={home.players} awayLineup={away.lineup} awayPlayers={away.players} />
      </MemoryRouter>,
    )
    const armbands = screen.getAllByRole('img', { name: 'Captain' })
    expect(armbands).toHaveLength(1)
    expect(armbands[0].closest('a')).toHaveAttribute('href', '/player/11')
  })

  it('shows no armband for a match played without captains', () => {
    const home = side(1, 10)
    const away = side(2, 20)
    render(
      <MemoryRouter>
        <MatchHeadToHeadPitch homeLineup={home.lineup} homePlayers={home.players} awayLineup={away.lineup} awayPlayers={away.players} />
      </MemoryRouter>,
    )
    expect(screen.queryByRole('img', { name: 'Captain' })).toBeNull()
  })

  it('marks the captain in the ratings list, in the viewer language', async () => {
    await i18n.changeLanguage('es')
    const home = side(1, 10, 10)
    const ratings = [10, 11].map((playerId) => ({ playerId, teamId: 1, position: POSITION_DEFENSE, minutes: 90, rated: true, rating: 6.5 }) as PlayerRating)
    render(
      <MemoryRouter>
        <TeamRatingsColumn teamId={1} title="Home" startingLineup={home.lineup} players={home.players} ratings={ratings} />
      </MemoryRouter>,
    )
    expect(screen.getAllByRole('img', { name: 'Capitán' })).toHaveLength(1)
  })
})
