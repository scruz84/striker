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

// The midfield battle is a genuinely symmetric contest - both teams' midfielders compete
// directly in the same flank zone for the ball - so comparing each side's own midfield rating
// zone-for-zone is the correct matchup, unlike attack/defense below.
function midfieldRows(homeQua: TeamQualification, awayQua: TeamQualification): ZoneRow[] {
  return [
    { labelKey: 'match.positionBattle.leftMidfield', home: homeQua.leftMidfield, away: awayQua.leftMidfield },
    { labelKey: 'match.positionBattle.centerMidfield', home: homeQua.midfield, away: awayQua.midfield },
    { labelKey: 'match.positionBattle.rightMidfield', home: homeQua.rightMidfield, away: awayQua.rightMidfield },
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

/** A zone-by-zone strength comparison built from the exact same cross-zone matchups
 * engine.processChance actually compares to decide the probability of a goal: each team's attack
 * on one flank against the opponent's defense on the MIRRORED flank plus the opponent's
 * goalkeeper - never a team's own zone against the opponent's identical zone (that would be
 * comparing e.g. both teams' goalkeepers to each other, which the engine never does and which
 * previously made this panel confusingly show "goalkeeping: home vs away" as if a keeper ever
 * faced the opposing keeper). Midfield is the one genuinely symmetric zone-vs-same-zone contest,
 * shown as-is. */
export function MatchPositionComparison({
  homeQua,
  awayQua,
  homeGoalkeeperRating,
  awayGoalkeeperRating,
  homeName,
  awayName,
}: MatchPositionComparisonProps) {
  const { t } = useTranslation()

  const homeAttackRows: ZoneRow[] = [
    { labelKey: 'match.positionBattle.leftAttack', home: homeQua.leftAttack, away: awayQua.rightDefense + awayGoalkeeperRating },
    { labelKey: 'match.positionBattle.centerAttack', home: homeQua.attack, away: awayQua.defense + awayGoalkeeperRating },
    { labelKey: 'match.positionBattle.rightAttack', home: homeQua.rightAttach, away: awayQua.leftDefense + awayGoalkeeperRating },
  ]
  const awayAttackRows: ZoneRow[] = [
    { labelKey: 'match.positionBattle.leftAttack', home: homeQua.rightDefense + homeGoalkeeperRating, away: awayQua.leftAttack },
    { labelKey: 'match.positionBattle.centerAttack', home: homeQua.defense + homeGoalkeeperRating, away: awayQua.attack },
    { labelKey: 'match.positionBattle.rightAttack', home: homeQua.leftDefense + homeGoalkeeperRating, away: awayQua.rightAttach },
  ]

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
        <Box sx={{ width: { xs: 90, sm: 120 } }} />
        <Typography variant="caption" sx={{ flex: 1, fontWeight: 700, pl: 1 }} noWrap>
          {awayName}
        </Typography>
      </Stack>
      <Stack spacing={1.5}>
        <Stack spacing={1}>
          {midfieldRows(homeQua, awayQua).map((row) => (
            <ZoneBar key={row.labelKey} label={t(row.labelKey)} home={row.home} away={row.away} />
          ))}
        </Stack>
        <Stack spacing={0.5}>
          <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
            {t('match.positionBattle.attackingHeader', { team: homeName })}
          </Typography>
          <Stack spacing={1}>
            {homeAttackRows.map((row) => (
              <ZoneBar key={`home-${row.labelKey}`} label={t(row.labelKey)} home={row.home} away={row.away} />
            ))}
          </Stack>
        </Stack>
        <Stack spacing={0.5}>
          <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
            {t('match.positionBattle.attackingHeader', { team: awayName })}
          </Typography>
          <Stack spacing={1}>
            {awayAttackRows.map((row) => (
              <ZoneBar key={`away-${row.labelKey}`} label={t(row.labelKey)} home={row.home} away={row.away} />
            ))}
          </Stack>
        </Stack>
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
      <Box sx={{ width: { xs: 90, sm: 120 } }}>
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
