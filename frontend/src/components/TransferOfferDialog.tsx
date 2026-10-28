import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMakeOffer, useTransferComparables } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { ContractTermsFields } from './ContractTermsFields'
import { NumberField } from './NumberField'
import { ResponsiveDialog } from './ResponsiveDialog'
import { TransferComparables } from './TransferComparables'

interface TransferOfferDialogProps {
  player: Player
  open: boolean
  onClose: () => void
  /** The selling club's asking price when the offer answers a market listing - shown as context
   * and used as a one-tap starting point. */
  askingPrice?: number
}

/** The buyer's side of a transfer negotiation - propose a fee to the selling club plus personal
 * terms (wage/duration) to the player. The player decides on the personal terms immediately,
 * before the offer ever reaches the selling club (see backend's MakeTransferOffer) - a decline
 * reveals exactly what they want instead, the same retry flow as ContractRenewalDialog. */
export function TransferOfferDialog({ player, open, onClose, askingPrice }: TransferOfferDialogProps) {
  const { t } = useTranslation()
  const makeOffer = useMakeOffer()
  const comparables = useTransferComparables(player.id, open)
  const [amount, setAmount] = useState('')
  const [wage, setWage] = useState(String(player.wage || ''))
  const [years, setYears] = useState('1')

  useEffect(() => {
    if (open) {
      setAmount(askingPrice ? String(askingPrice) : '')
      setWage(String(player.wage || ''))
      setYears('1')
      makeOffer.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const result = makeOffer.data
  const accepted = result?.accepted ?? false

  const handleSubmit = () => {
    const amountNum = Number(amount)
    const wageNum = Number(wage)
    const yearsNum = Number(years)
    if (!amountNum || amountNum <= 0 || !wageNum || wageNum <= 0 || !yearsNum || yearsNum < 1) return
    makeOffer.mutate({ playerId: player.id, amount: amountNum, wage: wageNum, years: yearsNum })
  }

  const handleUseExpectations = () => {
    if (!result) return
    setWage(String(result.expectedWage))
    setYears(String(Math.min(result.maxAcceptableYears, MAX_CONTRACT_YEARS_REMAINING)))
  }

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('transfers.offerDialog.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
          {!accepted && (
            <Button variant="contained" onClick={handleSubmit} disabled={makeOffer.isPending}>
              {t('transfers.offerDialog.submit')}
            </Button>
          )}
        </>
      }
    >
      <Stack spacing={2}>
        {!accepted && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t('transfers.offerDialog.subtitle')}
            </Typography>
            <NumberField
              label={t('transfers.offerDialog.amount')}
              value={amount}
              onValueChange={setAmount}
              currency
              fullWidth
              helperText={askingPrice ? t('transfers.offerDialog.askingPrice', { price: formatCurrency(askingPrice) }) : undefined}
            />
            <TransferComparables result={comparables.data} isLoading={comparables.isLoading} isError={comparables.isError} />
            <ContractTermsFields
              wage={wage}
              years={years}
              onWageChange={setWage}
              onYearsChange={setYears}
              currentWage={player.wage}
            />
          </>
        )}

        {result &&
          (accepted ? (
            <Alert severity="success">{t('transfers.offerDialog.sent')}</Alert>
          ) : (
            <Stack spacing={1}>
              <Alert severity="warning">
                {t('transfers.offerDialog.declined', {
                  wage: formatCurrency(result.expectedWage),
                  years: t('team.contractDialog.yearsOption', { count: result.maxAcceptableYears }),
                })}
              </Alert>
              <Button size="small" onClick={handleUseExpectations} sx={{ alignSelf: 'flex-start' }}>
                {t('team.contractDialog.useExpectations')}
              </Button>
            </Stack>
          ))}

        {makeOffer.isError && <Alert severity="error">{t('team.offerError')}</Alert>}
      </Stack>
    </ResponsiveDialog>
  )
}
