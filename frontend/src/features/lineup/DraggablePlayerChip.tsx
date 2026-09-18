import { useState } from 'react'
import { useDraggable } from '@dnd-kit/core'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerStatsPopover } from '../../components/PlayerStatsPopover'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'

interface DraggablePlayerChipProps {
  player: Player
  /** "field" shows the avatar with a name label and squad number underneath, sized to fit a
   * crowded pitch; "bench" shows the full name inline in a wider card. */
  variant?: 'field' | 'bench'
}

export function DraggablePlayerChip({ player, variant = 'bench' }: DraggablePlayerChipProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  // The moving drag visual is a DragOverlay in LineupEditorPage (a fixed-size ghost, so a
  // dropped-onto pitch slot always sees a consistently small hit target regardless of whether
  // the drag started from this compact field chip or the much wider bench card) - this element
  // just dims in place while dragging, it doesn't follow the pointer itself.
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `player-${player.id}`,
    data: { playerId: player.id },
  })
  // A click that doesn't move far enough to clear the DndContext's activation distance never
  // starts a drag, so it still reaches this onClick normally - that's what lets a plain tap open
  // stats instead of always picking the player up.
  const [statsAnchor, setStatsAnchor] = useState<HTMLElement | null>(null)
  const openStats = (event: React.MouseEvent<HTMLElement>) => setStatsAnchor(event.currentTarget)
  const closeStats = () => setStatsAnchor(null)

  const rating = overallRating(player)
  const position = bestCorePosition(player)

  const style = {
    cursor: 'grab',
    // 'pan-y' (rather than 'none') leaves native vertical scrolling to the browser until
    // dnd-kit's TouchSensor decides a press-and-hold is actually a drag (see its delay-based
    // activationConstraint in LineupEditorPage/MatchLineupForm) - with 'none' here, the browser
    // blocked scrolling the instant a touch landed on any card, which is most of the bench list's
    // scrollable area, so any attempt to scroll past a card dragged the player under your finger
    // instead.
    touchAction: 'pan-y' as const,
    userSelect: 'none' as const,
    opacity: isDragging ? 0.4 : 1,
  }

  const label = `${player.firstName} ${player.lastName} (${player.age})`

  if (variant === 'field') {
    return (
      <>
        <Tooltip title={label}>
          <Box
            ref={setNodeRef}
            {...listeners}
            {...attributes}
            onClick={openStats}
            sx={{ ...style, display: 'flex', flexDirection: 'column', alignItems: 'center', width: 68 }}
          >
            <Box sx={{ position: 'relative', lineHeight: 0 }}>
              <CartoonAvatar seed={player.id} size={44} />
              {!!player.shirtNumber && (
                <Box
                  sx={{
                    position: 'absolute',
                    top: -4,
                    right: -4,
                    width: 18,
                    height: 18,
                    borderRadius: '50%',
                    bgcolor: 'rgba(20,20,20,0.85)',
                    border: '1px solid rgba(255,255,255,0.85)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Typography sx={{ color: '#fff', fontSize: '0.6rem', fontWeight: 700, lineHeight: 1 }}>
                    {player.shirtNumber}
                  </Typography>
                </Box>
              )}
              {/* Average skill, quick-glance only (see PlayerBench's fuller badge for the bench,
                  where there's room for the suggested position too) - bottom-left mirrors the
                  shirt number's top-right placement so neither badge crowds the face. */}
              <Box
                sx={{
                  position: 'absolute',
                  bottom: -4,
                  left: -4,
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  bgcolor: ratingTierColor(rating, theme.palette),
                  border: '1px solid rgba(255,255,255,0.85)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Typography sx={{ color: '#fff', fontSize: '0.55rem', fontWeight: 700, lineHeight: 1 }}>
                  {rating}
                </Typography>
              </Box>
            </Box>
            <Box sx={{ mt: 0.25, px: 0.5, borderRadius: 1, bgcolor: 'rgba(0,0,0,0.6)', maxWidth: '100%' }}>
              <Typography
                noWrap
                sx={{ color: '#fff', fontSize: '0.62rem', fontWeight: 600, lineHeight: 1.3, textAlign: 'center' }}
              >
                {player.lastName}
              </Typography>
            </Box>
          </Box>
        </Tooltip>
        <PlayerStatsPopover player={player} anchorEl={statsAnchor} onClose={closeStats} />
      </>
    )
  }

  return (
    <>
      <Paper
        ref={setNodeRef}
        {...listeners}
        {...attributes}
        onClick={openStats}
        elevation={isDragging ? 6 : 1}
        sx={{ ...style, display: 'flex', alignItems: 'center', gap: 1, p: 0.5, width: '100%' }}
      >
        <CartoonAvatar seed={player.id} size={32} />
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="body2" noWrap sx={{ fontWeight: 600, lineHeight: 1.2 }}>
            {player.firstName} {player.lastName}
          </Typography>
          <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
            <Typography variant="caption" color="text.secondary">
              {player.age}
            </Typography>
            <Chip
              size="small"
              variant="outlined"
              label={t(POSITION_SHORT_LABEL_KEYS[position])}
              sx={{ height: 16, fontSize: '0.6rem', '& .MuiChip-label': { px: 0.5 } }}
            />
            <Typography variant="caption" sx={{ fontWeight: 700, color: ratingTierColor(rating, theme.palette) }}>
              {rating}
            </Typography>
          </Stack>
        </Box>
      </Paper>
      <PlayerStatsPopover player={player} anchorEl={statsAnchor} onClose={closeStats} />
    </>
  )
}
