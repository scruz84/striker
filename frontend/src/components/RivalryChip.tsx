import Chip from '@mui/material/Chip'
import WhatshotOutlinedIcon from '@mui/icons-material/WhatshotOutlined'
import { useTranslation } from 'react-i18next'
import { RIVALRY_CLASICO, RIVALRY_RIVAL } from '../api/types'

/** "Rival" / "Clásico" chip for a match or pair with a rivalry level (openspec rivalries "Rivalry
 * visibility"); nothing for a level of none. */
export function RivalryChip({ level, size = 'small' }: { level?: number; size?: 'small' | 'medium' }) {
  const { t } = useTranslation()
  if (level !== RIVALRY_RIVAL && level !== RIVALRY_CLASICO) {
    return null
  }
  const clasico = level === RIVALRY_CLASICO
  return (
    <Chip
      size={size}
      icon={<WhatshotOutlinedIcon />}
      color={clasico ? 'error' : 'warning'}
      variant={clasico ? 'filled' : 'outlined'}
      label={t(clasico ? 'rivalries.clasico' : 'rivalries.rival')}
      data-testid="rivalry-chip"
    />
  )
}
