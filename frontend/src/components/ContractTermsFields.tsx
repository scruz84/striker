import Chip from '@mui/material/Chip'
import MenuItem from '@mui/material/MenuItem'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { useTranslation } from 'react-i18next'
import { MAX_CONTRACT_YEARS_REMAINING } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { NumberField } from './NumberField'

interface ContractTermsFieldsProps {
  wage: string
  years: string
  onWageChange: (digits: string) => void
  onYearsChange: (years: string) => void
  /** The player's current wage - shown as context and as the base for the quick +% buttons. */
  currentWage?: number
}

/** The wage + duration pair shared by the renewal, free-agent signing and transfer-offer dialogs:
 * a numeric money field with a visible label, the current wage as context and one-tap raises. */
export function ContractTermsFields({ wage, years, onWageChange, onYearsChange, currentWage }: ContractTermsFieldsProps) {
  const { t } = useTranslation()
  const base = currentWage && currentWage > 0 ? currentWage : undefined

  return (
    <>
      <NumberField
        label={t('team.contractDialog.wage')}
        value={wage}
        onValueChange={onWageChange}
        currency
        fullWidth
        helperText={base ? t('team.contractDialog.currentWage', { wage: formatCurrency(base) }) : undefined}
      />
      {base && (
        <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', rowGap: 1 }}>
          {[0, 10, 25].map((percent) => (
            <Chip
              key={percent}
              size="small"
              variant="outlined"
              label={percent === 0 ? t('team.contractDialog.sameWage') : `+${percent}%`}
              onClick={() => onWageChange(String(Math.round(base * (1 + percent / 100))))}
            />
          ))}
        </Stack>
      )}
      <TextField select label={t('team.contractDialog.years')} value={years} onChange={(e) => onYearsChange(e.target.value)} fullWidth>
        {Array.from({ length: MAX_CONTRACT_YEARS_REMAINING }, (_, i) => i + 1).map((y) => (
          <MenuItem key={y} value={y}>
            {t('team.contractDialog.yearsOption', { count: y })}
          </MenuItem>
        ))}
      </TextField>
    </>
  )
}
