import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAcceptCounterOffer, useDeclineCounterOffer } from '../../api/hooks'
import type { TransferOffer } from '../../api/types'
import { isPast } from '../../api/useLiveMatch'
import { useConfirm } from '../../components/confirmDialogContext'
import { CounterOfferDialog } from '../../components/CounterOfferDialog'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime, formatShortDate } from '../../utils/formatDate'

interface ReceivedOfferActionsProps {
  offer: TransferOffer
  onAccept: (offer: TransferOffer) => void
  onReject: (offer: TransferOffer) => void
  acceptPending: boolean
  rejectPending: boolean
}

/** What the seller can do with one received offer: reject, counter once, or accept once its
 * negotiation window has passed. A countered offer waits for the buyer (openspec/specs/
 * transfer-negotiation), so it only shows the counter and when it lapses. */
export function ReceivedOfferActions({ offer, onAccept, onReject, acceptPending, rejectPending }: ReceivedOfferActionsProps) {
  const { t } = useTranslation()
  const [counterOpen, setCounterOpen] = useState(false)

  if (offer.counterAmount !== undefined) {
    return (
      <Stack spacing={0.5} sx={{ alignItems: 'flex-end', textAlign: 'right' }}>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {t('transfers.offers.counteredBySeller', { amount: formatCurrency(offer.counterAmount), team: offer.fromTeamName })}
        </Typography>
        {offer.counterExpiresAt && (
          <Typography variant="caption" color="text.secondary">
            {t('transfers.offers.counterExpires', { date: formatDateTime(offer.counterExpiresAt) })}
          </Typography>
        )}
      </Stack>
    )
  }

  const acceptReady = isPast(offer.acceptableAt)
  return (
    <Stack spacing={0.5} sx={{ alignItems: 'flex-end' }}>
      <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', justifyContent: 'flex-end', rowGap: 1 }}>
        <Button color="secondary" variant="outlined" onClick={() => onReject(offer)} disabled={rejectPending}>
          {t('transfers.reject')}
        </Button>
        <Button variant="outlined" onClick={() => setCounterOpen(true)}>
          {t('transfers.offers.counter')}
        </Button>
        <Button variant="contained" onClick={() => onAccept(offer)} disabled={acceptPending || !acceptReady}>
          {t('transfers.accept')}
        </Button>
      </Stack>
      {!acceptReady && (
        <Typography variant="caption" color="text.secondary" sx={{ textAlign: 'right' }}>
          {t('transfers.offers.acceptableAt', { date: formatDateTime(offer.acceptableAt) })}
        </Typography>
      )}
      <CounterOfferDialog offer={offer} open={counterOpen} onClose={() => setCounterOpen(false)} />
    </Stack>
  )
}

interface SentOfferCardProps {
  offer: TransferOffer
  onWithdraw: (offer: TransferOffer) => void
  withdrawPending: boolean
}

/** One of the buyer's own offers. A countered one shows the seller's counter fee, when it lapses
 * and Accept / Decline instead of Withdraw. */
export function SentOfferCard({ offer, onWithdraw, withdrawPending }: SentOfferCardProps) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const acceptCounter = useAcceptCounterOffer()
  const declineCounter = useDeclineCounterOffer()
  const name = `${offer.player.firstName} ${offer.player.lastName}`
  const countered = offer.counterAmount !== undefined

  const handleAcceptCounter = async () => {
    if (offer.counterAmount === undefined) return
    const ok = await confirm({
      message: t('transfers.confirmAcceptCounter', { name, team: offer.toTeamName, amount: formatCurrency(offer.counterAmount) }),
      confirmLabel: t('transfers.offers.acceptCounter'),
    })
    if (ok) acceptCounter.mutate(offer.offerId)
  }

  const handleDeclineCounter = async () => {
    if (offer.counterAmount === undefined) return
    const ok = await confirm({
      message: t('transfers.confirmDeclineCounter', { name, team: offer.toTeamName, amount: formatCurrency(offer.counterAmount) }),
      confirmLabel: t('transfers.offers.declineCounter'),
      destructive: true,
    })
    if (ok) declineCounter.mutate(offer.offerId)
  }

  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <Box sx={{ minWidth: 0 }}>
          <Link component={RouterLink} to={`/player/${offer.player.id}`} underline="hover" color="inherit" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
            {name}
          </Link>
          <Typography variant="caption" color="text.secondary" component="div">
            <TeamNameLink teamId={offer.toTeamId}>{offer.toTeamName}</TeamNameLink>
          </Typography>
        </Box>
        <Box sx={{ textAlign: 'right', flexShrink: 0 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.table.amount')}
          </Typography>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
            {formatCurrency(offer.amount)}
          </Typography>
        </Box>
      </Stack>
      <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
        {t('transfers.offers.termsLine', {
          wage: formatCurrency(offer.offeredWage),
          years: t('team.contractDialog.yearsOption', { count: offer.offeredYears }),
        })}
        {' · '}
        {t('transfers.offers.made', { date: formatShortDate(offer.createdAt) })}
      </Typography>
      {countered && offer.counterAmount !== undefined ? (
        <>
          <Alert severity="warning" icon={false} sx={{ mt: 1 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {t('transfers.offers.counterReceived', { team: offer.toTeamName, amount: formatCurrency(offer.counterAmount) })}
            </Typography>
            {offer.counterExpiresAt && (
              <Typography variant="caption" component="div">
                {t('transfers.offers.counterExpires', { date: formatDateTime(offer.counterExpiresAt) })}
              </Typography>
            )}
          </Alert>
          {(acceptCounter.isError || declineCounter.isError) && (
            <Alert severity="error" sx={{ mt: 1 }}>
              {t('transfers.offers.answerError')}
            </Alert>
          )}
          <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', mt: 1 }}>
            <Button color="secondary" variant="outlined" onClick={handleDeclineCounter} disabled={declineCounter.isPending || acceptCounter.isPending}>
              {t('transfers.offers.declineCounter')}
            </Button>
            <Button variant="contained" onClick={handleAcceptCounter} disabled={acceptCounter.isPending || declineCounter.isPending}>
              {t('transfers.offers.acceptCounter')}
            </Button>
          </Stack>
        </>
      ) : (
        <Stack direction="row" sx={{ justifyContent: 'flex-end', mt: 1 }}>
          <Button color="secondary" variant="outlined" onClick={() => onWithdraw(offer)} disabled={withdrawPending}>
            {t('transfers.withdraw')}
          </Button>
        </Stack>
      )}
    </Paper>
  )
}
