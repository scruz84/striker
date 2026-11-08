import { useState, type ReactNode, type SyntheticEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import Link from '@mui/material/Link'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
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
  useListPlayerForLoan,
  useLoanListings,
  useMe,
  useOffersReceived,
  useOffersSent,
  usePreContractsReceived,
  usePreContractsSent,
  useWithdrawPreContractOffer,
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
import { POSITION_SHORT_LABEL_KEYS, type FreeAgentListing, type LoanListing, type Player, type PreContractOffer, type Trainer, type TransferListing, type TransferOffer } from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { EmptyState } from '../../components/EmptyState'
import { FreeAgentSigningDialog } from '../../components/FreeAgentSigningDialog'
import { NumberField } from '../../components/NumberField'
import { PageHeader } from '../../components/PageHeader'
import { Paginated } from '../../components/Paginated'
import { ResponsiveDialog } from '../../components/ResponsiveDialog'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { WatchlistTab } from './WatchlistTab'
import { LastSeasonTab } from './LastSeasonTab'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatShortDate } from '../../utils/formatDate'
import { SKILL_FIELDS, SKILL_LABEL_KEYS, bestCorePosition, overallRating } from '../../utils/playerRating'
import {
  ActiveFilterChips,
  FilterBar,
  FilterSheet,
  ListingCard,
  MetaDot,
  MetaItem,
  PositionFilter,
  RangeFields,
  SkillFilter,
  SortControl,
  usePersistedFilter,
  useFilterSheet,
  type ActiveFilter,
  type SortDirection,
} from './marketUi'
import { CountryFlag } from '../../components/CountryFlag'
import { ListPlayerDialog } from './ListPlayerDialog'
import { ReceivedOfferActions, SentOfferCard } from './OfferCards'
import { PreContractsSection } from './PreContractCards'

type TransferSortField = 'age' | 'price' | 'listed' | 'salary' | 'skill'
type LoanSortField = 'age' | 'duration' | 'listed' | 'skill'
type FreeAgentSortField = 'age' | 'wage' | 'skill'
type TrainerSortField = 'skill' | 'wage'

const LOAN_DURATION_OPTIONS = [1, 2, 3]

// Order matches the <Tab> elements below - used to deep-link a specific tab (e.g. `?tab=coaching`
// from TeamDetailsPage's coach card), rather than always landing on the first one.
const TAB_NAMES = ['transfers', 'loans', 'offers', 'freeAgents', 'lastSeason', 'coaching', 'watchlist'] as const

// Parses a persisted filter string ("" when unset) into a number the filters can compare with.
const num = (value: string) => Number(value)

// The market's activity panel (issue #72): how many players are listed, how busy the offer
// market has been in the last hour/day, the currently-listed player getting the most attention,
// and the biggest transfer fee ever paid. A single horizontally scrollable row of compact tiles
// (it used to be a 260px block repeated on every tab); only the Transfers tab shows it. Renders
// nothing while stats are still loading rather than a skeleton, since it sits above the results
// and shouldn't block the rest of the page.
function MarketStatsStrip() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: stats } = useTransferMarketStats(!!loggedUser)

  if (!stats) {
    return null
  }

  const tiles: { key: string; label: string; value: ReactNode; caption?: ReactNode }[] = [
    { key: 'listed', label: t('transfers.stats.listedPlayers'), value: stats.listedPlayersCount },
    {
      key: 'offers',
      label: t('transfers.stats.offersRecently'),
      value: t('transfers.stats.offersShort', { hour: stats.offersLastHour, day: stats.offersLastDay }),
    },
    {
      key: 'viewed',
      label: t('transfers.stats.mostViewed'),
      value: stats.mostViewedPlayer ? (
        <Link component={RouterLink} to={`/player/${stats.mostViewedPlayer.player.id}`} underline="hover" color="inherit">
          {stats.mostViewedPlayer.player.lastName}
        </Link>
      ) : (
        t('transfers.stats.none')
      ),
      caption: stats.mostViewedPlayer ? t('transfers.stats.viewCount', { count: stats.mostViewedPlayer.viewCount }) : undefined,
    },
    {
      key: 'fee',
      label: t('transfers.stats.topFee'),
      value: stats.topTransferFee ? formatCurrency(stats.topTransferFee.amount, { compact: true }) : t('transfers.stats.none'),
      caption: stats.topTransferFee ? (
        <Link component={RouterLink} to={`/player/${stats.topTransferFee.player.id}`} underline="hover" color="inherit">
          {stats.topTransferFee.player.lastName}
        </Link>
      ) : undefined,
    },
  ]

  return (
    <Stack direction="row" spacing={1} sx={{ overflowX: 'auto', pb: 1, mb: 1, scrollSnapType: 'x proximity' }}>
      {tiles.map((tile) => (
        <Paper key={tile.key} variant="outlined" sx={{ px: 1.5, py: 1, minWidth: 128, flexShrink: 0, scrollSnapAlign: 'start' }}>
          <Typography variant="chipLabel" color="text.secondary" component="div" noWrap>
            {tile.label}
          </Typography>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.25 }} noWrap>
            {tile.value}
          </Typography>
          {tile.caption && (
            <Typography variant="caption" color="text.secondary" component="div" noWrap>
              {tile.caption}
            </Typography>
          )}
        </Paper>
      ))}
    </Stack>
  )
}

export function TransferMarketPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: windowStatus } = useTransferWindowStatus(!!loggedUser)
  const [tabName, setTabName] = usePersistedFilter<(typeof TAB_NAMES)[number]>('tab', TAB_NAMES[0])
  const tab = Math.max(TAB_NAMES.indexOf(tabName), 0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTabName(TAB_NAMES[newValue])

  return (
    <Box>
      <PageHeader title={t('transfers.title')} />
      {windowStatus && !windowStatus.open && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          {t('transfers.windowClosed')}
        </Alert>
      )}
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
          '& .MuiTab-root': { minWidth: 'auto', px: { xs: 1.25, sm: 2 } },
        }}
      >
        <Tab label={t('transfers.tabsShort.transfers')} />
        <Tab label={t('transfers.tabsShort.loans')} />
        <Tab label={t('transfers.tabsShort.offers')} />
        <Tab label={t('transfers.tabsShort.freeAgents')} />
        <Tab label={t('transfers.tabsShort.lastSeason')} />
        <Tab label={t('transfers.tabsShort.coaching')} />
        <Tab label={t('transfers.tabsShort.watchlist')} />
      </Tabs>
      {tab === 0 && <TransfersTab />}
      {tab === 1 && <LoansTab />}
      {tab === 2 && <OffersTab />}
      {tab === 3 && <FreeAgentsTab />}
      {tab === 4 && <LastSeasonTab />}
      {tab === 5 && <CoachingTab />}
      {tab === 6 && <WatchlistTab />}
    </Box>
  )
}

function TransfersTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: listings, isLoading: listingsLoading } = useTransferListings(!!loggedUser)

  const unlistPlayer = useUnlistPlayer()
  const buyListing = useBuyListing()
  const [listOpen, setListOpen] = useState(false)
  const sheet = useFilterSheet()

  const [search, setSearch] = usePersistedFilter<string>('tSearch', '')
  const [minAge, setMinAge] = usePersistedFilter<string>('tMinAge', '')
  const [maxAge, setMaxAge] = usePersistedFilter<string>('tMaxAge', '')
  const [minPrice, setMinPrice] = usePersistedFilter<string>('tMinPrice', '')
  const [maxPrice, setMaxPrice] = usePersistedFilter<string>('tMaxPrice', '')
  const [minSkill, setMinSkill] = usePersistedFilter<string>('tMinSkill', '')
  const [maxSkill, setMaxSkill] = usePersistedFilter<string>('tMaxSkill', '')
  const [skillFieldStr, setSkillFieldStr] = usePersistedFilter<string>('tSkillField', SKILL_FIELDS[0])
  const skillField = skillFieldStr as (typeof SKILL_FIELDS)[number]
  const [minSkillValue, setMinSkillValue] = usePersistedFilter<string>('tMinSkillValue', '')
  const [minCaps, setMinCaps] = usePersistedFilter<string>('tMinCaps', '')
  const [positionStr, setPositionStr] = usePersistedFilter<string>('tPosition', '')
  const position = positionStr === '' ? '' : Number(positionStr)
  const setPosition = (value: number | '') => setPositionStr(value === '' ? '' : String(value))
  const [sortField, setSortField] = usePersistedFilter<TransferSortField>('tSort', 'listed')
  const [sortDir, setSortDir] = usePersistedFilter<SortDirection>('tDir', 'desc')

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const listedPlayerIds = new Set(listings.filter((l) => l.sellingTeamId === myTeamId).map((l) => l.player.id))
  const listablePlayers = userTeam.players.filter((p) => !listedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.sellingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.sellingTeamId !== myTeamId)

  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMinPrice('')
    setMaxPrice('')
    setMinSkill('')
    setMaxSkill('')
    setMinSkillValue('')
    setMinCaps('')
    setPosition('')
  }

  const rangeChip = (key: string, min: string, max: string, labels: { both: string; from: string; to: string }, clear: () => void, format: (v: string) => string = (v) => v): ActiveFilter[] =>
    min || max
      ? [
          {
            key,
            label:
              min && max
                ? t(labels.both, { min: format(min), max: format(max) })
                : min
                  ? t(labels.from, { value: format(min) })
                  : t(labels.to, { value: format(max) }),
            onDelete: clear,
          },
        ]
      : []

  const activeFilters: ActiveFilter[] = [
    ...(search.trim() ? [{ key: 'search', label: t('transfers.filters.active.search', { value: search.trim() }), onDelete: () => setSearch('') }] : []),
    ...rangeChip('age', minAge, maxAge, { both: 'transfers.filters.active.age', from: 'transfers.filters.active.ageFrom', to: 'transfers.filters.active.ageTo' }, () => {
      setMinAge('')
      setMaxAge('')
    }),
    ...rangeChip('price', minPrice, maxPrice, { both: 'transfers.filters.active.price', from: 'transfers.filters.active.priceFrom', to: 'transfers.filters.active.priceTo' }, () => {
      setMinPrice('')
      setMaxPrice('')
    }, (v) => formatCurrency(num(v), { compact: true })),
    ...rangeChip('rating', minSkill, maxSkill, { both: 'transfers.filters.active.rating', from: 'transfers.filters.active.ratingFrom', to: 'transfers.filters.active.ratingTo' }, () => {
      setMinSkill('')
      setMaxSkill('')
    }),
    ...(minSkillValue ? [{ key: 'skill', label: t('transfers.filters.active.skill', { skill: t(SKILL_LABEL_KEYS[skillField]), value: minSkillValue }), onDelete: () => setMinSkillValue('') }] : []),
    ...(minCaps ? [{ key: 'caps', label: t('transfers.filters.active.caps', { value: minCaps }), onDelete: () => setMinCaps('') }] : []),
    ...(position !== '' ? [{ key: 'position', label: t(POSITION_SHORT_LABEL_KEYS[position]), onDelete: () => setPosition('') }] : []),
  ]

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
      if (minSkillValue && l.player[skillField] < Number(minSkillValue)) return false
      if (minCaps && l.caps < Number(minCaps)) return false
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
      <MarketStatsStrip />

      <Stack direction="row" sx={{ justifyContent: 'flex-end', mb: 1.5 }}>
        <Button variant="contained" onClick={() => setListOpen(true)} disabled={listablePlayers.length === 0}>
          {t('transfers.listPlayer.title')}
        </Button>
      </Stack>
      <ListPlayerDialog open={listOpen} onClose={() => setListOpen(false)} players={listablePlayers} />

      {myListings.length > 0 && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
            {t('transfers.yourListings')}
          </Typography>
          <Stack spacing={1}>
            {myListings.map((listing) => (
              <ListingCard
                key={listing.listingId}
                player={listing.player}
                meta={
                  <>
                    <MetaItem>{t('transfers.ageLabel', { age: listing.player.age })}</MetaItem>
                    <MetaDot />
                    <MetaItem>{t('transfers.listedOn', { date: formatShortDate(listing.listedAt) })}</MetaItem>
                  </>
                }
                priceLabel={t('transfers.table.price')}
                priceValue={formatCurrency(listing.askingPrice)}
                salary={t('transfers.salaryPerMatch', { amount: formatCurrency(listing.player.wage) })}
                actions={
                  <Button color="secondary" variant="outlined" onClick={() => unlistPlayer.mutate(listing.player.id)}>
                    {t('transfers.unlist')}
                  </Button>
                }
              />
            ))}
          </Stack>
        </Box>
      )}

      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
        {t('transfers.market')}
      </Typography>
      {marketListings.length === 0 ? (
        <EmptyState compact title={t('transfers.noMarketListings')} />
      ) : (
        <>
          <FilterBar search={search} onSearch={setSearch} activeCount={activeFilters.length} onOpenFilters={sheet.show} />
          <ActiveFilterChips filters={activeFilters} />
          <FilterSheet
            open={sheet.open}
            onClose={sheet.hide}
            resultCount={filteredMarketListings.length}
            onClear={clearFilters}
            hasActiveFilters={activeFilters.length > 0}
          >
            <SortControl<TransferSortField>
              value={sortField}
              onChange={setSortField}
              direction={sortDir}
              onToggleDirection={() => setSortDir(sortDir === 'asc' ? 'desc' : 'asc')}
              options={[
                { value: 'listed', label: t('transfers.table.listed') },
                { value: 'price', label: t('transfers.table.price') },
                { value: 'age', label: t('transfers.table.age') },
                { value: 'skill', label: t('transfers.table.rating') },
                { value: 'salary', label: t('transfers.table.salary') },
              ]}
            />
            <PositionFilter value={position} onChange={setPosition} />
            <RangeFields label={t('transfers.filters.age')} min={minAge} max={maxAge} onMin={setMinAge} onMax={setMaxAge} />
            <RangeFields label={t('transfers.filters.price')} min={minPrice} max={maxPrice} onMin={setMinPrice} onMax={setMaxPrice} currency />
            <RangeFields label={t('transfers.filters.rating')} min={minSkill} max={maxSkill} onMin={setMinSkill} onMax={setMaxSkill} />
            <SkillFilter field={skillField} onField={setSkillFieldStr} minValue={minSkillValue} onMinValue={setMinSkillValue} />
            <NumberField label={t('transfers.filters.minCapsLabel')} value={minCaps} onValueChange={setMinCaps} fullWidth />
          </FilterSheet>
          {filteredMarketListings.length === 0 ? (
            <EmptyState compact title={t('transfers.filters.noResults')} />
          ) : (
            <Paginated
              rows={filteredMarketListings}
              resetKey={JSON.stringify([search, minAge, maxAge, minPrice, maxPrice, minSkill, maxSkill, skillField, minSkillValue, minCaps, position, sortField, sortDir])}
            >
              {(pageListings) => (
                <Stack spacing={1}>
                  {pageListings.map((listing) => (
                    <ListingCard
                      key={listing.listingId}
                      player={listing.player}
                      meta={
                        <>
                          <TeamNameLink teamId={listing.sellingTeamId} variant="caption" color="text.secondary" underline="hover">
                            {listing.sellingTeamName}
                          </TeamNameLink>
                          <MetaDot />
                          <MetaItem>{t('transfers.ageLabel', { age: listing.player.age })}</MetaItem>
                          {listing.caps > 0 && (
                            <>
                              <MetaDot />
                              <MetaItem>{t('transfers.filters.capsCount', { count: listing.caps })}</MetaItem>
                            </>
                          )}
                          <MetaDot />
                          <MetaItem>{t('transfers.listedOn', { date: formatShortDate(listing.listedAt) })}</MetaItem>
                        </>
                      }
                      priceLabel={t('transfers.table.price')}
                      priceValue={formatCurrency(listing.askingPrice)}
                      salary={t('transfers.salaryPerMatch', { amount: formatCurrency(listing.player.wage) })}
                      actions={<MarketListingActions listing={listing} onBuy={handleBuy} buyPending={buyListing.isPending} />}
                    />
                  ))}
                </Stack>
              )}
            </Paginated>
          )}
        </>
      )}
    </Box>
  )
}

// MarketListingActions lets a browser either buy a listing outright at its asking price, or
// open the same negotiation dialog as offering on an unlisted player (see TeamDetailsPage's
// squad sheet) instead - the seller then accepts/rejects the resulting offer from the Offers tab.
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
    <>
      <Button variant="outlined" onClick={() => setOfferOpen(true)}>
        {t('team.makeOffer')}
      </Button>
      <Button variant="contained" onClick={() => onBuy(listing)} disabled={buyPending}>
        {t('transfers.buy')}
      </Button>
      <TransferOfferDialog player={listing.player} open={offerOpen} onClose={() => setOfferOpen(false)} askingPrice={listing.askingPrice} />
    </>
  )
}

/** Dialog for loaning out one of your players. Every term gets its own full-width, fully labelled
 * field with a line of help - the old inline row had three ~90px fields whose placeholders read
 * "Cuota …", "% de c…", "Cuota …". */
function OfferLoanDialog({ open, onClose, players }: { open: boolean; onClose: () => void; players: Player[] }) {
  const { t } = useTranslation()
  const listForLoan = useListPlayerForLoan()
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | ''>('')
  const [duration, setDuration] = useState<number>(1)
  const [loanFee, setLoanFee] = useState('')
  const [wageContributionPercent, setWageContributionPercent] = useState('')
  const [buyOptionFee, setBuyOptionFee] = useState('')

  const handleList = () => {
    if (selectedPlayerId === '') {
      return
    }
    listForLoan.mutate(
      {
        playerId: selectedPlayerId,
        durationSeasons: duration,
        loanFee: loanFee ? Number(loanFee) : 0,
        wageContributionPercent: wageContributionPercent ? Number(wageContributionPercent) : 0,
        buyOptionFee: buyOptionFee ? Number(buyOptionFee) : 0,
      },
      {
        onSuccess: () => {
          setSelectedPlayerId('')
          setLoanFee('')
          setWageContributionPercent('')
          setBuyOptionFee('')
          onClose()
        },
      },
    )
  }

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('transfers.offerLoan.title')}
      actions={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="contained" disabled={selectedPlayerId === '' || listForLoan.isPending} onClick={handleList}>
            {t('transfers.list')}
          </Button>
        </>
      }
    >
      <Stack spacing={2}>
        <Typography variant="body2" color="text.secondary">
          {t('transfers.offerLoan.subtitle')}
        </Typography>
        {listForLoan.isError && <Alert severity="error">{t('transfers.offerLoan.error')}</Alert>}
        <FormControl fullWidth>
          <InputLabel id="loan-player-label">{t('transfers.offerLoan.selectPlayer')}</InputLabel>
          <Select
            labelId="loan-player-label"
            label={t('transfers.offerLoan.selectPlayer')}
            value={selectedPlayerId}
            onChange={(event: SelectChangeEvent<number | ''>) => setSelectedPlayerId(event.target.value === '' ? '' : Number(event.target.value))}
          >
            {players.map((player) => (
              <MenuItem key={player.id} value={player.id}>
                <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
                {player.firstName} {player.lastName}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
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
        <NumberField
          label={t('transfers.offerLoan.loanFee')}
          value={loanFee}
          onValueChange={setLoanFee}
          currency
          fullWidth
          helperText={t('transfers.offerLoan.loanFeeHelp')}
        />
        <NumberField
          label={t('transfers.offerLoan.wageContributionPercent')}
          value={wageContributionPercent}
          onValueChange={setWageContributionPercent}
          suffix="%"
          max={100}
          fullWidth
          helperText={t('transfers.offerLoan.wageContributionHelp')}
        />
        <NumberField
          label={t('transfers.offerLoan.buyOptionFee')}
          value={buyOptionFee}
          onValueChange={setBuyOptionFee}
          currency
          fullWidth
          helperText={t('transfers.offerLoan.buyOptionHelp')}
        />
      </Stack>
    </ResponsiveDialog>
  )
}

// A compact read of a loan listing's optional economic terms (openspec/specs/team-finances,
// "Loan money") - "free" when none were set (the original, still-default behavior), each
// present term on its own line otherwise.
function LoanTermsSummary({ listing }: { listing: LoanListing }) {
  const { t } = useTranslation()
  if (!listing.loanFee && !listing.wageContributionPercent && !listing.buyOptionFee) {
    return (
      <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
        {t('transfers.offerLoan.free')}
      </Typography>
    )
  }
  return (
    <Stack spacing={0} sx={{ mt: 0.5 }}>
      {!!listing.loanFee && (
        <Typography variant="caption" color="text.secondary" component="div">
          {t('transfers.offerLoan.loanFeeSummary', { amount: formatCurrency(listing.loanFee) })}
        </Typography>
      )}
      {!!listing.wageContributionPercent && (
        <Typography variant="caption" color="text.secondary" component="div">
          {t('transfers.offerLoan.wageContributionSummary', { percent: listing.wageContributionPercent })}
        </Typography>
      )}
      {!!listing.buyOptionFee && (
        <Typography variant="caption" color="text.secondary" component="div">
          {t('transfers.offerLoan.buyOptionSummary', { amount: formatCurrency(listing.buyOptionFee) })}
        </Typography>
      )}
    </Stack>
  )
}

function LoanCard({ listing, action }: { listing: LoanListing; action: ReactNode }) {
  const { t } = useTranslation()
  return (
    <ListingCard
      player={listing.player}
      meta={
        <>
          <TeamNameLink teamId={listing.lendingTeamId} variant="caption" color="text.secondary" underline="hover">
            {listing.lendingTeamName}
          </TeamNameLink>
          <MetaDot />
          <MetaItem>{t('transfers.ageLabel', { age: listing.player.age })}</MetaItem>
          {listing.caps > 0 && (
            <>
              <MetaDot />
              <MetaItem>{t('transfers.filters.capsCount', { count: listing.caps })}</MetaItem>
            </>
          )}
          <MetaDot />
          <MetaItem>{t('transfers.listedOn', { date: formatShortDate(listing.listedAt) })}</MetaItem>
        </>
      }
      extra={<LoanTermsSummary listing={listing} />}
      priceLabel={t('transfers.table.duration')}
      priceValue={t('transfers.seasonCount', { count: listing.durationSeasons })}
      salary={t('transfers.salaryPerMatch', { amount: formatCurrency(listing.player.wage) })}
      actions={action}
    />
  )
}

function LoansTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: listings, isLoading: listingsLoading } = useLoanListings(!!loggedUser)

  const unlistFromLoan = useUnlistPlayerForLoan()
  const acceptLoan = useAcceptLoan()
  const [offerOpen, setOfferOpen] = useState(false)
  const sheet = useFilterSheet()

  const [search, setSearch] = usePersistedFilter<string>('lSearch', '')
  const [minAge, setMinAge] = usePersistedFilter<string>('lMinAge', '')
  const [maxAge, setMaxAge] = usePersistedFilter<string>('lMaxAge', '')
  const [maxDurationStr, setMaxDurationStr] = usePersistedFilter<string>('lMaxDuration', '')
  const maxDuration = maxDurationStr === '' ? '' : Number(maxDurationStr)
  const setMaxDuration = (value: number | '') => setMaxDurationStr(value === '' ? '' : String(value))
  const [skillFieldStr, setSkillFieldStr] = usePersistedFilter<string>('lSkillField', SKILL_FIELDS[0])
  const skillField = skillFieldStr as (typeof SKILL_FIELDS)[number]
  const [minSkillValue, setMinSkillValue] = usePersistedFilter<string>('lMinSkillValue', '')
  const [minCaps, setMinCaps] = usePersistedFilter<string>('lMinCaps', '')
  const [positionStr, setPositionStr] = usePersistedFilter<string>('lPosition', '')
  const position = positionStr === '' ? '' : Number(positionStr)
  const setPosition = (value: number | '') => setPositionStr(value === '' ? '' : String(value))
  const [sortField, setSortField] = usePersistedFilter<LoanSortField>('lSort', 'listed')
  const [sortDir, setSortDir] = usePersistedFilter<SortDirection>('lDir', 'desc')

  if (teamLoading || listingsLoading || !userTeam || !listings) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const myTeamId = userTeam.team.id
  const loanListedPlayerIds = new Set(listings.filter((l) => l.lendingTeamId === myTeamId).map((l) => l.player.id))
  const loanablePlayers: Player[] = userTeam.players.filter((p) => !loanListedPlayerIds.has(p.id))
  const myListings = listings.filter((l) => l.lendingTeamId === myTeamId)
  const marketListings = listings.filter((l) => l.lendingTeamId !== myTeamId)

  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMaxDuration('')
    setMinSkillValue('')
    setMinCaps('')
    setPosition('')
  }

  const activeFilters: ActiveFilter[] = [
    ...(search.trim() ? [{ key: 'search', label: t('transfers.filters.active.search', { value: search.trim() }), onDelete: () => setSearch('') }] : []),
    ...(minAge || maxAge
      ? [
          {
            key: 'age',
            label:
              minAge && maxAge
                ? t('transfers.filters.active.age', { min: minAge, max: maxAge })
                : minAge
                  ? t('transfers.filters.active.ageFrom', { value: minAge })
                  : t('transfers.filters.active.ageTo', { value: maxAge }),
            onDelete: () => {
              setMinAge('')
              setMaxAge('')
            },
          },
        ]
      : []),
    ...(maxDuration ? [{ key: 'duration', label: t('transfers.filters.active.duration', { count: maxDuration }), onDelete: () => setMaxDuration('') }] : []),
    ...(minSkillValue ? [{ key: 'skill', label: t('transfers.filters.active.skill', { skill: t(SKILL_LABEL_KEYS[skillField]), value: minSkillValue }), onDelete: () => setMinSkillValue('') }] : []),
    ...(minCaps ? [{ key: 'caps', label: t('transfers.filters.active.caps', { value: minCaps }), onDelete: () => setMinCaps('') }] : []),
    ...(position !== '' ? [{ key: 'position', label: t(POSITION_SHORT_LABEL_KEYS[position]), onDelete: () => setPosition('') }] : []),
  ]

  const filteredMarketListings = marketListings
    .filter((l) => {
      const term = search.trim().toLowerCase()
      if (term && !`${l.player.firstName} ${l.player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && l.player.age < Number(minAge)) return false
      if (maxAge && l.player.age > Number(maxAge)) return false
      if (maxDuration && l.durationSeasons > Number(maxDuration)) return false
      if (minSkillValue && l.player[skillField] < Number(minSkillValue)) return false
      if (minCaps && l.caps < Number(minCaps)) return false
      if (position !== '' && bestCorePosition(l.player) !== position) return false
      return true
    })
    .sort((a, b) => {
      const cmp =
        sortField === 'duration'
          ? a.durationSeasons - b.durationSeasons
          : sortField === 'age'
            ? a.player.age - b.player.age
            : sortField === 'skill'
              ? overallRating(a.player) - overallRating(b.player)
              : new Date(a.listedAt).getTime() - new Date(b.listedAt).getTime()
      return sortDir === 'asc' ? cmp : -cmp
    })

  const handleAcceptLoan = async (listing: LoanListing) => {
    const name = `${listing.player.firstName} ${listing.player.lastName}`
    const message = listing.loanFee
      ? t('transfers.confirmAcceptLoanWithFee', { name, seasons: listing.durationSeasons, fee: formatCurrency(listing.loanFee) })
      : t('transfers.confirmAcceptLoan', { name, seasons: listing.durationSeasons })
    const ok = await confirm({ message, confirmLabel: t('transfers.accept'), destructive: !!listing.loanFee })
    if (ok) {
      acceptLoan.mutate(listing.listingId)
    }
  }

  return (
    <Box>
      <Stack direction="row" sx={{ justifyContent: 'flex-end', mb: 1.5 }}>
        <Button variant="contained" onClick={() => setOfferOpen(true)} disabled={loanablePlayers.length === 0}>
          {t('transfers.offerLoan.button')}
        </Button>
      </Stack>
      <OfferLoanDialog open={offerOpen} onClose={() => setOfferOpen(false)} players={loanablePlayers} />

      {myListings.length > 0 && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
            {t('transfers.yourLoanListings')}
          </Typography>
          <Stack spacing={1}>
            {myListings.map((listing) => (
              <LoanCard
                key={listing.listingId}
                listing={listing}
                action={
                  <Button color="secondary" variant="outlined" onClick={() => unlistFromLoan.mutate(listing.player.id)}>
                    {t('transfers.unlist')}
                  </Button>
                }
              />
            ))}
          </Stack>
        </Box>
      )}

      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
        {t('transfers.loanMarket')}
      </Typography>
      {marketListings.length === 0 ? (
        <EmptyState compact title={t('transfers.noLoanListings')} />
      ) : (
        <>
          <FilterBar search={search} onSearch={setSearch} activeCount={activeFilters.length} onOpenFilters={sheet.show} />
          <ActiveFilterChips filters={activeFilters} />
          <FilterSheet
            open={sheet.open}
            onClose={sheet.hide}
            resultCount={filteredMarketListings.length}
            onClear={clearFilters}
            hasActiveFilters={activeFilters.length > 0}
          >
            <SortControl<LoanSortField>
              value={sortField}
              onChange={setSortField}
              direction={sortDir}
              onToggleDirection={() => setSortDir(sortDir === 'asc' ? 'desc' : 'asc')}
              options={[
                { value: 'listed', label: t('transfers.table.listed') },
                { value: 'duration', label: t('transfers.table.duration') },
                { value: 'age', label: t('transfers.table.age') },
                { value: 'skill', label: t('transfers.table.rating') },
              ]}
            />
            <PositionFilter value={position} onChange={setPosition} />
            <RangeFields label={t('transfers.filters.age')} min={minAge} max={maxAge} onMin={setMinAge} onMax={setMaxAge} />
            <FormControl fullWidth>
              <InputLabel id="loan-max-duration-label">{t('transfers.filters.maxDurationLabel')}</InputLabel>
              <Select
                labelId="loan-max-duration-label"
                label={t('transfers.filters.maxDurationLabel')}
                value={maxDuration}
                onChange={(event: SelectChangeEvent<number | ''>) => setMaxDuration(event.target.value === '' ? '' : Number(event.target.value))}
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
            <SkillFilter field={skillField} onField={setSkillFieldStr} minValue={minSkillValue} onMinValue={setMinSkillValue} />
            <NumberField label={t('transfers.filters.minCapsLabel')} value={minCaps} onValueChange={setMinCaps} fullWidth />
          </FilterSheet>
          {filteredMarketListings.length === 0 ? (
            <EmptyState compact title={t('transfers.filters.noResults')} />
          ) : (
            <Paginated
              rows={filteredMarketListings}
              resetKey={JSON.stringify([search, minAge, maxAge, maxDuration, skillField, minSkillValue, minCaps, position, sortField, sortDir])}
            >
              {(pageListings) => (
                <Stack spacing={1}>
                  {pageListings.map((listing) => (
                    <LoanCard
                      key={listing.listingId}
                      listing={listing}
                      action={
                        <Button variant="contained" onClick={() => handleAcceptLoan(listing)} disabled={acceptLoan.isPending}>
                          {t('transfers.accept')}
                        </Button>
                      }
                    />
                  ))}
                </Stack>
              )}
            </Paginated>
          )}
        </>
      )}
    </Box>
  )
}

function OffersTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data: received, isLoading: receivedLoading } = useOffersReceived(!!loggedUser)
  const { data: sent, isLoading: sentLoading } = useOffersSent(!!loggedUser)
  const { data: preReceived } = usePreContractsReceived(!!loggedUser)
  const { data: preSent } = usePreContractsSent(!!loggedUser)
  const withdrawPreContract = useWithdrawPreContractOffer()

  const acceptOffer = useAcceptOffer()
  const rejectOffer = useRejectOffer()
  const withdrawOffer = useWithdrawOffer()

  if (receivedLoading || sentLoading || !received || !sent) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const offerParams = (offer: TransferOffer) => ({
    name: `${offer.player.firstName} ${offer.player.lastName}`,
    team: offer.fromTeamName,
    amount: formatCurrency(offer.amount),
  })

  // Selling a player is irreversible, so accepting asks twice: the second prompt is the last chance to back out.
  const handleAccept = async (offer: TransferOffer) => {
    const params = offerParams(offer)
    const first = await confirm({
      message: t('transfers.confirmAcceptOffer', params),
      confirmLabel: t('transfers.accept'),
      destructive: true,
    })
    if (!first) return
    const second = await confirm({
      title: t('transfers.confirmAcceptOfferFinalTitle'),
      message: t('transfers.confirmAcceptOfferFinal', params),
      confirmLabel: t('transfers.accept'),
      destructive: true,
    })
    if (second) {
      acceptOffer.mutate(offer.offerId)
    }
  }

  const handleReject = async (offer: TransferOffer) => {
    const ok = await confirm({
      message: t('transfers.confirmRejectOffer', offerParams(offer)),
      confirmLabel: t('transfers.reject'),
      destructive: true,
    })
    if (ok) {
      rejectOffer.mutate(offer.offerId)
    }
  }

  const handleWithdraw = async (offer: TransferOffer) => {
    const ok = await confirm({
      message: t('transfers.confirmWithdrawOffer', {
        name: `${offer.player.firstName} ${offer.player.lastName}`,
        amount: formatCurrency(offer.amount),
      }),
      confirmLabel: t('transfers.withdraw'),
      destructive: true,
    })
    if (ok) {
      withdrawOffer.mutate(offer.offerId)
    }
  }

  const handleWithdrawPreContract = async (offer: PreContractOffer) => {
    const ok = await confirm({
      message: t('transfers.preContract.confirmWithdraw', { name: `${offer.player.firstName} ${offer.player.lastName}` }),
      confirmLabel: t('transfers.withdraw'),
      destructive: true,
    })
    if (ok) {
      withdrawPreContract.mutate(offer.offerId)
    }
  }

  return (
    <Box>
      <Alert severity="info" sx={{ mb: 2 }}>
        {t('transfers.offers.intro')}
      </Alert>

      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
        {t('transfers.offers.onYourPlayers')}
      </Typography>
      {received.length === 0 ? (
        <EmptyState compact title={t('transfers.offers.noPending')} />
      ) : (
        <ReceivedOffersByPlayer
          offers={received}
          renderAction={(offer) => (
            <ReceivedOfferActions
              offer={offer}
              onAccept={handleAccept}
              onReject={handleReject}
              acceptPending={acceptOffer.isPending}
              rejectPending={rejectOffer.isPending}
            />
          )}
        />
      )}

      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mt: 3, mb: 1 }}>
        {t('transfers.offers.yourOffers')}
      </Typography>
      {sent.length === 0 ? (
        <EmptyState compact title={t('transfers.offers.noneSent')} />
      ) : (
        <Stack spacing={1}>
          {sent.map((offer) => (
            <SentOfferCard key={offer.offerId} offer={offer} onWithdraw={handleWithdraw} withdrawPending={withdrawOffer.isPending} />
          ))}
        </Stack>
      )}

      <PreContractsSection
        received={preReceived ?? []}
        sent={preSent ?? []}
        onWithdraw={handleWithdrawPreContract}
        withdrawPending={withdrawPreContract.isPending}
      />
    </Box>
  )
}

// Groups offers received on your own players by player (openspec/specs/team-finances,
// "Transfers") - GetOffersReceived already returns every pending offer from every
// interested buyer, each at most one per team (backend enforces that uniquely); this is purely
// how they're laid out, so a seller comparing bids for the same player sees them side by side
// instead of interleaved by date with every other player's offers. Sorted by amount desc within
// each group, so the best offer always leads.
function ReceivedOffersByPlayer({
  offers,
  renderAction,
}: {
  offers: TransferOffer[]
  renderAction: (offer: TransferOffer) => React.ReactNode
}) {
  const { t } = useTranslation()
  const groups: TransferOffer[][] = []
  const groupIndexByPlayerId = new Map<number, number>()
  for (const offer of offers) {
    const existingIndex = groupIndexByPlayerId.get(offer.player.id)
    if (existingIndex === undefined) {
      groupIndexByPlayerId.set(offer.player.id, groups.length)
      groups.push([offer])
    } else {
      groups[existingIndex].push(offer)
    }
  }
  for (const group of groups) {
    group.sort((a, b) => b.amount - a.amount)
  }

  return (
    <Stack spacing={1.5}>
      {groups.map((group) => {
        const player = group[0].player
        return (
          <Paper key={player.id} variant="outlined" sx={{ p: 1.5 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1, flexWrap: 'wrap' }}>
              <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" color="inherit" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
                {player.firstName} {player.lastName}
              </Link>
              {group.length > 1 && <Chip size="small" label={t('transfers.offers.competingCount', { count: group.length })} />}
            </Stack>
            <Stack spacing={1.5}>
              {group.map((offer, index) => (
                <Box
                  key={offer.offerId}
                  sx={{ pt: index === 0 ? 0 : 1.5, borderTop: index === 0 ? 0 : 1, borderColor: 'divider' }}
                >
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', justifyContent: 'space-between' }}>
                    <Box sx={{ minWidth: 0 }}>
                      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                        <TeamNameLink teamId={offer.fromTeamId} variant="body2" sx={{ fontWeight: 600 }}>
                          {offer.fromTeamName}
                        </TeamNameLink>
                        {index === 0 && group.length > 1 && (
                          <Chip size="small" color="success" variant="outlined" label={t('transfers.offers.bestOffer')} sx={{ height: 20 }} />
                        )}
                      </Stack>
                      <Typography variant="caption" color="text.secondary" component="div">
                        {t('transfers.offers.termsLine', {
                          wage: formatCurrency(offer.offeredWage),
                          years: t('team.contractDialog.yearsOption', { count: offer.offeredYears }),
                        })}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" component="div">
                        {t('transfers.offers.made', { date: formatShortDate(offer.createdAt) })}
                      </Typography>
                    </Box>
                    <Typography variant="subtitle1" sx={{ fontWeight: 700, flexShrink: 0 }}>
                      {formatCurrency(offer.amount)}
                    </Typography>
                  </Stack>
                  <Stack direction="row" sx={{ justifyContent: 'flex-end', mt: 1 }}>
                    {renderAction(offer)}
                  </Stack>
                </Box>
              ))}
            </Stack>
          </Paper>
        )
      })}
    </Stack>
  )
}

function FreeAgentsTab() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: freeAgents, isLoading } = useFreeAgents(!!loggedUser)
  const [signingPlayer, setSigningPlayer] = useState<FreeAgentListing | null>(null)
  const sheet = useFilterSheet()

  const [search, setSearch] = usePersistedFilter<string>('faSearch', '')
  const [minAge, setMinAge] = usePersistedFilter<string>('faMinAge', '')
  const [maxAge, setMaxAge] = usePersistedFilter<string>('faMaxAge', '')
  const [positionStr, setPositionStr] = usePersistedFilter<string>('faPosition', '')
  const position = positionStr === '' ? '' : Number(positionStr)
  const setPosition = (value: number | '') => setPositionStr(value === '' ? '' : String(value))
  const [skillFieldStr, setSkillFieldStr] = usePersistedFilter<string>('faSkillField', SKILL_FIELDS[0])
  const skillField = skillFieldStr as (typeof SKILL_FIELDS)[number]
  const [minSkillValue, setMinSkillValue] = usePersistedFilter<string>('faMinSkillValue', '')
  const [minCaps, setMinCaps] = usePersistedFilter<string>('faMinCaps', '')
  const [sortField, setSortField] = usePersistedFilter<FreeAgentSortField>('faSort', 'wage')
  const [sortDir, setSortDir] = usePersistedFilter<SortDirection>('faDir', 'desc')

  if (isLoading || !freeAgents) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMinSkillValue('')
    setMinCaps('')
    setPosition('')
  }

  const activeFilters: ActiveFilter[] = [
    ...(search.trim() ? [{ key: 'search', label: t('transfers.filters.active.search', { value: search.trim() }), onDelete: () => setSearch('') }] : []),
    ...(minAge || maxAge
      ? [
          {
            key: 'age',
            label:
              minAge && maxAge
                ? t('transfers.filters.active.age', { min: minAge, max: maxAge })
                : minAge
                  ? t('transfers.filters.active.ageFrom', { value: minAge })
                  : t('transfers.filters.active.ageTo', { value: maxAge }),
            onDelete: () => {
              setMinAge('')
              setMaxAge('')
            },
          },
        ]
      : []),
    ...(minSkillValue ? [{ key: 'skill', label: t('transfers.filters.active.skill', { skill: t(SKILL_LABEL_KEYS[skillField]), value: minSkillValue }), onDelete: () => setMinSkillValue('') }] : []),
    ...(minCaps ? [{ key: 'caps', label: t('transfers.filters.active.caps', { value: minCaps }), onDelete: () => setMinCaps('') }] : []),
    ...(position !== '' ? [{ key: 'position', label: t(POSITION_SHORT_LABEL_KEYS[position]), onDelete: () => setPosition('') }] : []),
  ]

  const filteredFreeAgents = freeAgents
    .filter((p) => {
      const term = search.trim().toLowerCase()
      if (term && !`${p.firstName} ${p.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && p.age < Number(minAge)) return false
      if (maxAge && p.age > Number(maxAge)) return false
      if (minSkillValue && p[skillField] < Number(minSkillValue)) return false
      if (minCaps && p.caps < Number(minCaps)) return false
      if (position !== '' && bestCorePosition(p) !== position) return false
      return true
    })
    .sort((a, b) => {
      const cmp = sortField === 'age' ? a.age - b.age : sortField === 'skill' ? overallRating(a) - overallRating(b) : a.wage - b.wage
      return sortDir === 'asc' ? cmp : -cmp
    })

  return (
    <Box>
      <Alert severity="info" sx={{ mb: 2 }}>
        {t('transfers.freeAgents.intro')}
      </Alert>
      {freeAgents.length === 0 ? (
        <EmptyState compact title={t('transfers.freeAgents.none')} />
      ) : (
        <>
          <FilterBar search={search} onSearch={setSearch} activeCount={activeFilters.length} onOpenFilters={sheet.show} />
          <ActiveFilterChips filters={activeFilters} />
          <FilterSheet
            open={sheet.open}
            onClose={sheet.hide}
            resultCount={filteredFreeAgents.length}
            onClear={clearFilters}
            hasActiveFilters={activeFilters.length > 0}
          >
            <SortControl<FreeAgentSortField>
              value={sortField}
              onChange={setSortField}
              direction={sortDir}
              onToggleDirection={() => setSortDir(sortDir === 'asc' ? 'desc' : 'asc')}
              options={[
                { value: 'wage', label: t('transfers.freeAgents.wage') },
                { value: 'age', label: t('transfers.table.age') },
                { value: 'skill', label: t('transfers.table.rating') },
              ]}
            />
            <PositionFilter value={position} onChange={setPosition} />
            <RangeFields label={t('transfers.filters.age')} min={minAge} max={maxAge} onMin={setMinAge} onMax={setMaxAge} />
            <SkillFilter field={skillField} onField={setSkillFieldStr} minValue={minSkillValue} onMinValue={setMinSkillValue} />
            <NumberField label={t('transfers.filters.minCapsLabel')} value={minCaps} onValueChange={setMinCaps} fullWidth />
          </FilterSheet>
          {filteredFreeAgents.length === 0 ? (
            <EmptyState compact title={t('transfers.filters.noResults')} />
          ) : (
            <Paginated
              rows={filteredFreeAgents}
              resetKey={JSON.stringify([search, minAge, maxAge, skillField, minSkillValue, minCaps, position, sortField, sortDir])}
            >
              {(pageFreeAgents) => (
                <Stack spacing={1}>
                  {pageFreeAgents.map((player) => (
                    <ListingCard
                      key={player.id}
                      player={player}
                      formerClub={player.lastTeamName ? t('transfers.freeAgents.formerClub', { team: player.lastTeamName }) : undefined}
                      badge={
                        player.freeAgentSeasons === 0 && !!player.lastTeamId ? (
                          <Chip size="small" label={t('transfers.freeAgents.premiumChip')} />
                        ) : undefined
                      }
                      meta={
                        <>
                          <MetaItem>{t('transfers.ageLabel', { age: player.age })}</MetaItem>
                          {player.caps > 0 && (
                            <>
                              <MetaDot />
                              <MetaItem>{t('transfers.filters.capsCount', { count: player.caps })}</MetaItem>
                            </>
                          )}
                        </>
                      }
                      priceLabel={t('transfers.freeAgents.wage')}
                      priceValue={t('team.perMatch', { amount: formatCurrency(player.wage) })}
                      actions={
                        <Button variant="contained" onClick={() => setSigningPlayer(player)}>
                          {t('transfers.freeAgents.sign')}
                        </Button>
                      }
                    />
                  ))}
                </Stack>
              )}
            </Paginated>
          )}
        </>
      )}
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
        <Alert severity="info" sx={{ mb: 2 }}>
          {t('transfers.coaching.intro')}
        </Alert>
        <Paper sx={{ p: 2, maxWidth: 420 }}>
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

  return (
    <Box>
      <Alert severity="info" sx={{ mb: 2 }}>
        {t('transfers.coaching.intro')}
      </Alert>
      {available.length === 0 ? (
        <EmptyState compact title={t('transfers.coaching.none')} />
      ) : (
        <>
          <Box sx={{ mb: 1.5 }}>
            <SortControl<TrainerSortField>
              value={sortField}
              onChange={setSortField}
              direction={sortDir}
              onToggleDirection={() => setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))}
              options={[
                { value: 'skill', label: t('transfers.coaching.skill') },
                { value: 'wage', label: t('transfers.coaching.wage') },
              ]}
            />
          </Box>
          <Paginated rows={sorted} resetKey={JSON.stringify([sortField, sortDir])}>
            {(pageCandidates) => (
              <Stack spacing={1}>
                {pageCandidates.map((candidate) => (
                  <Paper key={candidate.id} variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
                      {candidate.firstName} {candidate.lastName}
                    </Typography>
                    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', mt: 0.5 }}>
                      <Box>
                        <Typography variant="body2">{t('transfers.coaching.skillLabel', { value: candidate.skill })}</Typography>
                        <Typography variant="caption" color="text.secondary">
                          {t('transfers.coaching.wage')}: {formatCurrency(candidate.wage)}
                        </Typography>
                      </Box>
                      <Button variant="contained" onClick={() => handleHire(candidate)} disabled={hireTrainer.isPending}>
                        {t('transfers.coaching.hire')}
                      </Button>
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            )}
          </Paginated>
        </>
      )}
    </Box>
  )
}
