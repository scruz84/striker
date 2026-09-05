import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'
import { useTranslation } from 'react-i18next'
import { useThemeMode } from './themeMode'

export function ThemeModeToggle() {
  const { t } = useTranslation()
  const { mode, toggleMode } = useThemeMode()

  return (
    <Tooltip title={mode === 'dark' ? t('header.lightMode') : t('header.darkMode')}>
      <IconButton onClick={toggleMode} color="inherit">
        {mode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}
      </IconButton>
    </Tooltip>
  )
}
