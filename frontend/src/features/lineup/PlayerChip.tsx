import { useState } from 'react'
import Box from '@mui/material/Box'
import ButtonBase from '@mui/material/ButtonBase'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerStatsPopover } from '../../components/PlayerStatsPopover'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'

interface PlayerChipProps {
  player: Player
  /** "field" shows the avatar with a name label and squad number underneath, sized to fit a
   * crowded pitch; "bench" shows the full name inline in a wider card. */
  variant?: 'field' | 'bench'
  /** This is the player currently picked up, waiting to be placed - see useLineupSelections. */
  selected: boolean
  /** A selection is active (elsewhere) and tapping this player would place it here (a swap) -
   * distinct from `selected` itself, which is the one actually picked up. */
  highlight: boolean
  onSelect: () => void
}

// The ring drawn around a selected/highlighted player's avatar - a plain colored ring for the one
// actually picked up, a pulsing one for every other player tapping it would swap with, so a
// manager always has an unambiguous answer to "where is this going to land" without needing to
// track a small avatar dragged across a crowded pitch (the previous drag-and-drop editor's own
// occupied slots never highlighted at all during a drag, which was the whole complaint).
function useSelectionRingSx(selected: boolean, highlight: boolean) {
  const theme = useTheme()
  // Deliberately different colors, not just different intensities of the same one - "this is the
  // player you picked up" (amber) and "tapping this swaps them in" (green) need to read as two
  // different roles at a glance, not as the same ring seen at two moments.
  if (selected) {
    return {
      boxShadow: `0 0 0 3px ${theme.palette.warning.main}, 0 0 8px 2px ${alpha(theme.palette.warning.main, 0.7)}`,
    }
  }
  if (highlight) {
    return {
      '@keyframes lineupTargetPulse': {
        '0%, 100%': { boxShadow: `0 0 0 2px ${alpha(theme.palette.success.main, 0.35)}` },
        '50%': { boxShadow: `0 0 0 2px ${alpha(theme.palette.success.main, 0.9)}` },
      },
      animation: 'lineupTargetPulse 1.3s ease-in-out infinite',
    }
  }
  return {}
}

export function PlayerChip({ player, variant = 'bench', selected, highlight, onSelect }: PlayerChipProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const ringSx = useSelectionRingSx(selected, highlight)
  const [statsAnchor, setStatsAnchor] = useState<HTMLElement | null>(null)
  const openStats = (event: React.MouseEvent<HTMLElement>) => {
    event.stopPropagation()
    setStatsAnchor(event.currentTarget)
  }
  const closeStats = () => setStatsAnchor(null)

  const rating = overallRating(player)
  const position = bestCorePosition(player)
  const label = `${player.firstName} ${player.lastName} (${player.age})`

  if (variant === 'field') {
    return (
      <>
        <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 68 }}>
          <Tooltip title={label}>
            <ButtonBase
              onClick={onSelect}
              sx={{ borderRadius: '50%' }}
              aria-pressed={selected}
              aria-label={label}
            >
              <Box sx={{ position: 'relative', lineHeight: 0, borderRadius: '50%', transition: 'box-shadow 150ms ease', ...ringSx }}>
                <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={44} />
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
                {/* Average skill, quick-glance only (see the bench variant's fuller badge, where
                    there's room for the suggested position too) - bottom-left mirrors the shirt
                    number's top-right placement so neither badge crowds the face. */}
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
            </ButtonBase>
          </Tooltip>
          <ButtonBase onClick={openStats} sx={{ mt: 0.25, px: 0.5, borderRadius: 1, bgcolor: 'rgba(0,0,0,0.6)', maxWidth: '100%' }}>
            <Typography
              noWrap
              sx={{ color: '#fff', fontSize: '0.62rem', fontWeight: 600, lineHeight: 1.3, textAlign: 'center' }}
            >
              {player.lastName}
            </Typography>
          </ButtonBase>
        </Box>
        <PlayerStatsPopover player={player} anchorEl={statsAnchor} onClose={closeStats} />
      </>
    )
  }

  return (
    <>
      <Paper elevation={selected ? 4 : 1} sx={{ display: 'flex', alignItems: 'center', gap: 1, p: 0.5, width: '100%' }}>
        <ButtonBase onClick={onSelect} sx={{ borderRadius: '50%' }} aria-pressed={selected} aria-label={label}>
          <Box sx={{ lineHeight: 0, borderRadius: '50%', transition: 'box-shadow 150ms ease', ...ringSx }}>
            <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={32} />
          </Box>
        </ButtonBase>
        <ButtonBase onClick={openStats} sx={{ minWidth: 0, flex: 1, justifyContent: 'flex-start', textAlign: 'left', borderRadius: 1, p: 0.5 }}>
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
        </ButtonBase>
      </Paper>
      <PlayerStatsPopover player={player} anchorEl={statsAnchor} onClose={closeStats} />
    </>
  )
}
