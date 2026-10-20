import { useMemo, useState } from 'react'
import Autocomplete from '@mui/material/Autocomplete'
import Box from '@mui/material/Box'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import type { SxProps, Theme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { useSearch } from '../../../api/hooks'
import { useDebouncedValue } from '../../../utils/useDebouncedValue'

// A team the admin picked: its id (what the API wants) plus how to show it.
export interface PickedTeam {
  id: number
  /** Full text shown in the field once chosen, e.g. "Real Betis (#12)". */
  label: string
  /** The team's name alone, for messages. */
  name: string
  /** Secondary text in the dropdown (country and division), empty for an id typed by hand. */
  detail: string
}

interface TeamPickerProps {
  label: string
  value: PickedTeam | null
  onChange: (team: PickedTeam | null) => void
  sx?: SxProps<Theme>
}

const SEARCH_DEBOUNCE_MS = 300

// Lets an admin pick a team by searching for its name (the same search the global search page
// uses) instead of having to know its numeric id. Typing a plain number offers that id directly,
// so the old "I only have the id" workflow still works.
export function TeamPicker({ label, value, onChange, sx }: TeamPickerProps) {
  const { t } = useTranslation()
  const [input, setInput] = useState('')
  const debounced = useDebouncedValue(input, SEARCH_DEBOUNCE_MS)
  const { data, isFetching } = useSearch(debounced)

  const typedId = /^\d+$/.test(input.trim()) ? Number(input.trim()) : null

  const options = useMemo<PickedTeam[]>(() => {
    const found = (data?.teams ?? []).map((team) => ({
      id: team.id,
      name: team.name,
      label: `${team.name} (#${team.id})`,
      detail: [team.country.toUpperCase(), team.divisionName].filter(Boolean).join(' · '),
    }))
    if (typedId !== null && typedId > 0 && !found.some((team) => team.id === typedId)) {
      const byId = t('admin.teamPicker.teamNumber', { id: typedId })
      return [{ id: typedId, name: byId, label: byId, detail: '' }, ...found]
    }
    return found
  }, [data, typedId, t])

  return (
    <Autocomplete<PickedTeam>
      value={value}
      onChange={(_event, team) => onChange(team)}
      inputValue={input}
      onInputChange={(_event, text) => setInput(text)}
      options={options}
      loading={isFetching}
      // The server already filtered by the typed text; filtering again here would drop results that
      // match on something other than the label (a stadium or a former name).
      filterOptions={(all) => all}
      getOptionLabel={(team) => team.label}
      isOptionEqualToValue={(a, b) => a.id === b.id}
      noOptionsText={input.trim() ? t('admin.teamPicker.noResults') : t('admin.teamPicker.typeToSearch')}
      loadingText={t('admin.teamPicker.searching')}
      size="small"
      sx={sx}
      renderOption={(props, team) => {
        const { key, ...rest } = props
        return (
          <Box component="li" key={key} {...rest}>
            <Box sx={{ minWidth: 0 }}>
              <Typography variant="body2" sx={{ overflowWrap: 'anywhere' }}>
                {team.label}
              </Typography>
              {team.detail && (
                <Typography variant="caption" color="text.secondary">
                  {team.detail}
                </Typography>
              )}
            </Box>
          </Box>
        )
      }}
      renderInput={(params) => <TextField {...params} label={label} helperText={t('admin.teamPicker.helper')} />}
    />
  )
}
