# Tasks

## 1. Rest rule for friendly pairing (bug: same country twice on one evening)

- [ ] 1.1 Add `NextLocalSaturdayKickoffAfter(country, hour, minute, earliest)` in `common/timezones.go` and make `NextLocalSaturdayKickoff` delegate to it with `time.Now()`; verify with table tests across UTC-10, UTC+0 and UTC+13 (earliest on a Saturday before/after 21:00 local, earliest on a Sunday)
- [ ] 1.2 Add `getLatestNationalTeamKickoffsStmtSql` (MAX kickoff per country over `friendly_matches` and `cup_ties`) and use it in `scheduleNationalTeamFriendliesLocked` with `earliest = max(now, last(home)+6d, last(away)+6d)`; verify with an integration test reproducing the São Tomé/Nigeria case (one friendly simulated 14 min early, pass runs) asserting the new fixture is 7 days out
- [ ] 1.3 Extend the existing concurrent pairing integration test (two replicas) to assert no country has two national-team kickoffs within 6 days; verify `go test ./common -run NationalTeam`
- [ ] 1.4 Update the `national-teams` spec Friendly pairing Code map (`NextLocalSaturdayKickoffAfter`, `getLatestNationalTeamKickoffsStmtSql`)

## 2. Broadcast-safe national match model (bug: live box spoils the result)

- [ ] 2.1 Refactor `enrichSuggestedLiveMatch` so its minute/score-as-of-now logic is a reusable `liveScoreAsOfNow(dbCtx, kind, id, kickoff)`; verify the existing livematches tests still pass unchanged
- [ ] 2.2 Add the `NationalMatch` type and `BuildNationalMatchStatus` in `common/nationalmatches.go`; verify unit tests for scheduled (no score), simulated-before-kickoff (live, minute 0, 0-0), live mid-match (score as of minute, not final) and finished (final score)
- [ ] 2.3 Add `GetNationalTeamRound` (union of national friendlies, World Cup group matches and active knockout ties; unplayed or kicked off in the last 24 h; ordered by kickoff; uncapped); verify an integration test with 3 kinds of rows and a live one returning its as-of-minute score
- [ ] 2.4 Add `GetNationalTeamResults(page, country)` (finished broadcasts only, 25 per page plus `hasMore`, optional country filter on home/away team id); verify integration tests for paging, country filter and a live match being excluded
- [ ] 2.5 Add `GetMyPlayersInNationalMatches(teamId, matchIds, tieIds)` counting the owner's players in saved national lineups; verify an integration test where 2 of a club's players are in a saved XI and a fixture without a lineup returns 0
- [ ] 2.6 Add `NationalTeamRoundHandler` (`GET /v1/national-teams/round`, fills `myPlayers` when the user owns a club) and `NationalTeamResultsHandler` (`GET /v1/national-teams/results`, 400 on a bad page/country); remove the `friendlies/upcoming|recent` handlers, routes and `nationalTeamFriendliesLimit`; verify handler tests for both endpoints

## 3. Broadcast-safe World Cup (bug: group table moves during the broadcast)

- [ ] 3.1 Make `GetWorldCupGroups` derive standings from group matches that are played with kickoff ≤ `LiveBroadcastCutoff()` and rank them with `RankGroupStandings`, returning group matches through `BuildNationalMatchStatus`; verify an integration test where a just-simulated 2-0 leaves the table unchanged and, after moving the kickoff back 2 h, the table equals `world_cup_group_teams`
- [ ] 3.2 Add `GetWorldCupHistory` and `GET /v1/national-teams/world-cups`; verify an integration test with two completed World Cups and one in progress
- [ ] 3.3 Add `nextWorldCupEstimate` to the current World Cup response (none active, ≥ `MinWorldCupCountries` active countries, last check + 60 d × cycles to the next multiple of `WorldCupFrequencySeasons`); verify unit tests for counter 6 → +120 days, counter 8 with no active World Cup → +240 days, and too few countries → absent
- [ ] 3.4 Update the `world-cup` spec Code map (derived standings, history, estimate)

## 4. Frontend: data layer

- [ ] 4.1 Add `NationalMatch`, `NationalTeamResultsPage`, `WorldCupSummary` and `nextWorldCupEstimate` to `api/types.ts`; replace `useUpcomingNationalTeamFriendlies`/`useRecentNationalTeamFriendlies` with `useNationalTeamRound` (30 s poll while a row is live or due within 15 min, else 5 min), `useNationalTeamResults` (infinite query) and `useWorldCupHistory` in `api/hooks.ts`; verify `npx tsc --noEmit`
- [ ] 4.2 Unit-test the round grouping helper (rows grouped by local kickoff, live first) and the filter helper (localized name search, region, "only my players"); verify `npx vitest run src/features/nationalteams`

## 5. Frontend: national teams page

- [ ] 5.1 Build `NationalMatchRow` (two-line compact row: competition chip + kickoff or live minute; flags, ellipsized names, score; 44 px tap target; "N of your players" badge); verify a component test rendering "Irlanda del Norte" vs "Santo Tomé y Príncipe" with `noWrap` names and a live row showing the minute and the as-of-minute score
- [ ] 5.2 Split `NationalTeamsPage.tsx` into tabs kept in `?tab=round|results|world-cup` with `RoundTab`, `ResultsTab` (load more) and `NationalMatchFilters`; remove `LiveNationalTeamMatches`, `FriendlyMatchList` and their client-side `mightStillBeLive` masking; verify a page test for tab-from-URL and the round tab listing more than 10 matches
- [ ] 5.3 Rebuild `WorldCupTab`: full group table (P W D L GF GA GD Pts, horizontally scrollable on phones), fixtures grouped by matchday with dates, roll of honour, next World Cup estimate in the empty state; verify a component test for the table columns and the estimate text
- [ ] 5.4 Add the `matches` tab to `TeamDetailsPage` for national teams (next fixture, last 10 results, last-5 form strip); verify a test that a national team shows the tab and a club does not
- [ ] 5.5 Add every new string to `frontend/src/i18n/locales/en.json` and `es.json` (tabs, filters, region chips, badge with plural, statuses, matchday, roll of honour, next World Cup estimate, form letters); verify the locale key parity test
- [ ] 5.6 Check the page at 360 px wide in the browser (Playwright screenshot of the round and World Cup tabs with seeded data); verify no horizontal page scroll and no team name on more than one line

## 6. Specs and verification

- [ ] 6.1 Update the `national-teams` spec Code map (new files, endpoints, frontend components) and the frontend part of the `world-cup` Code map; verify `openspec validate improve-national-teams-page --strict`
- [ ] 6.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass

## Workflow follow-up

- Archive the change and sync the `national-teams` and `world-cup` specs once merged.
