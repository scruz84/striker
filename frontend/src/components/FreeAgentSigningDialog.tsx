import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useSignFreeAgent } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { ContractTermsFields } from './ContractTermsFields'
import { clauseValue } from '../utils/contractTerms'
import { ResponsiveDialog } from './ResponsiveDialog'

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
  const [clause, setClause] = useState('')

  useEffect(() => {
    if (open) {
      setWage(String(player.wage || ''))
      setYears('1')
      setClause('')
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
    sign.mutate({ playerId: player.id, wage: wageNum, years: yearsNum, releaseClause: clauseValue(clause) })
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
      title={t('transfers.signDialog.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
          {!accepted && (
            <Button variant="contained" onClick={handleSubmit} disabled={sign.isPending}>
              {t('transfers.signDialog.submit')}
            </Button>
          )}
        </>
      }
    >
      <Stack spacing={2}>
        {!accepted && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t('transfers.signDialog.subtitle')}
            </Typography>
            {premiumMayApply && <Alert severity="info">{t('transfers.signDialog.premiumWarning')}</Alert>}
            <ContractTermsFields
              wage={wage}
              years={years}
              onWageChange={setWage}
              onYearsChange={setYears}
              currentWage={player.wage}
            clause={clause} onClauseChange={setClause} />
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
    </ResponsiveDialog>
  )
}
