import FlightTakeoffIcon from '@mui/icons-material/FlightTakeoff'
import HomeOutlinedIcon from '@mui/icons-material/HomeOutlined'
import { useTranslation } from 'react-i18next'

/** Home / away marker as an icon (house / plane) with an accessible name - the old one-letter tag
 * ("V") clashed with "Victoria" in the form guide. */
export function HomeAwayIcon({ isHome, size = 16 }: { isHome: boolean; size?: number }) {
  const { t } = useTranslation()
  const label = t(isHome ? 'common.home' : 'common.away')
  const Icon = isHome ? HomeOutlinedIcon : FlightTakeoffIcon
  return <Icon role="img" aria-label={label} titleAccess={label} sx={{ fontSize: size, color: 'text.secondary', flexShrink: 0 }} />
}
