import type { SxProps, Theme } from '@mui/material/styles'

// The pulsing "LIVE" label shared by the match viewer's scoreboard and the dashboard's live hero.
export const PULSING_LIVE_SX: SxProps<Theme> = {
  fontWeight: 700,
  animation: 'pulse 1.6s ease-in-out infinite',
  '@keyframes pulse': {
    '0%, 100%': { opacity: 1 },
    '50%': { opacity: 0.55 },
  },
}
