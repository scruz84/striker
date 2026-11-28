import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import { useTranslation } from 'react-i18next'
import type { RetiredPlayerDetails } from '../../api/types'
import { POSITION_LABEL_KEYS } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { CountryFlag } from '../../components/CountryFlag'
import { TeamNameLink } from '../../components/TeamNameLink'
import { PlayerSeasonStatsSection } from '../stats/PlayerSeasonStatsSection'
import { ClubHistorySection } from './ClubHistorySection'
import { PlayerTransferHistory } from './PlayerTransferHistory'

// The profile of a player who has retired (openspec retired-players "Retired player profile"): who
// he was and what he did, read-only - no skills, contract or market actions.
export function RetiredPlayerView({ profile }: { profile: RetiredPlayerDetails }) {
  const { t } = useTranslation()
  const player = profile.retired
  const rows: [string, number][] = [
    [t('playerPage.stats.appearances'), profile.appearances],
    [t('playerPage.stats.goals'), profile.goals],
    [t('playerPage.stats.caps'), profile.caps],
    [t('playerPage.stats.goldenBoots'), profile.seasonAwards.goldenBoots],
    [t('playerPage.stats.playerOfSeasons'), profile.seasonAwards.playerOfSeasons],
    [t('playerPage.stats.teamsOfTheWeek'), profile.bestXi.weeks],
    [t('playerPage.stats.teamsOfTheSeason'), profile.bestXi.seasons],
    [t('playerPage.stats.yellowCards'), profile.yellowCards],
    [t('playerPage.stats.redCards'), profile.redCards],
  ]
  return (
    <Stack spacing={{ xs: 1.5, sm: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Stack direction="row" spacing={2} sx={{ alignItems: 'center', minWidth: 0 }}>
          <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={64} />
          <Box sx={{ minWidth: 0 }}>
            <Typography variant="h5" component="h1" sx={{ overflowWrap: 'anywhere' }}>
              {player.birthCountry && <CountryFlag code={player.birthCountry} height={18} marginRight={0.75} />}
              {player.firstName} {player.lastName}
            </Typography>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
              <Chip size="small" label={t('retiredPlayer.chip')} />
              <Typography variant="body2" color="text.secondary">
                {t(POSITION_LABEL_KEYS[player.bestPosition] ?? 'positions.midfield')}
              </Typography>
            </Stack>
          </Box>
        </Stack>
        <Typography variant="body2" sx={{ mt: 1.5 }}>
          {player.retiredSeason !== undefined
            ? t('retiredPlayer.retiredInSeason', { season: player.retiredSeason, age: player.age })
            : t('retiredPlayer.retiredAtAge', { age: player.age })}
          {player.lastTeamId !== undefined && player.lastTeamName && (
            <>
              {' · '}
              {t('retiredPlayer.lastClub')} <TeamNameLink teamId={player.lastTeamId}>{player.lastTeamName}</TeamNameLink>
            </>
          )}
        </Typography>
        <Stack spacing={0.75} sx={{ mt: 2 }}>
          {rows.map(([label, value]) => (
            <Stack key={label} direction="row" sx={{ justifyContent: 'space-between', gap: 1 }}>
              <Typography variant="body2" color="text.secondary">
                {label}
              </Typography>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                {value}
              </Typography>
            </Stack>
          ))}
        </Stack>
      </Paper>
      <PlayerSeasonStatsSection lines={profile.seasonStats} />
      <ClubHistorySection entries={profile.clubHistory} />
      <PlayerTransferHistory transfers={profile.transfers} />
    </Stack>
  )
}
