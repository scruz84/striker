import Link from '@mui/material/Link'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTeamDetails } from '../../api/hooks'

export function TeamDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: teamDetails, isLoading } = useTeamDetails(id)

  if (isLoading || !teamDetails) {
    return <div>Loading...</div>
  }

  return (
    <Stack spacing={1}>
      <Typography variant="h4" gutterBottom>
        {teamDetails.team.name} ({teamDetails.team.id})
      </Typography>
      <Typography sx={{ fontWeight: 'bold' }}>Competitions:</Typography>
      <Link component={RouterLink} to={`/division/${teamDetails.team.divisionId}`}>
        {teamDetails.team.divisionName}
      </Link>
    </Stack>
  )
}
