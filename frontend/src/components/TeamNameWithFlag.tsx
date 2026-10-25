import Box from '@mui/material/Box'
import { useTranslation } from 'react-i18next'
import { localizedTeamName, nationalTeamCountry } from '../utils/nationalTeamName'
import { CountryFlag } from './CountryFlag'

/** A team's name as stored, localized when it is a national team's, with that country's flag in
 * front. An ordinary club name renders as plain text - a club row carries no country here. */
export function TeamNameWithFlag({ rawName, flagHeight = 14 }: { rawName: string; flagHeight?: number }) {
  const { t } = useTranslation()
  const text = localizedTeamName(t, rawName)
  const code = nationalTeamCountry(rawName)
  if (!code) return <>{text}</>
  return (
    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, verticalAlign: 'middle' }}>
      <CountryFlag code={code} height={flagHeight} decorative />
      <span>{text}</span>
    </Box>
  )
}
