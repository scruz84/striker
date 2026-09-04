import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useMakeOffer, useMe, useTeamDetails, useUserTeam } from '../../api/hooks'
import type { Player } from '../../api/types'

export function TeamDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: loggedUser } = useMe()
  const { data: teamDetails, isLoading } = useTeamDetails(id)
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !teamDetails) {
    return <div>Loading...</div>
  }

  const isOwnTeam = userTeam?.team.id === teamDetails.team.id

  return (
    <Stack spacing={1}>
      <Typography variant="h4" gutterBottom>
        {teamDetails.team.name} ({teamDetails.team.id})
      </Typography>
      <Typography sx={{ fontWeight: 'bold' }}>Competitions:</Typography>
      <Link component={RouterLink} to={`/division/${teamDetails.team.divisionId}`}>
        {teamDetails.team.divisionName}
      </Link>

      <Typography variant="h6" sx={{ mt: 2 }}>
        Squad
      </Typography>
      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Player</TableCell>
              <TableCell align="right">Age</TableCell>
              {!isOwnTeam && <TableCell align="right" />}
            </TableRow>
          </TableHead>
          <TableBody>
            {teamDetails.players.map((player) => (
              <PlayerRow key={player.id} player={player} showOfferAction={!isOwnTeam} />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}

function PlayerRow({ player, showOfferAction }: { player: Player; showOfferAction: boolean }) {
  const makeOffer = useMakeOffer()
  const [amount, setAmount] = useState('')
  const [showForm, setShowForm] = useState(false)

  const handleSubmit = () => {
    const amountNum = Number(amount)
    if (!amountNum || amountNum <= 0) {
      return
    }
    makeOffer.mutate({ playerId: player.id, amount: amountNum }, { onSuccess: () => setShowForm(false) })
  }

  return (
    <TableRow>
      <TableCell>
        {player.firstName} {player.lastName}
      </TableCell>
      <TableCell align="right">{player.age}</TableCell>
      {showOfferAction && (
        <TableCell align="right">
          {showForm ? (
            <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', alignItems: 'center' }}>
              <TextField
                size="small"
                label="Amount"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                sx={{ width: 120 }}
              />
              <Button size="small" variant="contained" onClick={handleSubmit} disabled={makeOffer.isPending}>
                Send
              </Button>
              <Button size="small" onClick={() => setShowForm(false)}>
                Cancel
              </Button>
            </Stack>
          ) : (
            <Button size="small" onClick={() => setShowForm(true)}>
              Make offer
            </Button>
          )}
          {makeOffer.isError && (
            <Alert severity="error" sx={{ mt: 1 }}>
              Could not make that offer.
            </Alert>
          )}
        </TableCell>
      )}
    </TableRow>
  )
}
