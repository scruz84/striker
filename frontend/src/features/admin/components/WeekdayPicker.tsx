import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import { useTranslation } from 'react-i18next'
import { WEEKDAY_OPTIONS } from '../../../api/types'
import { bitsFromMask, maskFromBits } from '../weekdayMask'

interface WeekdayPickerProps {
  /** Bitmask of the selected weekdays (see WEEKDAY_OPTIONS). */
  value: number
  onChange: (mask: number) => void
}

// The row of weekday toggles used by the match schedule, the friendly schedule and each league's
// own schedule. On a phone it spans the width with equally sized buttons so all seven fit.
export function WeekdayPicker({ value, onChange }: WeekdayPickerProps) {
  const { t } = useTranslation()
  return (
    <ToggleButtonGroup
      size="small"
      value={bitsFromMask(value)}
      onChange={(_event, bits: number[]) => onChange(maskFromBits(bits))}
      sx={{ width: { xs: '100%', sm: 'auto' }, '& .MuiToggleButton-root': { flex: { xs: 1, sm: 'none' }, px: { xs: 0.5, sm: 1.5 } } }}
    >
      {WEEKDAY_OPTIONS.map((d) => (
        <ToggleButton key={d.bit} value={d.bit}>
          {t(`common.weekdaysShort.${d.key}`)}
        </ToggleButton>
      ))}
    </ToggleButtonGroup>
  )
}
