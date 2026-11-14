# Proposal

## Why

A team's history tab (the trophy cabinet) should show every title the club has won: leagues,
domestic cups and international cups. Today it only shows domestic cups partly, and international
cups wrongly:

- When any cup completes, `engine.advanceCup` writes `cup_result = 'Champion'` into
  `season_history` for `cups.season`. For a regional cup (European Cup, Copa America, Asian Cup,
  African Cup) `cups.season` is the edition number, not a league season. So winning European Cup
  edition 2 marks the team "Champion" in its league season 2. That mark looks like a domestic
  cup win, and the World Cup knockout (national teams, `region = 'world'`) does the same.
- The domestic cup chip shows the raw English word "Champion", untranslated, without saying which
  cup it was.

## What Changes

- **International titles in their own column.** `season_history.international_cup_id` points to
  the regional cup edition the team won. It is recorded in the league season the team's country
  was playing when the final was resolved. A team can win the domestic and the international cup
  in the same season, and both show.
- **No more edition numbers as seasons.** Only a domestic cup writes `cup_result`, under its own
  season. A regional cup writes `international_cup_id`. The World Cup knockout writes nothing to
  `season_history` (it has its own roll of honour on the World Cup tab).
- **Migration `init111_team_cup_titles.sql`** adds the column, clears every `cup_result` that no
  domestic cup backs, deletes the rows that existed only because of that, and backfills every
  completed regional cup into its champion's season.
- **API.** Each history entry gains `leagueTier` and `internationalCup`
  (`{cupId, region, edition}` or null).
- **Trophy cabinet.**
  - The domestic cup chip reads "Cup winner" / "Campeón de copa".
  - A new gold chip reads "European Cup winner (Ed. 3)" / "Campeón de la Copa de Europa (Ed. 3)".
    It links to that edition on the International Cup page.
  - A "Honours" summary above the table counts the titles: league titles (tier 1), cups and each
    international cup.
- **International Cup page** accepts `?region=` and `?cup=` so a link can open a past edition.

## Out of scope

- Runners-up or rounds reached in a cup.
- The World Cup in a national team's history.
- Cup prize money (unchanged: a regional cup champion still gets `CupChampionBonus`).

## Owner decisions

- An international title is recorded in the league season in which its final is resolved, not in
  the season the team qualified from.
- The "Honours" summary is included.

Open questions for the owner: none.

## Capabilities

### New Capabilities
- `team-history`: the trophy cabinet: which seasons and titles it shows, where each cup title is
  recorded, the honours summary.

### Modified Capabilities
- `international-cup`: a completed edition is recorded in its champion's history; the page opens a
  given edition from the URL.

## Impact

- Backend: `resources/sql/init111_team_cup_titles.sql`; `engine/cups.go` (`advanceCup`);
  `common/seasonhistory.go` (`RecordSeasonHistoryInternationalCup`, `GetSeasonHistory`).
- Frontend: `features/team/TeamDetailsPage.tsx` (`TrophyCabinet`, `HonoursSummary`),
  `features/cup/InternationalCupPage.tsx`, `api/types.ts`, locales en/es.
- No new `TRANSACTION_*` reason and no new notification.
