import { useState, type SyntheticEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
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
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import {
  useAcceptLoan,
  useAcceptOffer,
  useBuyListing,
  useListPlayer,
  useListPlayerForLoan,
  useLoanListings,
  useMe,
  useOffersReceived,
  useOffersSent,
  useRejectOffer,
  useTransferListings,
  useTransferWindowStatus,
  useUnlistPlayer,
  useUnlistPlayerForLoan,
  useUserTeam,
  useWithdrawOffer,
} from '../../api/hooks'
import type { LoanListing, Player, TransferListing, TransferOffer } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function TransferMarketPage() {
  const { data: loggedUser } = useMe()
  const { data: windowStatus } = useTransferWindowStatus(!!loggedUser)
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Transfer market
      </Typography>
      {windowStatus && !windowStatus.open && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          The transfer window is closed - listing, buying, and accepting loans and offers are
          unavailable until it reopens.
        </Alert>
      )}
      <Tabs value={tab} onChange={handleChange} sx={{ mb: 2 }}>
        <Tab label="Transfers" />
        <Tab label="Loans" />
        <Tab label="Offers" />
      </Tabs>
      {tab === 0 && <TransfersTab />}
      {tab === 1 && <LoansTab />}
      {tab === 2 && <OffersTab />}
    </Box>
  )
}

function TransfersTab() {
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: listings, isLoading: listingsLoading } = useTransferListings(!!loggedUser)

  const listPlayer = useListPlayer()
  const unlistPlayer = useUnlistPlayer()
  const buyListing = useBuyListing()

  const [selectedPlayerId, setSelectedPlayerId] = useState<number | ''>('')
  const [askingPrice, setAskingPrice] = useState('')

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const listedPlayerIds = new Set(listings.filter((l) => l.sellingTeamId === myTeamId).map((l) => l.player.id))
  const listablePlayers = userTeam.players.filter((p) => !listedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.sellingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.sellingTeamId !== myTeamId)

  const handleList = () => {
    const price = Number(askingPrice)
    if (selectedPlayerId === '' || !price || price <= 0) {
      return
    }
    listPlayer.mutate(
      { playerId: selectedPlayerId, askingPrice: price },
      {
        onSuccess: () => {
          setSelectedPlayerId('')
          setAskingPrice('')
        },
      },
    )
  }

  return (
    <Box>
      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom>
          List a player
        </Typography>
        {listPlayer.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            Could not list that player.
          </Alert>
        )}
        <Grid container spacing={2} sx={{ alignItems: 'center' }}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <FormControl fullWidth>
              <InputLabel id="list-player-label">Player</InputLabel>
              <Select
                labelId="list-player-label"
                label="Player"
                value={selectedPlayerId}
                onChange={(event: SelectChangeEvent<number | ''>) =>
                  setSelectedPlayerId(event.target.value === '' ? '' : Number(event.target.value))
                }
              >
                {listablePlayers.map((player) => (
                  <MenuItem key={player.id} value={player.id}>
                    {player.firstName} {player.lastName}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 8, sm: 4 }}>
            <TextField
              fullWidth
              label="Asking price"
              type="number"
              value={askingPrice}
              onChange={(event) => setAskingPrice(event.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 4, sm: 2 }}>
            <Button
              fullWidth
              variant="contained"
              disabled={selectedPlayerId === '' || !askingPrice || listPlayer.isPending}
              onClick={handleList}
            >
              List
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {myListings.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
            Your listings
          </Typography>
          <ListingsTable
            listings={myListings}
            renderAction={(listing) => (
              <Button size="small" color="secondary" onClick={() => unlistPlayer.mutate(listing.player.id)}>
                Unlist
              </Button>
            )}
          />
        </Paper>
      )}

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          Market
        </Typography>
        {marketListings.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            No players listed right now.
          </Typography>
        ) : (
          <ListingsTable
            listings={marketListings}
            renderAction={(listing) => (
              <Button size="small" variant="contained" onClick={() => buyListing.mutate(listing.listingId)} disabled={buyListing.isPending}>
                Buy
              </Button>
            )}
          />
        )}
      </Paper>
    </Box>
  )
}

interface ListingsTableProps {
  listings: TransferListing[]
  renderAction: (listing: TransferListing) => React.ReactNode
}

function ListingsTable({ listings, renderAction }: ListingsTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Player</TableCell>
            <TableCell>Team</TableCell>
            <TableCell>Listed</TableCell>
            <TableCell align="right">Price</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {listings.map((listing) => (
            <TableRow key={listing.listingId}>
              <TableCell>
                {listing.player.firstName} {listing.player.lastName}
              </TableCell>
              <TableCell>{listing.sellingTeamName}</TableCell>
              <TableCell>{formatDateTime(listing.listedAt)}</TableCell>
              <TableCell align="right">{formatCurrency(listing.askingPrice)}</TableCell>
              <TableCell align="right">{renderAction(listing)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

const LOAN_DURATION_OPTIONS = [1, 2, 3]

function LoansTab() {
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: listings, isLoading: listingsLoading } = useLoanListings(!!loggedUser)

  const listForLoan = useListPlayerForLoan()
  const unlistFromLoan = useUnlistPlayerForLoan()
  const acceptLoan = useAcceptLoan()

  const [selectedPlayerId, setSelectedPlayerId] = useState<number | ''>('')
  const [duration, setDuration] = useState<number>(1)

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const loanListedPlayerIds = new Set(listings.filter((l) => l.lendingTeamId === myTeamId).map((l) => l.player.id))
  const loanablePlayers: Player[] = userTeam.players.filter((p) => !loanListedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.lendingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.lendingTeamId !== myTeamId)

  const handleList = () => {
    if (selectedPlayerId === '') {
      return
    }
    listForLoan.mutate(
      { playerId: selectedPlayerId, durationSeasons: duration },
      { onSuccess: () => setSelectedPlayerId('') },
    )
  }

  return (
    <Box>
      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom>
          Offer a player on loan
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Free, no fee - the player returns automatically once the duration elapses.
        </Typography>
        {listForLoan.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            Could not list that player for loan.
          </Alert>
        )}
        <Grid container spacing={2} sx={{ alignItems: 'center' }}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <FormControl fullWidth>
              <InputLabel id="loan-player-label">Player</InputLabel>
              <Select
                labelId="loan-player-label"
                label="Player"
                value={selectedPlayerId}
                onChange={(event: SelectChangeEvent<number | ''>) =>
                  setSelectedPlayerId(event.target.value === '' ? '' : Number(event.target.value))
                }
              >
                {loanablePlayers.map((player) => (
                  <MenuItem key={player.id} value={player.id}>
                    {player.firstName} {player.lastName}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 8, sm: 4 }}>
            <FormControl fullWidth>
              <InputLabel id="loan-duration-label">Duration</InputLabel>
              <Select
                labelId="loan-duration-label"
                label="Duration"
                value={duration}
                onChange={(event: SelectChangeEvent<number>) => setDuration(Number(event.target.value))}
              >
                {LOAN_DURATION_OPTIONS.map((seasons) => (
                  <MenuItem key={seasons} value={seasons}>
                    {seasons} season{seasons > 1 ? 's' : ''}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 4, sm: 2 }}>
            <Button fullWidth variant="contained" disabled={selectedPlayerId === '' || listForLoan.isPending} onClick={handleList}>
              List
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {myListings.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
            Your loan listings
          </Typography>
          <LoanListingsTable
            listings={myListings}
            renderAction={(listing) => (
              <Button size="small" color="secondary" onClick={() => unlistFromLoan.mutate(listing.player.id)}>
                Unlist
              </Button>
            )}
          />
        </Paper>
      )}

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          Loan market
        </Typography>
        {marketListings.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            No players available on loan right now.
          </Typography>
        ) : (
          <LoanListingsTable
            listings={marketListings}
            renderAction={(listing) => (
              <Button size="small" variant="contained" onClick={() => acceptLoan.mutate(listing.listingId)} disabled={acceptLoan.isPending}>
                Accept
              </Button>
            )}
          />
        )}
      </Paper>
    </Box>
  )
}

interface LoanListingsTableProps {
  listings: LoanListing[]
  renderAction: (listing: LoanListing) => React.ReactNode
}

function LoanListingsTable({ listings, renderAction }: LoanListingsTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Player</TableCell>
            <TableCell>Team</TableCell>
            <TableCell>Listed</TableCell>
            <TableCell align="right">Duration</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {listings.map((listing) => (
            <TableRow key={listing.listingId}>
              <TableCell>
                {listing.player.firstName} {listing.player.lastName}
              </TableCell>
              <TableCell>{listing.lendingTeamName}</TableCell>
              <TableCell>{formatDateTime(listing.listedAt)}</TableCell>
              <TableCell align="right">
                {listing.durationSeasons} season{listing.durationSeasons > 1 ? 's' : ''}
              </TableCell>
              <TableCell align="right">{renderAction(listing)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

function OffersTab() {
  const { data: loggedUser } = useMe()
  const { data: received, isLoading: receivedLoading } = useOffersReceived(!!loggedUser)
  const { data: sent, isLoading: sentLoading } = useOffersSent(!!loggedUser)

  const acceptOffer = useAcceptOffer()
  const rejectOffer = useRejectOffer()
  const withdrawOffer = useWithdrawOffer()

  if (receivedLoading || sentLoading || !received || !sent) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Offers are made directly on a player from their team page - listed or not.
      </Typography>

      <Paper sx={{ mb: 3 }}>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          Offers on your players
        </Typography>
        {received.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            No pending offers.
          </Typography>
        ) : (
          <OffersTable
            offers={received}
            teamColumn="from"
            renderAction={(offer) => (
              <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
                <Button size="small" variant="contained" onClick={() => acceptOffer.mutate(offer.offerId)} disabled={acceptOffer.isPending}>
                  Accept
                </Button>
                <Button size="small" color="secondary" onClick={() => rejectOffer.mutate(offer.offerId)} disabled={rejectOffer.isPending}>
                  Reject
                </Button>
              </Stack>
            )}
          />
        )}
      </Paper>

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          Your offers
        </Typography>
        {sent.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            You haven't made any offers.
          </Typography>
        ) : (
          <OffersTable
            offers={sent}
            teamColumn="to"
            renderAction={(offer) => (
              <Button size="small" color="secondary" onClick={() => withdrawOffer.mutate(offer.offerId)} disabled={withdrawOffer.isPending}>
                Withdraw
              </Button>
            )}
          />
        )}
      </Paper>
    </Box>
  )
}

interface OffersTableProps {
  offers: TransferOffer[]
  teamColumn: 'from' | 'to'
  renderAction: (offer: TransferOffer) => React.ReactNode
}

function OffersTable({ offers, teamColumn, renderAction }: OffersTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Player</TableCell>
            <TableCell>{teamColumn === 'from' ? 'From' : 'To'}</TableCell>
            <TableCell>Made</TableCell>
            <TableCell align="right">Amount</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {offers.map((offer) => (
            <TableRow key={offer.offerId}>
              <TableCell>
                {offer.player.firstName} {offer.player.lastName}
              </TableCell>
              <TableCell>{teamColumn === 'from' ? offer.fromTeamName : offer.toTeamName}</TableCell>
              <TableCell>{formatDateTime(offer.createdAt)}</TableCell>
              <TableCell align="right">{formatCurrency(offer.amount)}</TableCell>
              <TableCell align="right">{renderAction(offer)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
