import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_ADDED_TIME,
  EVENT_TYPE_GOAL,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  type MatchEvent,
} from '../../api/types'
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

// openspec/specs/match-viewer "Mentality change in the chronicle".
describe('MatchEventsReport mentality change', () => {
  const change: MatchEvent = { ...defaultResult, eventType: EVENT_TYPE_MENTALITY_CHANGE, minute: 70, teamOwner: 2, otherTeam: 1, mentality: 1 }

  it.each([
    ['en', 'attacking'],
    ['es', 'ataque'],
  ])('narrates the away side moving to an attacking approach at its minute in %s', async (language, word) => {
    await i18n.changeLanguage(language)
    const { container } = render(
      <MemoryRouter>
        <MatchEventsReport events={[change]} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" />
      </MemoryRouter>,
    )
    expect(container.textContent).toContain('Atlético Sur')
    expect(container.textContent?.toLowerCase()).toContain(word)
    expect(container.textContent).toContain('70')
  })

  it('shows nothing of the change while the replay has not reached its minute', () => {
    // The live view only holds the events broadcast so far (see BroadcastMatchLive) - at minute 50
    // the minute-70 change hasn't arrived.
    const { container } = render(
      <MemoryRouter>
        <MatchEventsReport events={[]} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" />
      </MemoryRouter>,
    )
    expect(container.textContent?.toLowerCase()).not.toContain('attacking')
  })
})

// openspec/specs/match-viewer "Added-time board in the chronicle" and "Minute labels with added
// time".
describe('MatchEventsReport added time', () => {
  const goal = (minute: number, addedMinute: number, homeGoals: number): MatchEvent => ({
    ...defaultResult,
    eventType: EVENT_TYPE_GOAL,
    minute,
    addedMinute,
    teamOwnerPlayers: [7],
    teamOwnerPlayerNames: ['Berlana'],
    homeGoals,
  })
  const board = (minute: number, addedTime: number): MatchEvent => ({ ...defaultResult, eventType: EVENT_TYPE_ADDED_TIME, minute, addedTime })

  function minutes(container: HTMLElement): string[] {
    return Array.from(container.querySelectorAll('p'))
      .map((p) => p.textContent ?? '')
      .filter((text) => /^\d+(\+\d+)?'$/.test(text))
  }

  it.each([
    ['en', '+5 minutes', '+1 minute', 'Added time'],
    ['es', '+5 minutos', '+1 minuto', 'Tiempo añadido'],
  ])('shows the board at its minute, after the minute\'s other events, in %s', async (language, five, one, label) => {
    await i18n.changeLanguage(language)
    const events = [goal(46, 0, 3), board(90, 5), goal(45, 2, 2), board(45, 1), goal(45, 0, 1), goal(90, 3, 4)]
    const { container } = render(
      <MemoryRouter>
        <MatchEventsReport events={events} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" />
      </MemoryRouter>,
    )
    expect(container.textContent).toContain(five)
    expect(container.textContent).toContain(one)
    expect(screen.getAllByLabelText(label)).toHaveLength(2)
    expect(minutes(container)).toEqual(["45'", "45'", "45+2'", "46'", "90'", "90+3'"])
  })

  it('leaves the board out of a live chronicle that has not reached it yet', async () => {
    await i18n.changeLanguage('en')
    // A live viewer at minute 88 has only the events published so far: no second-half board.
    const { container } = render(
      <MemoryRouter>
        <MatchEventsReport events={[goal(30, 0, 1)]} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" />
      </MemoryRouter>,
    )
    expect(container.textContent).not.toContain('minutes')
    expect(minutes(container)).toEqual(["30'"])
  })
})

