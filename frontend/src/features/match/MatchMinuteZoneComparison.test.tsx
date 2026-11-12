import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import type { MatchInstant, TeamQualification } from '../../api/types'
import i18n from '../../i18n'
import { matchTimeline } from '../../utils/matchMinute'
import { MatchMinuteZoneComparison } from './MatchMinuteZoneComparison'

const qua = (midfield: number): TeamQualification => ({
  goalkeeper: 0,
  leftDefense: 10,
  rightDefense: 10,
  defense: 10,
  leftMidfield: 10,
  rightMidfield: 10,
  midfield,
  leftAttack: 10,
  rightAttach: 10,
  attack: 10,
})

// match-viewer "Scrubbers cover added minutes": the zone duel steps through 45+1 and 45+2 between
// 45 and 46, showing each added minute's own snapshot.
describe('MatchMinuteZoneComparison with added time', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('steps into the first half’s added time', () => {
    const instants = [
      ...Array.from({ length: 45 }, (_, i) => ({ minute: i + 1, homeQua: qua(40), awayQua: qua(40) })),
      { minute: 45, addedMinute: 1, homeQua: qua(41), awayQua: qua(40) },
      { minute: 45, addedMinute: 2, homeQua: qua(42), awayQua: qua(40) },
      { minute: 46, homeQua: qua(43), awayQua: qua(40) },
    ] as unknown as MatchInstant[]
    render(
      <MemoryRouter>
        <MatchMinuteZoneComparison
          homeId={1}
          homeName="Home"
          awayId={2}
          awayName="Away"
          homeGoalkeeperRating={0}
          awayGoalkeeperRating={0}
          matchInstants={instants}
          timeline={matchTimeline(instants)}
        />
      </MemoryRouter>,
    )
    const previous = () => fireEvent.click(screen.getByRole('button', { name: i18n.t('match.ratings.previousMinute') }))
    expect(screen.getAllByText("46'").length).toBeGreaterThan(0)
    previous()
    expect(screen.getAllByText("45+2'").length).toBeGreaterThan(0)
    expect(screen.getByText(i18n.t('match.positionBattle.atMinute', { minute: '45+2' }))).toBeInTheDocument()
    previous()
    expect(screen.getAllByText("45+1'").length).toBeGreaterThan(0)
    previous()
    expect(screen.getAllByText("45'").length).toBeGreaterThan(0)
  })
})
