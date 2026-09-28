import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  type MatchEvent,
} from '../../api/types'
import { commentaryFor } from '../../utils/matchCommentary'

const NOTABLE_EVENT_TYPES: number[] = [
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_YELLOW_CARD,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_SUBSTITUTION,
]

interface TeamStats {
  goals: number
  missedChances: number
  yellowCards: number
  redCards: number
  injuries: number
}

function emptyStats(): TeamStats {
  return { goals: 0, missedChances: 0, yellowCards: 0, redCards: 0, injuries: 0 }
}

// Tallies each team's occasions (split into the ones that became a goal and the ones that
// didn't), cards, and injuries from the same event list the timeline below narrates - a quick
// scoreline-style summary of the match rather than a play-by-play.
function statsByTeam(events: MatchEvent[], homeId: number): { home: TeamStats; away: TeamStats } {
  const home = emptyStats()
  const away = emptyStats()
  for (const event of events) {
    const stats = event.teamOwner === homeId ? home : away
    switch (event.eventType) {
      case EVENT_TYPE_GOAL:
        stats.goals++
        break
      case EVENT_TYPE_NO_GOAL:
        stats.missedChances++
        break
      case EVENT_TYPE_YELLOW_CARD:
        stats.yellowCards++
        break
      case EVENT_TYPE_RED_CARD:
        stats.redCards++
        break
      case EVENT_TYPE_INJURY:
        stats.injuries++
        break
    }
  }
  return { home, away }
}

export interface MatchEventsReportProps {
  events: MatchEvent[]
  homeId: number
  homeName: string
  awayName: string
  // Match-flavor fields (see backend common.UpdateMatchFlavor) - all four are required together
  // for the intro paragraph to render (see hasFlavor below); a live broadcast simply doesn't have
  // any of these yet, so the intro is skipped exactly like a flavor-less cup tie/friendly already
  // does.
  weather?: string
  attendance?: number
  refereeName?: string
  assistant1Name?: string
  assistant2Name?: string
  // True once the match has actually finished - adds the closing "full time" line. Omitted for a
  // still-broadcasting match, which has no full-time moment yet.
  final?: boolean
}

// A Hattrick-style narrative report: an intro paragraph (weather/crowd/referee flavor, when
// known), then the notable events replayed as flowing prose via commentaryFor, each prefixed with
// the running score at that point (event.homeGoals/awayGoals) the way a real match report marks
// up its timeline. Shared by a finished match's report tab (MatchReportTab) and every live view
// (MatchDetailsPage/FriendlyMatchPage/CupTieDetailsPage), which used to show a bare event ticker
// instead - narrating live events the same way as a finished report, just without the "full time"
// closing line and without the flavor fields it doesn't have yet.
export function MatchEventsReport({
  events,
  homeId,
  homeName,
  awayName,
  weather,
  attendance,
  refereeName,
  assistant1Name,
  assistant2Name,
  final = false,
}: MatchEventsReportProps) {
  const { t } = useTranslation()

  const notable = events.filter((event) => NOTABLE_EVENT_TYPES.includes(event.eventType)).sort((a, b) => a.minute - b.minute)

  const hasFlavor = !!(weather && refereeName && assistant1Name && assistant2Name)

  return (
    <Stack spacing={1.5}>
      <MatchStatsSummary events={events} homeId={homeId} homeName={homeName} awayName={awayName} />
      {hasFlavor && (
        <ReportParagraph
          text={t('match.report.intro', {
            weather: t(`match.weather.${weather}`),
            attendance: attendance?.toLocaleString(),
            referee: refereeName,
            assistant1: assistant1Name,
            assistant2: assistant2Name,
          })}
        />
      )}
      <ReportParagraph text={t('match.report.kickoff')} />
      {notable.map((event, idx) => {
        const isHome = event.teamOwner === homeId
        const teamName = isHome ? homeName : awayName
        const opponentName = isHome ? awayName : homeName
        const commentary = commentaryFor(event, t, teamName, opponentName)
        if (!commentary) {
          return null
        }
        return (
          <ReportRow
            key={idx}
            minute={event.minute}
            homeGoals={event.homeGoals}
            awayGoals={event.awayGoals}
            text={commentary}
            isHome={isHome}
            icon={iconFor(event.eventType)}
          />
        )
      })}
      {final && <ReportParagraph text={t('match.report.fullTime')} />}
    </Stack>
  )
}

// A box-score-style summary above the narrative: total chances each team created, plus cards and
// injuries - the numbers a reader would otherwise have to tally themselves by scanning the whole
// event timeline below. Deliberately omits the goal/missed split (already implicit in the
// scoreline shown above) and just gives the total.
function MatchStatsSummary({
  events,
  homeId,
  homeName,
  awayName,
}: {
  events: MatchEvent[]
  homeId: number
  homeName: string
  awayName: string
}) {
  const { t } = useTranslation()
  const { home, away } = statsByTeam(events, homeId)

  const rows: Array<{ labelKey: string; home: number; away: number }> = [
    { labelKey: 'match.report.summary.chances', home: home.goals + home.missedChances, away: away.goals + away.missedChances },
    { labelKey: 'match.report.summary.yellowCards', home: home.yellowCards, away: away.yellowCards },
    { labelKey: 'match.report.summary.redCards', home: home.redCards, away: away.redCards },
    { labelKey: 'match.report.summary.injuries', home: home.injuries, away: away.injuries },
  ]

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="subtitle2" gutterBottom>
        {t('match.report.summary.title')}
      </Typography>
      <Stack spacing={1} sx={{ mb: 1 }}>
        <Stack direction="row">
          <Typography variant="caption" sx={{ flex: 1, fontWeight: 700, textAlign: 'right', pr: 1 }} noWrap>
            {homeName}
          </Typography>
          <Box sx={{ width: { xs: 130, sm: 160 } }} />
          <Typography variant="caption" sx={{ flex: 1, fontWeight: 700, pl: 1 }} noWrap>
            {awayName}
          </Typography>
        </Stack>
        {rows.map((row) => (
          <StatRow key={row.labelKey} label={t(row.labelKey)} home={row.home} away={row.away} />
        ))}
      </Stack>
    </Paper>
  )
}

function StatRow({ label, home, away }: { label: string; home: number; away: number }) {
  const homeStronger = home > away
  const awayStronger = away > home
  return (
    <Stack direction="row" sx={{ alignItems: 'center' }}>
      <Typography variant="body2" sx={{ flex: 1, textAlign: 'right', pr: 1, fontWeight: homeStronger ? 700 : 400 }}>
        {home}
      </Typography>
      <Typography variant="caption" color="text.secondary" sx={{ width: { xs: 130, sm: 160 }, textAlign: 'center' }}>
        {label}
      </Typography>
      <Typography variant="body2" sx={{ flex: 1, pl: 1, fontWeight: awayStronger ? 700 : 400 }}>
        {away}
      </Typography>
    </Stack>
  )
}

// Same emoji-per-event-type mapping the old live ticker (Events/EventContent in
// MatchScoreboard.tsx) used - kept as a small local switch since a plain glyph before the text
// suits this report layout better than that ticker's title/aria-label chip.
function iconFor(eventType: number): string | null {
  switch (eventType) {
    case EVENT_TYPE_GOAL:
      return '⚽'
    case EVENT_TYPE_NO_GOAL:
      return '🚫'
    case EVENT_TYPE_YELLOW_CARD:
      return '🟨'
    case EVENT_TYPE_RED_CARD:
      return '🟥'
    case EVENT_TYPE_INJURY:
      return '🚑'
    case EVENT_TYPE_SUBSTITUTION:
      return '🔄'
    default:
      return null
  }
}

function ReportParagraph({ text }: { text: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 2, borderLeftWidth: 4, borderLeftStyle: 'solid', borderLeftColor: 'outlineVariant' }}>
      <Typography variant="body2" color="text.secondary">
        {text}
      </Typography>
    </Paper>
  )
}

// A row colored by which team the event belongs to - a home-team goal/card/etc. gets the home
// color's left bar, an away-team one gets the away color's, so the timeline reads at a glance
// without having to parse the sentence each time.
function ReportRow({
  minute,
  homeGoals,
  awayGoals,
  text,
  isHome,
  icon,
}: {
  minute: number
  homeGoals: number
  awayGoals: number
  text: ReactNode
  isHome: boolean
  icon: string | null
}) {
  return (
    <Stack
      direction="row"
      spacing={1.5}
      sx={{
        alignItems: 'flex-start',
        borderLeftWidth: 4,
        borderLeftStyle: 'solid',
        borderLeftColor: isHome ? 'primary.main' : 'error.main',
        pl: 1.5,
        py: 0.5,
      }}
    >
      <Box sx={{ minWidth: 44, textAlign: 'center', flexShrink: 0 }}>
        <Typography variant="body2" sx={{ fontWeight: 700 }}>
          {homeGoals} - {awayGoals}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {minute}&apos;
        </Typography>
      </Box>
      {icon && (
        <Typography component="span" sx={{ fontSize: '1.1rem', lineHeight: '1.5rem' }}>
          {icon}
        </Typography>
      )}
      <Typography variant="body2" sx={{ flex: 1, pt: 0.25 }}>
        {text}
      </Typography>
    </Stack>
  )
}
