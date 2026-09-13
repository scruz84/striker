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
import { useSignFreeAgent } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'

interface FreeAgentSigningDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** Propose wage/contract-length terms to sign a free agent - the player decides on the spot,
 * exactly like ContractRenewalDialog's renewal flow. If they were released within their current
 * (first) season of free agency, their former club is owed a transfer premium on top of the wage
 * the signing team is taking on - see backend's SignFreeAgent/FreeAgentPremiumWageMultiplier. */
export function FreeAgentSigningDialog({ player, open, onClose }: FreeAgentSigningDialogProps) {
  const { t } = useTranslation()
  const sign = useSignFreeAgent()
  const [wage, setWage] = useState(String(player.wage || ''))
  const [years, setYears] = useState('1')

  useEffect(() => {
    if (open) {
      setWage(String(player.wage || ''))
      setYears('1')
      sign.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const result = sign.data
  const accepted = result?.accepted ?? false
  const premiumMayApply = player.freeAgentSeasons === 0 && !!player.lastTeamId

  const handleSubmit = () => {
    const wageNum = Number(wage)
    const yearsNum = Number(years)
    if (!wageNum || wageNum <= 0 || !yearsNum || yearsNum < 1) return
    sign.mutate({ playerId: player.id, wage: wageNum, years: yearsNum })
  }

  const handleUseExpectations = () => {
    if (!result) return
    setWage(String(result.expectedWage))
    setYears(String(Math.min(result.maxAcceptableYears, MAX_CONTRACT_YEARS_REMAINING)))
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t('transfers.signDialog.title', { name: `${player.firstName} ${player.lastName}` })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          {!accepted && (
            <>
              <Typography variant="body2" color="text.secondary">
                {t('transfers.signDialog.subtitle')}
              </Typography>
              {premiumMayApply && <Alert severity="info">{t('transfers.signDialog.premiumWarning')}</Alert>}
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
              <Stack spacing={1}>
                <Alert severity="success">
                  {t('transfers.signDialog.accepted', {
                    years: t('team.contractDialog.yearsOption', { count: result.contractYearsRemaining ?? 0 }),
                    wage: formatCurrency(result.wage ?? 0),
                  })}
                </Alert>
                {!!result.premium && (
                  <Alert severity="info">{t('transfers.signDialog.premiumCharged', { amount: formatCurrency(result.premium) })}</Alert>
                )}
              </Stack>
            ) : (
              <Stack spacing={1}>
                <Alert severity="warning">
                  {t('transfers.signDialog.declined', {
                    wage: formatCurrency(result.expectedWage),
                    years: t('team.contractDialog.yearsOption', { count: result.maxAcceptableYears }),
                  })}
                </Alert>
                <Button size="small" onClick={handleUseExpectations} sx={{ alignSelf: 'flex-start' }}>
                  {t('team.contractDialog.useExpectations')}
                </Button>
              </Stack>
            ))}

          {sign.isError && <Alert severity="error">{t('transfers.signDialog.error')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
        {!accepted && (
          <Button variant="contained" onClick={handleSubmit} disabled={sign.isPending}>
            {t('transfers.signDialog.submit')}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
