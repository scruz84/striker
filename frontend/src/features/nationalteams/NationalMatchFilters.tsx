import Chip from '@mui/material/Chip'
import FormControlLabel from '@mui/material/FormControlLabel'
import Stack from '@mui/material/Stack'
import Switch from '@mui/material/Switch'
import { useTranslation } from 'react-i18next'
import { SearchBar } from '../../components/m3/SearchBar'
import { NATIONAL_REGIONS, type NationalMatchFilter } from './nationalMatches'

/** Country search, region chips and (when the viewer owns a club) the "only my players" toggle -
 * shared by the Jornada and Resultados tabs. */
export function NationalMatchFilters({
  filter,
  onChange,
  showMineToggle,
  showRegions = true,
}: {
  filter: NationalMatchFilter
  onChange: (filter: NationalMatchFilter) => void
  showMineToggle: boolean
  showRegions?: boolean
}) {
  const { t } = useTranslation()
  return (
    <Stack spacing={1}>
      <SearchBar compact value={filter.query} onChange={(query) => onChange({ ...filter, query })} placeholder={t('nationalTeams.filters.search')} />
      {showRegions && (
        <Stack direction="row" sx={{ flexWrap: 'wrap', gap: 0.75 }}>
          <Chip
            label={t('nationalTeams.filters.allRegions')}
            color={filter.region === null ? 'primary' : 'default'}
            variant={filter.region === null ? 'filled' : 'outlined'}
            onClick={() => onChange({ ...filter, region: null })}
          />
          {NATIONAL_REGIONS.map((region) => (
            <Chip
              key={region}
              label={t(`nationalTeams.regions.${region}`)}
              color={filter.region === region ? 'primary' : 'default'}
              variant={filter.region === region ? 'filled' : 'outlined'}
              onClick={() => onChange({ ...filter, region: filter.region === region ? null : region })}
            />
          ))}
        </Stack>
      )}
      {showMineToggle && (
        <FormControlLabel
          control={<Switch checked={filter.onlyMine} onChange={(event) => onChange({ ...filter, onlyMine: event.target.checked })} />}
          label={t('nationalTeams.filters.onlyMine')}
        />
      )}
    </Stack>
  )
}
