import { useState, type SyntheticEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import Link from '@mui/material/Link'
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
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcceptLoan,
  useAcceptOffer,
  useAvailableTrainers,
  useBuyListing,
  useFireTrainer,
  useFreeAgents,
  useHireTrainer,
  useListPlayer,
  useListPlayerForLoan,
  useLoanListings,
  useMe,
  useOffersReceived,
  useOffersSent,
  useRejectOffer,
  useSignFreeAgent,
  useTeamTrainer,
  useTransferListings,
  useTransferWindowStatus,
  useUnlistPlayer,
  useUnlistPlayerForLoan,
  useUserTeam,
  useWithdrawOffer,
} from '../../api/hooks'
import type { LoanListing, Player, Trainer, TransferListing, TransferOffer } from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

type SortDirection = 'asc' | 'desc'
type TransferSortField = 'age' | 'price' | 'listed'
type LoanSortField = 'age' | 'duration' | 'listed'
type FreeAgentSortField = 'age' | 'wage'
type TrainerSortField = 'skill' | 'wage'

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
        <Tab label={t('transfers.tabs.freeAgents')} />
        <Tab label={t('transfers.tabs.coaching')} />
      </Tabs>
      {tab === 0 && <TransfersTab />}
      {tab === 1 && <LoansTab />}
      {tab === 2 && <OffersTab />}
      {tab === 3 && <FreeAgentsTab />}
      {tab === 4 && <CoachingTab />}
    </Box>
  )
}

function TransfersTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
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

  const handleBuy = async (listing: TransferListing) => {
    const ok = await confirm({
      message: t('transfers.confirmBuy', {
        name: `${listing.player.firstName} ${listing.player.lastName}`,
        price: formatCurrency(listing.askingPrice),
      }),
      confirmLabel: t('transfers.buy'),
      destructive: true,
    })
    if (ok) {
      buyListing.mutate(listing.listingId)
    }
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
                    onClick={() => handleBuy(listing)}
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
                <Link component={RouterLink} to={`/player/${listing.player.id}`} underline="hover" color="inherit">
                  {listing.player.firstName} {listing.player.lastName}
                </Link>
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
  const confirm = useConfirm()
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

  const handleAcceptLoan = async (listing: LoanListing) => {
    const ok = await confirm({
      message: t('transfers.confirmAcceptLoan', {
        name: `${listing.player.firstName} ${listing.player.lastName}`,
        seasons: listing.durationSeasons,
      }),
      confirmLabel: t('transfers.accept'),
    })
    if (ok) {
      acceptLoan.mutate(listing.listingId)
    }
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
                    onClick={() => handleAcceptLoan(listing)}
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
                <Link component={RouterLink} to={`/player/${listing.player.id}`} underline="hover" color="inherit">
                  {listing.player.firstName} {listing.player.lastName}
                </Link>
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
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: received, isLoading: receivedLoading } = useOffersReceived(!!loggedUser)
  const { data: sent, isLoading: sentLoading } = useOffersSent(!!loggedUser)

  const acceptOffer = useAcceptOffer()
  const rejectOffer = useRejectOffer()
  const withdrawOffer = useWithdrawOffer()

  if (receivedLoading || sentLoading || !received || !sent) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const handleAccept = async (offer: TransferOffer) => {
    const ok = await confirm({
      message: t('transfers.confirmAcceptOffer', {
        name: `${offer.player.firstName} ${offer.player.lastName}`,
        team: offer.fromTeamName,
        amount: formatCurrency(offer.amount),
      }),
      confirmLabel: t('transfers.accept'),
      destructive: true,
    })
    if (ok) {
      acceptOffer.mutate(offer.offerId)
    }
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
                <Button size="small" variant="contained" onClick={() => handleAccept(offer)} disabled={acceptOffer.isPending}>
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
                <Link component={RouterLink} to={`/player/${offer.player.id}`} underline="hover" color="inherit">
                  {offer.player.firstName} {offer.player.lastName}
                </Link>
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

function FreeAgentsTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: freeAgents, isLoading } = useFreeAgents(!!loggedUser)
  const signFreeAgent = useSignFreeAgent()

  const handleSign = async (player: Player) => {
    const ok = await confirm({
      message: t('transfers.freeAgents.confirmSign', {
        name: `${player.firstName} ${player.lastName}`,
        wage: formatCurrency(player.wage),
      }),
      confirmLabel: t('transfers.freeAgents.sign'),
    })
    if (ok) {
      signFreeAgent.mutate(player.id)
    }
  }

  const [search, setSearch] = useState('')
  const [minAge, setMinAge] = useState('')
  const [maxAge, setMaxAge] = useState('')
  const [sortField, setSortField] = useState<FreeAgentSortField>('wage')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  if (isLoading || !freeAgents) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const hasActiveFilters = !!(search || minAge || maxAge)
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
  }

  const filteredFreeAgents = freeAgents
    .filter((p) => {
      const term = search.trim().toLowerCase()
      if (term && !`${p.firstName} ${p.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && p.age < Number(minAge)) return false
      if (maxAge && p.age > Number(maxAge)) return false
      return true
    })
    .sort((a, b) => {
      const cmp = sortField === 'age' ? a.age - b.age : a.wage - b.wage
      return sortDir === 'asc' ? cmp : -cmp
    })

  const handleSort = (field: FreeAgentSortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('desc')
    }
  }

  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('transfers.freeAgents.intro')}
      </Typography>
      <Paper>
        {freeAgents.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.freeAgents.none')}
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
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Stack>
            {filteredFreeAgents.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('transfers.table.player')}</TableCell>
                      <TableCell align="right" sortDirection={sortField === 'age' ? sortDir : false}>
                        <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => handleSort('age')}>
                          {t('transfers.table.age')}
                        </TableSortLabel>
                      </TableCell>
                      <TableCell align="right" sortDirection={sortField === 'wage' ? sortDir : false}>
                        <TableSortLabel active={sortField === 'wage'} direction={sortField === 'wage' ? sortDir : 'asc'} onClick={() => handleSort('wage')}>
                          {t('transfers.freeAgents.wage')}
                        </TableSortLabel>
                      </TableCell>
                      <TableCell align="right" />
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {filteredFreeAgents.map((player) => (
                      <TableRow key={player.id}>
                        <TableCell>
                          <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" color="inherit">
                            {player.firstName} {player.lastName}
                          </Link>
                        </TableCell>
                        <TableCell align="right">{player.age}</TableCell>
                        <TableCell align="right">{formatCurrency(player.wage)}</TableCell>
                        <TableCell align="right">
                          <Button
                            size="small"
                            variant="contained"
                            onClick={() => handleSign(player)}
                            disabled={signFreeAgent.isPending}
                          >
                            {t('transfers.freeAgents.sign')}
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </>
        )}
      </Paper>
    </Box>
  )
}

// A team can only employ one trainer at a time (see common.ErrTeamAlreadyHasTrainer on the
// backend), so this tab shows either the current head coach - with a fire action - or, once
// there isn't one, the open market to hire from. Never both at once.
function CoachingTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: trainer, isLoading: trainerLoading } = useTeamTrainer(!!loggedUser)
  const { data: available, isLoading: availableLoading } = useAvailableTrainers(!!loggedUser && !trainerLoading && !trainer)
  const hireTrainer = useHireTrainer()
  const fireTrainer = useFireTrainer()

  const [sortField, setSortField] = useState<TrainerSortField>('skill')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  const handleHire = async (candidate: Trainer) => {
    const ok = await confirm({
      message: t('transfers.coaching.confirmHire', {
        name: `${candidate.firstName} ${candidate.lastName}`,
        wage: formatCurrency(candidate.wage),
      }),
      confirmLabel: t('transfers.coaching.hire'),
    })
    if (ok) {
      hireTrainer.mutate(candidate.id)
    }
  }

  const handleFire = async () => {
    if (!trainer) return
    const ok = await confirm({
      message: t('transfers.coaching.confirmFire', { name: `${trainer.firstName} ${trainer.lastName}` }),
      confirmLabel: t('transfers.coaching.fire'),
      destructive: true,
    })
    if (ok) {
      fireTrainer.mutate(trainer.id)
    }
  }

  if (trainerLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (trainer) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t('transfers.coaching.intro')}
        </Typography>
        <Paper sx={{ p: 3, maxWidth: 420 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
            {trainer.firstName} {trainer.lastName}
          </Typography>
          <Stack direction="row" spacing={4} sx={{ mt: 1.5, mb: 2.5 }}>
            <Box>
              <Typography variant="caption" color="text.secondary" component="div">
                {t('transfers.coaching.skill')}
              </Typography>
              <Typography variant="h6">{trainer.skill}</Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary" component="div">
                {t('transfers.coaching.wage')}
              </Typography>
              <Typography variant="h6">{formatCurrency(trainer.wage)}</Typography>
            </Box>
          </Stack>
          <Button color="secondary" variant="outlined" onClick={handleFire} disabled={fireTrainer.isPending}>
            {t('transfers.coaching.fire')}
          </Button>
        </Paper>
      </Box>
    )
  }

  if (availableLoading || !available) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const sorted = [...available].sort((a, b) => {
    const cmp = sortField === 'skill' ? a.skill - b.skill : a.wage - b.wage
    return sortDir === 'asc' ? cmp : -cmp
  })

  const handleSort = (field: TrainerSortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('desc')
    }
  }

  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('transfers.coaching.intro')}
      </Typography>
      <Paper>
        {available.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {t('transfers.coaching.none')}
          </Typography>
        ) : (
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{t('transfers.coaching.table.name')}</TableCell>
                  <TableCell align="right" sortDirection={sortField === 'skill' ? sortDir : false}>
                    <TableSortLabel active={sortField === 'skill'} direction={sortField === 'skill' ? sortDir : 'asc'} onClick={() => handleSort('skill')}>
                      {t('transfers.coaching.skill')}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell align="right" sortDirection={sortField === 'wage' ? sortDir : false}>
                    <TableSortLabel active={sortField === 'wage'} direction={sortField === 'wage' ? sortDir : 'asc'} onClick={() => handleSort('wage')}>
                      {t('transfers.coaching.wage')}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell align="right" />
                </TableRow>
              </TableHead>
              <TableBody>
                {sorted.map((candidate) => (
                  <TableRow key={candidate.id}>
                    <TableCell>
                      {candidate.firstName} {candidate.lastName}
                    </TableCell>
                    <TableCell align="right">{candidate.skill}</TableCell>
                    <TableCell align="right">{formatCurrency(candidate.wage)}</TableCell>
                    <TableCell align="right">
                      <Button size="small" variant="contained" onClick={() => handleHire(candidate)} disabled={hireTrainer.isPending}>
                        {t('transfers.coaching.hire')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>
    </Box>
  )
}
