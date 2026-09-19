import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme, type Theme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { TRANSACTION_REASON_KEYS, type TeamTransaction } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'

// A qualitative palette cycling through the theme's own colors - there's no state (success/
// warning/error) being represented here, just several unrelated income categories, so reusing
// theme colors keeps it on-brand without introducing a separate hardcoded palette.
function paletteColors(theme: Theme): string[] {
  return [theme.palette.primary.main, theme.palette.secondary.main, theme.palette.info.main, theme.palette.success.main, theme.palette.warning.main, theme.palette.error.main]
}

// Groups every income transaction (amount > 0) in the visible window by reason and charts each
// total as a horizontal bar, longest first - a quick read on which income stream actually pays
// the bills, alongside the recurring-vs-lump distinction the finances table already shows per row.
export function IncomeBreakdownChart({ transactions }: { transactions: TeamTransaction[] }) {
  const { t } = useTranslation()
  const theme = useTheme()

  const totalsByReason = new Map<number, number>()
  for (const tx of transactions) {
    if (tx.amount <= 0) {
      continue
    }
    totalsByReason.set(tx.reason, (totalsByReason.get(tx.reason) ?? 0) + tx.amount)
  }

  const bars = Array.from(totalsByReason.entries())
    .map(([reason, total]) => ({ reason, total }))
    .sort((a, b) => b.total - a.total)

  if (bars.length === 0) {
    return null
  }

  const maxTotal = bars[0].total
  const colors = paletteColors(theme)

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('finances.chart.incomeTitle')}
      </Typography>
      <Stack spacing={1.5}>
        {bars.map((bar, i) => (
          <Box key={bar.reason}>
            <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="body2">{t(TRANSACTION_REASON_KEYS[bar.reason] ?? 'finances.reasons.other')}</Typography>
              <Typography variant="body2" color="text.secondary">
                {formatCurrency(bar.total)}
              </Typography>
            </Stack>
            <Box sx={{ width: '100%', height: 8, borderRadius: 1, bgcolor: 'action.hover' }}>
              <Box
                sx={{
                  width: `${(bar.total / maxTotal) * 100}%`,
                  height: '100%',
                  borderRadius: 1,
                  bgcolor: colors[i % colors.length],
                }}
              />
            </Box>
          </Box>
        ))}
      </Stack>
    </Paper>
  )
}
