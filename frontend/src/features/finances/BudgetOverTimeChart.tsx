import { useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useTranslation } from 'react-i18next'
import type { TeamTransaction } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime, formatShortDate } from '../../utils/formatDate'
import { useElementWidth } from '../../utils/useElementWidth'

const FONT_SIZE = 11 // never smaller than the app's 11px floor, whatever the screen width
const PADDING = { top: 12, right: 12, bottom: 24, left: 58 }

// Reconstructs and charts the running budget balance across the visible transaction history
// (mirroring SeasonPositionChart's hand-rolled SVG line approach, the existing convention for a
// single-series chart in this codebase). transactions only covers the most recent window (see
// backend RecentTransactionsLimit), not the team's whole lifetime, so this necessarily shows a
// recent trend rather than a complete history - still enough to spot a budget sliding the wrong
// way before it becomes a crisis.
//
// The SVG is drawn at the container's real pixel width (not scaled from a fixed viewBox), so the
// axis labels stay legible on a phone; the plot is shorter there and a tap shows the tooltip.
export function BudgetOverTimeChart({ transactions, budget }: { transactions: TeamTransaction[]; budget: number }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isPhone = useMediaQuery(theme.breakpoints.down('sm'))
  const { ref, width } = useElementWidth<HTMLDivElement>()
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)

  if (transactions.length === 0) {
    return null
  }

  const graphWidth = Math.max(width, 240)
  const graphHeight = isPhone ? 170 : 220
  const plotWidth = graphWidth - PADDING.left - PADDING.right
  const plotHeight = graphHeight - PADDING.top - PADDING.bottom

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
    chronological.length === 1 ? PADDING.left + plotWidth / 2 : PADDING.left + (index / (chronological.length - 1)) * plotWidth
  const yFor = (balance: number) => PADDING.top + (1 - (balance - minBalance) / span) * plotHeight

  const linePath = chronological.map((p, i) => `${i === 0 ? 'M' : 'L'} ${xFor(i)} ${yFor(p.balance)}`).join(' ')
  const zeroLineY = minBalance <= 0 && maxBalance >= 0 ? yFor(0) : null

  const yTicks = Array.from(new Set([minBalance, Math.round((minBalance + maxBalance) / 2), maxBalance]))
  // X axis: first and last date of the window (a middle one too when there is room).
  const xTicks = Array.from(new Set(plotWidth > 360 ? [0, Math.floor((chronological.length - 1) / 2), chronological.length - 1] : [0, chronological.length - 1]))

  const hovered = hoverIndex !== null ? chronological[hoverIndex] : null
  // A tap anywhere on the plot selects the nearest point (hover targets are too small for a finger).
  const nearestIndex = (clientX: number, rect: DOMRect) => {
    const x = clientX - rect.left
    let best = 0
    chronological.forEach((_p, i) => {
      if (Math.abs(xFor(i) - x) < Math.abs(xFor(best) - x)) best = i
    })
    return best
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('finances.chart.budgetTitle')}
      </Typography>
      <Box ref={ref} sx={{ position: 'relative', width: '100%' }}>
        <svg
          width={graphWidth}
          height={graphHeight}
          role="img"
          aria-label={t('finances.chart.budgetAriaLabel')}
          style={{ display: 'block', touchAction: 'pan-y' }}
          onPointerDown={(event) => setHoverIndex(nearestIndex(event.clientX, event.currentTarget.getBoundingClientRect()))}
          onPointerMove={(event) => {
            if (event.pointerType === 'mouse') {
              setHoverIndex(nearestIndex(event.clientX, event.currentTarget.getBoundingClientRect()))
            }
          }}
          onPointerLeave={(event) => {
            if (event.pointerType === 'mouse') setHoverIndex(null)
          }}
        >
          {yTicks.map((tick) => (
            <g key={tick}>
              <line x1={PADDING.left} x2={graphWidth - PADDING.right} y1={yFor(tick)} y2={yFor(tick)} stroke={theme.palette.divider} strokeWidth={1} />
              <text x={PADDING.left - 8} y={yFor(tick)} textAnchor="end" dominantBaseline="middle" fontSize={FONT_SIZE} fill={theme.palette.text.secondary}>
                {formatCurrency(tick, { compact: true })}
              </text>
            </g>
          ))}

          {xTicks.map((index) => (
            <text
              key={index}
              x={xFor(index)}
              y={graphHeight - 6}
              textAnchor={index === 0 ? 'start' : index === chronological.length - 1 ? 'end' : 'middle'}
              fontSize={FONT_SIZE}
              fill={theme.palette.text.secondary}
            >
              {formatShortDate(chronological[index].createdAt)}
            </text>
          ))}

          {zeroLineY !== null && (
            <line x1={PADDING.left} x2={graphWidth - PADDING.right} y1={zeroLineY} y2={zeroLineY} stroke={theme.palette.error.main} strokeWidth={1} strokeDasharray="4 4" />
          )}

          <path d={linePath} fill="none" stroke={theme.palette.primary.main} strokeWidth={2} />

          <circle
            cx={xFor(chronological.length - 1)}
            cy={yFor(chronological[chronological.length - 1].balance)}
            r={4}
            fill={theme.palette.primary.main}
            stroke={theme.palette.background.paper}
            strokeWidth={1.5}
          />
          {hoverIndex !== null && (
            <circle cx={xFor(hoverIndex)} cy={yFor(chronological[hoverIndex].balance)} r={6} fill={theme.palette.primary.main} stroke={theme.palette.background.paper} strokeWidth={1.5} />
          )}
        </svg>
        {hovered && hoverIndex !== null && (
          <Paper
            elevation={3}
            sx={{
              position: 'absolute',
              left: Math.min(Math.max(xFor(hoverIndex), 70), graphWidth - 70),
              top: yFor(hovered.balance),
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
