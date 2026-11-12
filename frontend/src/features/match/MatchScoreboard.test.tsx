import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { SET_PIECE_PENALTY } from '../../api/types'
import i18n from '../../i18n'
import { Scoreboard, type ScoreboardProps } from './MatchScoreboard'

const base: ScoreboardProps = {
  homeId: 1,
  homeName: 'Real Norte',
  awayId: 2,
  awayName: 'Atlético Sur',
  homeGoals: 1,
  awayGoals: 0,
  status: 'live',
}

function renderScoreboard(props: Partial<ScoreboardProps>) {
  return render(
    <MemoryRouter>
      <Scoreboard {...base} {...props} />
    </MemoryRouter>,
  )
}

// openspec/specs/match-viewer "Minute labels with added time".
describe('Scoreboard added time', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('marks a goal in first-half added time, with its set piece', () => {
    renderScoreboard({ homeScorers: [{ name: 'Berlana', goals: [{ minute: 45, addedMinute: 2, setPiece: SET_PIECE_PENALTY }] }] })
    expect(screen.getByText("Berlana 45+2' (pen.)")).toBeInTheDocument()
  })

  it('shows the live clock in added time', () => {
    renderScoreboard({ minute: 90, addedMinute: 3 })
    expect(screen.getByText("90+3'")).toBeInTheDocument()
  })

  it('reads an old match exactly as before', () => {
    renderScoreboard({ minute: 63, homeScorers: [{ name: 'Berlana', goals: [{ minute: 16, addedMinute: 0, setPiece: 0 }] }] })
    expect(screen.getByText("63'")).toBeInTheDocument()
    expect(screen.getByText("Berlana 16'")).toBeInTheDocument()
  })
})
