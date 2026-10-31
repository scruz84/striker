import Chip from '@mui/material/Chip'
import FormControlLabel from '@mui/material/FormControlLabel'
import InputAdornment from '@mui/material/InputAdornment'
import Stack from '@mui/material/Stack'
import Switch from '@mui/material/Switch'
import TextField from '@mui/material/TextField'
import SearchIcon from '@mui/icons-material/Search'
import { useTranslation } from 'react-i18next'
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
      <TextField
        size="small"
        fullWidth
        value={filter.query}
        onChange={(event) => onChange({ ...filter, query: event.target.value })}
        placeholder={t('nationalTeams.filters.search')}
        slotProps={{
          htmlInput: { 'aria-label': t('nationalTeams.filters.search') },
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          },
        }}
      />
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
