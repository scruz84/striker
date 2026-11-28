import createCache from '@emotion/cache'
import { CacheProvider } from '@emotion/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { createBrowserRouter, RouterProvider, type RouteObject } from 'react-router-dom'
import { getGoogleOAuthClientId, getLegalConfig } from '../config'
import { ConfirmProvider } from '../components/ConfirmDialog'
import { NotifyProvider } from '../components/NotifyProvider'
import { HomePage } from '../features/home/HomePage'
import { Layout } from './Layout'
import { queryClient } from './queryClient'
import { ThemeModeProvider } from './ThemeModeProvider'

const emotionCache = createCache({ key: 'css' })

// Every page except the landing page is loaded on demand (the route's `lazy`), so the first load
// of any page downloads only the code it needs instead of one 1.4 MB bundle holding the whole app.
// A data router (rather than <BrowserRouter>) because pages with unsaved edits - the lineup editor
// - need useBlocker to confirm before navigating away.
const routes: RouteObject[] = [
  {
    element: <Layout />,
    children: [
      { path: '/', element: <HomePage /> },
      { path: '/dashboard', lazy: () => import('../features/dashboard/DashboardPage').then((m) => ({ Component: m.DashboardPage })) },
      { path: '/dashboard/lineup', lazy: () => import('../features/lineup/LineupEditorPage').then((m) => ({ Component: m.LineupEditorPage })) },
      { path: '/dashboard/finances', lazy: () => import('../features/finances/FinancesPage').then((m) => ({ Component: m.FinancesPage })) },
      { path: '/transfers', lazy: () => import('../features/transfers/TransferMarketPage').then((m) => ({ Component: m.TransferMarketPage })) },
      { path: '/cup', lazy: () => import('../features/cup/CupPage').then((m) => ({ Component: m.CupPage })) },
      { path: '/cup/ties/:id', lazy: () => import('../features/cup/CupTieDetailsPage').then((m) => ({ Component: m.CupTieDetailsPage })) },
      { path: '/international-cup', lazy: () => import('../features/cup/InternationalCupPage').then((m) => ({ Component: m.InternationalCupPage })) },
      { path: '/friendlies', lazy: () => import('../features/friendlies/FriendliesPage').then((m) => ({ Component: m.FriendliesPage })) },
      { path: '/friendlies/:id', lazy: () => import('../features/friendlies/FriendlyMatchPage').then((m) => ({ Component: m.FriendlyMatchPage })) },
      { path: '/national-teams', lazy: () => import('../features/nationalteams/NationalTeamsPage').then((m) => ({ Component: m.NationalTeamsPage })) },
      { path: '/groups', lazy: () => import('../features/groups/GroupsPage').then((m) => ({ Component: m.GroupsPage })) },
      { path: '/help', lazy: () => import('../features/help/HelpPage').then((m) => ({ Component: m.HelpPage })) },
      { path: '/admin', lazy: () => import('../features/admin/AdminPage').then((m) => ({ Component: m.AdminPage })) },
      { path: '/leagues', lazy: () => import('../features/leagues/LeaguesPage').then((m) => ({ Component: m.LeaguesPage })) },
      { path: '/division/:id', lazy: () => import('../features/division/DivisionDetailsPage').then((m) => ({ Component: m.DivisionDetailsPage })) },
      { path: '/division/:id/records', lazy: () => import('../features/stats/LeagueRecordsPage').then((m) => ({ Component: m.LeagueRecordsPage })) },
      { path: '/match/:id', lazy: () => import('../features/match/MatchDetailsPage').then((m) => ({ Component: m.MatchDetailsPage })) },
      { path: '/team/:id', lazy: () => import('../features/team/TeamDetailsPage').then((m) => ({ Component: m.TeamDetailsPage })) },
      { path: '/team/:id/matches', lazy: () => import('../features/team/TeamMatchesPage').then((m) => ({ Component: m.TeamMatchesPage })) },
      { path: '/player/:id', lazy: () => import('../features/player/PlayerDetailsPage').then((m) => ({ Component: m.PlayerDetailsPage })) },
      { path: '/players/evolutions', lazy: () => import('../features/player/LatestEvolutionsPage').then((m) => ({ Component: m.LatestEvolutionsPage })) },
      { path: '/search', lazy: () => import('../features/search/SearchPage').then((m) => ({ Component: m.SearchPage })) },
      { path: '/account', lazy: () => import('../features/account/AccountPage').then((m) => ({ Component: m.AccountPage })) },
      { path: '/privacy', lazy: () => import('../features/legal/PrivacyPage').then((m) => ({ Component: m.PrivacyPage })) },
      { path: '/terms', lazy: () => import('../features/legal/TermsPage').then((m) => ({ Component: m.TermsPage })) },
      { path: '/accept-terms', lazy: () => import('../features/legal/AcceptTermsPage').then((m) => ({ Component: m.AcceptTermsPage })) },
      // The legal notice only exists on deployments that enabled it (LEGAL_NOTICE_ENABLED=true);
      // otherwise /legal-notice falls through to the home page like any unknown path.
      ...(getLegalConfig().noticeEnabled
        ? [{ path: '/legal-notice', lazy: () => import('../features/legal/LegalNoticePage').then((m) => ({ Component: m.LegalNoticePage })) }]
        : []),
      { path: '/login', lazy: () => import('../features/auth/LoginPage').then((m) => ({ Component: m.LoginPage })) },
      { path: '/onboarding', lazy: () => import('../features/onboarding/SelectCountryPage').then((m) => ({ Component: m.SelectCountryPage })) },
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
