import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Divider from '@mui/material/Divider'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useCurrentWorldCup, useWorldCupHistory } from '../../api/hooks'
import type { NationalMatch, WorldCupGroup, WorldCupGroupMatch } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TeamNameWithFlag } from '../../components/TeamNameWithFlag'
import { formatDate } from '../../utils/formatDate'
import { localizedTeamName, nationalTeamCountry } from '../../utils/nationalTeamName'
import { expectedTotalRounds, groupByRound, roundName, TieRow } from '../cup/CupPage'
import { NationalMatchRow } from './NationalMatchRow'

/** The World Cup: group tables (broadcast-safe, see backend common.GetWorldCupGroups), fixtures by
 * matchday, the knockout bracket (CupPage's own components via useCupDetails), the roll of honour
 * and, while none is running, when the next one is expected. */
export function WorldCupTab() {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useCurrentWorldCup(true)
  const worldCup = data?.worldCup ?? null
  const { data: knockoutCup, isLoading: knockoutLoading } = useCupDetails(
    worldCup?.stage === 'knockout' ? (worldCup.knockoutCupId ?? undefined) : undefined,
  )

  if (isLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const estimate = data?.nextWorldCupEstimate ? t('nationalTeams.worldCup.nextEstimate', { date: formatDate(data.nextWorldCupEstimate) }) : undefined

  return (
    <Stack spacing={2}>
      {isError || !worldCup ? (
        <EmptyState icon={<PublicOutlinedIcon fontSize="inherit" />} title={t('nationalTeams.worldCup.noneTitle')} text={estimate ?? t('nationalTeams.worldCup.none')} />
      ) : (
        <>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 1 }}>
            <Typography variant="h6" component="h2">
              {t('nationalTeams.worldCup.titleWithSeason', { name: worldCup.name, season: worldCup.season })}
            </Typography>
            {worldCup.completed && (
              <Chip color="success" label={t('nationalTeams.worldCup.champion', { name: localizedTeamName(t, worldCup.championTeamName) })} />
            )}
          </Stack>
          {worldCup.completed && estimate && (
            <Typography variant="body2" color="text.secondary">
              {estimate}
            </Typography>
          )}

          {worldCup.stage === 'knockout' && (
            <Stack spacing={2}>
              <Typography variant="subtitle1" component="h3">
                {t('nationalTeams.worldCup.knockoutStage')}
              </Typography>
              {knockoutLoading || !knockoutCup ? (
                <Skeleton variant="rectangular" height={200} />
              ) : (
                (() => {
                  const totalRounds = expectedTotalRounds(knockoutCup.ties)
                  return groupByRound(knockoutCup.ties).map(({ round, ties }) => (
                    <Paper key={round} variant="outlined" sx={{ p: 2 }}>
                      <Typography variant="subtitle2" gutterBottom>
                        {roundName(t, round, totalRounds)}
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

          <Typography variant="subtitle1" component="h3">
            {t('nationalTeams.worldCup.groupStage')}
          </Typography>
          <Grid container spacing={2}>
            {(data?.groups ?? []).map((group) => (
              <Grid key={group.groupId} size={{ xs: 12, md: 6 }}>
                <WorldCupGroupCard group={group} />
              </Grid>
            ))}
          </Grid>
        </>
      )}

      <RollOfHonour />
    </Stack>
  )
}

const STANDING_COLUMNS = ['played', 'won', 'drawn', 'lost', 'goalsFor', 'goalsAgainst', 'goalDifference', 'points'] as const

function WorldCupGroupCard({ group }: { group: WorldCupGroup }) {
  const { t } = useTranslation()
  const matchdays = new Map<number, WorldCupGroupMatch[]>()
  for (const match of group.matches) {
    matchdays.set(match.round, [...(matchdays.get(match.round) ?? []), match])
  }

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1, sm: 2 } }}>
      <Typography variant="subtitle2" component="h4" gutterBottom>
        {t('nationalTeams.worldCup.group', { name: group.groupName })}
      </Typography>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table
          size="small"
          sx={{
            tableLayout: 'fixed',
            '& td, & th': { px: { xs: 0.25, sm: 0.75 }, fontSize: { xs: '0.75rem', sm: '0.875rem' } },
            '& td:not(:first-of-type), & th:not(:first-of-type)': { width: { xs: 22, sm: 32 } },
          }}
        >
          <TableHead>
            <TableRow>
              <TableCell>{t('nationalTeams.worldCup.standings.team')}</TableCell>
              {STANDING_COLUMNS.map((column) => (
                <TableCell key={column} align="right">
                  {t(`nationalTeams.worldCup.standings.${column}`)}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {group.standings.map((standing, i) => {
              const values: Record<(typeof STANDING_COLUMNS)[number], number> = {
                played: standing.played,
                won: standing.won,
                drawn: standing.drawn,
                lost: standing.lost,
                goalsFor: standing.goalsFor,
                goalsAgainst: standing.goalsAgainst,
                goalDifference: standing.goalsFor - standing.goalsAgainst,
                points: standing.won * 3 + standing.drawn,
              }
              const qualifies = i < 2
              return (
                <TableRow key={standing.teamId} data-testid="group-standing-row">
                  <TableCell
                    sx={{
                      fontWeight: qualifies ? 700 : 400,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      borderLeft: 3,
                      borderLeftColor: qualifies ? 'success.main' : 'transparent',
                    }}
                  >
                    <TeamNameLink teamId={standing.teamId} title={localizedTeamName(t, standing.teamName)}>
                      <TeamNameWithFlag rawName={standing.teamName} />
                    </TeamNameLink>
                  </TableCell>
                  {STANDING_COLUMNS.map((column) => (
                    <TableCell key={column} align="right" sx={{ fontWeight: column === 'points' ? 700 : 400 }}>
                      {values[column]}
                    </TableCell>
                  ))}
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </TableContainer>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
        {t('nationalTeams.worldCup.qualifiersNote')}
      </Typography>

      <Stack spacing={1} sx={{ mt: 1.5 }}>
        {Array.from(matchdays.entries())
          .sort(([a], [b]) => a - b)
          .map(([matchday, matches]) => (
            <Box key={matchday}>
              <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1.6 }}>
                {t('nationalTeams.worldCup.matchday', { matchday, date: formatDate(matches[0].matchDate) })}
              </Typography>
              <Stack divider={<Divider flexItem />}>
                {matches.map((match) => (
                  <NationalMatchRow key={match.matchId} match={groupMatchAsNational(match, group.groupName)} hideLabel />
                ))}
              </Stack>
            </Box>
          ))}
      </Stack>
    </Paper>
  )
}

function groupMatchAsNational(match: WorldCupGroupMatch, groupName: string): NationalMatch {
  return {
    kind: 'worldCupGroup',
    id: match.matchId,
    homeTeamId: match.homeTeamId,
    homeTeamName: match.homeTeamName,
    homeCountry: nationalTeamCountry(match.homeTeamName) ?? '',
    awayTeamId: match.awayTeamId,
    awayTeamName: match.awayTeamName,
    awayCountry: nationalTeamCountry(match.awayTeamName) ?? '',
    kickoff: match.matchDate,
    status: match.status,
    minute: match.minute,
    homeGoals: match.homeGoals,
    awayGoals: match.awayGoals,
    groupName,
    round: match.round,
  }
}

function RollOfHonour() {
  const { t } = useTranslation()
  const { data } = useWorldCupHistory(true)
  if (!data || data.length === 0) return null

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
        <EmojiEventsOutlinedIcon color="warning" />
        <Typography variant="subtitle1" component="h3">
          {t('nationalTeams.worldCup.rollOfHonour')}
        </Typography>
      </Stack>
      <Stack divider={<Divider flexItem />}>
        {data.map((worldCup) => (
          <Stack key={worldCup.worldCupId} direction="row" spacing={1} sx={{ alignItems: 'center', minHeight: 44 }}>
            <Typography variant="body2" sx={{ flex: 1, minWidth: 0 }} noWrap>
              {t('nationalTeams.worldCup.titleWithSeason', { name: worldCup.name, season: worldCup.season })}
            </Typography>
            {worldCup.completed && worldCup.championTeamId ? (
              <TeamNameLink teamId={worldCup.championTeamId} variant="body2" sx={{ fontWeight: 700 }}>
                <TeamNameWithFlag rawName={worldCup.championTeamName} />
              </TeamNameLink>
            ) : (
              <Chip size="small" label={t('nationalTeams.worldCup.inProgress')} />
            )}
          </Stack>
        ))}
      </Stack>
    </Paper>
  )
}
