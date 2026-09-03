import createCache from '@emotion/cache'
import { CacheProvider } from '@emotion/react'
import CssBaseline from '@mui/material/CssBaseline'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClientProvider } from '@tanstack/react-query'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { getGoogleOAuthClientId } from '../config'
import { LoginPage } from '../features/auth/LoginPage'
import { DashboardPage } from '../features/dashboard/DashboardPage'
import { DivisionDetailsPage } from '../features/division/DivisionDetailsPage'
import { FinancesPage } from '../features/finances/FinancesPage'
import { HomePage } from '../features/home/HomePage'
import { LineupEditorPage } from '../features/lineup/LineupEditorPage'
import { MatchDetailsPage } from '../features/match/MatchDetailsPage'
import { TeamDetailsPage } from '../features/team/TeamDetailsPage'
import { TransferMarketPage } from '../features/transfers/TransferMarketPage'
import { lightTheme } from '../theme'
import { Layout } from './Layout'
import { queryClient } from './queryClient'

const emotionCache = createCache({ key: 'css' })

function App() {
  return (
    <GoogleOAuthProvider clientId={getGoogleOAuthClientId()}>
      <QueryClientProvider client={queryClient}>
        <CacheProvider value={emotionCache}>
          <ThemeProvider theme={lightTheme}>
            <CssBaseline enableColorScheme />
            <BrowserRouter>
              <Routes>
                <Route element={<Layout />}>
                  <Route path="/" element={<HomePage />} />
                  <Route path="/dashboard" element={<DashboardPage />} />
                  <Route path="/dashboard/lineup" element={<LineupEditorPage />} />
                  <Route path="/dashboard/finances" element={<FinancesPage />} />
                  <Route path="/transfers" element={<TransferMarketPage />} />
                  <Route path="/division/:id" element={<DivisionDetailsPage />} />
                  <Route path="/match/:id" element={<MatchDetailsPage />} />
                  <Route path="/team/:id" element={<TeamDetailsPage />} />
                  <Route path="/login" element={<LoginPage />} />
                  <Route path="*" element={<HomePage />} />
                </Route>
              </Routes>
            </BrowserRouter>
          </ThemeProvider>
        </CacheProvider>
      </QueryClientProvider>
    </GoogleOAuthProvider>
  )
}

export default App
