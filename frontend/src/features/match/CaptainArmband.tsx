import Box from '@mui/material/Box'
import { useTranslation } from 'react-i18next'

/** The captain's "C" armband (see openspec match-viewer "Captain's armband"): a small badge next
 * to the captain in the match viewer's lineups pitch and ratings list. */
export function CaptainArmband({ size = 16 }: { size?: number }) {
  const { t } = useTranslation()
  return (
    <Box
      component="span"
      role="img"
      aria-label={t('match.captain')}
      title={t('match.captain')}
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: size,
        height: size,
        borderRadius: '4px',
        bgcolor: 'warning.main',
        color: 'warning.contrastText',
        border: '1px solid rgba(255,255,255,0.85)',
        fontSize: size * 0.62,
        fontWeight: 800,
        lineHeight: 1,
        flexShrink: 0,
      }}
    >
      C
    </Box>
  )
}
