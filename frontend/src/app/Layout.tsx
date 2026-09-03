import Container from '@mui/material/Container'
import { Outlet } from 'react-router-dom'
import { Header } from './Header'
import { Subheader } from './Subheader'

export function Layout() {
  return (
    <>
      <Header />
      <Subheader />
      <Container maxWidth="lg" sx={{ py: 3 }}>
        <Outlet />
      </Container>
    </>
  )
}
