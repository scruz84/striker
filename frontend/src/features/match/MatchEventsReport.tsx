import { Fragment, useMemo, useState, type ReactNode } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  EVENT_TYPE_ADDED_TIME,
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  type MatchEvent,
} from '../../api/types'
import { formatNumber } from '../../utils/formatCurrency'
import { fullTimeLine, halfTimeLine, kickoffLine, narrateMatch } from '../../utils/matchCommentary'
import { compareMatchMinute, formatMatchMinute } from '../../utils/matchMinute'
import { isMinorMissedChance } from '../../utils/matchEvents'
import { TeamBadge } from '../../components/TeamBadge'
import { matchTeamColors } from '../../utils/teamColors'

const NOTABLE_EVENT_TYPES: number[] = [
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_YELLOW_CARD,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  EVENT_TYPE_ADDED_TIME,
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
  awayId: number
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
  // True once the first half is over, which adds the half-time line (match-viewer "Half-time line
  // in the chronicle"). Defaults to `final`: a live view passes whether its broadcast is in the
  // break or past minute 45.
  firstHalfOver?: boolean
  // Tells this fixture apart from another between the same sides (its kickoff time), so a rematch
  // gets different lines - see NarrationTeams.matchKey.
  matchKey?: string
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
  awayId,
  awayName,
  weather,
  attendance,
  refereeName,
  assistant1Name,
  assistant2Name,
  final = false,
  firstHalfOver = final,
  matchKey,
}: MatchEventsReportProps) {
  const { t } = useTranslation()
  const teams = useMemo(() => ({ homeId, homeName, awayId, awayName, matchKey }), [homeId, homeName, awayId, awayName, matchKey])

  // Missed chances ("¡Qué cerca!...") are most of the narration and bury the goals, cards and
  // substitutions, so they're hidden until the viewer asks for the full commentary - except a
  // missed penalty, which always shows (see isMissedPenalty).
  const [fullCommentary, setFullCommentary] = useState(false)
  // Every event is narrated, in match order (45+2 after 45, before 46; a stable sort keeps each
  // minute's events - and the added-time board after the minute's other events - in the order they
  // were recorded), before the "key events only" filter, so collapsing the missed chances never
  // changes another line (match-commentary "No repeated lines within a match").
  const narrated = useMemo(() => narrateMatch(events, teams, t), [events, teams, t])
  const notable = narrated.filter((entry) => NOTABLE_EVENT_TYPES.includes(entry.event.eventType))
  const hasMissedChances = notable.some((entry) => isMinorMissedChance(entry.event))
  const visible = fullCommentary ? notable : notable.filter((entry) => !isMinorMissedChance(entry.event))

  // The match-ending events: a default result or an abandoned match keeps the neutral closing line,
  // and one abandoned in the first half has no half-time line either.
  const ordered = [...events].sort(compareMatchMinute)
  const defaultResult = ordered.some((event) => event.eventType === EVENT_TYPE_MATCH_DEFAULT_RESULT)
  const abandoned = ordered.find((event) => event.eventType === EVENT_TYPE_MATCH_ABANDONED)
  const showHalfTime = firstHalfOver && !defaultResult && !(abandoned && abandoned.minute <= 45)
  const firstHalf = ordered.filter((event) => event.minute <= 45)
  const halfTimeScore = firstHalf.length > 0 ? firstHalf[firstHalf.length - 1] : { homeGoals: 0, awayGoals: 0 }
  const lastEvent = ordered.length > 0 ? ordered[ordered.length - 1] : { homeGoals: 0, awayGoals: 0 }
  // The half-time line sits before the first second-half row shown, or after every row so far.
  const firstSecondHalfRow = visible.findIndex((entry) => entry.event.minute > 45)
  const halfTimeAt = firstSecondHalfRow === -1 ? visible.length : firstSecondHalfRow
  const halfTime = showHalfTime ? halfTimeLine(t, teams, halfTimeScore.homeGoals, halfTimeScore.awayGoals) : undefined
  const kickoff = kickoffLine(t, teams)
  const fullTime = defaultResult || abandoned ? undefined : fullTimeLine(t, teams, lastEvent.homeGoals, lastEvent.awayGoals)

  const hasFlavor = !!(weather && refereeName && assistant1Name && assistant2Name)

  return (
    <Stack spacing={1}>
      <MatchStatsSummary events={events} homeId={homeId} homeName={homeName} awayId={awayId} awayName={awayName} />
      {hasFlavor && (
        <ReportParagraph
          text={t('match.report.intro', {
            weather: t(`match.weather.${weather}`),
            attendance: attendance === undefined ? undefined : formatNumber(attendance),
            referee: refereeName,
            assistant1: assistant1Name,
            assistant2: assistant2Name,
          })}
        />
      )}
      {kickoff && <ReportParagraph text={kickoff} />}
      {hasMissedChances && (
        <Box>
          <Button size="small" onClick={() => setFullCommentary((value) => !value)}>
            {t(fullCommentary ? 'match.report.keyEventsOnly' : 'match.report.fullCommentary')}
          </Button>
        </Box>
      )}
      {visible.map((entry, idx) => {
        const { event } = entry
        const halfTimeRow = halfTime && idx === halfTimeAt ? <ReportParagraph text={halfTime} /> : null
        // The fourth official's board belongs to neither side: no team badge, just how long the half
        // goes on for.
        if (event.eventType === EVENT_TYPE_ADDED_TIME) {
          return (
            <Fragment key={idx}>
              {halfTimeRow}
              <ReportRow
                minute={event.minute}
                addedMinute={event.addedMinute}
                homeGoals={event.homeGoals}
                awayGoals={event.awayGoals}
                text={t('match.addedTime.board', { count: event.addedTime ?? 0 })}
                icon={iconFor(event.eventType)}
                iconLabel={t('match.addedTime.label')}
                emphasized
              />
            </Fragment>
          )
        }
        const isHome = event.teamOwner === homeId
        const teamId = isHome ? homeId : awayId
        const teamName = isHome ? homeName : awayName
        if (!entry.text) {
          return <Fragment key={idx}>{halfTimeRow}</Fragment>
        }
        return (
          <Fragment key={idx}>
            {halfTimeRow}
            <ReportRow
              minute={event.minute}
              addedMinute={event.addedMinute}
              homeGoals={event.homeGoals}
              awayGoals={event.awayGoals}
              text={
                <>
                  {entry.text}
                  {entry.tail && <> {entry.tail}</>}
                </>
              }
              icon={iconFor(event.eventType)}
              iconLabel={event.eventType === EVENT_TYPE_MATCH_DEFAULT_RESULT ? t('match.report.defaultResultLabel') : undefined}
              teamId={teamId}
              teamName={teamName}
              emphasized={!isMinorMissedChance(event)}
            />
          </Fragment>
        )
      })}
      {halfTime && halfTimeAt === visible.length && <ReportParagraph text={halfTime} />}
      {final && <ReportParagraph text={fullTime ?? t('match.report.fullTime')} />}
    </Stack>
  )
}

// A box-score-style summary above the narrative: total chances each team created, plus cards and
// injuries - the numbers a reader would otherwise have to tally themselves by scanning the whole
// event timeline below. Deliberately omits the goal/missed split (already implicit in the
// scoreline shown above) and just gives the total. Teams are identified by their badge rather than
// their name, which truncated on a phone.
function MatchStatsSummary({
  events,
  homeId,
  homeName,
  awayId,
  awayName,
}: {
  events: MatchEvent[]
  homeId: number
  homeName: string
  awayId: number
  awayName: string
}) {
  const { t } = useTranslation()
  const { home, away } = statsByTeam(events, homeId)
  const colors = matchTeamColors(homeId, awayId)

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
      <Stack spacing={1}>
        <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
          <TeamBadge teamId={homeId} teamName={homeName} size={24} />
          <TeamBadge teamId={awayId} teamName={awayName} size={24} />
        </Stack>
        {rows.map((row) => (
          <StatRow key={row.labelKey} label={t(row.labelKey)} home={row.home} away={row.away} homeColor={colors.home} awayColor={colors.away} />
        ))}
      </Stack>
    </Paper>
  )
}

// One stat as a pair of opposing bars growing from the center, each sized by its share of the
// row's total - the figures stay at the edges so the comparison reads at a glance.
function StatRow({
  label,
  home,
  away,
  homeColor,
  awayColor,
}: {
  label: string
  home: number
  away: number
  homeColor: string
  awayColor: string
}) {
  const total = home + away
  const homeShare = total > 0 ? (home / total) * 100 : 0
  const awayShare = total > 0 ? (away / total) * 100 : 0
  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'baseline', justifyContent: 'space-between' }}>
        <Typography variant="body2" sx={{ fontWeight: home > away ? 700 : 400, minWidth: 24 }}>
          {home}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {label}
        </Typography>
        <Typography variant="body2" sx={{ fontWeight: away > home ? 700 : 400, minWidth: 24, textAlign: 'right' }}>
          {away}
        </Typography>
      </Stack>
      <Stack direction="row" spacing={0.5}>
        <Box sx={{ flex: 1, display: 'flex', justifyContent: 'flex-end', bgcolor: 'action.hover', borderRadius: 1, height: 6 }}>
          <Box sx={{ width: `${homeShare}%`, bgcolor: homeColor, borderRadius: 1 }} />
        </Box>
        <Box sx={{ flex: 1, display: 'flex', bgcolor: 'action.hover', borderRadius: 1, height: 6 }}>
          <Box sx={{ width: `${awayShare}%`, bgcolor: awayColor, borderRadius: 1 }} />
        </Box>
      </Stack>
    </Box>
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
    case EVENT_TYPE_MATCH_ABANDONED:
      return '⛔'
    case EVENT_TYPE_MATCH_DEFAULT_RESULT:
      return '🤝'
    case EVENT_TYPE_MENTALITY_CHANGE:
      return '📋'
    case EVENT_TYPE_ADDED_TIME:
      return '⏱️'
    default:
      return null
  }
}

function ReportParagraph({ text }: { text: ReactNode }) {
  return (
    <Typography variant="body2" color="text.secondary">
      {text}
    </Typography>
  )
}

// A compact timeline row: minute on the left, the event's icon, the owning team's badge, then the
// narration - bold for goals, cards, injuries and substitutions, regular for the missed chances
// shown in the full commentary. No per-row card or colored bar: the badge says whose event it is.
function ReportRow({
  minute,
  addedMinute,
  homeGoals,
  awayGoals,
  text,
  icon,
  iconLabel,
  teamId,
  teamName,
  emphasized,
}: {
  minute: number
  addedMinute?: number
  homeGoals: number
  awayGoals: number
  text: ReactNode
  icon: string | null
  iconLabel?: string
  // Omitted for an event that belongs to neither side (the added-time board).
  teamId?: number
  teamName?: string
  emphasized: boolean
}) {
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', py: 0.25 }}>
      <Box sx={{ minWidth: 40, flexShrink: 0 }}>
        <Typography variant="body2" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
          {formatMatchMinute(minute, addedMinute)}
        </Typography>
        {emphasized && (
          <Typography variant="caption" color="text.secondary" sx={{ lineHeight: 1.2 }}>
            {homeGoals}-{awayGoals}
          </Typography>
        )}
      </Box>
      <Typography
        component="span"
        title={iconLabel}
        aria-label={iconLabel}
        sx={{ fontSize: '1.1rem', lineHeight: '1.5rem', width: 24, textAlign: 'center', flexShrink: 0 }}
      >
        {icon}
      </Typography>
      <Box sx={{ pt: 0.25, flexShrink: 0, width: 18 }}>
        {teamId !== undefined && teamName !== undefined && <TeamBadge teamId={teamId} teamName={teamName} size={18} />}
      </Box>
      <Typography variant="body2" sx={{ flex: 1, pt: 0.25, fontWeight: emphasized ? 600 : 400, color: emphasized ? 'text.primary' : 'text.secondary' }}>
        {text}
      </Typography>
    </Stack>
  )
}
