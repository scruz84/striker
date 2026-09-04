import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import Box from '@mui/material/Box'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDivisionDetails } from '../../api/hooks'
import type { WeekMatch } from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'

export function DivisionDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: divisionDetails, isLoading } = useDivisionDetails(id)

  if (isLoading || !divisionDetails) {
    return <div>{t('common.loading')}</div>
  }

  const currentWeek = divisionDetails.classification.positions[0]?.matches ?? 0

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('division.titleWithWeek', { name: divisionDetails.name, week: currentWeek })}
      </Typography>

      <TableContainer component={Paper} sx={{ mb: 2 }}>
        <Table size="small" aria-label="Classification">
          <TableHead>
            <TableRow>
              <TableCell>{t('division.table.position')}</TableCell>
              <TableCell>{t('division.table.team')}</TableCell>
              <TableCell align="center">{t('division.table.points')}</TableCell>
              <TableCell align="center">{t('division.table.average')}</TableCell>
              <TableCell align="center">{t('division.table.wins')}</TableCell>
              <TableCell align="center">{t('division.table.draws')}</TableCell>
              <TableCell align="center">{t('division.table.losses')}</TableCell>
              <TableCell align="center">{t('division.table.goalsFor')}</TableCell>
              <TableCell align="center">{t('division.table.goalsAgainst')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {divisionDetails.classification.positions.map((row) => (
              <TableRow key={row.position}>
                <TableCell component="th" scope="row">
                  {row.position}
                </TableCell>
                <TableCell>
                  <RouterLink to={`/team/${row.teamId}`} style={{ color: 'inherit' }}>
                    {row.teamName}
                  </RouterLink>
                </TableCell>
                <TableCell align="center">{row.points}</TableCell>
                <TableCell align="center">{row.goalsScored - row.goalsReceived}</TableCell>
                <TableCell align="center">{row.wins}</TableCell>
                <TableCell align="center">{row.draws}</TableCell>
                <TableCell align="center">{row.losses}</TableCell>
                <TableCell align="center">{row.goalsScored}</TableCell>
                <TableCell align="center">{row.goalsReceived}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Paper sx={{ mb: 2, p: 1 }}>
        <Typography variant="subtitle1" sx={{ px: 1, pt: 1 }}>
          {t('division.lastWeek')}
        </Typography>
        <WeekMatchesList weekMatches={divisionDetails.previousWeekMatches} />
      </Paper>
      <Paper sx={{ p: 1 }}>
        <Typography variant="subtitle1" sx={{ px: 1, pt: 1 }}>
          {t('division.nextWeek')}
        </Typography>
        <WeekMatchesList weekMatches={divisionDetails.nextWeekMatches} />
      </Paper>
    </Box>
  )
}

function WeekMatchesList({ weekMatches }: { weekMatches: WeekMatch[] }) {
  const { t } = useTranslation()
  if (weekMatches.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ px: 1, pb: 1 }}>
        {t('division.noMatches')}
      </Typography>
    )
  }

  return (
    <List>
      {weekMatches.map((match) => (
        <ListItemButton key={match.matchId} component={RouterLink} to={`/match/${match.matchId}`}>
          <Stack
            direction="row"
            sx={{ width: '100%', justifyContent: 'space-between', alignItems: 'center' }}
          >
            <Box>
              <Typography variant="body2">
                {match.homeName} - {match.awayName}
              </Typography>
              <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
                <CalendarTodayTwoToneIcon fontSize="inherit" />
                <Typography variant="caption" color="text.secondary">
                  {formatDateTime(match.matchDate)}
                </Typography>
              </Stack>
            </Box>
            <Typography variant="body2">
              {match.played ? `${match.homeGoals} - ${match.awayGoals}` : '-'}
            </Typography>
          </Stack>
        </ListItemButton>
      ))}
    </List>
  )
}
