import { useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { SeasonHistoryEntry } from '../api/types'

const GRAPH_WIDTH = 640
const GRAPH_HEIGHT = 220
const PADDING = { top: 16, right: 16, bottom: 28, left: 32 }
const PLOT_WIDTH = GRAPH_WIDTH - PADDING.left - PADDING.right
const PLOT_HEIGHT = GRAPH_HEIGHT - PADDING.top - PADDING.bottom

// Charts a team's final league position across seasons - lower (better) positions plot higher,
// so the line visually climbs when the team is doing well. Promoted/relegated seasons are
// highlighted with the same success/warning colors already used for their chips in the trophy
// cabinet table below, rather than introducing a separate categorical palette for a single
// series (see the dataviz skill: status colors are reserved for state, which promotion/
// relegation genuinely are here).
export function SeasonPositionChart({ history }: { history: SeasonHistoryEntry[] }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)

  const points = history
    .filter((entry): entry is SeasonHistoryEntry & { finalPosition: number } => entry.finalPosition !== null)
    .sort((a, b) => a.season - b.season)

  if (points.length === 0) {
    return null
  }

  const maxPosition = Math.max(...points.map((p) => p.finalPosition), 1)
  const positionSpan = Math.max(maxPosition - 1, 1)

  const xFor = (index: number) => (points.length === 1 ? PADDING.left + PLOT_WIDTH / 2 : PADDING.left + (index / (points.length - 1)) * PLOT_WIDTH)
  const yFor = (position: number) => PADDING.top + ((position - 1) / positionSpan) * PLOT_HEIGHT

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${xFor(i)} ${yFor(p.finalPosition)}`).join(' ')

  const yTicks = Array.from(new Set([1, Math.round((1 + maxPosition) / 2), maxPosition]))

  const hovered = hoverIndex !== null ? points[hoverIndex] : null

  const colorFor = (entry: SeasonHistoryEntry) =>
    entry.promoted ? theme.palette.success.main : entry.relegated ? theme.palette.warning.main : theme.palette.primary.main

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('team.chart.title')}
      </Typography>
      <Box sx={{ position: 'relative', width: '100%', maxWidth: GRAPH_WIDTH, aspectRatio: `${GRAPH_WIDTH} / ${GRAPH_HEIGHT}` }}>
        <svg
          viewBox={`0 0 ${GRAPH_WIDTH} ${GRAPH_HEIGHT}`}
          width="100%"
          height="100%"
          role="img"
          aria-label={t('team.chart.ariaLabel')}
        >
          {yTicks.map((tick) => (
            <g key={tick}>
              <line
                x1={PADDING.left}
                x2={GRAPH_WIDTH - PADDING.right}
                y1={yFor(tick)}
                y2={yFor(tick)}
                stroke={theme.palette.divider}
                strokeWidth={1}
              />
              <text x={PADDING.left - 8} y={yFor(tick)} textAnchor="end" dominantBaseline="middle" fontSize={10} fill={theme.palette.text.secondary}>
                {tick}
              </text>
            </g>
          ))}

          <path d={linePath} fill="none" stroke={theme.palette.primary.main} strokeWidth={2} />

          {points.map((p, i) => (
            <g key={p.season}>
              <text x={xFor(i)} y={GRAPH_HEIGHT - 8} textAnchor="middle" fontSize={10} fill={theme.palette.text.secondary}>
                {p.season}
              </text>
              {/* Generous invisible hit target around the small visible marker, easier to hover/tap precisely. */}
              <circle
                cx={xFor(i)}
                cy={yFor(p.finalPosition)}
                r={12}
                fill="transparent"
                onMouseEnter={() => setHoverIndex(i)}
                onMouseLeave={() => setHoverIndex((current) => (current === i ? null : current))}
                style={{ cursor: 'pointer' }}
              />
              <circle
                cx={xFor(i)}
                cy={yFor(p.finalPosition)}
                r={hoverIndex === i ? 6 : 4}
                fill={colorFor(p)}
                stroke={theme.palette.background.paper}
                strokeWidth={1.5}
                style={{ pointerEvents: 'none' }}
              />
            </g>
          ))}
        </svg>
        {hovered && hoverIndex !== null && (
          <Paper
            elevation={3}
            sx={{
              position: 'absolute',
              left: `${(xFor(hoverIndex) / GRAPH_WIDTH) * 100}%`,
              top: `${(yFor(hovered.finalPosition) / GRAPH_HEIGHT) * 100}%`,
              transform: 'translate(-50%, -125%)',
              p: 1,
              pointerEvents: 'none',
              whiteSpace: 'nowrap',
            }}
          >
            <Typography variant="caption" sx={{ display: 'block', fontWeight: 600 }}>
              {t('team.chart.tooltipSeason', { season: hovered.season, position: hovered.finalPosition })}
            </Typography>
            {hovered.leagueName && (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                {hovered.leagueName}
              </Typography>
            )}
            {hovered.promoted && (
              <Typography variant="caption" sx={{ display: 'block', color: 'success.main' }}>
                {t('team.promoted')}
              </Typography>
            )}
            {hovered.relegated && (
              <Typography variant="caption" sx={{ display: 'block', color: 'warning.main' }}>
                {t('team.relegated')}
              </Typography>
            )}
          </Paper>
        )}
      </Box>
    </Paper>
  )
}
