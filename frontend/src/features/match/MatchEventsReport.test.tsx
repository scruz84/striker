import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_ADDED_TIME,
  EVENT_TYPE_GOAL,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  EVENT_TYPE_NO_GOAL,
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
    ['en', 'Default result'],
    ['es', 'Resultado por defecto'],
  ])('narrates the default result in %s', async (language, label) => {
    await i18n.changeLanguage(language)
    const { container } = renderReport()

    // One of the default-result lines, filled in with the two sides.
    const lines = (i18n.t('match.commentary.defaultResult', { returnObjects: true }) as string[]).map((line) =>
      line.replace('{{team}}', 'Real Norte').replace('{{opponent}}', 'Atlético Sur'),
    )
    expect(lines.some((line) => container.textContent?.includes(line))).toBe(true)
    expect(container.textContent).toContain('Real Norte')
    expect(container.textContent).toContain('Atlético Sur')
    expect(screen.getByLabelText(label)).toHaveTextContent('🤝')
  })
})

// openspec/specs/match-viewer "Mentality change in the chronicle".
describe('MatchEventsReport mentality change', () => {
  const change: MatchEvent = { ...defaultResult, eventType: EVENT_TYPE_MENTALITY_CHANGE, minute: 70, teamOwner: 2, otherTeam: 1, mentality: 1 }

  it.each(['en', 'es'])('narrates the away side moving to an attacking approach at its minute in %s', async (language) => {
    await i18n.changeLanguage(language)
    const { container } = render(
      <MemoryRouter>
        <MatchEventsReport events={[change]} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" />
      </MemoryRouter>,
    )
    const lines = (i18n.t('match.commentary.mentalityChange.attacking', { returnObjects: true }) as string[]).map((line) =>
      line.replace('{{team}}', 'Atlético Sur'),
    )
    expect(lines.some((line) => container.textContent?.includes(line))).toBe(true)
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


// openspec match-viewer "Half-time line in the chronicle" and match-commentary "Full-time line by
// result" / "No repeated lines within a match".
describe('MatchEventsReport commentary bookends', () => {
  const base = { ...defaultResult, teamOwnerPlayers: [7], teamOwnerPlayerNames: ['Berlana'] }
  const goal = (minute: number, homeGoals: number, awayGoals: number, side: 1 | 2 = 1, player = 7): MatchEvent => ({
    ...base,
    eventType: EVENT_TYPE_GOAL,
    minute,
    teamOwner: side,
    otherTeam: side === 1 ? 2 : 1,
    teamOwnerPlayers: [player],
    teamOwnerPlayerNames: [`Player${player}`],
    homeGoals,
    awayGoals,
  })
  const miss = (minute: number, homeGoals: number, awayGoals: number, player: number): MatchEvent => ({
    ...goal(minute, homeGoals, awayGoals, 1, player),
    eventType: EVENT_TYPE_NO_GOAL,
  })

  // The case's lines, filled in the way the report fills them.
  function linesOf(key: string, score: string): string[] {
    const pool = i18n.t(key, { returnObjects: true }) as string[]
    return pool.map((line) =>
      line
        .replaceAll('{{home}}', 'Real Norte')
        .replaceAll('{{away}}', 'Atlético Sur')
        .replaceAll('{{score}}', score)
        .replaceAll('{{winner}}', 'Atlético Sur')
        .replaceAll('{{loser}}', 'Real Norte'),
    )
  }

  function report(events: MatchEvent[], props: { final?: boolean; firstHalfOver?: boolean } = {}) {
    return render(
      <MemoryRouter>
        <MatchEventsReport events={events} homeId={1} homeName="Real Norte" awayId={2} awayName="Atlético Sur" {...props} />
      </MemoryRouter>,
    )
  }

  function foundLine(text: string, lines: string[]): string | undefined {
    return lines.find((line) => text.includes(line))
  }

  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('puts the half-time line between the halves of a finished match', () => {
    const { container } = report([goal(30, 1, 0, 1, 7), goal(70, 2, 0, 1, 8)], { final: true })
    const text = container.textContent ?? ''
    const line = foundLine(text, linesOf('match.report.halfTime.homeLeads', '1-0'))
    expect(line).toBeDefined()
    expect(text.indexOf('Player7')).toBeLessThan(text.indexOf(line!))
    expect(text.indexOf(line!)).toBeLessThan(text.indexOf('Player8'))
  })

  it('shows no half-time line live before the break, and shows it during the break', () => {
    const events = [goal(30, 1, 0)]
    const before = report(events, { firstHalfOver: false })
    expect(foundLine(before.container.textContent ?? '', linesOf('match.report.halfTime.homeLeads', '1-0'))).toBeUndefined()
    before.unmount()
    const during = report(events, { firstHalfOver: true })
    expect(foundLine(during.container.textContent ?? '', linesOf('match.report.halfTime.homeLeads', '1-0'))).toBeDefined()
  })

  it('has no half-time line for a match abandoned in the first half', () => {
    const abandoned: MatchEvent = { ...base, eventType: EVENT_TYPE_MATCH_ABANDONED, minute: 30, teamOwner: 1, otherTeam: 2 }
    const { container } = report([abandoned], { final: true })
    const text = container.textContent ?? ''
    for (const key of ['goalless', 'level', 'homeLeads', 'awayLeads']) {
      expect(foundLine(text, linesOf(`match.report.halfTime.${key}`, '0-0'))).toBeUndefined()
    }
    expect(text).toContain(i18n.t('match.report.fullTime'))
  })

  it('closes a 0-4 with a big-win line naming the away side', () => {
    const events = [goal(10, 0, 1, 2, 21), goal(20, 0, 2, 2, 22), goal(50, 0, 3, 2, 23), goal(80, 0, 4, 2, 24)]
    const { container } = report(events, { final: true })
    expect(foundLine(container.textContent ?? '', linesOf('match.report.fullTimeResult.bigWin', '0-4'))).toBeDefined()
  })

  it('keeps the neutral full-time line for a default result', () => {
    const { container } = report([defaultResult], { final: true })
    expect(container.textContent).toContain(i18n.t('match.report.fullTime'))
  })

  it('opens with the same kickoff line live and finished', () => {
    const kickoffs = linesOf('match.report.kickoff', '')
    const live = report([goal(30, 1, 0)])
    const liveLine = foundLine(live.container.textContent ?? '', kickoffs)
    live.unmount()
    const finished = report([goal(30, 1, 0)], { final: true })
    expect(liveLine).toBeDefined()
    expect(foundLine(finished.container.textContent ?? '', kickoffs)).toBe(liveLine)
  })

  it('reads every key event the same with the full commentary on or off', () => {
    const events = [miss(5, 0, 0, 30), goal(12, 1, 0, 1, 31), miss(20, 1, 0, 32), miss(33, 1, 0, 33), goal(60, 2, 0, 1, 34)]
    const { container } = report(events, { final: true })
    const rowText = (name: string) =>
      [...container.querySelectorAll('p, span, div')].find((node) => node.childElementCount > 0 && node.textContent?.includes(name) && node.textContent.length < 400)
        ?.textContent
    const collapsed = [rowText('Player31'), rowText('Player34')]
    fireEvent.click(screen.getByRole('button', { name: i18n.t('match.report.fullCommentary') }))
    expect(container.textContent).toContain('Player30')
    expect([rowText('Player31'), rowText('Player34')]).toEqual(collapsed)
  })
})
