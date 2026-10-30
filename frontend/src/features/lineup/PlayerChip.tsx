import { useState } from 'react'
import Box from '@mui/material/Box'
import ButtonBase from '@mui/material/ButtonBase'
import Chip from '@mui/material/Chip'
import Collapse from '@mui/material/Collapse'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type Player, type PlayerUnavailability } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { SkillChipsRow } from '../../components/SkillChipsRow'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'
import { TransferListedBadge } from './TransferListedBadge'
import { UnavailabilityBadge } from './UnavailabilityBadge'
import { CountryFlag } from '../../components/CountryFlag'

interface PlayerChipProps {
  player: Player
  /** "field" shows the avatar with a name label and squad number underneath, sized to fit a
   * crowded pitch; "bench" shows a compact one-row card with foldable skills. */
  variant?: 'field' | 'bench'
  /** This is the player currently picked up, waiting to be placed - see useLineupSelections.
   * Irrelevant (and left false) outside the lineup editor - see TeamDetailsPage's squad list,
   * which reuses the "bench" card purely as a display row. */
  selected?: boolean
  /** A selection is active (elsewhere) and tapping this player would place it here (a swap) -
   * distinct from `selected` itself, which is the one actually picked up. Same caveat as
   * `selected`. */
  highlight?: boolean
  /** Tapping the chip selects / places the player. */
  onSelect?: () => void
  /** Set while the player is suspended or injured - see useSquadUnavailability. On the bench
   * such a player is shown disabled, since they can't be fielded (see useLineupSelections). */
  unavailability?: PlayerUnavailability
  /** True while the player is listed on the transfer market - see useSquadTransferListed. */
  forSale?: boolean
  /** The player who would take this (unavailable) starter's slot if the match kicked off now -
   * see ProjectedReplacement. Only shown on the pitch, next to the unavailability badge. */
  replacedBy?: Player
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

export function PlayerChip({
  player,
  variant = 'bench',
  selected = false,
  highlight = false,
  onSelect = () => {},
  unavailability,
  forSale,
  replacedBy,
}: PlayerChipProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const ringSx = useSelectionRingSx(selected, highlight)
  const [expanded, setExpanded] = useState(false)

  const rating = overallRating(player)
  const position = bestCorePosition(player)
  const label = `${player.firstName} ${player.lastName} (${player.age})`
  const ratingColor = ratingTierColor(rating, theme.palette)

  // Field chips: ONE tap target - the avatar and the name under it select/move the player. The
  // player's details open from the selection panel under the pitch (SelectedPlayerPanel), so a
  // move can never open a profile by mistake.
  if (variant === 'field') {
    return (
      <ButtonBase
        onClick={onSelect}
        aria-pressed={selected}
        aria-label={label}
        sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 60, borderRadius: 2 }}
      >
        <Box sx={{ position: 'relative', lineHeight: 0, borderRadius: '50%', transition: 'box-shadow 150ms ease', ...ringSx }}>
          <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={44} />
          {!!player.shirtNumber && (
            <Box
              sx={{
                position: 'absolute',
                top: -5,
                right: -6,
                minWidth: 20,
                height: 20,
                px: 0.25,
                borderRadius: 10,
                bgcolor: 'rgba(20,20,20,0.85)',
                border: '1px solid rgba(255,255,255,0.85)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Typography variant="chipLabel" sx={{ color: '#fff', lineHeight: 1 }}>
                {player.shirtNumber}
              </Typography>
            </Box>
          )}
          {/* Top-left, the one corner the shirt number and rating badges leave free. */}
          {unavailability && (
            <Box sx={{ position: 'absolute', top: -5, left: -6, lineHeight: 0 }}>
              <UnavailabilityBadge
                unavailability={unavailability}
                size={20}
                replacedBy={replacedBy ? `${replacedBy.firstName} ${replacedBy.lastName}` : undefined}
              />
            </Box>
          )}
          {/* Average skill, quick-glance only - bottom-left mirrors the shirt number's top-right
              placement so neither badge crowds the face. */}
          <Box
            sx={{
              position: 'absolute',
              bottom: -5,
              left: -6,
              minWidth: 20,
              height: 20,
              px: 0.25,
              borderRadius: 10,
              bgcolor: ratingColor,
              border: '1px solid rgba(255,255,255,0.85)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography variant="chipLabel" sx={{ color: theme.palette.getContrastText(ratingColor), lineHeight: 1 }}>
              {rating}
            </Typography>
          </Box>
          {/* The one corner left free by the shirt number/unavailability/rating badges. */}
          {forSale && (
            <Box sx={{ position: 'absolute', bottom: -5, right: -6, lineHeight: 0 }}>
              <TransferListedBadge size={20} />
            </Box>
          )}
        </Box>
        <Typography
          noWrap
          variant="chipLabel"
          sx={{ mt: 0.5, px: 0.5, maxWidth: '100%', borderRadius: 1, bgcolor: 'rgba(0,0,0,0.65)', color: '#fff', textAlign: 'center' }}
        >
          {player.lastName}
        </Typography>
        {unavailability && replacedBy && (
          <Typography
            variant="chipLabel"
            sx={{ mt: 0.25, px: 0.5, maxWidth: '100%', borderRadius: 1, bgcolor: 'rgba(0,0,0,0.65)', color: '#fff', textAlign: 'center', lineHeight: 1.15 }}
          >
            {t('lineup.replacedBy', { name: replacedBy.lastName })}
          </Typography>
        )}
      </ButtonBase>
    )
  }

  // Bench rows: avatar + name + position + rating + form in one tap target (select / swap); the
  // skills and the profile link fold away behind a 44px chevron instead of filling 3 lines per
  // player.
  return (
    <Paper elevation={selected ? 4 : 1} sx={{ width: '100%', overflow: 'hidden', ...(selected && { outline: `2px solid ${theme.palette.warning.main}` }) }}>
      <Stack direction="row" sx={{ alignItems: 'center' }}>
        <ButtonBase
          onClick={onSelect}
          disabled={!!unavailability}
          aria-pressed={selected}
          aria-label={label}
          sx={{ flex: 1, minWidth: 0, minHeight: 52, gap: 1.25, p: 0.75, justifyContent: 'flex-start', textAlign: 'left' }}
        >
          <Box
            sx={{
              lineHeight: 0,
              borderRadius: '50%',
              transition: 'box-shadow 150ms ease',
              ...(unavailability && { opacity: 0.45, filter: 'grayscale(1)' }),
              ...ringSx,
            }}
          >
            <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={36} />
          </Box>
          <Box sx={{ minWidth: 0, flex: 1 }}>
            <Typography variant="body2" noWrap sx={{ fontWeight: 600, lineHeight: 1.25 }}>
              <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
              {player.firstName} {player.lastName}
            </Typography>
            <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
              <Chip
                size="small"
                variant="outlined"
                translate="no"
                className="notranslate"
                label={t(POSITION_SHORT_LABEL_KEYS[position])}
                sx={{ height: 20, '& .MuiChip-label': { px: 0.75 } }}
              />
              <Typography variant="caption" color="text.secondary">
                {t('lineup.ageShort', { age: player.age })}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {t('playerCard.skillsShort.form')} {player.form}
              </Typography>
              {unavailability && <UnavailabilityBadge unavailability={unavailability} size={18} />}
              {forSale && <TransferListedBadge size={18} />}
            </Stack>
          </Box>
          <Typography variant="stat" sx={{ fontSize: '1.25rem', color: ratingColor, flexShrink: 0, pr: 0.5 }}>
            {rating}
          </Typography>
        </ButtonBase>
        <IconButton
          onClick={() => setExpanded((open) => !open)}
          aria-expanded={expanded}
          aria-label={t(expanded ? 'lineup.hideSkills' : 'lineup.showSkills')}
        >
          {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
      </Stack>
      <Collapse in={expanded} unmountOnExit>
        <Box sx={{ px: 1, pb: 1 }}>
          <SkillChipsRow player={player} />
          <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" variant="body2" sx={{ display: 'inline-flex', alignItems: 'center', minHeight: 44 }}>
            {t('playerCard.viewProfile')}
          </Link>
        </Box>
      </Collapse>
    </Paper>
  )
}
