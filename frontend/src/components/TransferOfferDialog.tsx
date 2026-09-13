import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import MenuItem from '@mui/material/MenuItem'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMakeOffer } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'

interface TransferOfferDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** The buyer's side of a transfer negotiation - propose a fee to the selling club plus personal
 * terms (wage/duration) to the player. The player decides on the personal terms immediately,
 * before the offer ever reaches the selling club (see backend's MakeTransferOffer) - a decline
 * reveals exactly what they want instead, the same retry flow as ContractRenewalDialog. */
export function TransferOfferDialog({ player, open, onClose }: TransferOfferDialogProps) {
  const { t } = useTranslation()
  const makeOffer = useMakeOffer()
  const [amount, setAmount] = useState('')
  const [wage, setWage] = useState(String(player.wage || ''))
  const [years, setYears] = useState('1')

  useEffect(() => {
    if (open) {
      setAmount('')
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
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t('transfers.offerDialog.title', { name: `${player.firstName} ${player.lastName}` })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          {!accepted && (
            <>
              <Typography variant="body2" color="text.secondary">
                {t('transfers.offerDialog.subtitle')}
              </Typography>
              <TextField
                label={t('transfers.offerDialog.amount')}
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                fullWidth
                size="small"
              />
              <TextField
                label={t('team.contractDialog.wage')}
                type="number"
                value={wage}
                onChange={(e) => setWage(e.target.value)}
                fullWidth
                size="small"
              />
              <TextField
                select
                label={t('team.contractDialog.years')}
                value={years}
                onChange={(e) => setYears(e.target.value)}
                fullWidth
                size="small"
              >
                {Array.from({ length: MAX_CONTRACT_YEARS_REMAINING }, (_, i) => i + 1).map((y) => (
                  <MenuItem key={y} value={y}>
                    {t('team.contractDialog.yearsOption', { count: y })}
                  </MenuItem>
                ))}
              </TextField>
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
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
        {!accepted && (
          <Button variant="contained" onClick={handleSubmit} disabled={makeOffer.isPending}>
            {t('transfers.offerDialog.submit')}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
