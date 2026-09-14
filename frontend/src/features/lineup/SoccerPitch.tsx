import { useDroppable } from '@dnd-kit/core'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'
import { POSITION_LABEL_KEYS } from '../../api/types'
import { PitchField, PitchSlotPosition } from '../../components/pitch/PitchField'
import { layoutPitchSlots, PITCH_HEIGHT_HALF, type PitchSlot } from '../../components/pitch/pitchLayout'
import { DraggablePlayerChip } from './DraggablePlayerChip'

interface SoccerPitchProps {
  counts: Partial<Record<number, number>>
  getPlayer: (position: number, slotIndex: number) => Player | undefined
}

export function SoccerPitch({ counts, getPlayer }: SoccerPitchProps) {
  // Only the half of the pitch a lineup ever actually occupies is shown here - see
  // PITCH_HEIGHT_HALF's comment in pitchLayout.ts - so editing a lineup takes noticeably less
  // vertical space than the full-pitch view used for read-only match lineups.
  const slots = layoutPitchSlots(counts, true)
  return (
    <PitchField half>
      {slots.map((slot) => (
        <PitchSlotView
          key={`${slot.position}-${slot.slotIndex}`}
          slot={slot}
          player={getPlayer(slot.position, slot.slotIndex)}
        />
      ))}
    </PitchField>
  )
}

function PitchSlotView({ slot, player }: { slot: PitchSlot; player?: Player }) {
  const { t } = useTranslation()
  const { setNodeRef, isOver } = useDroppable({
    id: `slot-${slot.position}-${slot.slotIndex}`,
    data: { position: slot.position, slotIndex: slot.slotIndex },
  })

  return (
    <PitchSlotPosition ref={setNodeRef} x={slot.x} y={slot.y} pitchHeight={PITCH_HEIGHT_HALF}>
      {player ? (
        <DraggablePlayerChip player={player} variant="field" />
      ) : (
        <Box
          sx={{
            width: 44,
            height: 44,
            borderRadius: '50%',
            border: '2px dashed rgba(255,255,255,0.8)',
            bgcolor: isOver ? 'rgba(255,255,255,0.4)' : 'rgba(255,255,255,0.15)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'background-color 120ms ease',
          }}
        >
          <Typography variant="caption" sx={{ color: 'white', fontWeight: 700, fontSize: '0.65rem' }}>
            {t(POSITION_LABEL_KEYS[slot.position]).slice(0, 3).toUpperCase()}
          </Typography>
        </Box>
      )}
    </PitchSlotPosition>
  )
}
