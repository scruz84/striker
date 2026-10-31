import { useState } from 'react'
import Button from '@mui/material/Button'
import Divider from '@mui/material/Divider'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useNationalTeamResults } from '../../api/hooks'
import { EmptyState } from '../../components/EmptyState'
import { NationalMatchFilters } from './NationalMatchFilters'
import { NationalMatchRow } from './NationalMatchRow'
import { countriesMatching, EMPTY_FILTER, matchesFilter } from './nationalMatches'

/** Every finished national-team result, newest first, 25 at a time (see backend
 * common.GetNationalTeamResults). A search that names exactly one country asks the server for
 * that country's results; the region chips filter what is loaded. */
export function ResultsTab() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState(EMPTY_FILTER)
  const matchingCountries = countriesMatching(t, filter.query)
  const country = matchingCountries.length === 1 ? matchingCountries[0] : undefined
  const { data, isLoading, isError, fetchNextPage, hasNextPage, isFetchingNextPage } = useNationalTeamResults(country, true)

  const matches = (data?.pages ?? []).flatMap((page) => page.matches)
  const visible = matches.filter((m) => matchesFilter(t, m, { ...filter, onlyMine: false }))
  const ambiguous = filter.query.trim() !== '' && matchingCountries.length !== 1

  return (
    <Stack spacing={2}>
      <NationalMatchFilters filter={filter} onChange={setFilter} showMineToggle={false} />
      {ambiguous && matchingCountries.length > 1 && (
        <Typography variant="caption" color="text.secondary">
          {t('nationalTeams.results.refineSearch', { count: matchingCountries.length })}
        </Typography>
      )}
      {isLoading ? (
        <Skeleton variant="rectangular" height={240} />
      ) : isError ? (
        <EmptyState compact title={t('nationalTeams.results.error')} />
      ) : visible.length === 0 ? (
        <EmptyState compact title={matches.length === 0 ? t('nationalTeams.results.empty') : t('nationalTeams.filters.noMatches')} />
      ) : (
        <Paper variant="outlined" sx={{ p: 1 }}>
          <Stack divider={<Divider flexItem />}>
            {visible.map((match) => (
              <NationalMatchRow key={`${match.kind}-${match.id}`} match={match} />
            ))}
          </Stack>
        </Paper>
      )}
      {hasNextPage && (
        <Button variant="outlined" onClick={() => fetchNextPage()} disabled={isFetchingNextPage}>
          {t('nationalTeams.results.loadMore')}
        </Button>
      )}
    </Stack>
  )
}
