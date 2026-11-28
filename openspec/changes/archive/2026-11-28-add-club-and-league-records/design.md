# Design

## On demand, cached

No new table: every record is a query over what is stored, plus Go for the streaks. A club's
official fixtures (league matches and cup ties, `recordFixturesSql`) are a few dozen per season, so
`ClubMatchRecords` walks them in kickoff order in memory. The results are cached in the object cache
(`records:club:<id>`, `records:league:<id>`) for `RecordsCacheTTL` = 10 minutes: a record can show
up to 10 minutes late, never early, since the query itself applies `LiveBroadcastCutoff()`.

No migration, TRANSACTION_* reason or notification key.

## Tie-breaks

- Biggest win/defeat: widest margin, then most goals scored (conceded for a defeat), then earliest.
- Highest scoring and attendance: most goals / biggest crowd, then earliest.
- Streaks: league matches only; a strictly longer run replaces the best, so the first longest wins;
  ongoing when it reaches the latest league match.
- Best finish: highest tier (tier 1 first), then best position, then earliest season.
- Most titles: titles, then the latest title, then team id.

## Code map

- `backend/common/records.go` (`GetClubRecords`, `ClubMatchRecords`, `GetLeagueRecords`, `LeagueTitleCounts`)
- `backend/handlers/statsHandlers.go` `TeamRecordsHandler`, `LeagueRecordsHandler`
- `frontend/src/features/stats/ClubRecordsSection.tsx`, `LeagueRecordsPage.tsx`, `RecordRows.tsx`
- `frontend/src/features/team/TeamDetailsPage.tsx` (history tab), `features/division/DivisionDetailsPage.tsx` (`LeagueHistoryButton`)
