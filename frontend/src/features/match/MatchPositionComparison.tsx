import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { TeamQualification } from '../../api/types'

interface ZoneRow {
  labelKey: string
  home: number
  away: number
}

function buildRows(homeQua: TeamQualification, awayQua: TeamQualification, homeGk: number, awayGk: number): ZoneRow[] {
  // Top to bottom in the same order the pitch itself draws these rows (strikers near the top,
  // goalkeeper at the bottom - see components/pitch/pitchLayout.ts's ROW_Y).
  return [
    { labelKey: 'match.positionBattle.leftAttack', home: homeQua.leftAttack, away: awayQua.leftAttack },
    { labelKey: 'match.positionBattle.centerAttack', home: homeQua.attack, away: awayQua.attack },
    { labelKey: 'match.positionBattle.rightAttack', home: homeQua.rightAttach, away: awayQua.rightAttach },
    { labelKey: 'match.positionBattle.leftMidfield', home: homeQua.leftMidfield, away: awayQua.leftMidfield },
    { labelKey: 'match.positionBattle.centerMidfield', home: homeQua.midfield, away: awayQua.midfield },
    { labelKey: 'match.positionBattle.rightMidfield', home: homeQua.rightMidfield, away: awayQua.rightMidfield },
    { labelKey: 'match.positionBattle.leftDefense', home: homeQua.leftDefense, away: awayQua.leftDefense },
    { labelKey: 'match.positionBattle.centerDefense', home: homeQua.defense, away: awayQua.defense },
    { labelKey: 'match.positionBattle.rightDefense', home: homeQua.rightDefense, away: awayQua.rightDefense },
    { labelKey: 'match.positionBattle.goalkeeping', home: homeGk, away: awayGk },
  ]
}

interface MatchPositionComparisonProps {
  homeQua: TeamQualification
  awayQua: TeamQualification
  homeGoalkeeperRating: number
  awayGoalkeeperRating: number
  homeName: string
  awayName: string
}

/** A zone-by-zone home-vs-away strength comparison, in the same top-to-bottom order the pitch
 * itself draws (attack, midfield, defense, goalkeeper) - these are the exact numbers
 * engine.processChance compares (a team's attack on one flank against the opponent's defense on
 * the mirrored flank) to decide whether a scoring chance becomes a goal, not a cosmetic summary. */
export function MatchPositionComparison({
  homeQua,
  awayQua,
  homeGoalkeeperRating,
  awayGoalkeeperRating,
  homeName,
  awayName,
}: MatchPositionComparisonProps) {
  const { t } = useTranslation()
  const rows = buildRows(homeQua, awayQua, homeGoalkeeperRating, awayGoalkeeperRating)

  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('match.positionBattle.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('match.positionBattle.description')}
      </Typography>
      <Stack direction="row" sx={{ mb: 1 }}>
        <Typography variant="caption" sx={{ flex: 1, fontWeight: 700, textAlign: 'right', pr: 1 }} noWrap>
          {homeName}
        </Typography>
        <Box sx={{ width: { xs: 90, sm: 160 } }} />
        <Typography variant="caption" sx={{ flex: 1, fontWeight: 700, pl: 1 }} noWrap>
          {awayName}
        </Typography>
      </Stack>
      <Stack spacing={1}>
        {rows.map((row) => (
          <ZoneBar key={row.labelKey} label={t(row.labelKey)} home={row.home} away={row.away} />
        ))}
      </Stack>
    </Paper>
  )
}

function ZoneBar({ label, home, away }: { label: string; home: number; away: number }) {
  const total = home + away
  // Each side's share of the zone, as a 0-100 pair summing to 100 (split evenly when neither team
  // has anything here). The bar itself halves that share again so a side never overruns the
  // center divider even when the opponent's value is 0.
  const homeShare = total > 0 ? (home / total) * 100 : 50
  const awayShare = 100 - homeShare
  const homePct = homeShare / 2
  const awayPct = awayShare / 2
  const homeStronger = home > away
  const awayStronger = away > home

  return (
    <Stack direction="row" sx={{ alignItems: 'center' }}>
      <Typography
        variant="body2"
        sx={{ flex: 1, textAlign: 'right', pr: 1, fontWeight: homeStronger ? 700 : 400 }}
      >
        {Math.round(homeShare)}%
      </Typography>
      <Box sx={{ width: { xs: 90, sm: 160 } }}>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', mb: 0.25 }}>
          {label}
        </Typography>
        <Box sx={{ position: 'relative', height: 8, bgcolor: 'action.hover', borderRadius: 1 }}>
          <Box sx={{ position: 'absolute', left: '50%', top: 0, bottom: 0, width: '1px', bgcolor: 'divider' }} />
          <Box
            sx={{
              position: 'absolute',
              right: '50%',
              top: 0,
              bottom: 0,
              width: `${homePct}%`,
              bgcolor: 'primary.main',
              borderRadius: '4px 0 0 4px',
            }}
          />
          <Box
            sx={{
              position: 'absolute',
              left: '50%',
              top: 0,
              bottom: 0,
              width: `${awayPct}%`,
              bgcolor: 'secondary.main',
              borderRadius: '0 4px 4px 0',
            }}
          />
        </Box>
      </Box>
      <Typography variant="body2" sx={{ flex: 1, pl: 1, fontWeight: awayStronger ? 700 : 400 }}>
        {Math.round(awayShare)}%
      </Typography>
    </Stack>
  )
}
