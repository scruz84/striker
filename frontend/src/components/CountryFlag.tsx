import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import { useTranslation } from 'react-i18next'
import { COUNTRY_NAME_KEYS } from '../api/types'

// Flags are SVG files in public/flags/<code>.svg named by the game's own country code (see
// public/flags/README.md), not emoji: Scotland and Wales have no portable flag emoji and Windows
// renders none at all. A code with no entry in COUNTRY_NAME_KEYS shows nothing.
interface CountryFlagProps {
  /** A backend country code (Player.birthCountry, Team.country). */
  code?: string
  /** Rendered height in px; the 4:3 flag's width follows. */
  height?: number
  /** Skip the country-name tooltip and alt text, for a flag shown right beside the name itself. */
  decorative?: boolean
  /** Space after the flag in theme spacing units, for a flag placed inline before a name. */
  marginRight?: number
}

export function CountryFlag({ code, height = 16, decorative = false, marginRight = 0 }: CountryFlagProps) {
  const { t } = useTranslation()
  if (!code || !COUNTRY_NAME_KEYS[code]) return null
  const name = t(COUNTRY_NAME_KEYS[code])
  const img = (
    <Box
      component="img"
      src={`/flags/${code}.svg`}
      alt={decorative ? '' : name}
      loading="lazy"
      decoding="async"
      sx={{
        display: 'inline-block',
        flexShrink: 0,
        height,
        width: (height * 4) / 3,
        borderRadius: '2px',
        // A hairline keeps white-dominated flags (Japan, Poland) visible on a light card.
        boxShadow: '0 0 0 1px rgba(128, 128, 128, 0.35)',
        verticalAlign: 'middle',
        mr: marginRight,
      }}
    />
  )
  return decorative ? img : <Tooltip title={name}>{img}</Tooltip>
}
