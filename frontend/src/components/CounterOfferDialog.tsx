import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import { useTranslation } from 'react-i18next'
import { useCounterOffer, useTransferComparables } from '../api/hooks'
import type { TransferOffer } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { NumberField } from './NumberField'
import { ResponsiveDialog } from './ResponsiveDialog'
import { TransferComparables } from './TransferComparables'

// Mirrors backend common.MaxCounterOfferMultiple: a counter is above the offer and at most this
// many times it.
const MAX_COUNTER_MULTIPLE = 10

interface CounterOfferDialogProps {
  offer: TransferOffer
  open: boolean
  onClose: () => void
}

/** The seller's one counter-offer on a received offer (openspec/specs/transfer-negotiation), with
 * the player's recent comparable transfers as a reference for the fee. */
export function CounterOfferDialog({ offer, open, onClose }: CounterOfferDialogProps) {
  const { t } = useTranslation()
  const counterOffer = useCounterOffer()
  const comparables = useTransferComparables(offer.player.id, open)
  const [amount, setAmount] = useState('')

  useEffect(() => {
    if (open) {
      setAmount('')
      counterOffer.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, offer.offerId])

  const max = offer.amount * MAX_COUNTER_MULTIPLE
  const value = Number(amount)
  const valid = value > offer.amount && value <= max

  const handleSubmit = () => {
    if (!valid) return
    counterOffer.mutate({ offerId: offer.offerId, amount: value }, { onSuccess: onClose })
  }

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('transfers.offers.counterTitle')}
      subtitle={`${offer.player.firstName} ${offer.player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="contained" onClick={handleSubmit} disabled={!valid || counterOffer.isPending}>
            {t('transfers.offers.send')}
          </Button>
        </>
      }
    >
      <Stack spacing={2}>
        <NumberField
          label={t('transfers.offers.counterAmount')}
          value={amount}
          onValueChange={setAmount}
          currency
          fullWidth
          autoFocus
          error={amount !== '' && !valid}
          helperText={t('transfers.offers.counterHint', {
            team: offer.fromTeamName,
            amount: formatCurrency(offer.amount),
            max: formatCurrency(max),
          })}
        />
        <TransferComparables result={comparables.data} isLoading={comparables.isLoading} isError={comparables.isError} />
        {counterOffer.isError && <Alert severity="error">{t('transfers.offers.counterError')}</Alert>}
      </Stack>
    </ResponsiveDialog>
  )
}
