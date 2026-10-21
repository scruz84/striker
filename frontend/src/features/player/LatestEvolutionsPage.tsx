import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useMe, useTeamLatestEvolutions } from '../../api/hooks'
import { PlayerEvolutionBody } from '../../components/PlayerEvolutionCard'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import TrendingUpIcon from '@mui/icons-material/TrendingUp'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'

// The page a NOTIFICATION_PROGRESSION notification links to, via its own logIds query param -
// exactly the caller's own squad's players whose skills evolved from that one specific
// match-triggered progression run (see engine.updatePlayerSkillsAfterMatch), each with what
// changed (see PlayerEvolutionBody). Opened without logIds (direct navigation, no particular
// match in mind), it instead lists every current squad player's own all-time latest run - that
// case can span many different matches, so it's a browsing view rather than a single match's
// report.
export function LatestEvolutionsPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const logIds = searchParams.get('logIds') ?? undefined
  const { data: loggedUser } = useMe()
  const { data: evolutions, isLoading } = useTeamLatestEvolutions(!!loggedUser, logIds)

  if (isLoading || !evolutions) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Stack spacing={{ xs: 1.5, sm: 2 }}>
      <PageHeader title={t('evolution.pageTitle')} />
      {evolutions.length === 0 ? (
        <EmptyState icon={<TrendingUpIcon fontSize="inherit" />} title={t('evolution.noneTitle')} text={t('evolution.none')} />
      ) : (
        <Stack spacing={2}>
          {evolutions.map((evolution) => (
            <Paper key={evolution.playerId} variant="outlined" sx={{ p: 2 }}>
              <Typography variant="subtitle1" gutterBottom>
                <PlayerNameLink playerId={evolution.playerId}>
                  {evolution.firstName} {evolution.lastName}
                </PlayerNameLink>
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                {t('evolution.season', { season: evolution.season })}
              </Typography>
              <PlayerEvolutionBody evolution={evolution} />
            </Paper>
          ))}
        </Stack>
      )}
    </Stack>
  )
}
