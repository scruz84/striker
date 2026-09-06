import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Lineup, Player } from '../../api/types'
import { FORMATION_POSITION_COUNTS, FORMATION_BALANCED, POSITION_LABEL_KEYS } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PitchField, PitchSlotPosition } from '../../components/pitch/PitchField'
import { layoutPitchSlots } from '../../components/pitch/pitchLayout'

interface MatchLineupPitchProps {
  lineup: Lineup | null
  players: Record<string, Player> | null
}

/** A read-only version of the lineup editor's pitch - same visual language (formation-sheet
 * layout, cartoon avatars, squad numbers), just no drag-and-drop, for showing how a team lined
 * up in a specific match. */
export function MatchLineupPitch({ lineup, players }: MatchLineupPitchProps) {
  const { t } = useTranslation()

  if (!lineup || !players || lineup.positions.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
        {t('match.noLineupData')}
      </Typography>
    )
  }

  const counts = FORMATION_POSITION_COUNTS[lineup.formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const slots = layoutPitchSlots(counts)
  const playerIdBySlot = new Map<string, number>()
  for (const pos of lineup.positions) {
    playerIdBySlot.set(`${pos.position}-${slotIndexWithinPosition(lineup, pos)}`, pos.playerId)
  }

  return (
    <PitchField>
      {slots.map((slot, index) => {
        const playerId = playerIdBySlot.get(`${slot.position}-${slot.slotIndex}`)
        const player = playerId !== undefined ? players[String(playerId)] : undefined
        return (
          <PitchSlotPosition key={`${slot.position}-${slot.slotIndex}`} x={slot.x} y={slot.y}>
            {player ? (
              <MatchPitchChip player={player} shirtNumber={index + 1} />
            ) : (
              <Box
                sx={{
                  width: 44,
                  height: 44,
                  borderRadius: '50%',
                  border: '2px dashed rgba(255,255,255,0.6)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Typography variant="caption" sx={{ color: 'white', fontWeight: 700, fontSize: '0.65rem' }}>
                  {t(POSITION_LABEL_KEYS[slot.position]).slice(0, 3).toUpperCase()}
                </Typography>
              </Box>
            )}
          </PitchSlotPosition>
        )
      })}
    </PitchField>
  )
}

// Lineup.positions is a flat list; a formation's central slots (e.g. two center-backs) share a
// position id and are told apart only by their order of appearance for that position.
function slotIndexWithinPosition(lineup: Lineup, target: Lineup['positions'][number]): number {
  let index = 0
  for (const pos of lineup.positions) {
    if (pos === target) return index
    if (pos.position === target.position) index++
  }
  return index
}

function MatchPitchChip({ player, shirtNumber }: { player: Player; shirtNumber: number }) {
  const label = `${player.firstName} ${player.lastName} (${player.age})`
  return (
    <Tooltip title={label}>
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 68 }}>
        <Box sx={{ position: 'relative', lineHeight: 0 }}>
          <CartoonAvatar seed={player.id} size={40} />
          <Box
            sx={{
              position: 'absolute',
              top: -4,
              right: -4,
              width: 16,
              height: 16,
              borderRadius: '50%',
              bgcolor: 'rgba(20,20,20,0.85)',
              border: '1px solid rgba(255,255,255,0.85)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography sx={{ color: '#fff', fontSize: '0.55rem', fontWeight: 700, lineHeight: 1 }}>
              {shirtNumber}
            </Typography>
          </Box>
        </Box>
        <Box sx={{ mt: 0.25, px: 0.5, borderRadius: 1, bgcolor: 'rgba(0,0,0,0.6)', maxWidth: '100%' }}>
          <Typography noWrap sx={{ color: '#fff', fontSize: '0.6rem', fontWeight: 600, lineHeight: 1.3, textAlign: 'center' }}>
            {player.lastName}
          </Typography>
        </Box>
      </Box>
    </Tooltip>
  )
}
