# Proposal

## Why

Each regional cup (European Cup, Copa America, Asian Cup, African Cup) has one team per country,
its tier-1 champion. With two or three countries in a region an edition is one or two rounds, and
the runner-up of a close title race has nothing to play for. Sending the top two of every tier-1
league doubles the field and makes the second place of the top division worth chasing.

## What Changes

- **Two entrants per country.** A fresh country sends the champion and the runner-up of its latest
  tier-1 season (`season_history.final_position` 1 and 2 of that league). When the season has no
  recorded runner-up, the country enters with its champion alone.
- **When an edition starts does not change**: the same fresh / overdue rule, still at least 2
  countries (`MinRegionalCupCountries`). A region with one country still has no cup.
- **Seeded draw.** Byes, when the field is not a power of two, go to champions first (higher
  current league points first, ties at random), then runners-up. The remaining teams are paired
  champion vs runner-up of another country whenever possible. Two teams of the same country never
  meet in round 1 when another draw exists. Byes are spread through the bracket, so two bye
  holders do not meet in round 2.
- **Entries record the position.** `international_cup_entries` gets `final_position` (1 or 2) and
  its primary key becomes `(cup_id, team_id)`, in `init112_international_cup_runners_up.sql`.
- Texts: the International Cup empty state says each country sends its top two.

Bracket sizes: 2 countries → 4 teams, 3 → 6 (2 byes), 4 → 8. An edition gets one more round, which
is one more day.

## Out of scope

- A "1st / 2nd" label on each team in the bracket.
- Changing when an edition starts or the minimum number of countries.
- Editions already running: they keep their entrants; the change applies to editions started after
  it ships.
- Prize money, which is unchanged (the champion still gets `CupChampionBonus`).

## Owner decisions

- Byes go to champions first, and the draw pairs champions with foreign runners-up.
- A region with a single country still has no cup.
- No 1st / 2nd label in the bracket for now.

Open questions for the owner: none.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `international-cup`: entrants are the top two; the first-round draw.

## Impact

- Backend: `resources/sql/init112_international_cup_runners_up.sql` (and the backfill guard in
  `init103_international_cup_entries.sql`); `common/regionalcups.go`; `engine/cups.go`
  (`startRegionalCupIfDue`, `RegionalCupEntrantsPerCountry`), new pure `engine/regionalcupdraw.go`.
- Frontend: `features/cup/InternationalCupPage.tsx` comment, locale `cup.noInternationalCup` and the
  international cup FAQ answer, en/es.
- No new `TRANSACTION_*` reason and no new notification.
