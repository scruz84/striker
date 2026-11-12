import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { NationalMatch } from '../../api/types'
import { COUNTRY_NAME_KEYS } from '../../api/types'
import { CountryFlag } from '../../components/CountryFlag'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import { formatDayTime } from '../../utils/formatDate'
import { matchMinuteText } from '../../utils/matchMinute'
import { nationalMatchLabel, nationalMatchLink } from './nationalMatches'

/** One national-team match as a compact two-line row that fits a 360 px screen: the competition
 * and kickoff (or live minute) on top, then flag + one-line ellipsized name on each side of the
 * score. The score shown is whatever the backend reports for the match's status - live rows carry
 * the broadcast's score as of now, never the final simulated one (see backend
 * common.applyNationalMatchStatus), so nothing here has to hide anything. */
export function NationalMatchRow({ match, hideLabel = false }: { match: NationalMatch; hideLabel?: boolean }) {
  const { t } = useTranslation()
  const hasScore = match.status !== 'scheduled' && match.homeGoals !== undefined && match.awayGoals !== undefined

  return (
    <Box
      component={RouterLink}
      to={nationalMatchLink(match)}
      data-testid="national-match-row"
      sx={{
        display: 'block',
        minHeight: 44,
        py: 0.75,
        px: 1,
        borderRadius: 1,
        textDecoration: 'none',
        color: 'inherit',
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', minWidth: 0, mb: 0.5 }}>
        {!hideLabel && (
          <Chip
            label={nationalMatchLabel(t, match)}
            size="small"
            variant="outlined"
            sx={{ height: 20, fontSize: '0.6875rem', maxWidth: '60%' }}
          />
        )}
        {match.status === 'live' ? (
          <Chip
            label={t('nationalTeams.row.liveMinute', { minute: matchMinuteText(match.minute ?? 0, match.addedMinute) })}
            color="error"
            size="small"
            sx={{ height: 20, fontSize: '0.6875rem', fontWeight: 700 }}
          />
        ) : (
          <Typography variant="caption" color="text.secondary" noWrap>
            {match.status === 'finished' ? `${t('nationalTeams.row.fullTime')} · ${formatDayTime(match.kickoff)}` : formatDayTime(match.kickoff)}
          </Typography>
        )}
        {!!match.myPlayers && (
          <Chip
            icon={<GroupsOutlinedIcon />}
            label={t('nationalTeams.row.myPlayers', { count: match.myPlayers })}
            color="primary"
            size="small"
            sx={{ height: 20, fontSize: '0.6875rem', ml: 'auto !important' }}
          />
        )}
      </Stack>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto minmax(0, 1fr)', alignItems: 'center', columnGap: 1 }}>
        <TeamSide teamId={match.homeTeamId} country={match.homeCountry} rawName={match.homeTeamName} align="right" />
        <Typography
          variant="body2"
          sx={{ minWidth: 52, textAlign: 'center', fontWeight: 700, color: match.status === 'live' ? 'error.main' : 'text.primary' }}
        >
          {hasScore ? `${match.homeGoals} - ${match.awayGoals}` : t('nationalTeams.row.versus')}
        </Typography>
        <TeamSide teamId={match.awayTeamId} country={match.awayCountry} rawName={match.awayTeamName} align="left" />
      </Box>
    </Box>
  )
}

function TeamSide({ teamId, country, rawName, align }: { teamId: number; country: string; rawName: string; align: 'left' | 'right' }) {
  const { t } = useTranslation()
  const name = COUNTRY_NAME_KEYS[country] ? t(COUNTRY_NAME_KEYS[country]) : rawName
  return (
    <Stack
      direction={align === 'right' ? 'row-reverse' : 'row'}
      spacing={0.75}
      sx={{ alignItems: 'center', minWidth: 0, justifyContent: 'flex-start' }}
    >
      <Box sx={{ flexShrink: 0, display: 'flex' }}>
        <CountryFlag code={country} height={14} decorative />
      </Box>
      <NestedTeamNameLink teamId={teamId} variant="body2" noWrap title={name} sx={{ minWidth: 0, textAlign: align }}>
        {name}
      </NestedTeamNameLink>
    </Stack>
  )
}
