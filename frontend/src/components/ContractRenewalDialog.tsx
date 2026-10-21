import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useRenewContract } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { ContractTermsFields } from './ContractTermsFields'
import { ResponsiveDialog } from './ResponsiveDialog'

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
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('team.contractDialog.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
          {!accepted && (
            <Button variant="contained" onClick={handleSubmit} disabled={renew.isPending}>
              {t('team.contractDialog.submit')}
            </Button>
          )}
        </>
      }
    >
      <Stack spacing={2}>
        {!accepted && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t('team.contractDialog.subtitle')}
            </Typography>
            <ContractTermsFields
              wage={wage}
              years={years}
              onWageChange={setWage}
              onYearsChange={setYears}
              currentWage={player.wage}
            />
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
    </ResponsiveDialog>
  )
}
