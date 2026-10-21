import { useState } from 'react'
import IconButton from '@mui/material/IconButton'
import ListItemText from '@mui/material/ListItemText'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Tooltip from '@mui/material/Tooltip'
import TranslateIcon from '@mui/icons-material/Translate'
import { useTranslation } from 'react-i18next'
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from '../i18n'
import { useChangeLanguage } from './useChangeLanguage'

export function LanguageSwitcher() {
  const { t } = useTranslation()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const { currentLanguage, changeLanguage } = useChangeLanguage()

  const handleSelect = (language: SupportedLanguage) => {
    changeLanguage(language)
    setAnchorEl(null)
  }

  return (
    <>
      <Tooltip title={t('header.language')}>
        <IconButton onClick={(event) => setAnchorEl(event.currentTarget)} color="inherit" aria-label={t('header.language')}>
          <TranslateIcon />
        </IconButton>
      </Tooltip>
      <Menu
        sx={{ mt: '45px' }}
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        {(Object.keys(SUPPORTED_LANGUAGES) as SupportedLanguage[]).map((language) => (
          <MenuItem key={language} selected={language === currentLanguage} onClick={() => handleSelect(language)}>
            <ListItemText>{SUPPORTED_LANGUAGES[language]}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </>
  )
}
