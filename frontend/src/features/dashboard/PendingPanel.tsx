import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import SnoozeIcon from '@mui/icons-material/Snooze'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useCupTieLineup,
  useFriendlyChallengesReceived,
  useFriendlyLineup,
  useMatchLineup,
  useOffersReceived,
  useTeamTrainer,
} from '../../api/hooks'
import {
  MATCH_KIND_CUP,
  MATCH_KIND_FRIENDLY,
  MATCH_KIND_LEAGUE,
  type FinancesResponse,
  type Lineup,
  type Player,
  type TeamMatchSummary,
} from '../../api/types'
import { matchSummaryLink } from '../../utils/matchSummary'
import { computeCashFlowForecast } from '../finances/cashFlowForecast'
import { useSquadUnavailability } from '../lineup/useSquadUnavailability'
import { buildPendingItems, CONTRACT_EXPIRY_WARNING_YEARS, type PendingItem } from './pendingItems'
import { currentTime, isSnoozed, readSnoozed, snoozeItem, writeSnoozed, type SnoozeState } from './pendingSnooze'

/** How many items a phone shows before "Show all (N)". */
export const PHONE_VISIBLE_ITEMS = 3

// On a phone an Alert's action button squeezes into a narrow column beside the text and splits
// over two lines; here it drops under the message at full width instead.
const STACKED_ALERT_ACTION_SX = {
  '@media (max-width: 599.95px)': {
    '& .MuiAlert-root': { flexWrap: 'wrap' },
    '& .MuiAlert-message': { flex: 1 },
    '& .MuiAlert-action': { width: '100%', m: 0, p: 0, pl: '34px', justifyContent: 'flex-start' },
  },
} as const

const playerName = (p: Player) => `${p.firstName} ${p.lastName}`

export interface PendingPanelProps {
  teamId: number
  players: Player[]
  finances: FinancesResponse | undefined
  /** The team's next fixture (league, cup or friendly), whose lineup the unavailable starters come from. */
  nextMatch: TeamMatchSummary | undefined
}

/** Everything that needs the owner's action, in one panel: one row per item with its icon, text and
 * action, most severe first (see buildPendingItems). Renders nothing when nothing is pending. On a
 * phone only the first PHONE_VISIBLE_ITEMS show until "Show all (N)" is tapped. Every item but an
 * error can be snoozed until tomorrow in this browser (see pendingSnooze). */
export function PendingPanel({ teamId, players, finances, nextMatch }: PendingPanelProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isPhone = useMediaQuery(theme.breakpoints.down('sm'))
  const [showAll, setShowAll] = useState(false)
  const [snoozed, setSnoozed] = useState<SnoozeState>(readSnoozed)

  const { data: academy } = useAcademy()
  const { data: offersReceived } = useOffersReceived(true)
  const { data: challengesReceived } = useFriendlyChallengesReceived(true)
  const { data: trainer, isSuccess: trainerLoaded } = useTeamTrainer(true)

  const nextKind = nextMatch?.kind
  const nextId = nextMatch ? String(nextMatch.matchId) : undefined
  const leagueLineup = useMatchLineup(nextId, nextKind === MATCH_KIND_LEAGUE)
  const cupLineup = useCupTieLineup(nextId, nextKind === MATCH_KIND_CUP)
  const friendlyLineup = useFriendlyLineup(nextId, nextKind === MATCH_KIND_FRIENDLY)
  const nextLineup: Lineup | undefined = (nextKind === MATCH_KIND_LEAGUE ? leagueLineup : nextKind === MATCH_KIND_CUP ? cupLineup : friendlyLineup).data
  // A suspension carries no sanction into a friendly, so it only counts for league and cup fixtures.
  const unavailability = useSquadUnavailability(nextKind === MATCH_KIND_FRIENDLY)

  const unavailableStarters = (nextLineup?.positions ?? [])
    .filter((slot) => unavailability.has(slot.playerId))
    .flatMap((slot) => {
      const starter = players.find((p) => p.id === slot.playerId)
      if (!starter) return []
      const replacementId = nextLineup?.projectedReplacements?.find((r) => r.playerOutId === slot.playerId)?.playerInId
      const replacement = replacementId !== undefined ? players.find((p) => p.id === replacementId) : undefined
      return [{ playerId: starter.id, name: playerName(starter), replacementName: replacement && playerName(replacement) }]
    })

  const forecast = finances ? computeCashFlowForecast(finances.transactions, finances.budget, finances.maxDebt) : null

  const items = buildPendingItems({
    teamId,
    insolvent: !!finances?.insolvent,
    weeksUntilInsolvent: forecast?.weeksUntilInsolvent ?? null,
    unavailableStarters,
    nextFixture: nextMatch ? { opponentName: nextMatch.opponentName, path: matchSummaryLink(nextMatch) } : undefined,
    expiringContracts: players.filter((p) => p.contractYearsRemaining <= CONTRACT_EXPIRY_WARNING_YEARS).length,
    hasTrainer: trainerLoaded ? !!trainer : undefined,
    ticketPrice: finances?.stadium?.ticketPrice,
    recommendedTicketPrice: finances?.stadium ? finances.recommendedTicketPrice : undefined,
    offersReceived: offersReceived?.length ?? 0,
    challengesReceived: challengesReceived?.length ?? 0,
    academyCanPromote: !!academy?.canPromote,
  })

  const now = currentTime()
  const pending = items.filter((item) => !isSnoozed(snoozed, item, now))
  if (pending.length === 0) {
    return null
  }

  const collapsed = isPhone && !showAll && pending.length > PHONE_VISIBLE_ITEMS
  const visible = collapsed ? pending.slice(0, PHONE_VISIBLE_ITEMS) : pending

  const handleSnooze = (item: PendingItem) => {
    const next = snoozeItem(snoozed, item, currentTime())
    setSnoozed(next)
    writeSnoozed(next)
  }

  return (
    <Box component="section" aria-label={t('dashboard.pending.title')} sx={{ mb: 2 }}>
      <Typography variant="overline" color="text.secondary" sx={{ display: 'block', lineHeight: 1.5, mb: 0.5 }}>
        {t('dashboard.pending.title')}
      </Typography>
      <Stack spacing={1} sx={STACKED_ALERT_ACTION_SX}>
        {visible.map((item) => (
          <Alert
            key={item.id}
            severity={item.severity}
            action={
              <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
                <Button color="inherit" size="small" component={RouterLink} to={item.to}>
                  {t(item.actionKey)}
                </Button>
                {item.snoozable && (
                  <Tooltip title={t('dashboard.pending.snooze')}>
                    <IconButton color="inherit" size="small" aria-label={t('dashboard.pending.snooze')} onClick={() => handleSnooze(item)}>
                      <SnoozeIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
              </Stack>
            }
          >
            {t(item.textKey, item.params)}
          </Alert>
        ))}
      </Stack>
      {isPhone && pending.length > PHONE_VISIBLE_ITEMS && (
        <Button size="small" sx={{ mt: 0.5 }} onClick={() => setShowAll(!showAll)} aria-expanded={showAll}>
          {showAll ? t('dashboard.pending.showLess') : t('dashboard.pending.showAll', { count: pending.length })}
        </Button>
      )}
    </Box>
  )
}
