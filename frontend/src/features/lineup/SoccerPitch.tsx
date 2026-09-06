import { useDroppable } from '@dnd-kit/core'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'
import { POSITION_LABEL_KEYS } from '../../api/types'
import { DraggablePlayerChip } from './DraggablePlayerChip'
import { layoutPitchSlots, PITCH_HEIGHT, PITCH_WIDTH, type PitchSlot } from './pitchLayout'

interface SoccerPitchProps {
  counts: Partial<Record<number, number>>
  getPlayer: (position: number, slotIndex: number) => Player | undefined
}

export function SoccerPitch({ counts, getPlayer }: SoccerPitchProps) {
  const slots = layoutPitchSlots(counts)
  return (
    <Box
      sx={{
        position: 'relative',
        width: '100%',
        aspectRatio: `${PITCH_WIDTH} / ${PITCH_HEIGHT}`,
        borderRadius: 2,
        overflow: 'hidden',
        background: 'repeating-linear-gradient(180deg, #3f8f46 0px, #3f8f46 40px, #388240 40px, #388240 80px)',
      }}
    >
      <PitchMarkings />
      {slots.map((slot, index) => (
        <PitchSlotView
          key={`${slot.position}-${slot.slotIndex}`}
          slot={slot}
          shirtNumber={index + 1}
          player={getPlayer(slot.position, slot.slotIndex)}
        />
      ))}
    </Box>
  )
}

function PitchMarkings() {
  const line = 'rgba(255,255,255,0.7)'
  return (
    <svg
      viewBox={`0 0 ${PITCH_WIDTH} ${PITCH_HEIGHT}`}
      preserveAspectRatio="none"
      style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', pointerEvents: 'none' }}
    >
      <rect x={4} y={4} width={PITCH_WIDTH - 8} height={PITCH_HEIGHT - 8} fill="none" stroke={line} strokeWidth={2} />
      <line x1={4} y1={PITCH_HEIGHT / 2} x2={PITCH_WIDTH - 4} y2={PITCH_HEIGHT / 2} stroke={line} strokeWidth={2} />
      <circle cx={PITCH_WIDTH / 2} cy={PITCH_HEIGHT / 2} r={42} fill="none" stroke={line} strokeWidth={2} />
      <circle cx={PITCH_WIDTH / 2} cy={PITCH_HEIGHT / 2} r={2.5} fill={line} />
      {/* own penalty + goal area, near the goalkeeper */}
      <rect x={70} y={PITCH_HEIGHT - 90} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
      <rect x={112} y={PITCH_HEIGHT - 34} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
      {/* opponent penalty + goal area, near the strikers */}
      <rect x={70} y={4} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
      <rect x={112} y={4} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
    </svg>
  )
}

function PitchSlotView({ slot, player, shirtNumber }: { slot: PitchSlot; player?: Player; shirtNumber: number }) {
  const { t } = useTranslation()
  const { setNodeRef, isOver } = useDroppable({
    id: `slot-${slot.position}-${slot.slotIndex}`,
    data: { position: slot.position, slotIndex: slot.slotIndex },
  })

  return (
    <Box
      ref={setNodeRef}
      sx={{
        position: 'absolute',
        left: `${(slot.x / PITCH_WIDTH) * 100}%`,
        top: `${(slot.y / PITCH_HEIGHT) * 100}%`,
        // Anchor to the avatar's own center (half its 44px height), not the whole chip's center,
        // so the avatar lines up with the slot point the same way whether or not the name label
        // below it is present.
        transform: 'translate(-50%, -22px)',
      }}
    >
      {player ? (
        <DraggablePlayerChip player={player} variant="field" shirtNumber={shirtNumber} />
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
    </Box>
  )
}
