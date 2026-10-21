import { useEffect, useState, type Ref } from 'react'
import Autocomplete from '@mui/material/Autocomplete'
import Box from '@mui/material/Box'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useSearch } from '../api/hooks'
import type { TeamSearchResult } from '../api/types'
import { formatDivisionName } from '../utils/divisionName'
import { TeamBadge } from './TeamBadge'

interface TeamAutocompleteProps {
  label: string
  value: TeamSearchResult | null
  onChange: (team: TeamSearchResult | null) => void
  /** A team that must not be offered (the user's own club). */
  excludeTeamId?: number
  inputRef?: Ref<HTMLInputElement>
}

/** Picks a team by typing part of its name - backed by the same `/v1/teams/search` the Search page
 * uses - instead of asking people to know an internal numeric id. Options show the crest, the
 * name and the division. */
export function TeamAutocomplete({ label, value, onChange, excludeTeamId, inputRef }: TeamAutocompleteProps) {
  const { t } = useTranslation()
  const [input, setInput] = useState('')
  const [debounced, setDebounced] = useState('')

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(input), 250)
    return () => window.clearTimeout(handle)
  }, [input])

  const { data, isFetching } = useSearch(debounced.length >= 2 ? debounced : '')
  const options = (data?.teams ?? []).filter((team) => team.id !== excludeTeamId)

  return (
    <Autocomplete
      value={value}
      onChange={(_event, team) => onChange(team)}
      inputValue={input}
      onInputChange={(_event, text) => setInput(text)}
      options={options}
      loading={isFetching}
      filterOptions={(all) => all}
      isOptionEqualToValue={(a, b) => a.id === b.id}
      getOptionLabel={(team) => team.name}
      noOptionsText={input.length < 2 ? t('teamAutocomplete.hint') : t('teamAutocomplete.none')}
      loadingText={t('common.loading')}
      renderOption={(props, team) => {
        const { key, ...rest } = props as typeof props & { key: string }
        return (
          <Box component="li" key={key} {...rest} sx={{ gap: 1.5, minHeight: 48 }}>
            <TeamBadge teamId={team.id} teamName={team.name} size={28} />
            <Box sx={{ minWidth: 0 }}>
              <Typography variant="body2" noWrap>
                {team.name}
              </Typography>
              <Typography variant="caption" color="text.secondary" noWrap>
                {formatDivisionName(t, team.divisionName)}
              </Typography>
            </Box>
          </Box>
        )
      }}
      renderInput={(params) => <TextField {...params} label={label} inputRef={inputRef} />}
      fullWidth
    />
  )
}
