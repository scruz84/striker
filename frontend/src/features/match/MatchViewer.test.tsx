import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { EVENT_TYPE_GOAL, EVENT_TYPE_INJURY, EVENT_TYPE_RED_CARD, POSITION_GOALKEEPER, type Lineup, type MatchInstant, type Player } from '../../api/types'
import i18n from '../../i18n'
import { MatchViewer } from './MatchViewer'
import type { MatchView } from './matchViewModel'

// Which phase the (mocked) clock puts the match in, and what the (mocked) live socket reports.
const clock = vi.hoisted(() => ({
  past: false,
  mightStillBeLive: false,
  due: false,
  broadcastOver: false,
  lineupLocked: false,
  live: {} as Record<string, unknown>,
}))
const zoneProps = vi.hoisted(() => ({ calls: [] as Array<{ home: number; away: number; live: boolean }> }))
const ratingsProps = vi.hoisted(() => ({ calls: [] as Array<{ events: Array<{ eventType: number }>; instants: unknown[]; live?: boolean }> }))
const reportProps = vi.hoisted(() => ({ calls: [] as Array<{ firstHalfOver?: boolean }> }))

vi.mock('../../api/useLiveMatch', () => ({
  isPast: () => clock.past,
  mightStillBeLive: () => clock.mightStillBeLive,
  useIsDue: () => clock.due,
  useIsBroadcastOver: () => clock.broadcastOver,
  useIsLineupLocked: () => clock.lineupLocked,
  useLiveMatch: () => clock.live,
}))
vi.mock('./MatchScoreboard', () => ({
  Scoreboard: (props: { status: string; subtitle?: string; homeScorers?: Array<{ name: string }> }) => (
    <div data-testid="scoreboard">
      {props.status}|{props.subtitle}|{(props.homeScorers ?? []).map((s) => s.name).join(',')}
    </div>
  ),
}))
vi.mock('./MatchMinuteZoneComparison', () => ({
  MatchMinuteZoneComparison: (props: { homeGoalkeeperRating: number; awayGoalkeeperRating: number; live?: boolean }) => {
    zoneProps.calls.push({ home: props.homeGoalkeeperRating, away: props.awayGoalkeeperRating, live: !!props.live })
    return <div data-testid="zone-duel" />
  },
}))
vi.mock('./MatchEventsReport', () => ({
  MatchEventsReport: (props: { firstHalfOver?: boolean }) => {
    reportProps.calls.push(props)
    return <div data-testid="live-report" />
  },
}))
vi.mock('./MatchHeadToHeadPitch', () => ({ MatchHeadToHeadPitch: () => <div data-testid="pitch" /> }))
vi.mock('./MatchMinuteRatings', () => ({
  MatchMinuteRatings: (props: { events: Array<{ eventType: number }>; instants: unknown[]; live?: boolean }) => {
    ratingsProps.calls.push(props)
    return <div data-testid="live-ratings" />
  },
}))
vi.mock('./MatchReportTab', () => ({ MatchReportTab: () => <div data-testid="report-tab" /> }))
vi.mock('./MatchRatingsTab', () => ({ MatchRatingsTab: () => <div data-testid="ratings-tab" /> }))
vi.mock('./StadiumEntry', () => ({ StadiumEntry: () => <div data-testid="stadium-entry" /> }))
vi.mock('../../components/HeadToHeadCard', () => ({ HeadToHeadCard: () => <div data-testid="head-to-head" /> }))
vi.mock('../lineup/LineupLockedNotice', () => ({ LineupLockedNotice: () => <div>lineup locked</div> }))

function lineupWithKeeper(keeperId: number): Lineup {
  return { matchId: 1, teamId: 1, formation: 0, positions: [{ playerId: keeperId, position: POSITION_GOALKEEPER }], substitutions: [] }
}

function playersWithKeeper(keeperId: number, goalKeeper: number): Record<string, Player> {
  return { [String(keeperId)]: { goalKeeper } as Player }
}

const instant = { minute: 1 } as MatchInstant

function baseView(overrides: Partial<MatchView> = {}): MatchView {
  return {
    homeId: 1,
    homeName: 'Home',
    awayId: 2,
    awayName: 'Away',
    homeStadiumName: 'Estadio Norte',
    matchDate: '2026-10-21T20:00:00Z',
    played: false,
    homeGoals: 0,
    awayGoals: 0,
    events: null,
    matchInstants: null,
    homeLineup: lineupWithKeeper(10),
    awayLineup: lineupWithKeeper(20),
    homePlayers: playersWithKeeper(10, 70),
    awayPlayers: playersWithKeeper(20, 55),
    subtitleLabel: 'Amistoso',
    liveUrl: '/v1/friendlies/1/live',
    queryKey: ['friendlyMatch', '1'],
    ...overrides,
  }
}

function liveState(overrides: Record<string, unknown> = {}) {
  return {
    connected: true,
    homeGoals: 1,
    awayGoals: 0,
    minute: 12,
    events: [{ eventType: EVENT_TYPE_GOAL, minute: 12, teamOwner: 1, teamOwnerPlayerNames: ['Berlana'], setPiece: 1 }],
    instants: [instant],
    halftime: false,
    homeLineup: lineupWithKeeper(10),
    awayLineup: lineupWithKeeper(20),
    homePlayers: playersWithKeeper(10, 70),
    awayPlayers: playersWithKeeper(20, 55),
    complete: false,
    giveUp: false,
    ...overrides,
  }
}

function renderViewer(view: MatchView, extra: { lineupSection?: boolean; previewSection?: boolean; belowScoreboard?: boolean } = {}) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <MatchViewer
          view={view}
          lineupSection={extra.lineupSection ? <div>lineup editor</div> : undefined}
          previewSection={extra.previewSection ? <div>opponent preview</div> : undefined}
          belowScoreboard={extra.belowScoreboard ? <div>shootout line</div> : undefined}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('MatchViewer', () => {
  beforeEach(() => {
    zoneProps.calls.length = 0
    clock.past = false
    clock.mightStillBeLive = false
    clock.due = false
    clock.broadcastOver = false
    clock.lineupLocked = false
    clock.live = liveState()
  })

  describe('lineup section visibility', () => {
    it('shows the lineup editor to a participant before kickoff', () => {
      renderViewer(baseView(), { lineupSection: true })
      expect(screen.getByText('lineup editor')).toBeInTheDocument()
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('scheduled')
    })

    it('shows the opponent preview below the lineup editor to a participant before the lock', () => {
      renderViewer(baseView(), { lineupSection: true, previewSection: true })
      const editor = screen.getByText('lineup editor')
      const preview = screen.getByText('opponent preview')
      expect(editor.compareDocumentPosition(preview) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    })

    it('drops the opponent preview together with the editor once the lock window starts', () => {
      clock.lineupLocked = true
      renderViewer(baseView(), { lineupSection: true, previewSection: true })
      expect(screen.getByText('lineup locked')).toBeInTheDocument()
      expect(screen.queryByText('opponent preview')).not.toBeInTheDocument()
    })

    it('shows no opponent preview to someone who is not a participant', () => {
      renderViewer(baseView())
      expect(screen.queryByText('opponent preview')).not.toBeInTheDocument()
    })

    it('swaps the editor for the locked notice once the lock window starts', () => {
      clock.lineupLocked = true
      renderViewer(baseView(), { lineupSection: true })
      expect(screen.getByText('lineup locked')).toBeInTheDocument()
      expect(screen.queryByText('lineup editor')).not.toBeInTheDocument()
    })

    it('shows neither the editor nor the locked notice to someone who is not playing', () => {
      renderViewer(baseView())
      expect(screen.queryByText('lineup editor')).not.toBeInTheDocument()
      cleanup()
      clock.lineupLocked = true
      renderViewer(baseView())
      expect(screen.queryByText('lineup locked')).not.toBeInTheDocument()
    })

    it('shows neither once the match is live', () => {
      clock.past = true
      clock.mightStillBeLive = true
      clock.due = true
      renderViewer(baseView(), { lineupSection: true })
      expect(screen.queryByText('lineup editor')).not.toBeInTheDocument()
      expect(screen.queryByText('lineup locked')).not.toBeInTheDocument()
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('live')
    })
  })

  describe('live view', () => {
    beforeEach(() => {
      clock.past = true
      clock.mightStillBeLive = true
      clock.due = true
    })

    it('lists the live scorers under the scoreboard for any competition', () => {
      renderViewer(baseView())
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('Berlana')
    })

    it('shows the subtitle with the competition label', () => {
      renderViewer(baseView())
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('Amistoso')
    })

    it('rates live players from the instants, without the withheld red cards and injuries', () => {
      ratingsProps.calls.length = 0
      clock.live = liveState({
        events: [
          { eventType: EVENT_TYPE_GOAL, minute: 12, teamOwner: 1, teamOwnerPlayerNames: ['Berlana'], setPiece: 1 },
          { eventType: EVENT_TYPE_RED_CARD, minute: 20, teamOwner: 2, teamOwnerPlayers: [21] },
          { eventType: EVENT_TYPE_INJURY, minute: 25, teamOwner: 1, teamOwnerPlayers: [11] },
        ],
      })
      renderViewer(baseView())
      const call = ratingsProps.calls.at(-1)
      expect(call?.live).toBe(true)
      expect(call?.instants).toEqual([instant])
      expect(call?.events.map((e) => e.eventType)).toEqual([EVENT_TYPE_GOAL])
    })

    // match-viewer "Half-time line in the chronicle": live, the line waits for the break.
    it.each([
      [{ minute: 45, addedMinute: 1, halftime: false }, false],
      [{ minute: 45, halftime: true }, true],
      [{ minute: 46, halftime: false }, true],
    ])('tells the report whether the first half is over (%o)', (state, expected) => {
      reportProps.calls.length = 0
      clock.live = liveState(state)
      renderViewer(baseView())
      expect(reportProps.calls.at(-1)?.firstHalfOver).toBe(expected)
    })

    it('shows the stadium entry, the report and the head-to-head card', () => {
      renderViewer(baseView())
      expect(screen.getByTestId('stadium-entry')).toBeInTheDocument()
      expect(screen.getByTestId('live-report')).toBeInTheDocument()
      expect(screen.getByTestId('head-to-head')).toBeInTheDocument()
    })

    // A phone that suspends the page mid-match exhausts the socket's reconnect attempts while the
    // match (simulated, so `played`, before kickoff) is still being broadcast - that must not
    // drop the viewer into the finished report with the final score and every goal.
    it('stays live when the socket gives up while the broadcast is still running', () => {
      clock.live = liveState({ connected: false, giveUp: true })
      renderViewer(baseView({ played: true, homeGoals: 3, awayGoals: 1 }))
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('connecting')
      expect(screen.queryByTestId('report-tab')).not.toBeInTheDocument()
    })

    it('falls back to the finished view when the socket gives up after the broadcast window', () => {
      clock.live = liveState({ connected: false, giveUp: true })
      clock.broadcastOver = true
      renderViewer(baseView({ played: true, homeGoals: 3, awayGoals: 1 }))
      expect(screen.getByTestId('report-tab')).toBeInTheDocument()
    })
  })

  describe('zone duel goalkeepers', () => {
    it('counts the same starting goalkeepers live and after full time', () => {
      // Live: past kickoff, broadcast still running.
      clock.past = true
      clock.mightStillBeLive = true
      clock.due = true
      renderViewer(baseView())
      const live = zoneProps.calls.find((c) => c.live)
      expect(live).toEqual({ home: 70, away: 55, live: true })

      // Finished: long past, with the same lineups and stored instants.
      cleanup()
      zoneProps.calls.length = 0
      clock.mightStillBeLive = false
      renderViewer(baseView({ played: true, matchInstants: [instant] }))
      fireEvent.click(screen.getByRole('tab', { name: i18n.t('match.tabs.tactics') }))
      const finished = zoneProps.calls.find((c) => !c.live)
      expect(finished).toEqual({ home: 70, away: 55, live: false })
    })

    it('counts 0 live only until the lineups arrive', () => {
      clock.past = true
      clock.mightStillBeLive = true
      clock.due = true
      clock.live = liveState({ homeLineup: null, awayLineup: null, homePlayers: null, awayPlayers: null })
      renderViewer(baseView())
      expect(zoneProps.calls.find((c) => c.live)).toEqual({ home: 0, away: 0, live: true })
    })
  })

  describe('finished view', () => {
    beforeEach(() => {
      clock.past = true
    })

    it('renders the scoreboard, the extra under it, the tabs and the head-to-head card', () => {
      renderViewer(baseView({ played: true }), { belowScoreboard: true })
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('played')
      expect(screen.getByText('shootout line')).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: i18n.t('match.tabs.chronicle') })).toBeInTheDocument()
      expect(screen.getByTestId('report-tab')).toBeInTheDocument()
      expect(screen.getByTestId('head-to-head')).toBeInTheDocument()
    })

    it('shows the zone duel in the tactics tab only for a match with stored instants', () => {
      renderViewer(baseView({ played: true, matchInstants: [instant] }))
      fireEvent.click(screen.getByRole('tab', { name: i18n.t('match.tabs.tactics') }))
      expect(screen.getByTestId('zone-duel')).toBeInTheDocument()
    })

    it('shows only the pitch in the tactics tab for a match without stored instants', () => {
      renderViewer(baseView({ played: true, matchInstants: null }))
      fireEvent.click(screen.getByRole('tab', { name: i18n.t('match.tabs.tactics') }))
      expect(screen.queryByTestId('zone-duel')).not.toBeInTheDocument()
      expect(screen.getByTestId('pitch')).toBeInTheDocument()
    })
  })

  // openspec/specs/match-viewer "Mentality in the lineups".
  describe('mentality', () => {
    const attacking = (keeperId: number): Lineup => ({ ...lineupWithKeeper(keeperId), mentality: 1 })
    const defensive = (keeperId: number): Lineup => ({ ...lineupWithKeeper(keeperId), mentality: -1 })

    it('labels each side in the tactics tab of a finished match, in the viewer language', async () => {
      clock.past = true
      await i18n.changeLanguage('es')
      try {
        renderViewer(baseView({ played: true, homeLineup: attacking(10), awayLineup: defensive(20) }))
        fireEvent.click(screen.getByRole('tab', { name: i18n.t('match.tabs.tactics') }))
        expect(screen.getByText('Ofensiva')).toBeInTheDocument()
        expect(screen.getByText('Defensiva')).toBeInTheDocument()
      } finally {
        await i18n.changeLanguage('en')
      }
    })

    it('labels each side next to the live lineups', () => {
      clock.past = true
      clock.mightStillBeLive = true
      clock.due = true
      clock.live = liveState({ homeLineup: attacking(10), awayLineup: { ...lineupWithKeeper(20), mentality: 0 } })
      renderViewer(baseView())
      expect(screen.getByText('Attacking')).toBeInTheDocument()
      expect(screen.queryByText('Balanced')).toBeInTheDocument()
    })

    it('shows no mentality before kickoff', () => {
      renderViewer(baseView({ homeLineup: attacking(10), awayLineup: defensive(20) }))
      expect(screen.queryByText('Attacking')).not.toBeInTheDocument()
      expect(screen.queryByText('Defensive')).not.toBeInTheDocument()
    })

    it('shows no label for a match played before mentality existed', () => {
      clock.past = true
      renderViewer(baseView({ played: true }))
      fireEvent.click(screen.getByRole('tab', { name: i18n.t('match.tabs.tactics') }))
      expect(screen.queryByText('Balanced')).not.toBeInTheDocument()
    })
  })
})
