import ButtonBase from '@mui/material/ButtonBase'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'
import { POSITION_LABEL_KEYS } from '../../api/types'
import { PitchField, PitchSlotPosition } from '../../components/pitch/PitchField'
import { layoutPitchSlots, PITCH_HEIGHT_HALF, type PitchSlot } from '../../components/pitch/pitchLayout'
import { PlayerChip } from './PlayerChip'

interface SoccerPitchProps {
  counts: Partial<Record<number, number>>
  getPlayer: (position: number, slotIndex: number) => Player | undefined
  selectedPlayerId: number | null
  onSlotClick: (position: number, slotIndex: number) => void
}

export function SoccerPitch({ counts, getPlayer, selectedPlayerId, onSlotClick }: SoccerPitchProps) {
  // Only the half of the pitch a lineup ever actually occupies is shown here - see
  // PITCH_HEIGHT_HALF's comment in pitchLayout.ts - so editing a lineup takes noticeably less
  // vertical space than the full-pitch view used for read-only match lineups.
  const slots = layoutPitchSlots(counts, true)
  const hasSelection = selectedPlayerId !== null
  return (
    <PitchField half>
      {slots.map((slot) => {
        const player = getPlayer(slot.position, slot.slotIndex)
        const selected = player?.id === selectedPlayerId
        return (
          <PitchSlotView
            key={`${slot.position}-${slot.slotIndex}`}
            slot={slot}
            player={player}
            selected={selected}
            // Any slot other than the one already selected is a valid place to tap next while a
            // selection is active - an empty one moves the player there, an occupied one swaps.
            highlight={hasSelection && !selected}
            onClick={() => onSlotClick(slot.position, slot.slotIndex)}
          />
        )
      })}
    </PitchField>
  )
}

function PitchSlotView({
  slot,
  player,
  selected,
  highlight,
  onClick,
}: {
  slot: PitchSlot
  player?: Player
  selected: boolean
  highlight: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <PitchSlotPosition x={slot.x} y={slot.y} pitchHeight={PITCH_HEIGHT_HALF}>
      {player ? (
        <PlayerChip player={player} variant="field" selected={selected} highlight={highlight} onSelect={onClick} />
      ) : (
        <ButtonBase onClick={onClick} sx={{ borderRadius: '50%' }} aria-label={t(POSITION_LABEL_KEYS[slot.position])}>
          <Box
            sx={{
              width: 44,
              height: 44,
              borderRadius: '50%',
              border: '2px dashed rgba(255,255,255,0.8)',
              bgcolor: 'rgba(255,255,255,0.15)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'box-shadow 150ms ease, background-color 150ms ease',
              ...(highlight && {
                borderColor: theme.palette.success.light,
                '@keyframes lineupTargetPulse': {
                  '0%, 100%': { boxShadow: `0 0 0 2px ${alpha(theme.palette.success.main, 0.35)}` },
                  '50%': { boxShadow: `0 0 0 2px ${alpha(theme.palette.success.main, 0.9)}` },
                },
                animation: 'lineupTargetPulse 1.3s ease-in-out infinite',
              }),
            }}
          >
            <Typography variant="caption" sx={{ color: 'white', fontWeight: 700, fontSize: '0.65rem' }}>
              {t(POSITION_LABEL_KEYS[slot.position]).slice(0, 3).toUpperCase()}
            </Typography>
          </Box>
        </ButtonBase>
      )}
    </PitchSlotPosition>
  )
}
