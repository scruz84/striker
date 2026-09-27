import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMe, useTeamLatestEvolutions } from '../../api/hooks'
import { PlayerEvolutionBody } from '../../components/PlayerEvolutionCard'
import { PlayerNameLink } from '../../components/PlayerNameLink'

// The page a NOTIFICATION_PROGRESSION notification links to - every one of the caller's own
// squad's players who went through a match-triggered progression run (see
// engine.updatePlayerSkillsAfterMatch), each with what changed since then (see
// PlayerEvolutionBody). A player who never played a competitive minute simply has no row here.
export function LatestEvolutionsPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: evolutions, isLoading } = useTeamLatestEvolutions(!!loggedUser)

  if (isLoading || !evolutions) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Stack spacing={{ xs: 1.5, sm: 2 }}>
      <Typography variant="h5">{t('evolution.pageTitle')}</Typography>
      {evolutions.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {t('evolution.none')}
        </Typography>
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
