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
import TableSortLabel from '@mui/material/TableSortLabel'
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

type SortDirection = 'asc' | 'desc'
type TransferSortField = 'age' | 'price' | 'listed'
type LoanSortField = 'age' | 'duration' | 'listed'

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

  const [search, setSearch] = useState('')
  const [minAge, setMinAge] = useState('')
  const [maxAge, setMaxAge] = useState('')
  const [minPrice, setMinPrice] = useState('')
  const [maxPrice, setMaxPrice] = useState('')
  const [sortField, setSortField] = useState<TransferSortField>('listed')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const listedPlayerIds = new Set(listings.filter((l) => l.sellingTeamId === myTeamId).map((l) => l.player.id))
  const listablePlayers = userTeam.players.filter((p) => !listedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.sellingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.sellingTeamId !== myTeamId)

  const hasActiveFilters = !!(search || minAge || maxAge || minPrice || maxPrice)
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMinPrice('')
    setMaxPrice('')
  }

  const filteredMarketListings = marketListings
    .filter((l) => {
      const term = search.trim().toLowerCase()
      if (term && !`${l.player.firstName} ${l.player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && l.player.age < Number(minAge)) return false
      if (maxAge && l.player.age > Number(maxAge)) return false
      if (minPrice && l.askingPrice < Number(minPrice)) return false
      if (maxPrice && l.askingPrice > Number(maxPrice)) return false
      return true
    })
    .sort((a, b) => {
      const cmp =
        sortField === 'price'
          ? a.askingPrice - b.askingPrice
          : sortField === 'age'
            ? a.player.age - b.player.age
            : new Date(a.listedAt).getTime() - new Date(b.listedAt).getTime()
      return sortDir === 'asc' ? cmp : -cmp
    })

  const handleSort = (field: TransferSortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

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
          <>
            <Stack direction="row" spacing={1} sx={{ p: 2, flexWrap: 'wrap' }}>
              <TextField
                size="small"
                label={t('transfers.filters.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={{ minWidth: 160 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minAge')}
                value={minAge}
                onChange={(e) => setMinAge(e.target.value)}
                sx={{ width: 100 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxAge')}
                value={maxAge}
                onChange={(e) => setMaxAge(e.target.value)}
                sx={{ width: 100 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minPrice')}
                value={minPrice}
                onChange={(e) => setMinPrice(e.target.value)}
                sx={{ width: 120 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxPrice')}
                value={maxPrice}
                onChange={(e) => setMaxPrice(e.target.value)}
                sx={{ width: 120 }}
              />
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Stack>
            {filteredMarketListings.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <ListingsTable
                listings={filteredMarketListings}
                sortField={sortField}
                sortDir={sortDir}
                onSort={handleSort}
                renderAction={(listing) => (
                  <Button
                    size="small"
                    variant="contained"
                    onClick={() => buyListing.mutate(listing.listingId)}
                    disabled={buyListing.isPending}
                  >
                    {t('transfers.buy')}
                  </Button>
                )}
              />
            )}
          </>
        )}
      </Paper>
    </Box>
  )
}

interface ListingsTableProps {
  listings: TransferListing[]
  renderAction: (listing: TransferListing) => React.ReactNode
  sortField?: TransferSortField
  sortDir?: SortDirection
  onSort?: (field: TransferSortField) => void
}

function ListingsTable({ listings, renderAction, sortField, sortDir, onSort }: ListingsTableProps) {
  const { t } = useTranslation()
  const sortable = !!onSort
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell>{t('transfers.table.team')}</TableCell>
            {sortable ? (
              <TableCell align="right" sortDirection={sortField === 'age' ? sortDir : false}>
                <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => onSort('age')}>
                  {t('transfers.table.age')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right">{t('transfers.table.age')}</TableCell>
            )}
            {sortable ? (
              <TableCell sortDirection={sortField === 'listed' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'listed'}
                  direction={sortField === 'listed' ? sortDir : 'asc'}
                  onClick={() => onSort('listed')}
                >
                  {t('transfers.table.listed')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell>{t('transfers.table.listed')}</TableCell>
            )}
            {sortable ? (
              <TableCell align="right" sortDirection={sortField === 'price' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'price'}
                  direction={sortField === 'price' ? sortDir : 'asc'}
                  onClick={() => onSort('price')}
                >
                  {t('transfers.table.price')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right">{t('transfers.table.price')}</TableCell>
            )}
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
              <TableCell align="right">{listing.player.age}</TableCell>
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

  const [search, setSearch] = useState('')
  const [minAge, setMinAge] = useState('')
  const [maxAge, setMaxAge] = useState('')
  const [maxDuration, setMaxDuration] = useState<number | ''>('')
  const [sortField, setSortField] = useState<LoanSortField>('listed')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const loanListedPlayerIds = new Set(listings.filter((l) => l.lendingTeamId === myTeamId).map((l) => l.player.id))
  const loanablePlayers: Player[] = userTeam.players.filter((p) => !loanListedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.lendingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.lendingTeamId !== myTeamId)

  const hasActiveFilters = !!(search || minAge || maxAge || maxDuration)
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMaxDuration('')
  }

  const filteredMarketListings = marketListings
    .filter((l) => {
      const term = search.trim().toLowerCase()
      if (term && !`${l.player.firstName} ${l.player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && l.player.age < Number(minAge)) return false
      if (maxAge && l.player.age > Number(maxAge)) return false
      if (maxDuration && l.durationSeasons > Number(maxDuration)) return false
      return true
    })
    .sort((a, b) => {
      const cmp =
        sortField === 'duration'
          ? a.durationSeasons - b.durationSeasons
          : sortField === 'age'
            ? a.player.age - b.player.age
            : new Date(a.listedAt).getTime() - new Date(b.listedAt).getTime()
      return sortDir === 'asc' ? cmp : -cmp
    })

  const handleSort = (field: LoanSortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

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
          <>
            <Stack direction="row" spacing={1} sx={{ p: 2, flexWrap: 'wrap' }}>
              <TextField
                size="small"
                label={t('transfers.filters.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={{ minWidth: 160 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minAge')}
                value={minAge}
                onChange={(e) => setMinAge(e.target.value)}
                sx={{ width: 100 }}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxAge')}
                value={maxAge}
                onChange={(e) => setMaxAge(e.target.value)}
                sx={{ width: 100 }}
              />
              <FormControl size="small" sx={{ width: 160 }}>
                <InputLabel id="loan-max-duration-label">{t('transfers.filters.maxDuration')}</InputLabel>
                <Select
                  labelId="loan-max-duration-label"
                  label={t('transfers.filters.maxDuration')}
                  value={maxDuration}
                  onChange={(event: SelectChangeEvent<number | ''>) =>
                    setMaxDuration(event.target.value === '' ? '' : Number(event.target.value))
                  }
                >
                  <MenuItem value="">
                    <em>{t('transfers.filters.any')}</em>
                  </MenuItem>
                  {LOAN_DURATION_OPTIONS.map((seasons) => (
                    <MenuItem key={seasons} value={seasons}>
                      {t('transfers.seasonCount', { count: seasons })}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Stack>
            {filteredMarketListings.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <LoanListingsTable
                listings={filteredMarketListings}
                sortField={sortField}
                sortDir={sortDir}
                onSort={handleSort}
                renderAction={(listing) => (
                  <Button
                    size="small"
                    variant="contained"
                    onClick={() => acceptLoan.mutate(listing.listingId)}
                    disabled={acceptLoan.isPending}
                  >
                    {t('transfers.accept')}
                  </Button>
                )}
              />
            )}
          </>
        )}
      </Paper>
    </Box>
  )
}

interface LoanListingsTableProps {
  listings: LoanListing[]
  renderAction: (listing: LoanListing) => React.ReactNode
  sortField?: LoanSortField
  sortDir?: SortDirection
  onSort?: (field: LoanSortField) => void
}

function LoanListingsTable({ listings, renderAction, sortField, sortDir, onSort }: LoanListingsTableProps) {
  const { t } = useTranslation()
  const sortable = !!onSort
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell>{t('transfers.table.team')}</TableCell>
            {sortable ? (
              <TableCell align="right" sortDirection={sortField === 'age' ? sortDir : false}>
                <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => onSort('age')}>
                  {t('transfers.table.age')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right">{t('transfers.table.age')}</TableCell>
            )}
            {sortable ? (
              <TableCell sortDirection={sortField === 'listed' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'listed'}
                  direction={sortField === 'listed' ? sortDir : 'asc'}
                  onClick={() => onSort('listed')}
                >
                  {t('transfers.table.listed')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell>{t('transfers.table.listed')}</TableCell>
            )}
            {sortable ? (
              <TableCell align="right" sortDirection={sortField === 'duration' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'duration'}
                  direction={sortField === 'duration' ? sortDir : 'asc'}
                  onClick={() => onSort('duration')}
                >
                  {t('transfers.table.duration')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right">{t('transfers.table.duration')}</TableCell>
            )}
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
              <TableCell align="right">{listing.player.age}</TableCell>
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
