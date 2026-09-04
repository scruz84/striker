import { useEffect } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents'
import GroupsIcon from '@mui/icons-material/Groups'
import StadiumIcon from '@mui/icons-material/Stadium'
import SwapHorizIcon from '@mui/icons-material/SwapHoriz'
import TrendingUpIcon from '@mui/icons-material/TrendingUp'
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer'
import { useMe } from '../../api/hooks'

const FEATURES = [
  {
    icon: SportsSoccerIcon,
    title: 'Live match simulation',
    description: 'Set your lineup and formation, then follow every match minute-by-minute as it plays out.',
  },
  {
    icon: SwapHorizIcon,
    title: 'Transfers & loans',
    description: 'Buy, sell, scout, and negotiate direct offers - or loan players in and out between clubs.',
  },
  {
    icon: EmojiEventsIcon,
    title: 'Leagues & cups',
    description: 'Battle promotion and relegation across tiers, and chase silverware in knockout cup competitions.',
  },
  {
    icon: StadiumIcon,
    title: 'Club finances',
    description: 'Balance wages against ticket revenue, sponsorship, and prize money - then reinvest in your stadium.',
  },
  {
    icon: TrendingUpIcon,
    title: 'Squad development',
    description: 'Watch your academy prospects progress, and manage aging veterans as they approach retirement.',
  },
  {
    icon: GroupsIcon,
    title: 'Friend groups',
    description: 'Create a private leaderboard with friends and see who builds the most valuable club.',
  },
]

const STEPS = [
  { step: '1', title: 'Claim a club', description: 'Pick an unowned team from any league and take the reins.' },
  { step: '2', title: 'Manage every detail', description: 'Set your lineup, work the transfer market, and grow your finances.' },
  { step: '3', title: 'Climb the table', description: 'Win matches, earn promotion, and lift the cup.' },
]

export function HomePage() {
  const { data: loggedUser, isLoading } = useMe()
  const navigate = useNavigate()

  useEffect(() => {
    if (loggedUser) {
      navigate('/dashboard')
    }
  }, [loggedUser, navigate])

  if (isLoading || loggedUser) {
    return null
  }

  return (
    <Box>
      <Box
        sx={{
          background: (theme) =>
            `linear-gradient(135deg, ${theme.palette.primary.dark ?? theme.palette.primary.main} 0%, ${theme.palette.primary.main} 100%)`,
          color: 'primary.contrastText',
        }}
      >
        <Container maxWidth="md" sx={{ py: { xs: 8, sm: 12 }, textAlign: 'center' }}>
          <Typography variant="h2" component="h1" sx={{ fontWeight: 700, mb: 2, fontSize: { xs: '2.25rem', sm: '3rem' } }}>
            Build your club. Own every decision.
          </Typography>
          <Typography variant="h6" component="p" sx={{ mb: 4, opacity: 0.92, fontWeight: 400 }}>
            Striker is a free browser football manager - claim a team, set your tactics, work the transfer
            market, and simulate your way from the lower divisions to the title.
          </Typography>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ justifyContent: 'center' }}>
            <Button component={RouterLink} to="/login" variant="contained" color="secondary" size="large">
              Get started
            </Button>
            <Button
              component={RouterLink}
              to="/login"
              variant="outlined"
              size="large"
              sx={{ color: 'inherit', borderColor: 'currentColor' }}
            >
              Sign in
            </Button>
          </Stack>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: { xs: 6, sm: 8 } }}>
        <Typography variant="h4" component="h2" sx={{ textAlign: 'center', fontWeight: 600, mb: 1 }}>
          Everything a manager needs
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ textAlign: 'center', mb: 5 }}>
          A full season-long game, running continuously in the background.
        </Typography>
        <Grid container spacing={3}>
          {FEATURES.map(({ icon: Icon, title, description }) => (
            <Grid key={title} size={{ xs: 12, sm: 6, md: 4 }}>
              <Paper variant="outlined" sx={{ p: 3, height: '100%' }}>
                <Icon color="primary" sx={{ fontSize: 36, mb: 1.5 }} />
                <Typography variant="h6" component="h3" sx={{ fontWeight: 600, mb: 1 }}>
                  {title}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {description}
                </Typography>
              </Paper>
            </Grid>
          ))}
        </Grid>
      </Container>

      <Box sx={{ bgcolor: 'action.hover', py: { xs: 6, sm: 8 } }}>
        <Container maxWidth="md">
          <Typography variant="h4" component="h2" sx={{ textAlign: 'center', fontWeight: 600, mb: 5 }}>
            How it works
          </Typography>
          <Grid container spacing={4}>
            {STEPS.map(({ step, title, description }) => (
              <Grid key={step} size={{ xs: 12, sm: 4 }}>
                <Stack spacing={1} sx={{ alignItems: 'center', textAlign: 'center' }}>
                  <Box
                    sx={{
                      width: 40,
                      height: 40,
                      borderRadius: '50%',
                      bgcolor: 'primary.main',
                      color: 'primary.contrastText',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontWeight: 700,
                    }}
                  >
                    {step}
                  </Box>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                    {title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {description}
                  </Typography>
                </Stack>
              </Grid>
            ))}
          </Grid>
        </Container>
      </Box>

      <Container maxWidth="md" sx={{ py: { xs: 6, sm: 8 }, textAlign: 'center' }}>
        <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 2 }}>
          Ready to take charge?
        </Typography>
        <Button component={RouterLink} to="/login" variant="contained" size="large">
          Claim your club
        </Button>
      </Container>
    </Box>
  )
}
