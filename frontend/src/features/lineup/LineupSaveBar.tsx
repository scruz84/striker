import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined'
import { useTranslation } from 'react-i18next'
import { BOTTOM_NAV_HEIGHT } from '../../app/BottomNav'

/** The save action, pinned to the bottom of the viewport (above the phone bottom bar) so a lineup
 * can be saved without scrolling past the bench and substitutions - with a clear "unsaved changes"
 * status next to it. */
export function LineupSaveBar({
  dirty,
  canSave,
  saving,
  saved,
  hint,
  onSave,
}: {
  dirty: boolean
  canSave: boolean
  saving: boolean
  saved: boolean
  /** Why saving is not possible yet (shown when `canSave` is false). */
  hint?: string
  onSave: () => void
}) {
  const { t } = useTranslation()

  return (
    <Paper
      elevation={4}
      sx={{
        position: 'sticky',
        bottom: { xs: `calc(${BOTTOM_NAV_HEIGHT}px + env(safe-area-inset-bottom))`, md: 0 },
        zIndex: (theme) => theme.zIndex.appBar - 2,
        mt: 2,
        px: 2,
        py: 1,
        borderRadius: { xs: 0, md: 3 },
        borderTop: 1,
        borderColor: 'divider',
        mx: { xs: -2, md: 0 },
      }}
    >
      <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', minWidth: 0 }} aria-live="polite">
          {dirty ? (
            <>
              <span aria-hidden="true" style={{ width: 8, height: 8, borderRadius: '50%', background: 'currentColor', flexShrink: 0 }} />
              <Typography variant="body2" sx={{ fontWeight: 600 }} noWrap>
                {t('lineup.unsavedChanges')}
              </Typography>
            </>
          ) : saved ? (
            <>
              <CheckCircleOutlinedIcon fontSize="small" color="success" />
              <Typography variant="body2" color="text.secondary" noWrap>
                {t('lineup.savedStatus')}
              </Typography>
            </>
          ) : (
            <Typography variant="body2" color="text.secondary" noWrap>
              {!canSave && hint ? hint : t('lineup.noChanges')}
            </Typography>
          )}
        </Stack>
        <Button variant="contained" disabled={!canSave || saving || !dirty} onClick={onSave} sx={{ flexShrink: 0 }}>
          {t('lineup.save')}
        </Button>
      </Stack>
      {dirty && !canSave && hint && (
        <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
          {hint}
        </Typography>
      )}
    </Paper>
  )
}
