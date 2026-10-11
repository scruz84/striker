import { useState } from 'react'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useCurrentWorldCup, useRecentNationalTeamFriendlies, useUpcomingNationalTeamFriendlies } from '../../api/hooks'
import type { FriendlyMatch, WorldCupGroup, WorldCupGroupMatch } from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'
import { localizedTeamName } from '../../utils/nationalTeamName'
import { groupByRound, roundName, TieRow } from '../cup/CupPage'
import { NestedTeamNameLink, TeamNameLink } from '../../components/TeamNameLink'

/** Where a manager watches the parts of the game no single club owns: the weekly national-team
 * friendlies (see backend engine.ScheduleNationalTeamFriendlies) and, once one is running, the
 * World Cup's group stage and knockout bracket (see backend engine.StartWorldCupForNewSeason).
 * The knockout bracket itself reuses CupPage's rendering outright via useCupDetails - a tie is a
 * tie regardless of which competition it belongs to, same reasoning as InternationalCupPage. */
export function NationalTeamsPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)

  return (
    <Stack spacing={2}>
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        {t('nationalTeams.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {t('nationalTeams.subtitle')}
      </Typography>

      <Paper variant="outlined">
        <Tabs value={tab} onChange={(_e, value) => setTab(value)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tab label={t('nationalTeams.tabs.friendlies')} />
          <Tab label={t('nationalTeams.tabs.worldCup')} />
        </Tabs>
        <Box sx={{ p: 2 }}>{tab === 0 ? <NationalTeamFriendlies /> : <WorldCupSection />}</Box>
      </Paper>
    </Stack>
  )
}

function NationalTeamFriendlies() {
  const { t } = useTranslation()
  const { data: upcoming, isLoading: upcomingLoading } = useUpcomingNationalTeamFriendlies(true)
  const { data: recent, isLoading: recentLoading } = useRecentNationalTeamFriendlies(true)

  return (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 6 }}>
        <FriendlyMatchList
          title={t('nationalTeams.friendlies.upcoming')}
          emptyLabel={t('nationalTeams.friendlies.upcomingEmpty')}
          matches={upcoming}
          isLoading={upcomingLoading}
          showResult={false}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6 }}>
        <FriendlyMatchList
          title={t('nationalTeams.friendlies.recent')}
          emptyLabel={t('nationalTeams.friendlies.recentEmpty')}
          matches={recent}
          isLoading={recentLoading}
          showResult
        />
      </Grid>
    </Grid>
  )
}

// Every row on this page is a national team, by construction (useUpcomingNationalTeamFriendlies/
// useRecentNationalTeamFriendlies/useCurrentWorldCup never return a club) - so localizedTeamName
// is always safe to apply here, unlike a general match list that might also show a real club.

function FriendlyMatchList({
  title,
  emptyLabel,
  matches,
  isLoading,
  showResult,
}: {
  title: string
  emptyLabel: string
  matches?: FriendlyMatch[]
  isLoading: boolean
  showResult: boolean
}) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {title}
      </Typography>
      {isLoading || !matches ? (
        <Skeleton variant="rectangular" height={80} />
      ) : matches.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {emptyLabel}
        </Typography>
      ) : (
        <Stack spacing={1}>
          {matches.map((m) => (
            <Stack
              key={m.matchId}
              component={RouterLink}
              to={`/friendlies/${m.matchId}`}
              direction="row"
              spacing={1}
              sx={{ alignItems: 'center', textDecoration: 'none', color: 'inherit', '&:hover': { opacity: 0.8 } }}
            >
              <NestedTeamNameLink teamId={m.homeTeamId} variant="body2" sx={{ flex: 1 }}>
                {localizedTeamName(t, m.homeTeamName)}
              </NestedTeamNameLink>
              <Typography variant="body2" sx={{ minWidth: 90, textAlign: 'center' }}>
                {showResult ? `${m.homeGoals} - ${m.awayGoals}` : formatDateTime(m.matchDate)}
              </Typography>
              <NestedTeamNameLink teamId={m.awayTeamId} variant="body2" sx={{ flex: 1, textAlign: 'right' }}>
                {localizedTeamName(t, m.awayTeamName)}
              </NestedTeamNameLink>
            </Stack>
          ))}
        </Stack>
      )}
    </Paper>
  )
}

function WorldCupSection() {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useCurrentWorldCup(true)
  const { data: knockoutCup, isLoading: knockoutLoading } = useCupDetails(
    data?.worldCup.stage === 'knockout' ? (data.worldCup.knockoutCupId ?? undefined) : undefined,
  )

  if (isLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }
  if (isError || !data) {
    return (
      <Typography variant="body2" color="text.secondary">
        {t('nationalTeams.worldCup.none')}
      </Typography>
    )
  }

  const { worldCup, groups } = data

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography variant="h6">{t('nationalTeams.worldCup.titleWithSeason', { name: worldCup.name, season: worldCup.season })}</Typography>
        {worldCup.completed && (
          <Chip
            color="success"
            label={t('nationalTeams.worldCup.champion', { name: localizedTeamName(t, worldCup.championTeamName) })}
          />
        )}
      </Stack>

      <Typography variant="subtitle1">{t('nationalTeams.worldCup.groupStage')}</Typography>
      <Grid container spacing={2}>
        {groups.map((group) => (
          <Grid key={group.groupId} size={{ xs: 12, md: 6 }}>
            <WorldCupGroupCard group={group} />
          </Grid>
        ))}
      </Grid>

      {worldCup.stage === 'knockout' && (
        <Stack spacing={2}>
          <Typography variant="subtitle1">{t('nationalTeams.worldCup.knockoutStage')}</Typography>
          {knockoutLoading || !knockoutCup ? (
            <Skeleton variant="rectangular" height={200} />
          ) : (
            (() => {
              const rounds = groupByRound(knockoutCup.ties)
              return rounds.map(({ round, ties }) => (
                <Paper key={round} variant="outlined" sx={{ p: 2 }}>
                  <Typography variant="subtitle2" gutterBottom>
                    {roundName(t, round, rounds.length)}
                  </Typography>
                  <Stack spacing={1}>
                    {ties.map((tie) => (
                      <TieRow key={tie.tieId} tie={tie} />
                    ))}
                  </Stack>
                </Paper>
              ))
            })()
          )}
        </Stack>
      )}
    </Stack>
  )
}

function WorldCupGroupCard({ group }: { group: WorldCupGroup }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="subtitle2" gutterBottom>
        {t('nationalTeams.worldCup.group', { name: group.groupName })}
      </Typography>
      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('nationalTeams.worldCup.standings.team')}</TableCell>
              <TableCell align="right">{t('nationalTeams.worldCup.standings.played')}</TableCell>
              <TableCell align="right">{t('nationalTeams.worldCup.standings.goalDifference')}</TableCell>
              <TableCell align="right">{t('nationalTeams.worldCup.standings.points')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {group.standings.map((standing, i) => (
              <TableRow key={standing.teamId} sx={{ fontWeight: i < 2 ? 700 : 400 }}>
                <TableCell sx={{ fontWeight: i < 2 ? 700 : 400 }}>
                  <TeamNameLink teamId={standing.teamId}>{localizedTeamName(t, standing.teamName)}</TeamNameLink>
                </TableCell>
                <TableCell align="right">{standing.played}</TableCell>
                <TableCell align="right">{standing.goalsFor - standing.goalsAgainst}</TableCell>
                <TableCell align="right">{standing.won * 3 + standing.drawn}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Stack spacing={0.5} sx={{ mt: 1.5 }}>
        {group.matches.map((match) => (
          <WorldCupGroupMatchRow key={match.matchId} match={match} />
        ))}
      </Stack>
    </Paper>
  )
}

function WorldCupGroupMatchRow({ match }: { match: WorldCupGroupMatch }) {
  const { t } = useTranslation()
  return (
    <Stack
      component={RouterLink}
      to={`/friendlies/${match.matchId}`}
      direction="row"
      spacing={1}
      sx={{ alignItems: 'center', textDecoration: 'none', color: 'inherit', '&:hover': { opacity: 0.8 } }}
    >
      <NestedTeamNameLink teamId={match.homeTeamId} variant="caption" sx={{ flex: 1 }}>
        {localizedTeamName(t, match.homeTeamName)}
      </NestedTeamNameLink>
      <Typography variant="caption" sx={{ minWidth: 60, textAlign: 'center' }}>
        {match.played ? `${match.homeGoals} - ${match.awayGoals}` : '—'}
      </Typography>
      <NestedTeamNameLink teamId={match.awayTeamId} variant="caption" sx={{ flex: 1, textAlign: 'right' }}>
        {localizedTeamName(t, match.awayTeamName)}
      </NestedTeamNameLink>
    </Stack>
  )
}
