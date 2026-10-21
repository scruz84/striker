import createCache from '@emotion/cache'
import { CacheProvider } from '@emotion/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { createBrowserRouter, RouterProvider, type RouteObject } from 'react-router-dom'
import { getGoogleOAuthClientId } from '../config'
import { ConfirmProvider } from '../components/ConfirmDialog'
import { NotifyProvider } from '../components/NotifyProvider'
import { AdminPage } from '../features/admin/AdminPage'
import { LoginPage } from '../features/auth/LoginPage'
import { DashboardPage } from '../features/dashboard/DashboardPage'
import { CupPage } from '../features/cup/CupPage'
import { CupTieDetailsPage } from '../features/cup/CupTieDetailsPage'
import { InternationalCupPage } from '../features/cup/InternationalCupPage'
import { DivisionDetailsPage } from '../features/division/DivisionDetailsPage'
import { FinancesPage } from '../features/finances/FinancesPage'
import { FriendliesPage } from '../features/friendlies/FriendliesPage'
import { FriendlyMatchPage } from '../features/friendlies/FriendlyMatchPage'
import { GroupsPage } from '../features/groups/GroupsPage'
import { HelpPage } from '../features/help/HelpPage'
import { HomePage } from '../features/home/HomePage'
import { LeaguesPage } from '../features/leagues/LeaguesPage'
import { LineupEditorPage } from '../features/lineup/LineupEditorPage'
import { MatchDetailsPage } from '../features/match/MatchDetailsPage'
import { NationalTeamsPage } from '../features/nationalteams/NationalTeamsPage'
import { SelectCountryPage } from '../features/onboarding/SelectCountryPage'
import { LatestEvolutionsPage } from '../features/player/LatestEvolutionsPage'
import { PlayerDetailsPage } from '../features/player/PlayerDetailsPage'
import { SearchPage } from '../features/search/SearchPage'
import { TeamDetailsPage } from '../features/team/TeamDetailsPage'
import { TeamMatchesPage } from '../features/team/TeamMatchesPage'
import { TransferMarketPage } from '../features/transfers/TransferMarketPage'
import { Layout } from './Layout'
import { queryClient } from './queryClient'
import { ThemeModeProvider } from './ThemeModeProvider'

const emotionCache = createCache({ key: 'css' })

// A data router (rather than <BrowserRouter>) because pages with unsaved edits - the lineup editor
// - need useBlocker to confirm before navigating away.
const routes: RouteObject[] = [
  {
    element: <Layout />,
    children: [
      { path: '/', element: <HomePage /> },
      { path: '/dashboard', element: <DashboardPage /> },
      { path: '/dashboard/lineup', element: <LineupEditorPage /> },
      { path: '/dashboard/finances', element: <FinancesPage /> },
      { path: '/transfers', element: <TransferMarketPage /> },
      { path: '/cup', element: <CupPage /> },
      { path: '/cup/ties/:id', element: <CupTieDetailsPage /> },
      { path: '/international-cup', element: <InternationalCupPage /> },
      { path: '/friendlies', element: <FriendliesPage /> },
      { path: '/friendlies/:id', element: <FriendlyMatchPage /> },
      { path: '/national-teams', element: <NationalTeamsPage /> },
      { path: '/groups', element: <GroupsPage /> },
      { path: '/help', element: <HelpPage /> },
      { path: '/admin', element: <AdminPage /> },
      { path: '/leagues', element: <LeaguesPage /> },
      { path: '/division/:id', element: <DivisionDetailsPage /> },
      { path: '/match/:id', element: <MatchDetailsPage /> },
      { path: '/team/:id', element: <TeamDetailsPage /> },
      { path: '/team/:id/matches', element: <TeamMatchesPage /> },
      { path: '/player/:id', element: <PlayerDetailsPage /> },
      { path: '/players/evolutions', element: <LatestEvolutionsPage /> },
      { path: '/search', element: <SearchPage /> },
      { path: '/login', element: <LoginPage /> },
      { path: '/onboarding', element: <SelectCountryPage /> },
      { path: '*', element: <HomePage /> },
    ],
  },
]
const router = createBrowserRouter(routes)

function App() {
  return (
    <GoogleOAuthProvider clientId={getGoogleOAuthClientId()}>
      <QueryClientProvider client={queryClient}>
        <CacheProvider value={emotionCache}>
          <ThemeModeProvider>
            <ConfirmProvider>
              <NotifyProvider>
                <RouterProvider router={router} />
              </NotifyProvider>
            </ConfirmProvider>
          </ThemeModeProvider>
        </CacheProvider>
      </QueryClientProvider>
    </GoogleOAuthProvider>
  )
}

export default App
