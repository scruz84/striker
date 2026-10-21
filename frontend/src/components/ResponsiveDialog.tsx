import type { ReactNode } from 'react'
import AppBar from '@mui/material/AppBar'
import Box from '@mui/material/Box'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import IconButton from '@mui/material/IconButton'
import Toolbar from '@mui/material/Toolbar'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import CloseIcon from '@mui/icons-material/Close'
import { useTranslation } from 'react-i18next'

interface ResponsiveDialogProps {
  open: boolean
  onClose: () => void
  /** One short line ("Renovar contrato") - the subject goes in `subtitle`. */
  title: string
  /** Who/what the dialog is about ("Adrian Marcelo Manteca"). */
  subtitle?: string
  children: ReactNode
  /** Buttons for the action bar: fixed at the bottom of the screen on a phone, so they stay
   * visible with the on-screen keyboard open. */
  actions?: ReactNode
  maxWidth?: 'xs' | 'sm' | 'md'
}

/** A dialog that takes over the whole screen on a phone (app bar with a close button and a
 * single-line title, scrollable body, fixed action bar) and is the usual centered modal from the
 * `sm` breakpoint up. Every dialog in the app goes through this so they behave alike. */
export function ResponsiveDialog({ open, onClose, title, subtitle, children, actions, maxWidth = 'xs' }: ResponsiveDialogProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const fullScreen = useMediaQuery(theme.breakpoints.down('sm'))

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullScreen={fullScreen}
      maxWidth={maxWidth}
      fullWidth
      slotProps={{ paper: { sx: fullScreen ? { height: '100dvh', maxHeight: '100dvh', borderRadius: 0 } : undefined } }}
    >
      {fullScreen ? (
        <AppBar position="static" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}>
          <Toolbar sx={{ gap: 0.5, minHeight: 56 }}>
            <IconButton edge="start" onClick={onClose} aria-label={t('common.close')}>
              <CloseIcon />
            </IconButton>
            <Box sx={{ minWidth: 0, flexGrow: 1 }}>
              <Typography variant="subtitle1" component="h2" noWrap sx={{ fontWeight: 600 }}>
                {title}
              </Typography>
              {subtitle && (
                <Typography variant="body2" color="text.secondary" noWrap>
                  {subtitle}
                </Typography>
              )}
            </Box>
          </Toolbar>
        </AppBar>
      ) : (
        <DialogTitle>
          {title}
          {subtitle && (
            <Typography variant="body2" color="text.secondary" component="div">
              {subtitle}
            </Typography>
          )}
        </DialogTitle>
      )}
      <DialogContent sx={{ pt: { xs: 2, sm: 1 } }}>{children}</DialogContent>
      {actions && (
        <DialogActions
          sx={{
            gap: 1,
            ...(fullScreen && {
              borderTop: 1,
              borderColor: 'divider',
              px: 2,
              py: 1.5,
              pb: 'max(12px, env(safe-area-inset-bottom))',
              '& > :not(style)': { flex: 1, m: 0 },
            }),
          }}
        >
          {actions}
        </DialogActions>
      )}
    </Dialog>
  )
}
