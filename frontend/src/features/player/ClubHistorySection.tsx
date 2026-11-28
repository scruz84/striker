import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { PlayerClubHistoryEntry } from '../../api/types'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatDateTime } from '../../utils/formatDate'

// Every club a player has belonged to, most recent first; nothing without one.
export function ClubHistorySection({ entries }: { entries: PlayerClubHistoryEntry[] }) {
  const { t } = useTranslation()
  if (entries.length === 0) return null
  return (
    <Stack spacing={1}>
      <Typography variant="h6">{t('playerPage.clubHistory.title')}</Typography>
      <TableContainer component={Paper}>
        <Table size="small" aria-label="Club history" sx={{ '& td, & th': { px: { xs: 0.75, sm: 2 } } }}>
          <TableHead>
            <TableRow>
              <TableCell>{t('playerPage.clubHistory.club')}</TableCell>
              <TableCell align="right">{t('playerPage.clubHistory.joined')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {entries.map((entry) => (
              <TableRow key={`${entry.teamId}-${entry.joinedAt}`}>
                <TableCell>
                  <TeamNameLink teamId={entry.teamId}>{entry.teamName}</TeamNameLink>
                </TableCell>
                <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                  {formatDateTime(entry.joinedAt)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}
