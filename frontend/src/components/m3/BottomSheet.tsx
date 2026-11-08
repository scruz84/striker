import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import Drawer from '@mui/material/Drawer'
import IconButton from '@mui/material/IconButton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import CloseIcon from '@mui/icons-material/Close'
import { useTranslation } from 'react-i18next'

interface BottomSheetProps {
  open: boolean
  onClose: () => void
  /** Shown with a close button at the top of the sheet. */
  title?: string
  /** Fixed at the bottom (buttons), kept clear of the bottom safe area. */
  footer?: ReactNode
  children: ReactNode
}

/** M3 modal bottom sheet: surfaceContainerLow, 28px top corners and a drag handle, at most 640px
 * wide and 90% of the window tall (see the ui-theme spec). The handle is a visual cue only; the
 * backdrop, Escape and the close button dismiss it. */
export function BottomSheet({ open, onClose, title, footer, children }: BottomSheetProps) {
  const { t } = useTranslation()
  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          sx: {
            bgcolor: 'surfaceContainerLow',
            borderTopLeftRadius: 28,
            borderTopRightRadius: 28,
            maxHeight: '90dvh',
            maxWidth: 640,
            width: '100%',
            mx: 'auto',
          },
        },
      }}
    >
      <Box
        className="bottom-sheet-handle"
        aria-hidden="true"
        sx={{ width: 32, height: 4, borderRadius: '2px', bgcolor: 'onSurfaceVariant', opacity: 0.4, mx: 'auto', mt: 2, mb: title ? 0 : 1 }}
      />
      {title && (
        <Stack direction="row" sx={{ alignItems: 'center', px: 2, pt: 1, pb: 0.5 }}>
          <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, flex: 1 }}>
            {title}
          </Typography>
          <IconButton onClick={onClose} aria-label={t('common.close')} edge="end">
            <CloseIcon />
          </IconButton>
        </Stack>
      )}
      <Box sx={{ overflowY: 'auto', px: 2, pb: 2 }}>{children}</Box>
      {footer && (
        <Stack direction="row" spacing={1} sx={{ p: 2, pb: 'max(16px, env(safe-area-inset-bottom))', borderTop: 1, borderColor: 'outlineVariant' }}>
          {footer}
        </Stack>
      )}
    </Drawer>
  )
}
