import { useState } from 'react'
import Box from '@mui/material/Box'
import Container from '@mui/material/Container'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useMe } from '../api/hooks'
import { Header } from './Header'
import { NavDrawer } from './NavDrawer'
import { Subheader } from './Subheader'

export function Layout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const { data: loggedUser } = useMe()
  const location = useLocation()

  // A brand-new sign-in has no team yet until they pick a country (see
  // LoggedUser.pendingCountrySelection) - every other route is off-limits until then, so this
  // renders the redirect in place of the whole chrome rather than letting NavDrawer/Subheader
  // mount for a user who has nothing to show yet.
  if (loggedUser?.pendingCountrySelection && location.pathname !== '/onboarding') {
    return <Navigate to="/onboarding" replace />
  }

  return (
    <Box>
      <Header onMenuClick={() => setMobileNavOpen(true)} />
      <Box sx={{ display: 'flex' }}>
        <NavDrawer mobileOpen={mobileNavOpen} onClose={() => setMobileNavOpen(false)} />
        <Box component="main" sx={{ flexGrow: 1, minWidth: 0 }}>
          <Subheader />
          <Container maxWidth="lg" sx={{ py: 3 }}>
            <Outlet />
          </Container>
        </Box>
      </Box>
    </Box>
  )
}
