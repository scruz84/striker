import { useDroppable } from '@dnd-kit/core'
import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'
import { DraggablePlayerChip } from './DraggablePlayerChip'

interface PlayerBenchProps {
  players: Player[]
}

/** The drop target for players not currently assigned to a pitch slot - drop a fielded player here to bench them. */
export function PlayerBench({ players }: PlayerBenchProps) {
  const { t } = useTranslation()
  const { setNodeRef, isOver } = useDroppable({ id: 'bench' })

  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {t('lineup.bench')}
      </Typography>
      <Stack
        ref={setNodeRef}
        spacing={1}
        sx={{
          minHeight: 120,
          p: 1,
          borderRadius: 1,
          bgcolor: isOver ? 'action.hover' : 'action.selected',
          border: '1px dashed',
          borderColor: 'divider',
        }}
      >
        {players.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 2 }}>
            {t('lineup.benchEmpty')}
          </Typography>
        ) : (
          players.map((player) => <DraggablePlayerChip key={player.id} player={player} variant="bench" />)
        )}
      </Stack>
    </Box>
  )
}
