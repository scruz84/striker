import { useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { TeamTransaction } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

const GRAPH_WIDTH = 640
const GRAPH_HEIGHT = 220
const PADDING = { top: 16, right: 16, bottom: 16, left: 56 }
const PLOT_WIDTH = GRAPH_WIDTH - PADDING.left - PADDING.right
const PLOT_HEIGHT = GRAPH_HEIGHT - PADDING.top - PADDING.bottom

// Reconstructs and charts the running budget balance across the visible transaction history
// (mirroring SeasonPositionChart's hand-rolled SVG line approach, the existing convention for a
// single-series chart in this codebase). transactions only covers the most recent window (see
// backend RecentTransactionsLimit), not the team's whole lifetime, so this necessarily shows a
// recent trend rather than a complete history - still enough to spot a budget sliding the wrong
// way before it becomes a crisis.
export function BudgetOverTimeChart({ transactions, budget }: { transactions: TeamTransaction[]; budget: number }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)

  if (transactions.length === 0) {
    return null
  }

  // transactions is ordered most-recent-first; walk backward from the current budget, undoing
  // each transaction's amount to reconstruct the balance right after the previous one, then
  // reverse so the chart reads left (oldest) to right (most recent, i.e. today's budget).
  let runningBalance = budget
  const chronological: { createdAt: string; balance: number }[] = []
  for (const tx of transactions) {
    chronological.push({ createdAt: tx.createdAt, balance: runningBalance })
    runningBalance -= tx.amount
  }
  chronological.reverse()

  const balances = chronological.map((p) => p.balance)
  const minBalance = Math.min(...balances, 0)
  const maxBalance = Math.max(...balances, 0)
  const span = Math.max(maxBalance - minBalance, 1)

  const xFor = (index: number) =>
    chronological.length === 1 ? PADDING.left + PLOT_WIDTH / 2 : PADDING.left + (index / (chronological.length - 1)) * PLOT_WIDTH
  const yFor = (balance: number) => PADDING.top + (1 - (balance - minBalance) / span) * PLOT_HEIGHT

  const linePath = chronological.map((p, i) => `${i === 0 ? 'M' : 'L'} ${xFor(i)} ${yFor(p.balance)}`).join(' ')
  const zeroLineY = minBalance <= 0 && maxBalance >= 0 ? yFor(0) : null

  const yTicks = Array.from(new Set([minBalance, Math.round((minBalance + maxBalance) / 2), maxBalance]))

  const hovered = hoverIndex !== null ? chronological[hoverIndex] : null

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('finances.chart.budgetTitle')}
      </Typography>
      <Box sx={{ position: 'relative', width: '100%', maxWidth: GRAPH_WIDTH, aspectRatio: `${GRAPH_WIDTH} / ${GRAPH_HEIGHT}` }}>
        <svg viewBox={`0 0 ${GRAPH_WIDTH} ${GRAPH_HEIGHT}`} width="100%" height="100%" role="img" aria-label={t('finances.chart.budgetAriaLabel')}>
          {yTicks.map((tick) => (
            <g key={tick}>
              <line x1={PADDING.left} x2={GRAPH_WIDTH - PADDING.right} y1={yFor(tick)} y2={yFor(tick)} stroke={theme.palette.divider} strokeWidth={1} />
              <text x={PADDING.left - 8} y={yFor(tick)} textAnchor="end" dominantBaseline="middle" fontSize={10} fill={theme.palette.text.secondary}>
                {formatCurrency(tick)}
              </text>
            </g>
          ))}

          {zeroLineY !== null && (
            <line x1={PADDING.left} x2={GRAPH_WIDTH - PADDING.right} y1={zeroLineY} y2={zeroLineY} stroke={theme.palette.error.main} strokeWidth={1} strokeDasharray="4 4" />
          )}

          <path d={linePath} fill="none" stroke={theme.palette.primary.main} strokeWidth={2} />

          {chronological.map((p, i) => (
            <g key={p.createdAt + i}>
              <circle
                cx={xFor(i)}
                cy={yFor(p.balance)}
                r={10}
                fill="transparent"
                onMouseEnter={() => setHoverIndex(i)}
                onMouseLeave={() => setHoverIndex((current) => (current === i ? null : current))}
                style={{ cursor: 'pointer' }}
              />
              {i === chronological.length - 1 && (
                <circle cx={xFor(i)} cy={yFor(p.balance)} r={hoverIndex === i ? 6 : 4} fill={theme.palette.primary.main} stroke={theme.palette.background.paper} strokeWidth={1.5} style={{ pointerEvents: 'none' }} />
              )}
            </g>
          ))}
        </svg>
        {hovered && hoverIndex !== null && (
          <Paper
            elevation={3}
            sx={{
              position: 'absolute',
              left: `${(xFor(hoverIndex) / GRAPH_WIDTH) * 100}%`,
              top: `${(yFor(hovered.balance) / GRAPH_HEIGHT) * 100}%`,
              transform: 'translate(-50%, -125%)',
              p: 1,
              pointerEvents: 'none',
              whiteSpace: 'nowrap',
            }}
          >
            <Typography variant="caption" sx={{ display: 'block', fontWeight: 600 }}>
              {formatCurrency(hovered.balance)}
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              {formatDateTime(hovered.createdAt)}
            </Typography>
          </Paper>
        )}
      </Box>
    </Paper>
  )
}
