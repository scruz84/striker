import { useState, type SyntheticEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
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
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
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
  useTeamTrainer,
  useTransferListings,
  useTransferMarketStats,
  useTransferWindowStatus,
  useUnlistPlayer,
  useUnlistPlayerForLoan,
  useUserTeam,
  useWithdrawOffer,
} from '../../api/hooks'
import { COUNTRY_NAME_KEYS, POSITION_DEFENSE, POSITION_GOALKEEPER, POSITION_MIDFIELD, POSITION_STRIKER, POSITION_SHORT_LABEL_KEYS } from '../../api/types'
import type { FreeAgentListing, LoanListing, Player, Trainer, TransferListing, TransferOffer } from '../../api/types'
import { isPast } from '../../api/useLiveMatch'
import { useConfirm } from '../../components/confirmDialogContext'
import { FreeAgentSigningDialog } from '../../components/FreeAgentSigningDialog'
import { Paginated } from '../../components/Paginated'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { countryFlag } from '../../utils/countryFlag'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'

type SortDirection = 'asc' | 'desc'
type TransferSortField = 'age' | 'price' | 'listed' | 'salary' | 'skill'
type LoanSortField = 'age' | 'duration' | 'listed' | 'skill'
type FreeAgentSortField = 'age' | 'wage' | 'skill'
type TrainerSortField = 'skill' | 'wage'

// The four core positions bestCorePosition ever returns - offered, in this order, by the
// market's "search by suggested position" filter on every browsable listing tab.
const CORE_POSITIONS = [POSITION_GOALKEEPER, POSITION_DEFENSE, POSITION_MIDFIELD, POSITION_STRIKER]

// Shared pill styling for every filter TextField/Select across the market's three search bars
// (transfers/loans/free agents) - a rounded, filled field with no visible border until focused,
// replacing the boxy default outlined look. Targets .MuiOutlinedInput-root as a plain CSS
// descendant selector, so it applies equally whether it's spread on the TextField/Select itself
// or on the FormControl wrapping one (see PositionFilterSelect and the loan duration filter).
const pillFilterSx = {
  width: '100%',
  '& .MuiOutlinedInput-root': {
    borderRadius: 999,
    backgroundColor: 'action.hover',
    transition: 'background-color 150ms ease',
    '& fieldset': { border: 'none' },
    '&:hover': { backgroundColor: 'action.selected' },
    '&.Mui-focused': {
      backgroundColor: 'background.paper',
      '& fieldset': { border: '2px solid', borderColor: 'primary.main' },
    },
  },
} as const

// Lays every filter field for a search bar out in an actual grid - fixed-width columns that wrap
// as a whole row rather than each field wrapping wherever it happens to run out of horizontal
// space (a plain flex-wrap row left ragged, misaligned rows once enough fields were on it - see
// issue feedback on the transfers/loans/free-agents search bars). Each field fills its own cell
// (see pillFilterSx's width: 100%) instead of carrying its own fixed pixel width.
const filterGridSx = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))',
  gap: 1.5,
  alignItems: 'center',
  p: 2,
} as const

// A player's name plus two quick-reference badges - their suggested position (see
// bestCorePosition) and average skill rating - so browsing a listings table doesn't require
// opening each player's profile just to see roughly what they're worth. formerClub renders as a
// caption underneath, for a free agent who was recently released (see FreeAgentListing). The name
// stays on a single line and truncates instead of wrapping, since these compound names (often two
// first/last names) would otherwise push every row on mobile to three lines just for the name.
function PlayerCell({ player, formerClub }: { player: Player; formerClub?: string }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const rating = overallRating(player)
  const flag = countryFlag(player.birthCountry)
  return (
    <Stack spacing={0.125} sx={{ minWidth: 0 }}>
      <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
        {flag && (
          <Tooltip title={t(COUNTRY_NAME_KEYS[player.birthCountry!] ?? player.birthCountry!)}>
            <Typography component="span" sx={{ flexShrink: 0, fontSize: '0.9rem', lineHeight: 1 }}>
              {flag}
            </Typography>
          </Tooltip>
        )}
        <Link
          component={RouterLink}
          to={`/player/${player.id}`}
          underline="hover"
          color="inherit"
          sx={{ flex: '1 1 auto', minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {player.firstName} {player.lastName}
        </Link>
        <Chip
          size="small"
          variant="outlined"
          label={t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(player)])}
          sx={{ height: 18, fontSize: '0.65rem', flexShrink: 0 }}
        />
        <Typography variant="caption" sx={{ fontWeight: 700, flexShrink: 0, color: ratingTierColor(rating, theme.palette) }}>
          {rating}
        </Typography>
      </Stack>
      {formerClub && (
        <Typography variant="caption" color="text.secondary" noWrap>
          {formerClub}
        </Typography>
      )}
    </Stack>
  )
}

// Shared "search by suggested position" filter dropdown, reused by every browsable listings tab
// (transfers/loans/free agents) - filters client-side by bestCorePosition since the market
// listing endpoints don't take a position parameter.
function PositionFilterSelect({ value, onChange }: { value: number | ''; onChange: (value: number | '') => void }) {
  const { t } = useTranslation()
  return (
    <FormControl size="small" sx={pillFilterSx}>
      <InputLabel id="position-filter-label">{t('transfers.filters.position')}</InputLabel>
      <Select
        labelId="position-filter-label"
        label={t('transfers.filters.position')}
        value={value}
        onChange={(event: SelectChangeEvent<number | ''>) => onChange(event.target.value === '' ? '' : Number(event.target.value))}
      >
        <MenuItem value="">
          <em>{t('transfers.filters.allPositions')}</em>
        </MenuItem>
        {CORE_POSITIONS.map((position) => (
          <MenuItem key={position} value={position}>
            {t(POSITION_SHORT_LABEL_KEYS[position])}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  )
}

// Order matches the <Tab> elements below - used to deep-link a specific tab (e.g. `?tab=coaching`
// from TeamDetailsPage's coach card), rather than always landing on the first one.
const TAB_NAMES = ['transfers', 'loans', 'offers', 'freeAgents', 'coaching'] as const

// The market's activity panel (issue #72): how many players are listed, how busy the offer
// market has been in the last hour/day, the currently-listed player getting the most attention,
// and the biggest transfer fee ever paid. Renders nothing while stats are still loading rather
// than a skeleton, since it sits above the tabs and shouldn't block the rest of the page.
function MarketStatsPanel() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: stats } = useTransferMarketStats(!!loggedUser)

  if (!stats) {
    return null
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.stats.listedPlayers')}
          </Typography>
          <Typography variant="h6">{stats.listedPlayersCount}</Typography>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.stats.offersRecently')}
          </Typography>
          <Typography variant="body1" sx={{ fontWeight: 700 }}>
            {t('transfers.stats.offersCount', { hour: stats.offersLastHour, day: stats.offersLastDay })}
          </Typography>
        </Grid>
        <Grid size={{ xs: 12, sm: 3 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.stats.mostViewed')}
          </Typography>
          {stats.mostViewedPlayer ? (
            <Typography variant="body2">
              <Link component={RouterLink} to={`/player/${stats.mostViewedPlayer.player.id}`} underline="hover" color="inherit">
                {stats.mostViewedPlayer.player.firstName} {stats.mostViewedPlayer.player.lastName}
              </Link>{' '}
              <Typography component="span" variant="caption" color="text.secondary">
                {t('transfers.stats.viewCount', { count: stats.mostViewedPlayer.viewCount })}
              </Typography>
            </Typography>
          ) : (
            <Typography variant="body2" color="text.secondary">
              {t('transfers.stats.none')}
            </Typography>
          )}
        </Grid>
        <Grid size={{ xs: 12, sm: 3 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.stats.topFee')}
          </Typography>
          {stats.topTransferFee ? (
            <Typography variant="body2">
              <Link component={RouterLink} to={`/player/${stats.topTransferFee.player.id}`} underline="hover" color="inherit">
                {stats.topTransferFee.player.firstName} {stats.topTransferFee.player.lastName}
              </Link>{' '}
              <Typography component="span" sx={{ fontWeight: 700 }}>
                {formatCurrency(stats.topTransferFee.amount)}
              </Typography>
              <Typography variant="caption" color="text.secondary" component="div">
                {t('transfers.stats.topFeeTeams', {
                  buyer: stats.topTransferFee.buyerTeamName,
                  seller: stats.topTransferFee.sellerTeamName,
                })}
              </Typography>
            </Typography>
          ) : (
            <Typography variant="body2" color="text.secondary">
              {t('transfers.stats.none')}
            </Typography>
          )}
        </Grid>
      </Grid>
    </Paper>
  )
}

export function TransferMarketPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: windowStatus } = useTransferWindowStatus(!!loggedUser)
  const [searchParams] = useSearchParams()
  const initialTabIndex = TAB_NAMES.indexOf(searchParams.get('tab') as (typeof TAB_NAMES)[number])
  const [tab, setTab] = useState(initialTabIndex >= 0 ? initialTabIndex : 0)
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
      <MarketStatsPanel />
      <Tabs
        value={tab}
        onChange={handleChange}
        variant="scrollable"
        scrollButtons="auto"
        allowScrollButtonsMobile
        sx={{
          mb: 2,
          borderBottom: 1,
          borderColor: 'divider',
          '& .MuiTab-root': { minWidth: 'auto', px: { xs: 1.25, sm: 2 }, fontSize: { xs: '0.8rem', sm: '0.875rem' } },
        }}
      >
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
  const [minSkill, setMinSkill] = useState('')
  const [maxSkill, setMaxSkill] = useState('')
  const [position, setPosition] = useState<number | ''>('')
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

  const hasActiveFilters = !!(search || minAge || maxAge || minPrice || maxPrice || minSkill || maxSkill || position !== '')
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMinPrice('')
    setMaxPrice('')
    setMinSkill('')
    setMaxSkill('')
    setPosition('')
  }

  const filteredMarketListings = marketListings
    .filter((l) => {
      const term = search.trim().toLowerCase()
      if (term && !`${l.player.firstName} ${l.player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && l.player.age < Number(minAge)) return false
      if (maxAge && l.player.age > Number(maxAge)) return false
      if (minPrice && l.askingPrice < Number(minPrice)) return false
      if (maxPrice && l.askingPrice > Number(maxPrice)) return false
      if (minSkill && overallRating(l.player) < Number(minSkill)) return false
      if (maxSkill && overallRating(l.player) > Number(maxSkill)) return false
      if (position !== '' && bestCorePosition(l.player) !== position) return false
      return true
    })
    .sort((a, b) => {
      const cmp =
        sortField === 'price'
          ? a.askingPrice - b.askingPrice
          : sortField === 'age'
            ? a.player.age - b.player.age
            : sortField === 'salary'
              ? a.player.wage - b.player.wage
              : sortField === 'skill'
                ? overallRating(a.player) - overallRating(b.player)
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
            <Box sx={filterGridSx}>
              <TextField
                size="small"
                label={t('transfers.filters.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minAge')}
                value={minAge}
                onChange={(e) => setMinAge(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxAge')}
                value={maxAge}
                onChange={(e) => setMaxAge(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minPrice')}
                value={minPrice}
                onChange={(e) => setMinPrice(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxPrice')}
                value={maxPrice}
                onChange={(e) => setMaxPrice(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minSkill')}
                value={minSkill}
                onChange={(e) => setMinSkill(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxSkill')}
                value={maxSkill}
                onChange={(e) => setMaxSkill(e.target.value)}
                sx={pillFilterSx}
              />
              <PositionFilterSelect value={position} onChange={setPosition} />
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters} sx={{ justifySelf: 'start' }}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Box>
            {filteredMarketListings.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <Paginated
                rows={filteredMarketListings}
                resetKey={JSON.stringify([search, minAge, maxAge, minPrice, maxPrice, minSkill, maxSkill, position, sortField, sortDir])}
              >
                {(pageListings) => (
                  <ListingsTable
                    listings={pageListings}
                    sortField={sortField}
                    sortDir={sortDir}
                    onSort={handleSort}
                    renderAction={(listing) => <MarketListingActions listing={listing} onBuy={handleBuy} buyPending={buyListing.isPending} />}
                  />
                )}
              </Paginated>
            )}
          </>
        )}
      </Paper>
    </Box>
  )
}

// MarketListingActions lets a browser either buy a listing outright at its asking price, or
// open the same negotiation dialog as offering on an unlisted player (see TeamDetailsPage's
// PlayerRow) instead - the seller then accepts/rejects the resulting offer from the Offers tab.
// The listing itself is untouched by an offer; only accepting one removes it.
function MarketListingActions({
  listing,
  onBuy,
  buyPending,
}: {
  listing: TransferListing
  onBuy: (listing: TransferListing) => void
  buyPending: boolean
}) {
  const { t } = useTranslation()
  const [offerOpen, setOfferOpen] = useState(false)

  return (
    <Stack spacing={0.25} sx={{ alignItems: 'flex-end' }}>
      <Button size="small" variant="contained" onClick={() => onBuy(listing)} disabled={buyPending} sx={{ minWidth: 0, px: 1.25, py: 0.25 }}>
        {t('transfers.buy')}
      </Button>
      <Button size="small" onClick={() => setOfferOpen(true)} sx={{ minWidth: 0, px: 1.25, py: 0.25 }}>
        {t('team.makeOffer')}
      </Button>
      <TransferOfferDialog player={listing.player} open={offerOpen} onClose={() => setOfferOpen(false)} />
    </Stack>
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
      <Table size="small" sx={{ tableLayout: 'fixed' }}>
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '22%' }}>{t('transfers.table.team')}</TableCell>
            {sortable ? (
              <TableCell
                align="right"
                sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 44 }}
                sortDirection={sortField === 'age' ? sortDir : false}
              >
                <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => onSort('age')}>
                  {t('transfers.table.age')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 44 }}>
                {t('transfers.table.age')}
              </TableCell>
            )}
            {sortable ? (
              <TableCell
                align="right"
                sx={{ display: { xs: 'none', md: 'table-cell' }, width: 56 }}
                sortDirection={sortField === 'skill' ? sortDir : false}
              >
                <TableSortLabel active={sortField === 'skill'} direction={sortField === 'skill' ? sortDir : 'asc'} onClick={() => onSort('skill')}>
                  {t('transfers.table.rating')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ display: { xs: 'none', md: 'table-cell' }, width: 56 }}>
                {t('transfers.table.rating')}
              </TableCell>
            )}
            {sortable ? (
              <TableCell
                sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '18%' }}
                sortDirection={sortField === 'listed' ? sortDir : false}
              >
                <TableSortLabel
                  active={sortField === 'listed'}
                  direction={sortField === 'listed' ? sortDir : 'asc'}
                  onClick={() => onSort('listed')}
                >
                  {t('transfers.table.listed')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '18%' }}>{t('transfers.table.listed')}</TableCell>
            )}
            {sortable ? (
              <TableCell
                align="right"
                sx={{ display: { xs: 'none', md: 'table-cell' }, width: 96 }}
                sortDirection={sortField === 'salary' ? sortDir : false}
              >
                <TableSortLabel
                  active={sortField === 'salary'}
                  direction={sortField === 'salary' ? sortDir : 'asc'}
                  onClick={() => onSort('salary')}
                >
                  {t('transfers.table.salary')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ display: { xs: 'none', md: 'table-cell' }, width: 96 }}>
                {t('transfers.table.salary')}
              </TableCell>
            )}
            {sortable ? (
              <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }} sortDirection={sortField === 'price' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'price'}
                  direction={sortField === 'price' ? sortDir : 'asc'}
                  onClick={() => onSort('price')}
                >
                  {t('transfers.table.price')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }}>
                {t('transfers.table.price')}
              </TableCell>
            )}
            <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }} />
          </TableRow>
        </TableHead>
        <TableBody>
          {listings.map((listing) => (
            <TableRow key={listing.listingId}>
              <TableCell>
                <PlayerCell player={listing.player} />
              </TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                <TeamNameLink teamId={listing.sellingTeamId}>{listing.sellingTeamName}</TeamNameLink>
              </TableCell>
              <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                {listing.player.age}
              </TableCell>
              <TableCell align="right" sx={{ display: { xs: 'none', md: 'table-cell' } }}>
                {overallRating(listing.player)}
              </TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{formatDateTime(listing.listedAt)}</TableCell>
              <TableCell align="right" sx={{ display: { xs: 'none', md: 'table-cell' } }}>
                {formatCurrency(listing.player.wage)}
              </TableCell>
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
  const [position, setPosition] = useState<number | ''>('')
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

  const hasActiveFilters = !!(search || minAge || maxAge || maxDuration || position !== '')
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMaxDuration('')
    setPosition('')
  }

  const filteredMarketListings = marketListings
    .filter((l) => {
      const term = search.trim().toLowerCase()
      if (term && !`${l.player.firstName} ${l.player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && l.player.age < Number(minAge)) return false
      if (maxAge && l.player.age > Number(maxAge)) return false
      if (maxDuration && l.durationSeasons > Number(maxDuration)) return false
      if (position !== '' && bestCorePosition(l.player) !== position) return false
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
            <Box sx={filterGridSx}>
              <TextField
                size="small"
                label={t('transfers.filters.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minAge')}
                value={minAge}
                onChange={(e) => setMinAge(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxAge')}
                value={maxAge}
                onChange={(e) => setMaxAge(e.target.value)}
                sx={pillFilterSx}
              />
              <FormControl size="small" sx={pillFilterSx}>
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
              <PositionFilterSelect value={position} onChange={setPosition} />
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters} sx={{ justifySelf: 'start' }}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Box>
            {filteredMarketListings.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <Paginated rows={filteredMarketListings} resetKey={JSON.stringify([search, minAge, maxAge, maxDuration, position, sortField, sortDir])}>
                {(pageListings) => (
                  <LoanListingsTable
                    listings={pageListings}
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
              </Paginated>
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
      <Table size="small" sx={{ tableLayout: 'fixed' }}>
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '22%' }}>{t('transfers.table.team')}</TableCell>
            {sortable ? (
              <TableCell
                align="right"
                sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 44 }}
                sortDirection={sortField === 'age' ? sortDir : false}
              >
                <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => onSort('age')}>
                  {t('transfers.table.age')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 44 }}>
                {t('transfers.table.age')}
              </TableCell>
            )}
            {sortable ? (
              <TableCell
                sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '18%' }}
                sortDirection={sortField === 'listed' ? sortDir : false}
              >
                <TableSortLabel
                  active={sortField === 'listed'}
                  direction={sortField === 'listed' ? sortDir : 'asc'}
                  onClick={() => onSort('listed')}
                >
                  {t('transfers.table.listed')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '18%' }}>{t('transfers.table.listed')}</TableCell>
            )}
            {sortable ? (
              <TableCell align="right" sx={{ width: { xs: 96, sm: 100 } }} sortDirection={sortField === 'duration' ? sortDir : false}>
                <TableSortLabel
                  active={sortField === 'duration'}
                  direction={sortField === 'duration' ? sortDir : 'asc'}
                  onClick={() => onSort('duration')}
                >
                  {t('transfers.table.duration')}
                </TableSortLabel>
              </TableCell>
            ) : (
              <TableCell align="right" sx={{ width: { xs: 96, sm: 100 } }}>
                {t('transfers.table.duration')}
              </TableCell>
            )}
            <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }} />
          </TableRow>
        </TableHead>
        <TableBody>
          {listings.map((listing) => (
            <TableRow key={listing.listingId}>
              <TableCell>
                <PlayerCell player={listing.player} />
              </TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                <TeamNameLink teamId={listing.lendingTeamId}>{listing.lendingTeamName}</TeamNameLink>
              </TableCell>
              <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                {listing.player.age}
              </TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{formatDateTime(listing.listedAt)}</TableCell>
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
            renderAction={(offer) => {
              const acceptReady = isPast(offer.acceptableAt)
              return (
                <Stack spacing={0.25} sx={{ alignItems: 'flex-end' }}>
                  <Stack direction="row" spacing={0.5} sx={{ justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                    <Button
                      size="small"
                      variant="contained"
                      onClick={() => handleAccept(offer)}
                      disabled={acceptOffer.isPending || !acceptReady}
                      sx={{ minWidth: 0, px: 1 }}
                    >
                      {t('transfers.accept')}
                    </Button>
                    <Button size="small" color="secondary" onClick={() => rejectOffer.mutate(offer.offerId)} disabled={rejectOffer.isPending} sx={{ minWidth: 0, px: 1 }}>
                      {t('transfers.reject')}
                    </Button>
                  </Stack>
                  {!acceptReady && (
                    <Typography variant="caption" color="text.secondary" sx={{ textAlign: 'right' }}>
                      {t('transfers.offers.acceptableAt', { date: formatDateTime(offer.acceptableAt) })}
                    </Typography>
                  )}
                </Stack>
              )
            }}
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
      <Table size="small" sx={{ tableLayout: 'fixed' }}>
        <TableHead>
          <TableRow>
            <TableCell>{t('transfers.table.player')}</TableCell>
            <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' }, width: '18%' }}>{t('transfers.table.made')}</TableCell>
            <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }}>
              {t('transfers.table.amount')}
            </TableCell>
            <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 150 }}>
              {t('transfers.table.terms')}
            </TableCell>
            <TableCell align="right" sx={{ width: { xs: 92, sm: 140 } }} />
          </TableRow>
        </TableHead>
        <TableBody>
          {offers.map((offer) => (
            <TableRow key={offer.offerId}>
              <TableCell>
                <Stack spacing={0.125} sx={{ minWidth: 0 }}>
                  <Link
                    component={RouterLink}
                    to={`/player/${offer.player.id}`}
                    underline="hover"
                    color="inherit"
                    sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                  >
                    {offer.player.firstName} {offer.player.lastName}
                  </Link>
                  <Typography variant="caption" color="text.secondary" noWrap>
                    {teamColumn === 'from' ? (
                      <TeamNameLink teamId={offer.fromTeamId}>{offer.fromTeamName}</TeamNameLink>
                    ) : (
                      <TeamNameLink teamId={offer.toTeamId}>{offer.toTeamName}</TeamNameLink>
                    )}
                  </Typography>
                </Stack>
              </TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{formatDateTime(offer.createdAt)}</TableCell>
              <TableCell align="right">{formatCurrency(offer.amount)}</TableCell>
              <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                {formatCurrency(offer.offeredWage)} · {t('team.contractDialog.yearsOption', { count: offer.offeredYears })}
              </TableCell>
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
  const { data: loggedUser } = useMe()
  const { data: freeAgents, isLoading } = useFreeAgents(!!loggedUser)
  const [signingPlayer, setSigningPlayer] = useState<FreeAgentListing | null>(null)

  const [search, setSearch] = useState('')
  const [minAge, setMinAge] = useState('')
  const [maxAge, setMaxAge] = useState('')
  const [position, setPosition] = useState<number | ''>('')
  const [sortField, setSortField] = useState<FreeAgentSortField>('wage')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  if (isLoading || !freeAgents) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const hasActiveFilters = !!(search || minAge || maxAge || position !== '')
  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setPosition('')
  }

  const filteredFreeAgents = freeAgents
    .filter((p) => {
      const term = search.trim().toLowerCase()
      if (term && !`${p.firstName} ${p.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && p.age < Number(minAge)) return false
      if (maxAge && p.age > Number(maxAge)) return false
      if (position !== '' && bestCorePosition(p) !== position) return false
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
            <Box sx={filterGridSx}>
              <TextField
                size="small"
                label={t('transfers.filters.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.minAge')}
                value={minAge}
                onChange={(e) => setMinAge(e.target.value)}
                sx={pillFilterSx}
              />
              <TextField
                size="small"
                type="number"
                label={t('transfers.filters.maxAge')}
                value={maxAge}
                onChange={(e) => setMaxAge(e.target.value)}
                sx={pillFilterSx}
              />
              <PositionFilterSelect value={position} onChange={setPosition} />
              {hasActiveFilters && (
                <Button size="small" onClick={clearFilters} sx={{ justifySelf: 'start' }}>
                  {t('transfers.filters.clear')}
                </Button>
              )}
            </Box>
            {filteredFreeAgents.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('transfers.filters.noResults')}
              </Typography>
            ) : (
              <Paginated rows={filteredFreeAgents} resetKey={JSON.stringify([search, minAge, maxAge, position, sortField, sortDir])}>
                {(pageFreeAgents) => (
                  <TableContainer>
                    <Table size="small" sx={{ tableLayout: 'fixed' }}>
                      <TableHead>
                        <TableRow>
                          <TableCell>{t('transfers.table.player')}</TableCell>
                          <TableCell
                            align="right"
                            sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 56 }}
                            sortDirection={sortField === 'age' ? sortDir : false}
                          >
                            <TableSortLabel active={sortField === 'age'} direction={sortField === 'age' ? sortDir : 'asc'} onClick={() => handleSort('age')}>
                              {t('transfers.table.age')}
                            </TableSortLabel>
                          </TableCell>
                          <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }} sortDirection={sortField === 'wage' ? sortDir : false}>
                            <TableSortLabel active={sortField === 'wage'} direction={sortField === 'wage' ? sortDir : 'asc'} onClick={() => handleSort('wage')}>
                              {t('transfers.freeAgents.wage')}
                            </TableSortLabel>
                          </TableCell>
                          <TableCell align="right" sx={{ width: { xs: 76, sm: 92 } }} />
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {pageFreeAgents.map((player) => (
                          <TableRow key={player.id}>
                            <TableCell>
                              <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
                                <PlayerCell
                                  player={player}
                                  formerClub={player.lastTeamName ? t('transfers.freeAgents.formerClub', { team: player.lastTeamName }) : undefined}
                                />
                                {player.freeAgentSeasons === 0 && !!player.lastTeamId && (
                                  <Chip size="small" label={t('transfers.freeAgents.premiumChip')} />
                                )}
                              </Stack>
                            </TableCell>
                            <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                              {player.age}
                            </TableCell>
                            <TableCell align="right">{formatCurrency(player.wage)}</TableCell>
                            <TableCell align="right">
                              <Button size="small" variant="contained" onClick={() => setSigningPlayer(player)} sx={{ minWidth: 0, px: 1 }}>
                                {t('transfers.freeAgents.sign')}
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Paginated>
            )}
          </>
        )}
      </Paper>
      {signingPlayer && (
        <FreeAgentSigningDialog player={signingPlayer} open={!!signingPlayer} onClose={() => setSigningPlayer(null)} />
      )}
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
          <Paginated rows={sorted} resetKey={JSON.stringify([sortField, sortDir])}>
            {(pageCandidates) => (
              <TableContainer>
                <Table size="small" sx={{ tableLayout: 'fixed' }}>
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('transfers.coaching.table.name')}</TableCell>
                      <TableCell
                        align="right"
                        sx={{ display: { xs: 'none', sm: 'table-cell' }, width: 72 }}
                        sortDirection={sortField === 'skill' ? sortDir : false}
                      >
                        <TableSortLabel active={sortField === 'skill'} direction={sortField === 'skill' ? sortDir : 'asc'} onClick={() => handleSort('skill')}>
                          {t('transfers.coaching.skill')}
                        </TableSortLabel>
                      </TableCell>
                      <TableCell align="right" sx={{ width: { xs: 84, sm: 96 } }} sortDirection={sortField === 'wage' ? sortDir : false}>
                        <TableSortLabel active={sortField === 'wage'} direction={sortField === 'wage' ? sortDir : 'asc'} onClick={() => handleSort('wage')}>
                          {t('transfers.coaching.wage')}
                        </TableSortLabel>
                      </TableCell>
                      <TableCell align="right" sx={{ width: { xs: 112, sm: 100 } }} />
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {pageCandidates.map((candidate) => (
                      <TableRow key={candidate.id}>
                        <TableCell sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {candidate.firstName} {candidate.lastName}
                        </TableCell>
                        <TableCell align="right" sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                          {candidate.skill}
                        </TableCell>
                        <TableCell align="right">{formatCurrency(candidate.wage)}</TableCell>
                        <TableCell align="right">
                          <Button size="small" variant="contained" onClick={() => handleHire(candidate)} disabled={hireTrainer.isPending} sx={{ minWidth: 0, px: 1 }}>
                            {t('transfers.coaching.hire')}
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Paginated>
        )}
      </Paper>
    </Box>
  )
}
