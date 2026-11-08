import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'

interface StatTileProps {
  icon?: ReactNode
  label: string
  value: ReactNode
  /** A meaning colour for the value (e.g. error red for a negative budget). */
  valueColor?: string
  /** Makes the whole tile one link. */
  to?: string
}

/** M3 stat tile: a glanceable figure on a filled card, icon and label quiet in onSurfaceVariant,
 * the value in the `stat` type style (see the ui-theme spec). */
export function StatTile({ icon, label, value, valueColor, to }: StatTileProps) {
  const content = (
    <>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mb: 0.5, color: 'onSurfaceVariant' }}>
        {icon && (
          <Box aria-hidden="true" sx={{ display: 'inline-flex', '& svg': { fontSize: 20, color: 'inherit' } }}>
            {icon}
          </Box>
        )}
        <Typography variant="overline" sx={{ lineHeight: 1, color: 'inherit' }}>
          {label}
        </Typography>
      </Stack>
      <Typography variant="stat" component="div" sx={{ fontSize: '1.5rem', color: valueColor ?? 'onSurface' }}>
        {value}
      </Typography>
    </>
  )
  const body = { height: '100%', p: 1.5, minHeight: 80 }

  return (
    <Card variant="filled" sx={{ height: '100%' }}>
      {to ? (
        <CardActionArea component={RouterLink} to={to} sx={body}>
          {content}
        </CardActionArea>
      ) : (
        <Box sx={body}>{content}</Box>
      )}
    </Card>
  )
}
