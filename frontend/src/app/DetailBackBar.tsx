import Box from '@mui/material/Box'
import ButtonBase from '@mui/material/ButtonBase'
import Typography from '@mui/material/Typography'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { DetailRoute } from './navigation'

/** Back bar for drill-down pages on a phone. Goes back in history when this page was reached from
 * inside the app, and to the section's parent when it was opened directly (a shared link, a
 * freshly launched PWA) so "back" never leaves the app. */
export function DetailBackBar({ route }: { route: DetailRoute }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()

  const goBack = () => {
    if (location.key === 'default') {
      navigate(route.fallback, { replace: true })
    } else {
      navigate(-1)
    }
  }

  return (
    <Box
      sx={{
        position: 'sticky',
        top: 0,
        zIndex: (theme) => theme.zIndex.appBar - 1,
        bgcolor: 'background.paper',
        borderBottom: 1,
        borderColor: 'divider',
        px: 0.5,
        display: 'flex',
        alignItems: 'center',
      }}
    >
      <ButtonBase
        onClick={goBack}
        aria-label={t('common.back')}
        sx={{ minHeight: 48, pl: 1, pr: 2, gap: 1, borderRadius: 24, justifyContent: 'flex-start' }}
      >
        <ArrowBackIcon />
        <Typography variant="subtitle1" component="span">
          {t(route.labelKey)}
        </Typography>
      </ButtonBase>
    </Box>
  )
}
