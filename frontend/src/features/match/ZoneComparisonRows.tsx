import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { TeamQualification } from '../../api/types'

interface ZoneRow {
  labelKey: string
  home: number
  away: number
  /** i18n key for the tooltip spelling out exactly which zone is being compared. */
  captionKey: string
}

// The midfield battle is a genuinely symmetric contest - both teams' midfielders compete
// directly in the same flank zone for the ball - so comparing each side's own midfield rating
// zone-for-zone is the correct matchup, unlike the defense/attack columns below. This is the one
// column where "left" means the same flank for both teams, unlike the mirrored columns - which
// is exactly what confuses readers, so every cell's tooltip spells out its own matchup instead of
// relying on the label alone.
function midfieldRows(homeQua: TeamQualification, awayQua: TeamQualification): ZoneRow[] {
  return [
    { labelKey: 'match.positionBattle.leftMidfield', home: homeQua.leftMidfield, away: awayQua.leftMidfield, captionKey: 'match.positionBattle.vsMidfieldLeft' },
    { labelKey: 'match.positionBattle.centerMidfield', home: homeQua.midfield, away: awayQua.midfield, captionKey: 'match.positionBattle.vsMidfieldCenter' },
    { labelKey: 'match.positionBattle.rightMidfield', home: homeQua.rightMidfield, away: awayQua.rightMidfield, captionKey: 'match.positionBattle.vsMidfieldRight' },
  ]
}

interface ZoneComparisonRowsProps {
  homeQua: TeamQualification
  awayQua: TeamQualification
  homeGoalkeeperRating: number
  awayGoalkeeperRating: number
  homeName: string
  awayName: string
}

// Fixed home/away colors (not the app's own green-seeded primary/secondary, which would blend
// into the pitch) - matches the blue-vs-red convention this panel's design is modeled on.
const HOME_COLOR = '#1e88e5'
const AWAY_COLOR = '#e53935'

/** A pitch-shaped, zone-by-zone strength comparison built from the exact same cross-zone
 * matchups engine.processChance actually compares to decide the probability of a goal: each
 * team's attack on one flank against the opponent's defense on the MIRRORED flank plus the
 * opponent's goalkeeper - never a team's own zone against the opponent's identical zone (that
 * would be comparing e.g. both teams' goalkeepers to each other, which the engine never does).
 * Midfield is the one genuinely symmetric zone-vs-same-zone contest, shown as-is.
 *
 * Both mirrored columns are anchored to the home team's own zone names (its defense fending off
 * the opponent's attack on the mirrored flank; its attack against the opponent's defense on the
 * mirrored flank) so the whole panel reads as one pitch - "how strong is MY team, zone by zone" -
 * rather than needing a separate "{team} attacking" section per side to say the same six numbers
 * from the other direction. Each cell's exact matchup (which raw values feed it) is on its
 * tooltip rather than printed inline, to keep the grid as scannable as a real pitch view.
 *
 * Just the team-name header and the zone grid, with no Paper/title/description of its own - every
 * caller (MatchMinuteZoneComparison, live or a finished match's "Jugadores" tab) wraps this in its
 * own Paper with a minute slider above it, the same way TeamRatingsColumn is shared between
 * MatchRatingsTab's static view and MatchMinuteRatings' scrubbable one. */
export function ZoneComparisonRows({
  homeQua,
  awayQua,
  homeGoalkeeperRating,
  awayGoalkeeperRating,
  homeName,
  awayName,
}: ZoneComparisonRowsProps) {
  const { t } = useTranslation()

  const defenseRows: ZoneRow[] = [
    { labelKey: 'match.positionBattle.leftDefenseZone', home: homeQua.leftDefense + homeGoalkeeperRating, away: awayQua.rightAttach, captionKey: 'match.positionBattle.vsAttackRight' },
    { labelKey: 'match.positionBattle.centerDefenseZone', home: homeQua.defense + homeGoalkeeperRating, away: awayQua.attack, captionKey: 'match.positionBattle.vsAttackCenter' },
    { labelKey: 'match.positionBattle.rightDefenseZone', home: homeQua.rightDefense + homeGoalkeeperRating, away: awayQua.leftAttack, captionKey: 'match.positionBattle.vsAttackLeft' },
  ]
  const midRows = midfieldRows(homeQua, awayQua)
  const attackRows: ZoneRow[] = [
    { labelKey: 'match.positionBattle.leftAttack', home: homeQua.leftAttack, away: awayQua.rightDefense + awayGoalkeeperRating, captionKey: 'match.positionBattle.vsDefenseRightGk' },
    { labelKey: 'match.positionBattle.centerAttack', home: homeQua.attack, away: awayQua.defense + awayGoalkeeperRating, captionKey: 'match.positionBattle.vsDefenseCenterGk' },
    { labelKey: 'match.positionBattle.rightAttack', home: homeQua.rightAttach, away: awayQua.leftDefense + awayGoalkeeperRating, captionKey: 'match.positionBattle.vsDefenseLeftGk' },
  ]
  const columns = [
    { headerKey: 'match.positionBattle.defenseHeader', rows: defenseRows },
    { headerKey: 'match.positionBattle.midfieldHeader', rows: midRows },
    { headerKey: 'match.positionBattle.attackHeader', rows: attackRows },
  ]

  return (
    <>
      <Stack direction="row" sx={{ mb: 1 }}>
        <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: 0.75, justifyContent: 'flex-end' }}>
          <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: HOME_COLOR, flexShrink: 0 }} />
          <Typography variant="caption" sx={{ fontWeight: 700 }} noWrap>
            {homeName}
          </Typography>
        </Box>
        <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: 0.75, pl: 2 }}>
          <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: AWAY_COLOR, flexShrink: 0 }} />
          <Typography variant="caption" sx={{ fontWeight: 700 }} noWrap>
            {awayName}
          </Typography>
        </Box>
      </Stack>
      <Stack direction="row" sx={{ mb: 0.5 }}>
        {columns.map((col) => (
          <Typography
            key={col.headerKey}
            variant="caption"
            color="text.secondary"
            sx={{ flex: 1, textAlign: 'center', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.5, fontSize: '0.65rem' }}
          >
            {t(col.headerKey)}
          </Typography>
        ))}
      </Stack>
      <Box
        sx={{
          position: 'relative',
          display: 'flex',
          gap: { xs: 0.75, sm: 1.5 },
          borderRadius: 2,
          p: { xs: 0.75, sm: 1.5 },
          bgcolor: (theme) => (theme.palette.mode === 'light' ? '#2e7d32' : '#173a1c'),
        }}
      >
        <PitchCorner corner="top-left" />
        <PitchCorner corner="top-right" />
        <PitchCorner corner="bottom-left" />
        <PitchCorner corner="bottom-right" />
        {columns.map((col) => (
          <Stack key={col.headerKey} spacing={{ xs: 0.5, sm: 0.75 }} sx={{ flex: 1, minWidth: 0 }}>
            {col.rows.map((row) => (
              <DuelCell key={row.labelKey} label={t(row.labelKey)} caption={t(row.captionKey)} home={row.home} away={row.away} />
            ))}
          </Stack>
        ))}
      </Box>
    </>
  )
}

function PitchCorner({ corner }: { corner: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right' }) {
  const size = 14
  const vertical = corner.startsWith('top') ? 'top' : 'bottom'
  const horizontal = corner.endsWith('left') ? 'left' : 'right'
  const radiusCorner = `${corner.startsWith('top') ? 'Top' : 'Bottom'}${corner.endsWith('left') ? 'Left' : 'Right'}`
  return (
    <Box
      sx={{
        position: 'absolute',
        [vertical]: 4,
        [horizontal]: 4,
        width: size,
        height: size,
        borderTop: vertical === 'top' ? '2px solid rgba(255,255,255,0.6)' : 'none',
        borderBottom: vertical === 'bottom' ? '2px solid rgba(255,255,255,0.6)' : 'none',
        borderLeft: horizontal === 'left' ? '2px solid rgba(255,255,255,0.6)' : 'none',
        borderRight: horizontal === 'right' ? '2px solid rgba(255,255,255,0.6)' : 'none',
        [`border${radiusCorner}Radius`]: '100%',
      }}
    />
  )
}

function formatValue(n: number): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

function DuelCell({ label, caption, home, away }: { label: string; caption: string; home: number; away: number }) {
  const total = home + away
  const homeShare = total > 0 ? (home / total) * 100 : 50
  const awayShare = 100 - homeShare
  const homeStronger = home > away
  const awayStronger = away > home

  return (
    <Tooltip title={`${label} — ${caption}`} arrow enterTouchDelay={0}>
      <Box
        sx={{
          display: 'flex',
          borderRadius: 1,
          overflow: 'hidden',
          boxShadow: '0 0 0 2px rgba(255,255,255,0.65)',
          cursor: 'default',
        }}
      >
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            bgcolor: HOME_COLOR,
            color: '#fff',
            textAlign: 'center',
            py: { xs: 0.75, sm: 1 },
            borderRight: '2px solid rgba(255,255,255,0.65)',
          }}
        >
          <Typography variant="body2" sx={{ fontWeight: homeStronger ? 800 : 500, lineHeight: 1.1 }}>
            {Math.round(homeShare)}%
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.85, fontSize: '0.65rem', display: 'block' }}>
            ({formatValue(home)})
          </Typography>
        </Box>
        <Box sx={{ flex: 1, minWidth: 0, bgcolor: AWAY_COLOR, color: '#fff', textAlign: 'center', py: { xs: 0.75, sm: 1 } }}>
          <Typography variant="body2" sx={{ fontWeight: awayStronger ? 800 : 500, lineHeight: 1.1 }}>
            {Math.round(awayShare)}%
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.85, fontSize: '0.65rem', display: 'block' }}>
            ({formatValue(away)})
          </Typography>
        </Box>
      </Box>
    </Tooltip>
  )
}
