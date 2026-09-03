import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useCupDetails, useLatestCupId, useMe } from '../../api/hooks'
import type { CupTie } from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'

export function CupPage() {
  const { data: loggedUser } = useMe()
  const { data: latestCup, isLoading: cupIdLoading } = useLatestCupId(loggedUser?.country)
  const { data: cupDetails, isLoading: detailsLoading } = useCupDetails(latestCup?.cupId)

  if (cupIdLoading || detailsLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!cupDetails) {
    return <Typography color="text.secondary">No cup found for your country yet.</Typography>
  }

  const { cup, ties } = cupDetails
  const rounds = groupByRound(ties)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {cup.name} - Season {cup.season}
      </Typography>
      {cup.completed && (
        <Chip color="success" label={`Champion: ${cup.championTeamName}`} sx={{ mb: 2 }} />
      )}

      {rounds.map(({ round, ties: roundTies }) => (
        <Paper key={round} sx={{ mb: 2, p: 2 }}>
          <Typography variant="subtitle1" gutterBottom>
            {roundName(round, rounds.length)}
          </Typography>
          <Stack spacing={1}>
            {roundTies.map((tie) => (
              <TieRow key={tie.tieId} tie={tie} />
            ))}
          </Stack>
        </Paper>
      ))}
    </Box>
  )
}

function TieRow({ tie }: { tie: CupTie }) {
  const isBye = tie.awayTeamId === null
  return (
    <Stack direction="row" spacing={1} sx={{ py: 0.5, alignItems: 'center' }}>
      <Typography variant="body2" sx={{ flex: 1 }}>
        {tie.homeTeamName || 'TBD'}
      </Typography>
      <Typography variant="body2" sx={{ minWidth: 80, textAlign: 'center' }}>
        {isBye ? 'Bye' : tie.played ? `${tie.homeGoals} - ${tie.awayGoals}` : formatDateTime(tie.matchDate ?? '')}
      </Typography>
      <Typography variant="body2" sx={{ flex: 1, textAlign: 'right' }}>
        {tie.awayTeamName || (isBye ? '' : 'TBD')}
      </Typography>
      {tie.wentToPenalties && (
        <Typography variant="caption" color="text.secondary">
          (pens)
        </Typography>
      )}
    </Stack>
  )
}

function groupByRound(ties: CupTie[]): { round: number; ties: CupTie[] }[] {
  const byRound = new Map<number, CupTie[]>()
  for (const tie of ties) {
    const list = byRound.get(tie.round) ?? []
    list.push(tie)
    byRound.set(tie.round, list)
  }
  return Array.from(byRound.entries())
    .sort(([a], [b]) => a - b)
    .map(([round, roundTies]) => ({
      round,
      ties: [...roundTies].sort((a, b) => a.slotNumber - b.slotNumber),
    }))
}

function roundName(round: number, totalRounds: number): string {
  const roundsFromEnd = totalRounds - round
  if (roundsFromEnd === 0) return 'Final'
  if (roundsFromEnd === 1) return 'Semifinal'
  if (roundsFromEnd === 2) return 'Quarterfinal'
  return `Round ${round}`
}
