import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { POSITION_LABEL_KEYS, type TransferComparablesResult } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { formatDate } from '../utils/formatDate'
import { PlayerNameLink } from './PlayerNameLink'

interface TransferComparablesProps {
  result?: TransferComparablesResult
  isLoading: boolean
  isError: boolean
}

/** Recent sales of players like the offer's target (same position, similar age and average skill
 * at the time of the sale) - a reference for the fee. Only public figures are ever shown. A failure
 * is a short note: it must never get in the way of sending the offer. */
export function TransferComparables({ result, isLoading, isError }: TransferComparablesProps) {
  const { t } = useTranslation()

  let body
  if (isLoading) {
    body = (
      <Stack spacing={1} data-testid="comparables-loading">
        <Skeleton variant="rounded" height={20} />
        <Skeleton variant="rounded" height={44} />
        <Skeleton variant="rounded" height={44} />
      </Stack>
    )
  } else if (isError || !result) {
    body = (
      <Alert severity="info" icon={false}>
        {t('transfers.offerDialog.comparables.error')}
      </Alert>
    )
  } else if (result.comparables.length === 0) {
    body = (
      <Typography variant="body2" color="text.secondary">
        {t('transfers.offerDialog.comparables.empty')}
      </Typography>
    )
  } else {
    body = (
      <Stack spacing={1}>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {t('transfers.offerDialog.comparables.summary', {
            count: result.count,
            median: formatCurrency(result.medianFee ?? 0),
            min: formatCurrency(result.minFee ?? 0),
            max: formatCurrency(result.maxFee ?? 0),
          })}
        </Typography>
        {result.broadened && (
          <Typography variant="caption" color="text.secondary">
            {t('transfers.offerDialog.comparables.broadened')}
          </Typography>
        )}
        <Stack component="ul" spacing={1} sx={{ listStyle: 'none', m: 0, p: 0 }}>
          {result.comparables.map((c) => (
            <Stack
              component="li"
              key={c.transferId}
              direction="row"
              spacing={1}
              sx={{ justifyContent: 'space-between', alignItems: 'flex-start' }}
            >
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
                  <PlayerNameLink playerId={c.playerId}>{c.playerName}</PlayerNameLink>
                </Typography>
                <Typography variant="caption" color="text.secondary" component="div">
                  {t('transfers.offerDialog.comparables.meta', {
                    position: t(POSITION_LABEL_KEYS[c.position]),
                    age: c.age,
                    average: c.averageSkill,
                  })}
                </Typography>
                <Typography variant="caption" color="text.secondary" component="div" sx={{ overflowWrap: 'anywhere' }}>
                  {`${c.fromTeamName || '-'} → ${c.toTeamName || '-'} · ${formatDate(c.createdAt)}`}
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ fontWeight: 700, whiteSpace: 'nowrap' }}>
                {formatCurrency(c.fee)}
              </Typography>
            </Stack>
          ))}
        </Stack>
      </Stack>
    )
  }

  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {t('transfers.offerDialog.comparables.title')}
      </Typography>
      {body}
    </Box>
  )
}
