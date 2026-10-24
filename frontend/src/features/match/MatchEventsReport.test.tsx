import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { EVENT_TYPE_MATCH_DEFAULT_RESULT, type MatchEvent } from '../../api/types'
import i18n from '../../i18n'
import { MatchEventsReport } from './MatchEventsReport'

// The only event a fixture settled by default stores: 0-0 at minute 0, owned by the home side.
const defaultResult: MatchEvent = {
  eventType: EVENT_TYPE_MATCH_DEFAULT_RESULT,
  minute: 0,
  teamOwner: 1,
  teamOwnerPlayers: null,
  teamOwnerPlayerNames: null,
  otherTeam: 2,
  otherTeamPlayers: null,
  otherTeamPlayerNames: null,
  homeGoals: 0,
  awayGoals: 0,
}

function renderReport() {
  return render(
    <MemoryRouter>
      <MatchEventsReport events={[defaultResult]} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" final />
    </MemoryRouter>,
  )
}

describe('MatchEventsReport default result', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it.each([
    ['en', 'recorded as a 0-0 default result between', 'Default result'],
    ['es', 'se registra como un 0-0 por defecto entre', 'Resultado por defecto'],
  ])('narrates the default result in %s', async (language, sentence, label) => {
    await i18n.changeLanguage(language)
    const { container } = renderReport()

    expect(container.textContent).toContain(sentence)
    expect(container.textContent).toContain('Real Norte')
    expect(container.textContent).toContain('Atlético Sur')
    expect(screen.getByLabelText(label)).toHaveTextContent('🤝')
  })
})
