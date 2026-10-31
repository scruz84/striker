import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../api/hooks'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { DevLoginWarning } from './DevLoginWarning'
import { BOTTOM_NAV_HEIGHT, BottomNav } from './BottomNav'
import { DetailBackBar } from './DetailBackBar'
import { Header } from './Header'
import { NavDrawer } from './NavDrawer'
import { findDetailRoute } from './navigation'
import { Subheader } from './Subheader'

// The legal texts stay readable by everyone, including a user still waiting at a gate.
const LEGAL_PATHS = ['/privacy', '/terms', '/legal-notice']
// Paths a signed-in user who has not accepted the terms can still open.
const TERMS_GATE_EXEMPT_PATHS = ['/accept-terms', ...LEGAL_PATHS]

export function Layout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const { t } = useTranslation()
  const { data: loggedUser, isError: sessionUnknown, isFetching: sessionRetrying, refetch: retrySession } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const location = useLocation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))

  // Nobody gets anywhere before accepting the current Privacy Policy and Terms (a first sign-in, or
  // after the policy version was raised): /accept-terms comes before the onboarding redirect below.
  // The legal texts themselves stay readable so the screen can link to them.
  if (loggedUser && !loggedUser.termsAccepted && !TERMS_GATE_EXEMPT_PATHS.includes(location.pathname)) {
    return <Navigate to="/accept-terms" replace />
  }

  // A brand-new sign-in has no team yet until they pick a country (see
  // LoggedUser.pendingCountrySelection) - every other route is off-limits until then, so this
  // renders the redirect in place of the whole chrome rather than letting NavDrawer/Subheader
  // mount for a user who has nothing to show yet.
  // The legal pages stay reachable while waiting for a country, and /accept-terms is where a user
  // who must accept first is sent - redirecting those to /onboarding too would bounce between the
  // two gates forever.
  const waitingAtTermsGate = loggedUser && !loggedUser.termsAccepted && TERMS_GATE_EXEMPT_PATHS.includes(location.pathname)
  if (loggedUser?.pendingCountrySelection && location.pathname !== '/onboarding' && !waitingAtTermsGate && !LEGAL_PATHS.includes(location.pathname)) {
    return <Navigate to="/onboarding" replace />
  }

  const showBottomNav = !!loggedUser && !loggedUser.pendingCountrySelection && !isDesktop
  const detailRoute = isDesktop ? undefined : findDetailRoute(location.pathname, userTeam?.team.id)

  return (
    <Box>
      <DevLoginWarning />
      <Header />
      <Box sx={{ display: 'flex' }}>
        <NavDrawer mobileOpen={mobileNavOpen} onClose={() => setMobileNavOpen(false)} />
        <Box
          component="main"
          sx={{
            flexGrow: 1,
            minWidth: 0,
            // Keep the last content clear of the fixed bottom bar (and the home indicator).
            pb: showBottomNav ? `calc(${BOTTOM_NAV_HEIGHT}px + env(safe-area-inset-bottom))` : 0,
          }}
        >
          {detailRoute ? <DetailBackBar route={detailRoute} /> : <Subheader />}
          <Container maxWidth="lg" sx={{ py: { xs: 2, md: 3 } }}>
            {/* The session check failed (no network, server down) and nothing is known about the
                user yet - say so instead of showing the guest UI; useMe keeps retrying. */}
            {sessionUnknown && loggedUser === undefined && (
              <Alert
                severity="warning"
                sx={{ mb: 2 }}
                action={
                  <Button color="inherit" size="small" disabled={sessionRetrying} onClick={() => void retrySession()}>
                    {t('common.tryAgain')}
                  </Button>
                }
              >
                {t('common.serverUnreachable')}
              </Alert>
            )}
            {/* Keyed by path so navigating to a different page always gets a fresh boundary
                instead of staying stuck on a previous page's error fallback. */}
            <ErrorBoundary key={location.pathname}>
              <Outlet />
            </ErrorBoundary>
          </Container>
        </Box>
      </Box>
      {showBottomNav && <BottomNav onMore={() => setMobileNavOpen(true)} />}
    </Box>
  )
}
