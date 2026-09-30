import { useMemo, useState } from 'react'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import Box from '@mui/material/Box'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player, PlayerUnavailability } from '../../api/types'
import { SKILL_FIELDS, SKILL_LABEL_KEYS, overallRating } from '../../utils/playerRating'
import { PlayerChip } from './PlayerChip'

type BenchSortField = 'default' | 'age' | 'overall' | (typeof SKILL_FIELDS)[number]

function sortValue(player: Player, field: BenchSortField): number {
  if (field === 'age') return player.age
  if (field === 'overall') return overallRating(player)
  // 'default' never reaches here - sortPlayers below skips sorting entirely for it.
  return player[field as (typeof SKILL_FIELDS)[number]]
}

function sortPlayers(players: Player[], field: BenchSortField, direction: 'asc' | 'desc'): Player[] {
  if (field === 'default') {
    return players
  }
  return [...players].sort((a, b) => {
    const diff = sortValue(a, field) - sortValue(b, field)
    return direction === 'asc' ? diff : -diff
  })
}

interface PlayerBenchProps {
  players: Player[]
  selectedPlayerId: number | null
  onSelectPlayer: (playerId: number) => void
  /** Tap the bench's own empty background (not a specific player) to bench whichever fielded
   * player is currently selected. */
  onBenchAreaClick: () => void
  unavailability: Map<number, PlayerUnavailability>
  transferListed: Set<number>
}

/** Where a player not currently assigned to a pitch slot shows up - tap a fielded player, then tap
 * a bench player (to swap) or the bench's own background (to just bench them) - see
 * useLineupSelections for the actual placement logic.
 *
 * Sortable by age, overall rating, or any single skill (goalkeeper/defense/lateral/pass/midfield/
 * dribbling/shot) - e.g. sorting by Shot surfaces the bench's best finisher at a glance instead of
 * needing to open each player's own stats popover one at a time to compare them for a striker
 * slot. Purely a display order: it doesn't touch useLineupSelections' own state, so switching sort
 * order never disturbs an in-progress selection. */
export function PlayerBench({ players, selectedPlayerId, onSelectPlayer, onBenchAreaClick, unavailability, transferListed }: PlayerBenchProps) {
  const { t } = useTranslation()
  const [sortField, setSortField] = useState<BenchSortField>('default')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  const sortedPlayers = useMemo(() => sortPlayers(players, sortField, sortDir), [players, sortField, sortDir])
  const hasSelection = selectedPlayerId !== null
  // The bench itself is only a meaningful tap target while the selection is a fielded player -
  // tapping it with a bench player already selected (or nothing selected) would do nothing.
  const selectedIsFielded = hasSelection && !players.some((p) => p.id === selectedPlayerId)

  return (
    <Box>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
        <Typography variant="subtitle2">{t('lineup.bench')}</Typography>
        <Box sx={{ flex: 1 }} />
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <InputLabel id="bench-sort-label">{t('lineup.sortBy')}</InputLabel>
          <Select
            labelId="bench-sort-label"
            label={t('lineup.sortBy')}
            value={sortField}
            onChange={(event: SelectChangeEvent) => setSortField(event.target.value as BenchSortField)}
          >
            <MenuItem value="default">{t('lineup.sortDefault')}</MenuItem>
            <MenuItem value="age">{t('lineup.sortAge')}</MenuItem>
            <MenuItem value="overall">{t('lineup.sortOverall')}</MenuItem>
            {SKILL_FIELDS.map((field) => (
              <MenuItem key={field} value={field}>
                {t(SKILL_LABEL_KEYS[field])}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        {sortField !== 'default' && (
          <Tooltip title={t(sortDir === 'asc' ? 'lineup.sortAscending' : 'lineup.sortDescending')}>
            <IconButton size="small" onClick={() => setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))}>
              {sortDir === 'asc' ? <ArrowUpwardIcon fontSize="small" /> : <ArrowDownwardIcon fontSize="small" />}
            </IconButton>
          </Tooltip>
        )}
      </Stack>
      <Stack
        spacing={0.5}
        onClick={(event) => {
          // Only fires for a click landing directly on this background, not one bubbling up from
          // a player row inside it (each row handles its own tap) - see PlayerChip.
          if (event.target === event.currentTarget) {
            onBenchAreaClick()
          }
        }}
        sx={{
          minHeight: 120,
          p: 0.75,
          borderRadius: 1,
          bgcolor: 'action.selected',
          border: '1px dashed',
          borderColor: selectedIsFielded ? 'success.main' : 'divider',
          transition: 'border-color 150ms ease',
          cursor: selectedIsFielded ? 'pointer' : 'default',
        }}
      >
        {players.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 2 }}>
            {t('lineup.benchEmpty')}
          </Typography>
        ) : (
          sortedPlayers.map((player) => (
            <PlayerChip
              key={player.id}
              player={player}
              variant="bench"
              selected={player.id === selectedPlayerId}
              // Swapping only makes sense when the current selection is a fielded player - two
              // bench players have nothing to trade, tapping one just changes the selection (see
              // useLineupSelections' handleBenchPlayerClick), so it's not highlighted as a target.
              highlight={selectedIsFielded && player.id !== selectedPlayerId && !unavailability.has(player.id)}
              onSelect={() => onSelectPlayer(player.id)}
              unavailability={unavailability.get(player.id)}
              forSale={transferListed.has(player.id)}
            />
          ))
        )}
      </Stack>
    </Box>
  )
}
