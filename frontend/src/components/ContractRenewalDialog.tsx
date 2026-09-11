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
import { useRenewContract } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'

interface ContractRenewalDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** The team's side of a contract renewal negotiation - propose a wage and duration, the player
 * decides on the spot against their own market wage and age-appropriate max term (see backend's
 * MakeContractRenewalOffer). A decline reveals exactly what they want instead, so retrying with
 * better terms is the whole "negotiation" - there's no other back-and-forth modeled server-side. */
export function ContractRenewalDialog({ player, open, onClose }: ContractRenewalDialogProps) {
  const { t } = useTranslation()
  const renew = useRenewContract()
  const [wage, setWage] = useState(String(player.wage))
  const [years, setYears] = useState('1')

  useEffect(() => {
    if (open) {
      setWage(String(player.wage))
      setYears('1')
      renew.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const offer = renew.data
  const accepted = offer?.accepted ?? false

  const handleSubmit = () => {
    const wageNum = Number(wage)
    const yearsNum = Number(years)
    if (!wageNum || wageNum <= 0 || !yearsNum || yearsNum < 1) return
    renew.mutate({ playerId: player.id, wage: wageNum, years: yearsNum })
  }

  const handleUseExpectations = () => {
    if (!offer) return
    setWage(String(offer.expectedWage))
    setYears(String(Math.min(offer.maxAcceptableYears, MAX_CONTRACT_YEARS_REMAINING)))
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t('team.contractDialog.title', { name: `${player.firstName} ${player.lastName}` })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          {!accepted && (
            <>
              <Typography variant="body2" color="text.secondary">
                {t('team.contractDialog.subtitle')}
              </Typography>
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

          {offer &&
            (accepted ? (
              <Alert severity="success">
                {t('team.contractDialog.accepted', {
                  years: t('team.contractDialog.yearsOption', { count: offer.contractYearsRemaining ?? 0 }),
                  wage: formatCurrency(offer.wage ?? 0),
                })}
              </Alert>
            ) : (
              <Stack spacing={1}>
                <Alert severity="warning">
                  {t('team.contractDialog.declined', {
                    wage: formatCurrency(offer.expectedWage),
                    years: t('team.contractDialog.yearsOption', { count: offer.maxAcceptableYears }),
                  })}
                </Alert>
                <Button size="small" onClick={handleUseExpectations} sx={{ alignSelf: 'flex-start' }}>
                  {t('team.contractDialog.useExpectations')}
                </Button>
              </Stack>
            ))}

          {renew.isError && <Alert severity="error">{t('team.contractDialog.error')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
        {!accepted && (
          <Button variant="contained" onClick={handleSubmit} disabled={renew.isPending}>
            {t('team.contractDialog.submit')}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
