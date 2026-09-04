import { GoogleLogin } from '@react-oauth/google'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import Paper from '@mui/material/Paper'
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useLogin, useMe } from '../../api/hooks'

export function LoginPage() {
  const { data: loggedUser } = useMe()
  const login = useLogin()
  const navigate = useNavigate()

  return (
    <Container maxWidth="xs" sx={{ py: { xs: 6, sm: 10 } }}>
      <Paper variant="outlined" sx={{ p: 4 }}>
        <Stack spacing={2} sx={{ alignItems: 'center', textAlign: 'center' }}>
          <SportsSoccerIcon color="primary" sx={{ fontSize: 40 }} />
          {loggedUser ? (
            <>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Welcome back, {loggedUser.name}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                You're already signed in.
              </Typography>
              <Button variant="contained" onClick={() => navigate('/dashboard')}>
                Go to dashboard
              </Button>
            </>
          ) : (
            <>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Sign in to Striker
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Use your Google account to claim a club and start managing.
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
                  Sign-in failed. Please try again.
                </Typography>
              )}
            </>
          )}
          <Typography variant="body2">
            <RouterLink to="/">Back to home</RouterLink>
          </Typography>
        </Stack>
      </Paper>
    </Container>
  )
}
