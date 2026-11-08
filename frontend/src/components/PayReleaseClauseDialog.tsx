import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { usePayReleaseClause } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { clauseValue } from '../utils/contractTerms'
import { formatCurrency } from '../utils/formatCurrency'
import { ContractTermsFields } from './ContractTermsFields'
import { ResponsiveDialog } from './ResponsiveDialog'

interface PayReleaseClauseDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** Pay another club's player's release clause and sign him at once, without his club's consent
 * (openspec/specs/release-clauses "Paying a clause"). The player still judges the personal terms;
 * a refusal reveals what he wants, as in the other contract dialogs. */
export function PayReleaseClauseDialog({ player, open, onClose }: PayReleaseClauseDialogProps) {
  const { t } = useTranslation()
  const pay = usePayReleaseClause()
  const [wage, setWage] = useState(String(player.wage || ''))
  const [years, setYears] = useState('1')
  const [clause, setClause] = useState('')

  useEffect(() => {
    if (open) {
      setWage(String(player.wage || ''))
      setYears('1')
      setClause('')
      pay.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const result = pay.data
  const accepted = result?.accepted ?? false

  const handleSubmit = () => {
    const wageNum = Number(wage)
    const yearsNum = Number(years)
    if (!wageNum || wageNum <= 0 || !yearsNum || yearsNum < 1) return
    pay.mutate({ playerId: player.id, wage: wageNum, years: yearsNum, releaseClause: clauseValue(clause) })
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
      title={t('transfers.releaseClause.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
          {!accepted && (
            <Button variant="contained" color="warning" onClick={handleSubmit} disabled={pay.isPending}>
              {t('transfers.releaseClause.submit', { fee: formatCurrency(player.releaseClause ?? 0) })}
            </Button>
          )}
        </>
      }
    >
      <Stack spacing={2}>
        {!accepted && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t('transfers.releaseClause.subtitle', { fee: formatCurrency(player.releaseClause ?? 0) })}
            </Typography>
            <ContractTermsFields
              wage={wage}
              years={years}
              onWageChange={setWage}
              onYearsChange={setYears}
              currentWage={player.wage}
              wageFloor={player.wage}
              clause={clause}
              onClauseChange={setClause}
            />
          </>
        )}
        {result &&
          (accepted ? (
            <Alert severity="success">{t('transfers.releaseClause.signed')}</Alert>
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
        {pay.isError && <Alert severity="error">{t('transfers.releaseClause.error')}</Alert>}
      </Stack>
    </ResponsiveDialog>
  )
}
