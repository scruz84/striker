import { useDraggable } from '@dnd-kit/core'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import type { Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'

interface DraggablePlayerChipProps {
  player: Player
  /** "field" shows just the avatar (name on hover) to fit a crowded pitch; "bench" shows name inline. */
  variant?: 'field' | 'bench'
}

export function DraggablePlayerChip({ player, variant = 'bench' }: DraggablePlayerChipProps) {
  // The moving drag visual is a DragOverlay in LineupEditorPage (a fixed-size ghost, so a
  // dropped-onto pitch slot always sees a consistently small hit target regardless of whether
  // the drag started from this compact field chip or the much wider bench card) - this element
  // just dims in place while dragging, it doesn't follow the pointer itself.
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `player-${player.id}`,
    data: { playerId: player.id },
  })

  const style = {
    cursor: 'grab',
    touchAction: 'none' as const,
    userSelect: 'none' as const,
    opacity: isDragging ? 0.4 : 1,
  }

  const label = `${player.firstName} ${player.lastName} (${player.age})`

  if (variant === 'field') {
    return (
      <Tooltip title={label}>
        <Box ref={setNodeRef} {...listeners} {...attributes} sx={{ ...style, lineHeight: 0 }}>
          <CartoonAvatar seed={player.id} size={44} />
        </Box>
      </Tooltip>
    )
  }

  return (
    <Paper
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      elevation={isDragging ? 6 : 1}
      sx={{ ...style, display: 'flex', alignItems: 'center', gap: 1, p: 0.75, width: '100%' }}
    >
      <CartoonAvatar seed={player.id} size={36} />
      <Box sx={{ minWidth: 0 }}>
        <Typography variant="body2" noWrap sx={{ fontWeight: 600, lineHeight: 1.2 }}>
          {player.firstName} {player.lastName}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {player.age}
        </Typography>
      </Box>
    </Paper>
  )
}
