import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMe, usePreContractMarket, useUserTeam } from '../../api/hooks'
import { POSITION_SHORT_LABEL_KEYS, type PreContractMarketEntry } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { PreContractDialog } from '../../components/PreContractDialog'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { WatchToggle } from '../../components/WatchToggle'
import { EmptyState } from '../../components/EmptyState'
import { Paginated } from '../../components/Paginated'
import { formatCurrency } from '../../utils/formatCurrency'
import {
  ActiveFilterChips,
  ContractMeta,
  FilterBar,
  FilterSheet,
  PositionFilter,
  RangeFields,
  SortControl,
  useFilterSheet,
  usePersistedFilter,
  type ActiveFilter,
  type SortDirection,
} from './marketUi'

type LastSeasonSortField = 'skill' | 'age' | 'salary' | 'clause'

// The fields a filter can compare: another club's skills stay hidden here (and caps aren't sent),
// so only the public facts the row shows - its best position, age, rating, wage and clause.
const clauseOf = (entry: PreContractMarketEntry) => entry.player.releaseClause ?? 0
const ratingOf = (entry: PreContractMarketEntry) => entry.player.averageSkill ?? 0

/** The transfer market's "Last season" tab (openspec/specs/pre-contracts "Pre-contract market"):
 * the best players of other clubs in the last season of their contract, with whether a
 * pre-contract offer is open yet (their league halfway through) and the actions it allows. */
export function LastSeasonTab() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: userTeam } = useUserTeam(!!me)
  const { data: entries, isLoading } = usePreContractMarket(!!userTeam)
  const sheet = useFilterSheet()

  const [search, setSearch] = usePersistedFilter<string>('lsSearch', '')
  const [minAge, setMinAge] = usePersistedFilter<string>('lsMinAge', '')
  const [maxAge, setMaxAge] = usePersistedFilter<string>('lsMaxAge', '')
  const [minSkill, setMinSkill] = usePersistedFilter<string>('lsMinSkill', '')
  const [maxSkill, setMaxSkill] = usePersistedFilter<string>('lsMaxSkill', '')
  const [minWage, setMinWage] = usePersistedFilter<string>('lsMinWage', '')
  const [maxWage, setMaxWage] = usePersistedFilter<string>('lsMaxWage', '')
  const [minClause, setMinClause] = usePersistedFilter<string>('lsMinClause', '')
  const [maxClause, setMaxClause] = usePersistedFilter<string>('lsMaxClause', '')
  const [positionStr, setPositionStr] = usePersistedFilter<string>('lsPosition', '')
  const position = positionStr === '' ? '' : Number(positionStr)
  const setPosition = (value: number | '') => setPositionStr(value === '' ? '' : String(value))
  const [sortField, setSortField] = usePersistedFilter<LastSeasonSortField>('lsSort', 'skill')
  const [sortDir, setSortDir] = usePersistedFilter<SortDirection>('lsDir', 'desc')

  if (!userTeam) {
    return <Alert severity="info">{t('watchlist.needTeam')}</Alert>
  }
  if (isLoading || !entries) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const clearFilters = () => {
    setSearch('')
    setMinAge('')
    setMaxAge('')
    setMinSkill('')
    setMaxSkill('')
    setMinWage('')
    setMaxWage('')
    setMinClause('')
    setMaxClause('')
    setPosition('')
  }

  const money = (v: string) => formatCurrency(Number(v), { compact: true })
  const rangeChip = (key: string, min: string, max: string, clear: () => void, format: (v: string) => string = (v) => v): ActiveFilter[] =>
    min || max
      ? [
          {
            key,
            label:
              min && max
                ? t(`transfers.filters.active.${key}`, { min: format(min), max: format(max) })
                : min
                  ? t(`transfers.filters.active.${key}From`, { value: format(min) })
                  : t(`transfers.filters.active.${key}To`, { value: format(max) }),
            onDelete: clear,
          },
        ]
      : []

  const activeFilters: ActiveFilter[] = [
    ...(search.trim() ? [{ key: 'search', label: t('transfers.filters.active.search', { value: search.trim() }), onDelete: () => setSearch('') }] : []),
    ...rangeChip('age', minAge, maxAge, () => {
      setMinAge('')
      setMaxAge('')
    }),
    ...rangeChip('rating', minSkill, maxSkill, () => {
      setMinSkill('')
      setMaxSkill('')
    }),
    ...rangeChip('salary', minWage, maxWage, () => {
      setMinWage('')
      setMaxWage('')
    }, money),
    ...rangeChip('clause', minClause, maxClause, () => {
      setMinClause('')
      setMaxClause('')
    }, money),
    ...(position !== '' ? [{ key: 'position', label: t(POSITION_SHORT_LABEL_KEYS[position]), onDelete: () => setPosition('') }] : []),
  ]

  const term = search.trim().toLowerCase()
  const filteredEntries = entries
    .filter((entry) => {
      const { player } = entry
      if (term && !`${player.firstName} ${player.lastName}`.toLowerCase().includes(term)) return false
      if (minAge && player.age < Number(minAge)) return false
      if (maxAge && player.age > Number(maxAge)) return false
      if (minSkill && ratingOf(entry) < Number(minSkill)) return false
      if (maxSkill && ratingOf(entry) > Number(maxSkill)) return false
      if (minWage && player.wage < Number(minWage)) return false
      if (maxWage && player.wage > Number(maxWage)) return false
      if (minClause && clauseOf(entry) < Number(minClause)) return false
      if (maxClause && clauseOf(entry) > Number(maxClause)) return false
      if (position !== '' && entry.bestPosition !== position) return false
      return true
    })
    .sort((a, b) => {
      const cmp =
        sortField === 'age'
          ? a.player.age - b.player.age
          : sortField === 'salary'
            ? a.player.wage - b.player.wage
            : sortField === 'clause'
              ? clauseOf(a) - clauseOf(b)
              : ratingOf(a) - ratingOf(b)
      return sortDir === 'asc' ? cmp : -cmp
    })

  return (
    <Box>
      <Alert severity="info" sx={{ mb: 1.5 }}>
        {t('transfers.lastSeason.intro')}
      </Alert>
      {entries.length === 0 ? (
        <Alert severity="info">{t('transfers.lastSeason.empty')}</Alert>
      ) : (
        <>
          <FilterBar search={search} onSearch={setSearch} activeCount={activeFilters.length} onOpenFilters={sheet.show} />
          <ActiveFilterChips filters={activeFilters} />
          <FilterSheet
            open={sheet.open}
            onClose={sheet.hide}
            resultCount={filteredEntries.length}
            onClear={clearFilters}
            hasActiveFilters={activeFilters.length > 0}
          >
            <SortControl<LastSeasonSortField>
              value={sortField}
              onChange={setSortField}
              direction={sortDir}
              onToggleDirection={() => setSortDir(sortDir === 'asc' ? 'desc' : 'asc')}
              options={[
                { value: 'skill', label: t('transfers.table.rating') },
                { value: 'age', label: t('transfers.table.age') },
                { value: 'salary', label: t('transfers.table.salary') },
                { value: 'clause', label: t('transfers.filters.releaseClause') },
              ]}
            />
            <PositionFilter value={position} onChange={setPosition} />
            <RangeFields label={t('transfers.filters.age')} min={minAge} max={maxAge} onMin={setMinAge} onMax={setMaxAge} />
            <RangeFields label={t('transfers.filters.rating')} min={minSkill} max={maxSkill} onMin={setMinSkill} onMax={setMaxSkill} />
            <RangeFields label={t('transfers.filters.salary')} min={minWage} max={maxWage} onMin={setMinWage} onMax={setMaxWage} currency />
            <RangeFields label={t('transfers.filters.releaseClause')} min={minClause} max={maxClause} onMin={setMinClause} onMax={setMaxClause} currency />
          </FilterSheet>
          {filteredEntries.length === 0 ? (
            <EmptyState compact title={t('transfers.filters.noResults')} />
          ) : (
            <Paginated
              rows={filteredEntries}
              resetKey={JSON.stringify([search, minAge, maxAge, minSkill, maxSkill, minWage, maxWage, minClause, maxClause, position, sortField, sortDir])}
            >
              {(pageEntries) => (
                <Stack spacing={1}>
                  {pageEntries.map((entry) => (
                    <LastSeasonRow key={entry.player.id} entry={entry} />
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

export function LastSeasonRow({ entry }: { entry: PreContractMarketEntry }) {
  const { t } = useTranslation()
  const [preContractOpen, setPreContractOpen] = useState(false)
  const [offerOpen, setOfferOpen] = useState(false)
  const { player } = entry
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }} data-testid="last-season-row">
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={40} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
            <PlayerNameLink playerId={player.id}>
              {player.firstName} {player.lastName}
            </PlayerNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" component="div">
            {t(POSITION_SHORT_LABEL_KEYS[entry.bestPosition])} · {t('lineup.ageShort', { age: player.age })} ·{' '}
            {t('transfers.lastSeason.averageSkill', { skill: player.averageSkill ?? 0 })} ·{' '}
            <TeamNameLink teamId={player.teamId}>{entry.teamName}</TeamNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.salaryPerMatch', { amount: formatCurrency(player.wage) })}
          </Typography>
          <ContractMeta player={player} />
          {!entry.open && (
            <Typography variant="caption" color="text.secondary" component="div">
              {t('transfers.lastSeason.notOpen')}
            </Typography>
          )}
        </Box>
        <WatchToggle player={player} size="small" />
      </Stack>
      <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', mt: 1 }}>
        <Button variant="outlined" onClick={() => setOfferOpen(true)}>
          {t('team.makeOffer')}
        </Button>
        <Button variant="contained" disabled={!entry.open || entry.offered} onClick={() => setPreContractOpen(true)}>
          {entry.offered ? t('transfers.lastSeason.offered') : t('transfers.preContract.button')}
        </Button>
      </Stack>
      {preContractOpen && <PreContractDialog player={player} open={preContractOpen} onClose={() => setPreContractOpen(false)} />}
      {offerOpen && <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />}
    </Paper>
  )
}
