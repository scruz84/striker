import createCache from '@emotion/cache'
import { CacheProvider } from '@emotion/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { getGoogleOAuthClientId } from '../config'
import { ConfirmProvider } from '../components/ConfirmDialog'
import { AdminPage } from '../features/admin/AdminPage'
import { LoginPage } from '../features/auth/LoginPage'
import { DashboardPage } from '../features/dashboard/DashboardPage'
import { CupPage } from '../features/cup/CupPage'
import { CupTieDetailsPage } from '../features/cup/CupTieDetailsPage'
import { DivisionDetailsPage } from '../features/division/DivisionDetailsPage'
import { FinancesPage } from '../features/finances/FinancesPage'
import { FriendliesPage } from '../features/friendlies/FriendliesPage'
import { FriendlyMatchPage } from '../features/friendlies/FriendlyMatchPage'
import { GroupsPage } from '../features/groups/GroupsPage'
import { HomePage } from '../features/home/HomePage'
import { LeaguesPage } from '../features/leagues/LeaguesPage'
import { LineupEditorPage } from '../features/lineup/LineupEditorPage'
import { MatchDetailsPage } from '../features/match/MatchDetailsPage'
import { SelectCountryPage } from '../features/onboarding/SelectCountryPage'
import { LatestEvolutionsPage } from '../features/player/LatestEvolutionsPage'
import { PlayerDetailsPage } from '../features/player/PlayerDetailsPage'
import { TeamDetailsPage } from '../features/team/TeamDetailsPage'
import { TeamMatchesPage } from '../features/team/TeamMatchesPage'
import { TransferMarketPage } from '../features/transfers/TransferMarketPage'
import { Layout } from './Layout'
import { queryClient } from './queryClient'
import { ThemeModeProvider } from './ThemeModeProvider'

const emotionCache = createCache({ key: 'css' })

function App() {
  return (
    <GoogleOAuthProvider clientId={getGoogleOAuthClientId()}>
      <QueryClientProvider client={queryClient}>
        <CacheProvider value={emotionCache}>
          <ThemeModeProvider>
            <ConfirmProvider>
              <BrowserRouter>
                <Routes>
                  <Route element={<Layout />}>
                    <Route path="/" element={<HomePage />} />
                    <Route path="/dashboard" element={<DashboardPage />} />
                    <Route path="/dashboard/lineup" element={<LineupEditorPage />} />
                    <Route path="/dashboard/finances" element={<FinancesPage />} />
                    <Route path="/transfers" element={<TransferMarketPage />} />
                    <Route path="/cup" element={<CupPage />} />
                    <Route path="/cup/ties/:id" element={<CupTieDetailsPage />} />
                    <Route path="/friendlies" element={<FriendliesPage />} />
                    <Route path="/friendlies/:id" element={<FriendlyMatchPage />} />
                    <Route path="/groups" element={<GroupsPage />} />
                    <Route path="/admin" element={<AdminPage />} />
                    <Route path="/leagues" element={<LeaguesPage />} />
                    <Route path="/division/:id" element={<DivisionDetailsPage />} />
                    <Route path="/match/:id" element={<MatchDetailsPage />} />
                    <Route path="/team/:id" element={<TeamDetailsPage />} />
                    <Route path="/team/:id/matches" element={<TeamMatchesPage />} />
                    <Route path="/player/:id" element={<PlayerDetailsPage />} />
                    <Route path="/players/evolutions" element={<LatestEvolutionsPage />} />
                    <Route path="/login" element={<LoginPage />} />
                    <Route path="/onboarding" element={<SelectCountryPage />} />
                    <Route path="*" element={<HomePage />} />
                  </Route>
                </Routes>
              </BrowserRouter>
            </ConfirmProvider>
          </ThemeModeProvider>
        </CacheProvider>
      </QueryClientProvider>
    </GoogleOAuthProvider>
  )
}

export default App
