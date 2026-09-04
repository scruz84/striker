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
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: windowStatus } = useTransferWindowStatus(!!loggedUser)
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('transfers.title')}
      </Typography>
      {windowStatus && !windowStatus.open && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          {t('transfers.windowClosed')}
        </Alert>
      )}
      <Tabs value={tab} onChange={handleChange} sx={{ mb: 2 }}>
        <Tab label={t('transfers.tabs.transfers')} />
        <Tab label={t('transfers.tabs.loans')} />
        <Tab label={t('transfers.tabs.offers')} />
      </Tabs>
      {tab === 0 && <TransfersTab />}
      {tab === 1 && <LoansTab />}
      {tab === 2 && <OffersTab />}
    </Box>
  )
}

function TransfersTab() {
  const { t } = useTranslation()
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
          {t('transfers.listPlayer.title')}
        </Typography>
        {listPlayer.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {t('transfers.listPlayer.error')}
          </Alert>
        )}
        <Grid container spacing={2} sx={{ alignItems: 'center' }}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <FormControl fullWidth>
              <InputLabel id="list-player-label">{t('transfers.player')}</InputLabel>
              <Select
                labelId="list-player-label"
                label={t('transfers.player')}
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
              label={t('transfers.askingPrice')}
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
              {t('transfers.list')}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {myListings.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
            {t('transfers.yourListings')}
          </Typography>
          <ListingsTable
            listings={myListings}
            renderAction={(listing) => (
              <Button size="small" color="secondary" onClick={() => unlistPlayer.mutate(listing.player.id)}>
                {t('transfers.unlist')}
              </Button>
            )}
          />
        </Paper>
      )}

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          {t('transfers.market')}
        </Typography>
        {marketListings.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.noMarketListings')}
          </Typography>
        ) : (
          <ListingsTable
            listings={marketListings}
            renderAction={(listing) => (
              <Button size="small" variant="contained" onClick={() => buyListing.mutate(listing.listingId)} disabled={buyListing.isPending}>
                {t('transfers.buy')}
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
  const { t } = useTranslation()
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell>{t('transfers.table.team')}</TableCell>
            <TableCell>{t('transfers.table.listed')}</TableCell>
            <TableCell align="right">{t('transfers.table.price')}</TableCell>
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
  const { t } = useTranslation()
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
          {t('transfers.offerLoan.title')}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t('transfers.offerLoan.subtitle')}
        </Typography>
        {listForLoan.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {t('transfers.offerLoan.error')}
          </Alert>
        )}
        <Grid container spacing={2} sx={{ alignItems: 'center' }}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <FormControl fullWidth>
              <InputLabel id="loan-player-label">{t('transfers.player')}</InputLabel>
              <Select
                labelId="loan-player-label"
                label={t('transfers.player')}
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
              <InputLabel id="loan-duration-label">{t('transfers.duration')}</InputLabel>
              <Select
                labelId="loan-duration-label"
                label={t('transfers.duration')}
                value={duration}
                onChange={(event: SelectChangeEvent<number>) => setDuration(Number(event.target.value))}
              >
                {LOAN_DURATION_OPTIONS.map((seasons) => (
                  <MenuItem key={seasons} value={seasons}>
                    {t('transfers.seasonCount', { count: seasons })}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 4, sm: 2 }}>
            <Button fullWidth variant="contained" disabled={selectedPlayerId === '' || listForLoan.isPending} onClick={handleList}>
              {t('transfers.list')}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {myListings.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
            {t('transfers.yourLoanListings')}
          </Typography>
          <LoanListingsTable
            listings={myListings}
            renderAction={(listing) => (
              <Button size="small" color="secondary" onClick={() => unlistFromLoan.mutate(listing.player.id)}>
                {t('transfers.unlist')}
              </Button>
            )}
          />
        </Paper>
      )}

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          {t('transfers.loanMarket')}
        </Typography>
        {marketListings.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.noLoanListings')}
          </Typography>
        ) : (
          <LoanListingsTable
            listings={marketListings}
            renderAction={(listing) => (
              <Button size="small" variant="contained" onClick={() => acceptLoan.mutate(listing.listingId)} disabled={acceptLoan.isPending}>
                {t('transfers.accept')}
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
  const { t } = useTranslation()
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell>{t('transfers.table.team')}</TableCell>
            <TableCell>{t('transfers.table.listed')}</TableCell>
            <TableCell align="right">{t('transfers.table.duration')}</TableCell>
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
              <TableCell align="right">{t('transfers.seasonCount', { count: listing.durationSeasons })}</TableCell>
              <TableCell align="right">{renderAction(listing)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

function OffersTab() {
  const { t } = useTranslation()
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
        {t('transfers.offers.intro')}
      </Typography>

      <Paper sx={{ mb: 3 }}>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          {t('transfers.offers.onYourPlayers')}
        </Typography>
        {received.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.offers.noPending')}
          </Typography>
        ) : (
          <OffersTable
            offers={received}
            teamColumn="from"
            renderAction={(offer) => (
              <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
                <Button size="small" variant="contained" onClick={() => acceptOffer.mutate(offer.offerId)} disabled={acceptOffer.isPending}>
                  {t('transfers.accept')}
                </Button>
                <Button size="small" color="secondary" onClick={() => rejectOffer.mutate(offer.offerId)} disabled={rejectOffer.isPending}>
                  {t('transfers.reject')}
                </Button>
              </Stack>
            )}
          />
        )}
      </Paper>

      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
          {t('transfers.offers.yourOffers')}
        </Typography>
        {sent.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.offers.noneSent')}
          </Typography>
        ) : (
          <OffersTable
            offers={sent}
            teamColumn="to"
            renderAction={(offer) => (
              <Button size="small" color="secondary" onClick={() => withdrawOffer.mutate(offer.offerId)} disabled={withdrawOffer.isPending}>
                {t('transfers.withdraw')}
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
  const { t } = useTranslation()
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell>{teamColumn === 'from' ? t('transfers.table.from') : t('transfers.table.to')}</TableCell>
            <TableCell>{t('transfers.table.made')}</TableCell>
            <TableCell align="right">{t('transfers.table.amount')}</TableCell>
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
