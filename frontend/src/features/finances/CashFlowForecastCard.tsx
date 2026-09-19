import Alert from '@mui/material/Alert'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { TeamTransaction } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { computeCashFlowForecast } from './cashFlowForecast'

export function CashFlowForecastCard({ transactions, budget, maxDebt }: { transactions: TeamTransaction[]; budget: number; maxDebt: number }) {
  const { t } = useTranslation()
  const forecast = computeCashFlowForecast(transactions, budget, maxDebt)

  if (!forecast) {
    return null
  }

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Typography variant="subtitle1" gutterBottom>
          {t('finances.forecast.title')}
        </Typography>
        <Grid container spacing={2} sx={{ mb: forecast.weeksUntilInsolvent !== null ? 2 : 0 }}>
          <Grid size={{ xs: 6, sm: 3 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              {t('finances.forecast.weeklyIncome')}
            </Typography>
            <Typography variant="body1">{formatCurrency(forecast.weeklyIncome)}</Typography>
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              {t('finances.forecast.weeklyWages')}
            </Typography>
            <Typography variant="body1">{formatCurrency(forecast.weeklyWages)}</Typography>
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              {t('finances.forecast.netWeekly')}
            </Typography>
            <Typography variant="body1" color={forecast.netWeekly < 0 ? 'error' : 'success.main'}>
              {formatCurrency(forecast.netWeekly)}
            </Typography>
          </Grid>
        </Grid>
        {forecast.weeksUntilInsolvent !== null && (
          <Alert severity="warning">{t('finances.forecast.weeksUntilInsolvent', { count: forecast.weeksUntilInsolvent })}</Alert>
        )}
      </CardContent>
    </Card>
  )
}
