import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMakePreContractOffer } from '../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { ContractTermsFields } from './ContractTermsFields'
import { PRE_CONTRACT_MOVE_PREMIUM, clauseValue } from '../utils/contractTerms'
import { ResponsiveDialog } from './ResponsiveDialog'

interface PreContractDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** Offer another club's player in his last season a contract for next season (openspec/specs/
 * pre-contracts). No fee: the player judges the wage and years at once and wants 10% more than
 * usual to leave his club; his club is told and may still renew him before he decides. */
export function PreContractDialog({ player, open, onClose }: PreContractDialogProps) {
  const { t } = useTranslation()
  const makeOffer = useMakePreContractOffer()
  const suggested = Math.ceil((player.wage || 0) * PRE_CONTRACT_MOVE_PREMIUM)
  const [wage, setWage] = useState(String(suggested || ''))
  const [years, setYears] = useState('1')
  const [clause, setClause] = useState('')

  useEffect(() => {
    if (open) {
      setWage(String(suggested || ''))
      setYears('1')
      setClause('')
      makeOffer.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const result = makeOffer.data
  const accepted = result?.accepted ?? false

  const handleSubmit = () => {
    const wageNum = Number(wage)
    const yearsNum = Number(years)
    if (!wageNum || wageNum <= 0 || !yearsNum || yearsNum < 1) return
    makeOffer.mutate({ playerId: player.id, wage: wageNum, years: yearsNum, releaseClause: clauseValue(clause) })
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
      title={t('transfers.preContract.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{accepted ? t('common.close') : t('common.cancel')}</Button>
          {!accepted && (
            <Button variant="contained" onClick={handleSubmit} disabled={makeOffer.isPending}>
              {t('transfers.preContract.submit')}
            </Button>
          )}
        </>
      }
    >
      <Stack spacing={2}>
        {!accepted && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t('transfers.preContract.subtitle')}
            </Typography>
            <ContractTermsFields wage={wage} years={years} onWageChange={setWage} onYearsChange={setYears} currentWage={player.wage} clause={clause} onClauseChange={setClause} />
          </>
        )}
        {result &&
          (accepted ? (
            <Alert severity="success">{t('transfers.preContract.sent')}</Alert>
          ) : (
            <Stack spacing={1}>
              <Alert severity="warning">
                {t('transfers.preContract.declined', {
                  wage: formatCurrency(result.expectedWage),
                  years: t('team.contractDialog.yearsOption', { count: result.maxAcceptableYears }),
                })}
              </Alert>
              <Button size="small" onClick={handleUseExpectations} sx={{ alignSelf: 'flex-start' }}>
                {t('team.contractDialog.useExpectations')}
              </Button>
            </Stack>
          ))}
        {makeOffer.isError && <Alert severity="error">{t('transfers.preContract.error')}</Alert>}
      </Stack>
    </ResponsiveDialog>
  )
}
