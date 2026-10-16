import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useTranslation } from 'react-i18next'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'

// Shared by the league match details page and the friendly match page - a live or played
// match's score header doesn't depend on anything competitive-match-specific.

export interface ScoreboardProps {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  homeGoals: number
  awayGoals: number
  status: 'scheduled' | 'live' | 'halftime' | 'connecting' | 'played'
  minute?: number
  subtitle?: string
  // The home team's stadium - shown as a caption next to the status chip when known (omitted
  // for a friendly, which doesn't track stadium flavor - see WeekMatch.homeStadiumName).
  stadiumName?: string
  // Each team's current league standing (see ClassificationPosition.position), shown as a small
  // chip under their name - only known for a league match (WeekMatch.divisionId), so undefined
  // for a friendly or cup tie simply omits it rather than showing something misleading.
  homePosition?: number
  awayPosition?: number
}

export function Scoreboard({
  homeId,
  homeName,
  awayId,
  awayName,
  homeGoals,
  awayGoals,
  status,
  minute,
  subtitle,
  stadiumName,
  homePosition,
  awayPosition,
}: ScoreboardProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
  const badgeSize = isMobile ? 40 : 56

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden', mb: 2 }}>
      <Box
        sx={{
          height: 4,
          background: `linear-gradient(90deg, ${theme.palette.primary.main}, ${theme.palette.primary.light})`,
        }}
      />
      <Box sx={{ p: { xs: 1.5, sm: 3 } }}>
        {stadiumName && (
          <Typography variant="caption" color="text.secondary" align="center" sx={{ display: 'block', mb: 1 }}>
            {t('match.venue', { name: stadiumName })}
          </Typography>
        )}
        <Stack direction="row" sx={{ justifyContent: 'center', mb: 1.5 }}>
          {status === 'live' && (
            <Chip
              size="small"
              color="error"
              label={t('match.live')}
              sx={{
                fontWeight: 700,
                animation: 'pulse 1.6s ease-in-out infinite',
                '@keyframes pulse': {
                  '0%, 100%': { opacity: 1 },
                  '50%': { opacity: 0.55 },
                },
              }}
            />
          )}
          {status === 'halftime' && <Chip size="small" variant="outlined" color="warning" label={t('match.halftime')} />}
          {status === 'connecting' && <Chip size="small" label={t('match.connecting')} />}
          {status === 'scheduled' && <Chip size="small" variant="outlined" label={t('match.scheduled')} />}
          {status === 'played' && <Chip size="small" variant="outlined" label={t('match.fullTime')} />}
        </Stack>
        {/* Flexbox rather than a 12-column Grid split - the center score needs a
            width that fits "88 - 3" without wrapping (see issue report: a fractional
            Grid column was too narrow on a phone screen and wrapped the score onto two
            lines), so it gets a fixed min-width and the two team columns share whatever
            space is left instead of a fixed fraction each. */}
        <Stack direction="row" spacing={{ xs: 0.5, sm: 2 }} sx={{ alignItems: 'center' }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Stack spacing={0.5} sx={{ alignItems: 'center', textAlign: 'center' }}>
              <TeamBadge teamId={homeId} teamName={homeName} size={badgeSize} />
              <Typography
                variant="subtitle1"
                noWrap
                sx={{ fontWeight: 600, maxWidth: '100%', fontSize: { xs: '0.75rem', sm: '1rem' } }}
              >
                <TeamNameLink teamId={homeId}>{homeName}</TeamNameLink>
              </Typography>
              {typeof homePosition === 'number' && (
                <Chip size="small" variant="outlined" label={t('match.leaguePosition', { position: homePosition })} />
              )}
            </Stack>
          </Box>
          <Box sx={{ flex: '0 0 auto', minWidth: { xs: 64, sm: 96 }, textAlign: 'center' }}>
            <Typography sx={{ fontWeight: 700, fontSize: { xs: '1.5rem', sm: '3rem' }, whiteSpace: 'nowrap' }}>
              {homeGoals} - {awayGoals}
            </Typography>
            {typeof minute === 'number' && minute > 0 && (
              <Typography variant="body2" align="center" color="text.secondary">
                {minute}&apos;
              </Typography>
            )}
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Stack spacing={0.5} sx={{ alignItems: 'center', textAlign: 'center' }}>
              <TeamBadge teamId={awayId} teamName={awayName} size={badgeSize} />
              <Typography
                variant="subtitle1"
                noWrap
                sx={{ fontWeight: 600, maxWidth: '100%', fontSize: { xs: '0.75rem', sm: '1rem' } }}
              >
                <TeamNameLink teamId={awayId}>{awayName}</TeamNameLink>
              </Typography>
              {typeof awayPosition === 'number' && (
                <Chip size="small" variant="outlined" label={t('match.leaguePosition', { position: awayPosition })} />
              )}
            </Stack>
          </Box>
        </Stack>
        {subtitle && (
          <Typography variant="caption" color="text.secondary" align="center" sx={{ display: 'block', mt: 1.5 }}>
            {subtitle}
          </Typography>
        )}
      </Box>
    </Paper>
  )
}
