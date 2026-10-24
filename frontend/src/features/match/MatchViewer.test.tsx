import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { EVENT_TYPE_GOAL, POSITION_GOALKEEPER, type Lineup, type MatchInstant, type Player } from '../../api/types'
import i18n from '../../i18n'
import { MatchViewer } from './MatchViewer'
import type { MatchView } from './matchViewModel'

// Which phase the (mocked) clock puts the match in, and what the (mocked) live socket reports.
const clock = vi.hoisted(() => ({
  past: false,
  mightStillBeLive: false,
  due: false,
  lineupLocked: false,
  live: {} as Record<string, unknown>,
}))
const zoneProps = vi.hoisted(() => ({ calls: [] as Array<{ home: number; away: number; live: boolean }> }))

vi.mock('../../api/useLiveMatch', () => ({
  isPast: () => clock.past,
  mightStillBeLive: () => clock.mightStillBeLive,
  useIsDue: () => clock.due,
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
vi.mock('./MatchEventsReport', () => ({ MatchEventsReport: () => <div data-testid="live-report" /> }))
vi.mock('./MatchHeadToHeadPitch', () => ({ MatchHeadToHeadPitch: () => <div data-testid="pitch" /> }))
vi.mock('./MatchMinuteRatings', () => ({ MatchMinuteRatings: () => <div data-testid="live-ratings" /> }))
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

function renderViewer(view: MatchView, extra: { lineupSection?: boolean; belowScoreboard?: boolean } = {}) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <MatchViewer
          view={view}
          lineupSection={extra.lineupSection ? <div>lineup editor</div> : undefined}
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
    clock.lineupLocked = false
    clock.live = liveState()
  })

  describe('lineup section visibility', () => {
    it('shows the lineup editor to a participant before kickoff', () => {
      renderViewer(baseView(), { lineupSection: true })
      expect(screen.getByText('lineup editor')).toBeInTheDocument()
      expect(screen.getByTestId('scoreboard')).toHaveTextContent('scheduled')
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

    it('shows the stadium entry, the report and the head-to-head card', () => {
      renderViewer(baseView())
      expect(screen.getByTestId('stadium-entry')).toBeInTheDocument()
      expect(screen.getByTestId('live-report')).toBeInTheDocument()
      expect(screen.getByTestId('head-to-head')).toBeInTheDocument()
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
})
