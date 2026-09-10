import { GoogleLogin } from '@react-oauth/google'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import Divider from '@mui/material/Divider'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useDevLogin, useDevLoginUsers, useLogin, useMe } from '../../api/hooks'

export function LoginPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const login = useLogin()
  const devLogin = useDevLogin()
  const { data: devUsers } = useDevLoginUsers()
  const navigate = useNavigate()

  return (
    <Container maxWidth="xs" sx={{ py: { xs: 6, sm: 10 } }}>
      <Paper variant="outlined" sx={{ p: 4 }}>
        <Stack spacing={2} sx={{ alignItems: 'center', textAlign: 'center' }}>
          <SportsSoccerIcon color="primary" sx={{ fontSize: 40 }} />
          {loggedUser ? (
            <>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                {t('login.welcomeBack', { name: loggedUser.name })}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {t('login.alreadySignedIn')}
              </Typography>
              <Button variant="contained" onClick={() => navigate('/dashboard')}>
                {t('login.goToDashboard')}
              </Button>
            </>
          ) : (
            <>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                {t('login.title')}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {t('login.subtitle')}
              </Typography>
              <Box>
                <GoogleLogin
                  onSuccess={(credentialResponse) => {
                    if (!credentialResponse.credential) {
                      return
                    }
                    login.mutate(credentialResponse.credential, {
                      onSuccess: () => navigate('/dashboard'),
                    })
                  }}
                  onError={() => {
                    console.error('Login failed')
                  }}
                />
              </Box>
              {login.isError && (
                <Typography variant="body2" color="error">
                  {t('login.signInFailed')}
                </Typography>
              )}
              {!!devUsers?.length && (
                <>
                  <Divider flexItem sx={{ width: '100%' }}>
                    {t('login.devLogin.divider')}
                  </Divider>
                  <Stack spacing={1} sx={{ width: '100%' }}>
                    {devUsers.map((devUser) => (
                      <Button
                        key={devUser.email}
                        variant="outlined"
                        size="small"
                        disabled={devLogin.isPending}
                        onClick={() =>
                          devLogin.mutate(devUser.email, { onSuccess: () => navigate('/dashboard') })
                        }
                      >
                        {devUser.name}
                        {devUser.isAdmin ? ` (${t('login.devLogin.admin')})` : ''}
                      </Button>
                    ))}
                  </Stack>
                  {devLogin.isError && (
                    <Typography variant="body2" color="error">
                      {t('login.signInFailed')}
                    </Typography>
                  )}
                </>
              )}
            </>
          )}
          <Typography variant="body2">
            <Link component={RouterLink} to="/" underline="hover">
              {t('login.backToHome')}
            </Link>
          </Typography>
        </Stack>
      </Paper>
    </Container>
  )
}
