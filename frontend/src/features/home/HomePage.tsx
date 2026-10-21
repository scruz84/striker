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
import { useTranslation } from 'react-i18next'
import { useMe } from '../../api/hooks'

const FEATURES = [
  { icon: SportsSoccerIcon, key: 'matchSimulation' },
  { icon: SwapHorizIcon, key: 'transfersLoans' },
  { icon: EmojiEventsIcon, key: 'leaguesCups' },
  { icon: StadiumIcon, key: 'clubFinances' },
  { icon: TrendingUpIcon, key: 'squadDevelopment' },
  { icon: GroupsIcon, key: 'friendGroups' },
] as const

const STEPS = ['claim', 'manage', 'climb'] as const

export function HomePage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const navigate = useNavigate()

  useEffect(() => {
    if (loggedUser) {
      navigate('/dashboard')
    }
  }, [loggedUser, navigate])

  // The landing page (with its sign-in buttons) only once the server has said there is no session:
  // not while that is loading or the server is unreachable (see useMe).
  if (loggedUser !== null) {
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
            {t('home.hero.title')}
          </Typography>
          <Typography variant="h6" component="p" sx={{ mb: 4, opacity: 0.92, fontWeight: 400 }}>
            {t('home.hero.subtitle')}
          </Typography>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ justifyContent: 'center' }}>
            <Button component={RouterLink} to="/login" variant="contained" color="secondary" size="large">
              {t('home.hero.getStarted')}
            </Button>
            <Button
              component={RouterLink}
              to="/login"
              variant="outlined"
              size="large"
              sx={{ color: 'inherit', borderColor: 'currentColor' }}
            >
              {t('home.hero.signIn')}
            </Button>
          </Stack>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: { xs: 6, sm: 8 } }}>
        <Typography variant="h4" component="h2" sx={{ textAlign: 'center', fontWeight: 600, mb: 1 }}>
          {t('home.features.title')}
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ textAlign: 'center', mb: 5 }}>
          {t('home.features.subtitle')}
        </Typography>
        <Grid container spacing={3}>
          {FEATURES.map(({ icon: Icon, key }) => (
            <Grid key={key} size={{ xs: 12, sm: 6, md: 4 }}>
              <Paper variant="outlined" sx={{ p: 3, height: '100%' }}>
                <Icon color="primary" sx={{ fontSize: 36, mb: 1.5 }} />
                <Typography variant="h6" component="h3" sx={{ fontWeight: 600, mb: 1 }}>
                  {t(`home.features.${key}.title`)}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {t(`home.features.${key}.description`)}
                </Typography>
              </Paper>
            </Grid>
          ))}
        </Grid>
      </Container>

      <Box sx={{ bgcolor: 'action.hover', py: { xs: 6, sm: 8 } }}>
        <Container maxWidth="md">
          <Typography variant="h4" component="h2" sx={{ textAlign: 'center', fontWeight: 600, mb: 5 }}>
            {t('home.howItWorks.title')}
          </Typography>
          <Grid container spacing={4}>
            {STEPS.map((key, index) => (
              <Grid key={key} size={{ xs: 12, sm: 4 }}>
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
                    {index + 1}
                  </Box>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                    {t(`home.howItWorks.steps.${key}.title`)}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {t(`home.howItWorks.steps.${key}.description`)}
                  </Typography>
                </Stack>
              </Grid>
            ))}
          </Grid>
        </Container>
      </Box>

      <Container maxWidth="md" sx={{ py: { xs: 6, sm: 8 }, textAlign: 'center' }}>
        <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 2 }}>
          {t('home.readyTitle')}
        </Typography>
        <Button component={RouterLink} to="/login" variant="contained" size="large">
          {t('home.claimClub')}
        </Button>
      </Container>
    </Box>
  )
}
