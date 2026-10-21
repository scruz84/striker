import { useState, type ReactNode } from 'react'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import CloseIcon from '@mui/icons-material/Close'
import TuneIcon from '@mui/icons-material/Tune'
import Badge from '@mui/material/Badge'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Drawer from '@mui/material/Drawer'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import Link from '@mui/material/Link'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  COUNTRY_NAME_KEYS,
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_MIDFIELD,
  POSITION_SHORT_LABEL_KEYS,
  POSITION_STRIKER,
  type Player,
} from '../../api/types'
import { NumberField } from '../../components/NumberField'
import { SkillChipsRow } from '../../components/SkillChipsRow'
import { countryFlag } from '../../utils/countryFlag'
import { SKILL_FIELDS, SKILL_LABEL_KEYS, bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'

export type SortDirection = 'asc' | 'desc'

// The four core positions bestCorePosition ever returns - offered, in this order, by the
// market's "search by suggested position" filter on every browsable listing tab.
export const CORE_POSITIONS = [POSITION_GOALKEEPER, POSITION_DEFENSE, POSITION_MIDFIELD, POSITION_STRIKER]

// usePersistedFilter keeps one search-bar field in the page's URL query string instead of plain
// component state. Every listing card links to that player's own page (PlayerCell/ListingCard),
// and clicking "back" from there remounts this whole tab fresh - a plain useState would lose
// whatever was typed/selected the instant that happens, since nothing survives the remount except
// what's in the URL the browser is navigating back to. Writes replace the current history entry
// (rather than pushing a new one), so typing in a filter field doesn't turn every keystroke into
// its own "back" stop - only the final state before navigating away ends up in the URL.
export function usePersistedFilter<T extends string>(key: string, defaultValue: T): [T, (value: T) => void] {
  const [searchParams, setSearchParams] = useSearchParams()
  const value = (searchParams.get(key) as T | null) ?? defaultValue
  const setValue = (next: T) => {
    setSearchParams(
      (prev) => {
        const params = new URLSearchParams(prev)
        if (next === defaultValue) {
          params.delete(key)
        } else {
          params.set(key, next)
        }
        return params
      },
      { replace: true },
    )
  }
  return [value, setValue]
}

// A player's name (shown in full, wrapping onto a second line rather than being cut to "Lucio
// Jose…") plus two quick-reference badges - their suggested position (see bestCorePosition) and
// average skill rating - so browsing a listing doesn't require opening each player's profile just
// to see roughly what they're worth. formerClub renders as a caption underneath, for a free agent
// who was recently released (see FreeAgentListing).
export function PlayerCell({ player, formerClub }: { player: Player; formerClub?: string }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const rating = overallRating(player)
  const flag = countryFlag(player.birthCountry)
  return (
    <Stack spacing={0.125} sx={{ minWidth: 0, flex: 1 }}>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'flex-start' }}>
        {flag && (
          <Tooltip title={t(COUNTRY_NAME_KEYS[player.birthCountry!] ?? player.birthCountry!)}>
            <Typography component="span" sx={{ flexShrink: 0, fontSize: '1rem', lineHeight: 1.4 }}>
              {flag}
            </Typography>
          </Tooltip>
        )}
        <Link
          component={RouterLink}
          to={`/player/${player.id}`}
          underline="hover"
          color="inherit"
          sx={{ flex: '1 1 auto', minWidth: 0, fontWeight: 600, overflowWrap: 'anywhere', lineHeight: 1.4 }}
        >
          {player.firstName} {player.lastName}
        </Link>
        <Chip
          size="small"
          variant="outlined"
          translate="no"
          className="notranslate"
          label={t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(player)])}
          sx={{ height: 20, flexShrink: 0 }}
        />
        <Typography variant="subtitle2" sx={{ fontWeight: 700, flexShrink: 0, color: ratingTierColor(rating, theme.palette) }}>
          {rating}
        </Typography>
      </Stack>
      {formerClub && (
        <Typography variant="caption" color="text.secondary">
          {formerClub}
        </Typography>
      )}
    </Stack>
  )
}

// A "·" separator for the meta line under a listing card's skill strip - plain text items are
// passed as children rather than a join() on strings so callers can mix in links (TeamNameLink)
// and translated fragments freely.
export function MetaDot() {
  return (
    <Typography component="span" variant="caption" color="text.secondary" sx={{ opacity: 0.6 }}>
      ·
    </Typography>
  )
}

/** A labelled figure ("Edad 22") for the meta line - every number carries its label. */
export function MetaItem({ children }: { children: ReactNode }) {
  return (
    <Typography variant="caption" color="text.secondary">
      {children}
    </Typography>
  )
}

// The listing card every browsable market tab renders one of per row. Top to bottom: who (full
// name, position, rating), the skill strip, a labelled meta line, then a bottom row with the
// price on the left and the actions on the right at a comfortable 44px height - so every
// attribute is visible at every viewport width, without a table's columns disappearing on mobile.
export function ListingCard({
  player,
  formerClub,
  badge,
  meta,
  priceLabel,
  priceValue,
  salary,
  extra,
  actions,
}: {
  player: Player
  formerClub?: string
  badge?: ReactNode
  meta?: ReactNode
  priceLabel: string
  priceValue: ReactNode
  /** "Salario 206 €/partido" - on the left of the bottom row, under the price. */
  salary?: ReactNode
  extra?: ReactNode
  actions: ReactNode
}) {
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'flex-start' }}>
        <PlayerCell player={player} formerClub={formerClub} />
        {badge}
      </Stack>
      <SkillChipsRow player={player} />
      {meta && (
        <Stack direction="row" spacing={0.75} sx={{ flexWrap: 'wrap', alignItems: 'center', rowGap: 0.25, mt: 0.75 }}>
          {meta}
        </Stack>
      )}
      {extra}
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', mt: 1 }}>
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {priceLabel}
          </Typography>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
            {priceValue}
          </Typography>
          {salary && (
            <Typography variant="caption" color="text.secondary" component="div">
              {salary}
            </Typography>
          )}
        </Box>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexShrink: 0 }}>
          {actions}
        </Stack>
      </Stack>
    </Paper>
  )
}

/** Search box + "Filters (n)" button: the only filter UI on screen until the sheet is opened. */
export function FilterBar({
  search,
  onSearch,
  activeCount,
  onOpenFilters,
}: {
  search: string
  onSearch: (value: string) => void
  activeCount: number
  onOpenFilters: () => void
}) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
      <TextField
        label={t('transfers.filters.search')}
        value={search}
        onChange={(event) => onSearch(event.target.value)}
        fullWidth
        type="search"
      />
      <Badge badgeContent={activeCount} color="primary" sx={{ flexShrink: 0 }}>
        <Button variant="outlined" startIcon={<TuneIcon />} onClick={onOpenFilters}>
          {t('transfers.filters.button')}
        </Button>
      </Badge>
    </Stack>
  )
}

export interface ActiveFilter {
  key: string
  label: string
  onDelete: () => void
}

/** The filters currently narrowing the list, as removable chips under the search bar. */
export function ActiveFilterChips({ filters }: { filters: ActiveFilter[] }) {
  if (filters.length === 0) {
    return null
  }
  return (
    <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', rowGap: 1, mb: 1.5 }}>
      {filters.map((filter) => (
        <Chip key={filter.key} size="small" label={filter.label} onDelete={filter.onDelete} />
      ))}
    </Stack>
  )
}

/** Bottom sheet holding every filter and the sort order of a market tab. */
export function FilterSheet({
  open,
  onClose,
  resultCount,
  onClear,
  hasActiveFilters,
  children,
}: {
  open: boolean
  onClose: () => void
  resultCount: number
  onClear: () => void
  hasActiveFilters: boolean
  children: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      slotProps={{ paper: { sx: { borderTopLeftRadius: 20, borderTopRightRadius: 20, maxHeight: '90dvh', maxWidth: 560, mx: 'auto', width: '100%' } } }}
    >
      <Stack direction="row" sx={{ alignItems: 'center', px: 2, pt: 1, pb: 0.5 }}>
        <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, flex: 1 }}>
          {t('transfers.filters.title')}
        </Typography>
        <IconButton onClick={onClose} aria-label={t('common.close')} edge="end">
          <CloseIcon />
        </IconButton>
      </Stack>
      <Box sx={{ overflowY: 'auto', px: 2, pb: 2 }}>
        <Stack spacing={2.5} sx={{ pt: 1 }}>
          {children}
        </Stack>
      </Box>
      <Stack
        direction="row"
        spacing={1}
        sx={{ p: 2, pb: 'max(16px, env(safe-area-inset-bottom))', borderTop: 1, borderColor: 'divider' }}
      >
        <Button onClick={onClear} disabled={!hasActiveFilters}>
          {t('transfers.filters.clear')}
        </Button>
        <Button variant="contained" onClick={onClose} sx={{ flex: 1 }}>
          {t('transfers.filters.showResults', { count: resultCount })}
        </Button>
      </Stack>
    </Drawer>
  )
}

/** A min - max pair under one visible group label ("Edad"), each field labelled as well. */
export function RangeFields({
  label,
  min,
  max,
  onMin,
  onMax,
  currency = false,
}: {
  label: string
  min: string
  max: string
  onMin: (value: string) => void
  onMax: (value: string) => void
  currency?: boolean
}) {
  const { t } = useTranslation()
  return (
    <Box component="fieldset" sx={{ border: 0, p: 0, m: 0, minWidth: 0 }}>
      <Typography variant="subtitle2" component="legend" sx={{ mb: 1 }}>
        {label}
      </Typography>
      <Stack direction="row" spacing={1.5}>
        <NumberField label={t('transfers.filters.min')} value={min} onValueChange={onMin} currency={currency} fullWidth />
        <NumberField label={t('transfers.filters.max')} value={max} onValueChange={onMax} currency={currency} fullWidth />
      </Stack>
    </Box>
  )
}

/** Sort field + direction, kept inside the filter sheet instead of a row of its own. */
export function SortControl<T extends string>({
  value,
  onChange,
  direction,
  onToggleDirection,
  options,
}: {
  value: T
  onChange: (value: T) => void
  direction: SortDirection
  onToggleDirection: () => void
  options: { value: T; label: string }[]
}) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
      <FormControl fullWidth>
        <InputLabel id="sort-field-label">{t('transfers.filters.sortBy')}</InputLabel>
        <Select
          labelId="sort-field-label"
          label={t('transfers.filters.sortBy')}
          value={value}
          onChange={(event: SelectChangeEvent) => onChange(event.target.value as T)}
        >
          {options.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      <Tooltip title={t(direction === 'asc' ? 'transfers.filters.sortAscending' : 'transfers.filters.sortDescending')}>
        <IconButton
          onClick={onToggleDirection}
          aria-label={t(direction === 'asc' ? 'transfers.filters.sortAscending' : 'transfers.filters.sortDescending')}
          sx={{ flexShrink: 0, bgcolor: 'action.hover' }}
        >
          {direction === 'asc' ? <ArrowUpwardIcon fontSize="small" /> : <ArrowDownwardIcon fontSize="small" />}
        </IconButton>
      </Tooltip>
    </Stack>
  )
}

// Shared "search by suggested position" filter, reused by every browsable listings tab - filters
// client-side by bestCorePosition since the market listing endpoints don't take a position
// parameter. A toggle group (one tap per position) instead of a dropdown.
export function PositionFilter({ value, onChange }: { value: number | ''; onChange: (value: number | '') => void }) {
  const { t } = useTranslation()
  return (
    <Box component="fieldset" sx={{ border: 0, p: 0, m: 0, minWidth: 0 }}>
      <Typography variant="subtitle2" component="legend" sx={{ mb: 1 }}>
        {t('transfers.filters.position')}
      </Typography>
      <ToggleButtonGroup
        exclusive
        fullWidth
        value={value}
        onChange={(_event, next: number | '' | null) => onChange(next ?? '')}
        aria-label={t('transfers.filters.position')}
      >
        {CORE_POSITIONS.map((position) => (
          <ToggleButton key={position} value={position} translate="no" className="notranslate">
            {t(POSITION_SHORT_LABEL_KEYS[position])}
          </ToggleButton>
        ))}
      </ToggleButtonGroup>
    </Box>
  )
}

// Shared "which skill" + "minimum value" pair for the market filters - unlike a rating range
// (which averages the seven skills), this targets one named skill.
export function SkillFilter({
  field,
  onField,
  minValue,
  onMinValue,
}: {
  field: (typeof SKILL_FIELDS)[number]
  onField: (value: (typeof SKILL_FIELDS)[number]) => void
  minValue: string
  onMinValue: (value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <Box component="fieldset" sx={{ border: 0, p: 0, m: 0, minWidth: 0 }}>
      <Typography variant="subtitle2" component="legend" sx={{ mb: 1 }}>
        {t('transfers.filters.skillGroup')}
      </Typography>
      <Stack direction="row" spacing={1.5}>
        <FormControl fullWidth>
          <InputLabel id="skill-field-filter-label">{t('transfers.filters.skill')}</InputLabel>
          <Select
            labelId="skill-field-filter-label"
            label={t('transfers.filters.skill')}
            value={field}
            onChange={(event: SelectChangeEvent) => onField(event.target.value as (typeof SKILL_FIELDS)[number])}
          >
            {SKILL_FIELDS.map((f) => (
              <MenuItem key={f} value={f}>
                {t(SKILL_LABEL_KEYS[f])}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <NumberField label={t('transfers.filters.minSkillValue')} value={minValue} onValueChange={onMinValue} max={99} fullWidth />
      </Stack>
    </Box>
  )
}

/** Open/close state for a tab's filter sheet. */
export function useFilterSheet() {
  const [open, setOpen] = useState(false)
  return { open, show: () => setOpen(true), hide: () => setOpen(false) }
}
