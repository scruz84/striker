import SwapHorizTwoToneIcon from '@mui/icons-material/SwapHorizTwoTone'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useRecentLeagueTransfers } from '../../api/hooks'
import { PLAYER_TRANSFER_LOAN } from '../../api/types'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { CountryFlag } from '../../components/CountryFlag'

// Shown on both the league details page and the dashboard - a league's most recent moves (bought
// listings, accepted offers, exercised buy options, premium-paying free-agent signings and loans
// involving at least one team from the league), newest first. A loan gets a "Loan" chip, its fee
// or "Free", and reads "{borrower} on loan from {lender}". See common.GetRecentLeagueTransfers.
export function RecentLeagueTransfers({ leagueId }: { leagueId: string | undefined }) {
  const { t } = useTranslation()
  const { data: transfers, isLoading } = useRecentLeagueTransfers(leagueId)

  return (
    <Paper variant="outlined">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <SwapHorizTwoToneIcon color="primary" fontSize="small" />
        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
          {t('recentTransfers.title')}
        </Typography>
      </Stack>
      {isLoading || !transfers ? (
        <Skeleton variant="rectangular" height={100} sx={{ m: 2, borderRadius: 1 }} />
      ) : transfers.length === 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
          {t('recentTransfers.empty')}
        </Typography>
      ) : (
        <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
          {transfers.map((transfer, index) => (
            <Stack
              key={`${transfer.player.id}-${transfer.occurredAt}-${index}`}
              direction="row"
              spacing={1.5}
              sx={{ alignItems: 'center', px: 2, py: 1 }}
            >
              <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
                  <CountryFlag code={transfer.player.birthCountry} height={14} marginRight={0.5} />
                  <PlayerNameLink playerId={transfer.player.id}>
                    {transfer.player.firstName} {transfer.player.lastName}
                  </PlayerNameLink>
                  {transfer.kind === PLAYER_TRANSFER_LOAN && (
                    <Chip size="small" variant="outlined" color="info" label={t('recentTransfers.loanChip')} sx={{ ml: 1, height: 18, fontSize: '0.6875rem' }} />
                  )}
                </Typography>
                <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                  <TeamNameLink teamId={transfer.buyerTeamId}>{transfer.buyerTeamName}</TeamNameLink>
                  {' '}
                  {transfer.kind === PLAYER_TRANSFER_LOAN ? t('recentTransfers.onLoanFrom') : t('recentTransfers.from')}
                  {' '}
                  <TeamNameLink teamId={transfer.sellerTeamId}>{transfer.sellerTeamName}</TeamNameLink>
                </Typography>
              </Box>
              <Box sx={{ textAlign: 'right', flexShrink: 0 }}>
                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                  {transfer.kind === PLAYER_TRANSFER_LOAN && transfer.amount === 0 ? t('recentTransfers.free') : formatCurrency(transfer.amount)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {formatDateTime(transfer.occurredAt)}
                </Typography>
              </Box>
            </Stack>
          ))}
        </Stack>
      )}
    </Paper>
  )
}
