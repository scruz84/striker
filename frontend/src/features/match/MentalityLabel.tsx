import Chip from '@mui/material/Chip'
import { useTranslation } from 'react-i18next'
import { MENTALITY_KEYS } from '../../api/types'

/** A side's mentality at kickoff (openspec/specs/match-viewer "Mentality in the lineups"), shown
 * with the lineups of a live or finished match - never before kickoff. Nothing when the lineup
 * carries none (a match played before mentality existed). */
export function MentalityLabel({ mentality }: { mentality?: number }) {
  const { t } = useTranslation()
  if (mentality === undefined || !(mentality in MENTALITY_KEYS)) return null
  return <Chip size="small" variant="outlined" label={t(`match.mentality.${MENTALITY_KEYS[mentality]}`)} sx={{ ml: 1 }} />
}
