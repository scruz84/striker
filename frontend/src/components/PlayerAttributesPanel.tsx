import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { SKILL_FIELDS, ratingTierColor } from '../utils/playerRating'

// Radar geometry, in the SVG's own viewBox units - R leaves enough margin (viewBox width minus
// cx minus R*labelFactor) for the longest label ("Goalkeeper 99") to fit without clipping at
// either edge, on every axis angle a 7-gon produces.
const VIEW_W = 300
const VIEW_H = 220
const CX = 150
const CY = 112
const R = 62
const LABEL_FACTOR = 1.22
const GRID_RINGS = [1 / 3, 2 / 3, 1]

function polarPoint(index: number, count: number, frac: number): [number, number] {
  const angle = -Math.PI / 2 + (index * 2 * Math.PI) / count
  return [CX + Math.cos(angle) * R * frac, CY + Math.sin(angle) * R * frac]
}

/** The player details page's attribute visualization - a radar chart for the whole seven-skill
 * shape at a glance (a keeper's strong-shot/weak-dribbling profile reads instantly as a shape),
 * paired with the same values as tier-colored bars for exact numbers. */
export function PlayerAttributesPanel({ player }: { player: Player }) {
  const { t } = useTranslation()
  const theme = useTheme()

  const skills = SKILL_FIELDS.map((field) => ({
    field,
    label: t(`playerCard.skills.${field === 'goalKeeper' ? 'goalkeeper' : field}`),
    value: player[field],
  }))
  const n = skills.length

  const shapePoints = skills.map((s, i) => polarPoint(i, n, s.value / 100).join(',')).join(' ')

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2.5 } }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: { xs: 1, sm: 2 } }}>
        {t('playerPage.attributes.title')}
      </Typography>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={{ xs: 1, md: 3 }} sx={{ alignItems: 'center' }}>
        {/* The radar chart repeats exactly what the bars below it already show, just less legibly
            at phone width - dropping it there (bars alone, as a compact list) keeps the same
            information in a fraction of the vertical space; there's room for both side by side
            from md up. */}
        <Box
          component="svg"
          viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
          sx={{ display: { xs: 'none', md: 'block' }, width: '100%', maxWidth: 300, flexShrink: 0 }}
        >
          {GRID_RINGS.map((frac) => (
            <polygon
              key={frac}
              points={skills.map((_, i) => polarPoint(i, n, frac).join(',')).join(' ')}
              fill="none"
              stroke={theme.palette.divider}
            />
          ))}
          {skills.map((s, i) => {
            const [x, y] = polarPoint(i, n, 1)
            return <line key={s.field} x1={CX} y1={CY} x2={x} y2={y} stroke={theme.palette.divider} />
          })}
          <polygon points={shapePoints} fill={alpha(theme.palette.primary.main, 0.22)} stroke={theme.palette.primary.main} strokeWidth={2} />
          {skills.map((s, i) => {
            const [x, y] = polarPoint(i, n, s.value / 100)
            return <circle key={s.field} cx={x} cy={y} r={2.6} fill={theme.palette.primary.main} />
          })}
          {skills.map((s, i) => {
            const [x, y] = polarPoint(i, n, LABEL_FACTOR)
            const anchor = Math.abs(x - CX) < 4 ? 'middle' : x > CX ? 'start' : 'end'
            return (
              <text
                key={s.field}
                x={x}
                y={y}
                fontSize={9.5}
                fill={theme.palette.text.secondary}
                textAnchor={anchor}
                dominantBaseline="middle"
              >
                {s.label} {s.value}
              </text>
            )
          })}
        </Box>

        <Stack spacing={{ xs: 0.75, sm: 1.25 }} sx={{ flex: 1, width: '100%' }}>
          {skills.map((s) => (
            <SkillTierBar key={s.field} label={s.label} value={s.value} color={ratingTierColor(s.value, theme.palette)} />
          ))}
        </Stack>
      </Stack>
    </Paper>
  )
}

function SkillTierBar({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <Stack direction="row" sx={{ alignItems: 'center', gap: { xs: 1, sm: 1.5 } }}>
      <Typography variant="body2" noWrap sx={{ width: { xs: 76, sm: 100 }, flexShrink: 0 }}>
        {label}
      </Typography>
      <Box sx={{ flex: 1, height: 6, borderRadius: 1, bgcolor: 'action.hover', overflow: 'hidden' }}>
        <Box sx={{ display: 'block', width: `${Math.min(value, 100)}%`, height: '100%', bgcolor: color, borderRadius: 1 }} />
      </Box>
      <Typography variant="body2" color="text.secondary" sx={{ width: 24, textAlign: 'right', flexShrink: 0, fontWeight: 600 }}>
        {value}
      </Typography>
    </Stack>
  )
}
