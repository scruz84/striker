import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { SkillBar } from './SkillBar'

export function PlayerCard({ player }: { player: Player }) {
  const { t } = useTranslation()
  return (
    <Card sx={{ mb: 1 }}>
      <CardContent>
        <Grid container spacing={2}>
          <Grid size={{ xs: 3, sm: 2 }}>
            <img src="/avatars/player.svg" height="80" loading="lazy" alt={player.firstName} />
          </Grid>
          <Grid size={{ xs: 9, sm: 4 }}>
            <Typography variant="subtitle1">
              {player.firstName} {player.lastName}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {t('playerCard.age', { age: player.age })}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <SkillBar label={t('playerCard.skills.goalkeeper')} value={player.goalKeeper} />
            <SkillBar label={t('playerCard.skills.defense')} value={player.defense} />
            <SkillBar label={t('playerCard.skills.lateral')} value={player.lateral} />
            <SkillBar label={t('playerCard.skills.pass')} value={player.pass} />
            <SkillBar label={t('playerCard.skills.midfield')} value={player.midfield} />
            <SkillBar label={t('playerCard.skills.dribbling')} value={player.dribbling} />
            <SkillBar label={t('playerCard.skills.shot')} value={player.shot} />
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  )
}
