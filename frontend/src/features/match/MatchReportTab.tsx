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
  type WeekMatch,
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

// A Hattrick-style narrative report: an intro paragraph (weather/crowd/referee flavor - see
// backend common.UpdateMatchFlavor), then the same notable events the Summary tab lists as a
// ticker, replayed here as flowing prose via the same commentaryFor phrasing, each prefixed with
// the running score at that point (event.homeGoals/awayGoals) the way a real match report marks
// up its timeline.
export function MatchReportTab({ match }: { match: WeekMatch }) {
  const { t } = useTranslation()

  const notable = (match.events ?? [])
    .filter((event) => NOTABLE_EVENT_TYPES.includes(event.eventType))
    .sort((a, b) => a.minute - b.minute)

  const hasFlavor = !!(match.weather && match.refereeName && match.assistant1Name && match.assistant2Name)

  return (
    <Stack spacing={1.5}>
      {hasFlavor && (
        <ReportParagraph
          text={t('match.report.intro', {
            weather: t(`match.weather.${match.weather}`),
            attendance: match.attendance?.toLocaleString(),
            referee: match.refereeName,
            assistant1: match.assistant1Name,
            assistant2: match.assistant2Name,
          })}
        />
      )}
      <ReportParagraph text={t('match.report.kickoff')} />
      {notable.map((event, idx) => {
        const isHome = event.teamOwner === match.home
        const teamName = isHome ? match.homeName : match.awayName
        const opponentName = isHome ? match.awayName : match.homeName
        const commentary = commentaryFor(event, t, teamName, opponentName)
        if (!commentary) {
          return null
        }
        return <ReportRow key={idx} minute={event.minute} homeGoals={event.homeGoals} awayGoals={event.awayGoals} text={commentary} />
      })}
      <ReportParagraph text={t('match.report.fullTime')} />
    </Stack>
  )
}

function ReportParagraph({ text }: { text: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="body2" color="text.secondary">
        {text}
      </Typography>
    </Paper>
  )
}

function ReportRow({ minute, homeGoals, awayGoals, text }: { minute: number; homeGoals: number; awayGoals: number; text: string }) {
  return (
    <Stack direction="row" spacing={2} sx={{ alignItems: 'flex-start' }}>
      <Box sx={{ minWidth: 44, textAlign: 'center', flexShrink: 0 }}>
        <Typography variant="body2" sx={{ fontWeight: 700 }}>
          {homeGoals} - {awayGoals}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {minute}&apos;
        </Typography>
      </Box>
      <Typography variant="body2" sx={{ flex: 1, pt: 0.25 }}>
        {text}
      </Typography>
    </Stack>
  )
}
