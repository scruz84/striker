import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Lineup, Player } from '../../api/types'
import { FORMATION_POSITION_COUNTS, FORMATION_BALANCED, POSITION_LABEL_KEYS } from '../../api/types'
import { PitchField, PitchSlotPosition } from '../../components/pitch/PitchField'
import { layoutHeadToHeadSlots, PITCH_HEIGHT_HEAD_TO_HEAD, type PitchSlot } from '../../components/pitch/pitchLayout'
import { MatchPitchChip, slotIndexWithinPosition } from './MatchPitchChip'

interface MatchHeadToHeadPitchProps {
  homeLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayLineup: Lineup | null
  awayPlayers: Record<string, Player> | null
  ratings?: Record<string, number> | null
}

/** Both teams' starting XIs on one shared pitch, facing each other - home in the bottom half
 * (own goal at the bottom), away mirrored into the top half (own goal at the top) - the same
 * layout a broadcast head-to-head graphic uses, rather than each team getting its own separate
 * pitch. See pitchLayout.layoutHeadToHeadSlots for the coordinate math. */
export function MatchHeadToHeadPitch({ homeLineup, homePlayers, awayLineup, awayPlayers, ratings }: MatchHeadToHeadPitchProps) {
  const { t } = useTranslation()

  if (!homeLineup || !homePlayers || !awayLineup || !awayPlayers || homeLineup.positions.length === 0 || awayLineup.positions.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
        {t('match.noLineupData')}
      </Typography>
    )
  }

  const homeCounts = FORMATION_POSITION_COUNTS[homeLineup.formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const awayCounts = FORMATION_POSITION_COUNTS[awayLineup.formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const { home: homeSlots, away: awaySlots } = layoutHeadToHeadSlots(homeCounts, awayCounts)

  return (
    <PitchField height={PITCH_HEIGHT_HEAD_TO_HEAD}>
      {renderTeamSlots(homeSlots, homeLineup, homePlayers, t, ratings)}
      {renderTeamSlots(awaySlots, awayLineup, awayPlayers, t, ratings)}
    </PitchField>
  )
}

function renderTeamSlots(
  slots: PitchSlot[],
  lineup: Lineup,
  players: Record<string, Player>,
  t: (key: string) => string,
  ratings?: Record<string, number> | null,
) {
  const playerIdBySlot = new Map<string, number>()
  for (const pos of lineup.positions) {
    playerIdBySlot.set(`${pos.position}-${slotIndexWithinPosition(lineup, pos)}`, pos.playerId)
  }

  return slots.map((slot) => {
    const playerId = playerIdBySlot.get(`${slot.position}-${slot.slotIndex}`)
    const player = playerId !== undefined ? players[String(playerId)] : undefined
    const rating = playerId !== undefined ? ratings?.[String(playerId)] : undefined
    return (
      <PitchSlotPosition
        key={`${lineup.teamId}-${slot.position}-${slot.slotIndex}`}
        x={slot.x}
        y={slot.y}
        pitchHeight={PITCH_HEIGHT_HEAD_TO_HEAD}
      >
        {player ? (
          <MatchPitchChip player={player} rating={rating} />
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
            <Typography variant="caption" sx={{ color: 'white', fontWeight: 700, fontSize: '0.6875rem' }}>
              {t(POSITION_LABEL_KEYS[slot.position]).slice(0, 3).toUpperCase()}
            </Typography>
          </Box>
        )}
      </PitchSlotPosition>
    )
  })
}
