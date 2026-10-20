import type { FormEvent, ReactNode } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'

interface SettingCardProps {
  title: string
  description?: string
  submitLabel: string
  onSubmit: () => void
  /** True while the request is in flight: the button is disabled so it can't be double-submitted. */
  pending?: boolean
  /** True when the current input is not valid to submit. */
  invalid?: boolean
  /** An explanation of why the input is invalid, shown under the fields. */
  errorText?: string
  color?: 'primary' | 'warning' | 'error'
  /** The input fields. They are laid out in a row from `sm` up and stacked below that. */
  children: ReactNode
}

// A single admin setting: a title, a short description, its input field(s) and a save button, as
// one self-contained form. Replaces the near-identical forms the admin page used to repeat; on a
// phone the fields stack and the button spans the full width instead of overflowing a fixed-width row.
export function SettingCard({
  title,
  description,
  submitLabel,
  onSubmit,
  pending = false,
  invalid = false,
  errorText,
  color = 'primary',
  children,
}: SettingCardProps) {
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!invalid && !pending) {
      onSubmit()
    }
  }

  return (
    <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
      <Box component="form" onSubmit={handleSubmit} noValidate>
        <Typography variant="subtitle1" gutterBottom>
          {title}
        </Typography>
        {description && (
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
            {description}
          </Typography>
        )}
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ alignItems: { xs: 'stretch', sm: 'flex-start' }, flexWrap: 'wrap', rowGap: 1.5 }}>
          {children}
          <Button type="submit" variant="contained" color={color} disabled={invalid || pending} sx={{ alignSelf: { xs: 'stretch', sm: 'flex-start' }, minHeight: 40 }}>
            {submitLabel}
          </Button>
        </Stack>
        {errorText && (
          <Typography variant="caption" color="error" component="p" sx={{ mt: 0.5 }}>
            {errorText}
          </Typography>
        )}
      </Box>
    </Paper>
  )
}
