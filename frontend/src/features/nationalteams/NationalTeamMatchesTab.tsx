import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useNationalTeamResults, useNationalTeamRound } from '../../api/hooks'
import { EmptyState } from '../../components/EmptyState'
import { NationalMatchRow } from './NationalMatchRow'
import { formStrip, type FormLetter } from './nationalMatches'

const FORM_COLORS: Record<FormLetter, string> = { W: 'success.main', D: 'warning.main', L: 'error.main' }

/** A national team's own matches on its team page: what it plays next (or is playing now), its
 * last 10 results and a five-result form strip. */
export function NationalTeamMatchesTab({ teamId, country }: { teamId: number; country: string }) {
  const { t } = useTranslation()
  const { data: round, isLoading: roundLoading } = useNationalTeamRound(true)
  const { data: results, isLoading: resultsLoading } = useNationalTeamResults(country, true)

  const current = (round ?? []).filter((m) => (m.homeTeamId === teamId || m.awayTeamId === teamId) && m.status !== 'finished')
  const last = (results?.pages[0]?.matches ?? []).slice(0, 10)
  const form = formStrip(last, teamId)

  return (
    <Stack spacing={2} sx={{ pt: 1 }}>
      <Paper variant="outlined" sx={{ p: 1 }}>
        <Typography variant="subtitle1" component="h2" sx={{ px: 1, pb: 1 }}>
          {t('nationalTeams.teamMatches.next')}
        </Typography>
        {roundLoading ? (
          <Skeleton variant="rectangular" height={56} />
        ) : current.length === 0 ? (
          <EmptyState compact title={t('nationalTeams.teamMatches.noNext')} />
        ) : (
          <Stack divider={<Divider flexItem />}>
            {current.map((match) => (
              <NationalMatchRow key={`${match.kind}-${match.id}`} match={match} />
            ))}
          </Stack>
        )}
      </Paper>

      <Paper variant="outlined" sx={{ p: 1 }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 1, pb: 1, flexWrap: 'wrap', rowGap: 0.5 }}>
          <Typography variant="subtitle1" component="h2" sx={{ flex: 1 }}>
            {t('nationalTeams.teamMatches.lastResults')}
          </Typography>
          {form.length > 0 && (
            <Stack direction="row" spacing={0.5} aria-label={t('nationalTeams.teamMatches.form')} data-testid="form-strip">
              {form.map((letter, i) => (
                <Box
                  key={i}
                  title={t(`nationalTeams.teamMatches.formLetters.${letter}`)}
                  sx={{
                    width: 22,
                    height: 22,
                    borderRadius: 0.5,
                    bgcolor: FORM_COLORS[letter],
                    color: 'common.white',
                    fontSize: '0.75rem',
                    fontWeight: 700,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  {t(`nationalTeams.teamMatches.formShort.${letter}`)}
                </Box>
              ))}
            </Stack>
          )}
        </Stack>
        {resultsLoading ? (
          <Skeleton variant="rectangular" height={120} />
        ) : last.length === 0 ? (
          <EmptyState compact title={t('nationalTeams.results.empty')} />
        ) : (
          <Stack divider={<Divider flexItem />}>
            {last.map((match) => (
              <NationalMatchRow key={`${match.kind}-${match.id}`} match={match} />
            ))}
          </Stack>
        )}
      </Paper>
    </Stack>
  )
}
