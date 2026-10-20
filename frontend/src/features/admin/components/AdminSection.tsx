import type { ReactNode } from 'react'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'

interface AdminSectionProps {
  title?: string
  description?: string
  /** 'danger' outlines the section in the error color, for irreversible actions. */
  tone?: 'default' | 'danger'
  children: ReactNode
}

// One block of the admin panel: a titled Paper with the spacing every section used to repeat by
// hand (`Paper sx={{ p: 2, mb: 2 }}` + a subtitle + a muted description).
export function AdminSection({ title, description, tone = 'default', children }: AdminSectionProps) {
  const danger = tone === 'danger'
  return (
    <Paper variant={danger ? 'outlined' : 'elevation'} sx={{ p: 2, mb: 2, ...(danger && { borderColor: 'error.main' }) }}>
      {title && (
        <Typography variant="subtitle1" color={danger ? 'error' : 'text.primary'} gutterBottom>
          {title}
        </Typography>
      )}
      {description && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {description}
        </Typography>
      )}
      {children}
    </Paper>
  )
}
