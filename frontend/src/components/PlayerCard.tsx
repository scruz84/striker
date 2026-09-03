import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import type { Player } from '../api/types'
import { SkillBar } from './SkillBar'

export function PlayerCard({ player }: { player: Player }) {
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
              Age {player.age}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <SkillBar label="Goalkeeper" value={player.goalKeeper} />
            <SkillBar label="Defense" value={player.defense} />
            <SkillBar label="Lateral" value={player.lateral} />
            <SkillBar label="Pass" value={player.pass} />
            <SkillBar label="Midfield" value={player.midfield} />
            <SkillBar label="Dribbling" value={player.dribbling} />
            <SkillBar label="Shot" value={player.shot} />
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  )
}
