import { useState } from 'react'
import Box from '@mui/material/Box'
import Container from '@mui/material/Container'
import { Outlet } from 'react-router-dom'
import { Header } from './Header'
import { NavDrawer } from './NavDrawer'
import { Subheader } from './Subheader'

export function Layout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

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
