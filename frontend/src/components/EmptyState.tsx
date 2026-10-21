import type { ReactNode } from 'react'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  text?: string
  /** Optional call to action (a Button). */
  action?: ReactNode
  /** Smaller, inline variant for a section that is empty inside a populated page. */
  compact?: boolean
}

/** Shared "nothing here yet" block: icon, title, explanation and an optional call to action. A
 * page whose only content is an empty state should use this rather than a bare paragraph, so it
 * never looks like an error. */
export function EmptyState({ icon, title, text, action, compact = false }: EmptyStateProps) {
  return (
    <Stack spacing={1} sx={{ alignItems: 'center', textAlign: 'center', py: compact ? 2 : 6, px: 2, color: 'text.secondary' }}>
      {icon && <span aria-hidden="true" style={{ display: 'inline-flex', fontSize: compact ? 32 : 48, opacity: 0.7 }}>{icon}</span>}
      <Typography variant={compact ? 'subtitle2' : 'subtitle1'} component="h2" color="text.primary" sx={{ fontWeight: 600 }}>
        {title}
      </Typography>
      {text && (
        <Typography variant="body2" sx={{ maxWidth: 420 }}>
          {text}
        </Typography>
      )}
      {action}
    </Stack>
  )
}
