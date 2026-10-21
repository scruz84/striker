import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { TeamTransaction } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { computeCashFlowForecast } from './cashFlowForecast'

function Kpi({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Typography variant="overline" color="text.secondary" component="div" noWrap>
          {label}
        </Typography>
        <Typography variant="stat" component="div" sx={{ fontSize: { xs: '1.375rem', sm: '1.75rem' }, color }}>
          {value}
        </Typography>
      </CardContent>
    </Card>
  )
}

/** The top of the finances page: budget plus the weekly income / expenses / net forecast as a
 * 2x2 grid of KPIs (merging what used to be two separate cards), and the insolvency warning. */
export function CashFlowKpis({ transactions, budget, maxDebt }: { transactions: TeamTransaction[]; budget: number; maxDebt: number }) {
  const { t } = useTranslation()
  const forecast = computeCashFlowForecast(transactions, budget, maxDebt)

  return (
    <Box sx={{ mb: 2 }}>
      <Grid container spacing={1.5}>
        <Grid size={{ xs: 6, sm: forecast ? 3 : 12 }}>
          <Kpi label={t('finances.budget')} value={formatCurrency(budget)} color={budget < 0 ? 'error.main' : undefined} />
        </Grid>
        {forecast && (
          <>
            <Grid size={{ xs: 6, sm: 3 }}>
              <Kpi label={t('finances.kpi.weeklyIncome')} value={formatCurrency(forecast.weeklyIncome, { compact: true })} />
            </Grid>
            <Grid size={{ xs: 6, sm: 3 }}>
              <Kpi label={t('finances.kpi.weeklyExpenses')} value={formatCurrency(forecast.weeklyWages, { compact: true })} />
            </Grid>
            <Grid size={{ xs: 6, sm: 3 }}>
              <Kpi
                label={t('finances.kpi.weeklyNet')}
                value={formatCurrency(forecast.netWeekly, { compact: true, signed: true })}
                color={forecast.netWeekly < 0 ? 'error.main' : 'success.main'}
              />
            </Grid>
          </>
        )}
      </Grid>
      {forecast?.weeksUntilInsolvent != null && (
        <Alert severity="warning" sx={{ mt: 1.5 }}>
          {t('finances.forecast.weeksUntilInsolvent', { count: forecast.weeksUntilInsolvent })}
        </Alert>
      )}
    </Box>
  )
}
