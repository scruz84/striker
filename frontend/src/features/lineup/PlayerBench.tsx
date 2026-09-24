import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player, PlayerUnavailability } from '../../api/types'
import { PlayerChip } from './PlayerChip'

interface PlayerBenchProps {
  players: Player[]
  selectedPlayerId: number | null
  onSelectPlayer: (playerId: number) => void
  /** Tap the bench's own empty background (not a specific player) to bench whichever fielded
   * player is currently selected. */
  onBenchAreaClick: () => void
  unavailability: Map<number, PlayerUnavailability>
}

/** Where a player not currently assigned to a pitch slot shows up - tap a fielded player, then tap
 * a bench player (to swap) or the bench's own background (to just bench them) - see
 * useLineupSelections for the actual placement logic. */
export function PlayerBench({ players, selectedPlayerId, onSelectPlayer, onBenchAreaClick, unavailability }: PlayerBenchProps) {
  const { t } = useTranslation()
  const hasSelection = selectedPlayerId !== null
  // The bench itself is only a meaningful tap target while the selection is a fielded player -
  // tapping it with a bench player already selected (or nothing selected) would do nothing.
  const selectedIsFielded = hasSelection && !players.some((p) => p.id === selectedPlayerId)

  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {t('lineup.bench')}
      </Typography>
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
          players.map((player) => (
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
            />
          ))
        )}
      </Stack>
    </Box>
  )
}
