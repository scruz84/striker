# Proposal

## Why

There are three match pages today, one per competition, and they no longer show the same thing:

- `/match/:id` (league): `MatchDetailsPage.tsx`
- `/cup/ties/:id` (domestic cup, international cup and World Cup knockout): `CupTieDetailsPage.tsx`
- `/friendlies/:id` (club friendlies, national-team friendlies and World Cup group matches): `FriendlyMatchPage.tsx`

The cup and friendly pages are copies of the league page that have drifted apart. Compared with
the league viewer, they lack the following:

| What the league viewer shows | Cup tie | Friendly / World Cup group |
|---|---|---|
| Zone duel ("Duelo por zonas") on a finished match | **missing**: cup ties don't store per-minute instants | yes |
| Weather, crowd and referee lines in the chronicle | **missing**: not stored | **missing**: not stored |
| Stadium name and stadium-entry banner | missing | missing |
| Scorers under the live scoreboard | missing | missing |
| Competition context in the subtitle ("Jornada 3 de 38 · fecha · ☀️") | date only | date only |
| Goalkeeper counted in the zone duel while live | no (0) | no (0) |
| Lineup editor shown only to a participant, before kickoff | also shown during the live match | shown to **any** logged-in user, also during the live match |

Any future improvement to the league page would have to be made three times. The zone duel also
shows only each side's share of a duel (for example 62% / 38%). Each team's actual strength in
that zone appears only in a tooltip, so a manager can't see at a glance how strong each line is.

## What Changes

- **One match viewer.** A single `MatchViewer` component renders every match: the pre-kickoff
  view with the lineup editor, the live broadcast and the finished-match tabs (Crónica, Notas,
  Tácticas, plus the head-to-head history). The three routes stay as they are, because
  notifications, brackets, team pages and the World Cup page already link to them. Each route
  becomes a thin adapter that loads its own endpoint and maps the result onto one shared view
  model. Cup-only extras, such as the penalty-shootout line and the "bye" card, become optional
  slots of the viewer instead of a separate page.
- **Cup ties store what league matches store.** A cup tie stores its per-minute zone instants and
  its report flavor (weather, attendance, referee and two assistants). Its weather is drawn once
  and drives both its attendance and its report.
- **Friendlies store report flavor too.** This covers weather, attendance, referee and assistants.
  Friendlies already store their instants.
- **Detail endpoints return the same fields.** Cup-tie and friendly detail responses add the
  home stadium name, the flavor fields, the instants (for cup ties) and a competition label
  (cup name and round, "Friendly", or World Cup group and matchday). The viewer can then build
  the same subtitle for any match.
- **The zone duel shows each team's strength.** Each half of a duel cell keeps its percentage as
  the main figure and adds that team's raw zone strength beneath it, rounded to a whole number.
  The tooltip keeps the matchup caption and both values to one decimal. While live, the panel
  counts each side's starting goalkeeper the same way the finished view does, so the numbers
  don't jump at full time.
- **Lineup-editor rule unified.** The editor (or the "lineup locked" notice) is shown only to an
  owner whose team plays the match, and only before kickoff, as on the league page today.

Out of scope:
- Merging the three backend tables (`matches`, `cup_ties`, `friendly_matches`) or their REST
  routes into one. The viewer is unified at the view-model level only.
- Backfilling instants or flavor for cup ties and friendlies played before the migration. Those
  matches keep rendering without the zone duel or the flavor lines, as old league matches already
  do.
- Any change to how a match is simulated, or to attendance and revenue amounts. This change only
  persists and displays values the engine already computes.
- Showing the league standing position on non-league scoreboards. It has no meaning outside a
  league.

Open questions for the owner:
1. **Where the strength number goes.** The proposal puts the team's strength under its
   percentage, inside the same cell (two lines: `62%` bold, `14` small). The alternative is a
   number next to the percentage (`62% · 14`). That fits on a desktop, but on a 360 px phone each
   half-cell is about 50 px wide, so it would have to wrap or shrink the text below a readable
   size. The two-line layout is the recommended default.
2. **The World Cup group label.** Should it read "Mundial · Grupo A · Jornada 2", or is "Mundial
   · Grupo A" enough? The proposal includes the matchday because the league subtitle shows one.

## Capabilities

### New Capabilities

- `match-viewer`: the single page that presents any match (league, cup tie or friendly) before
  kickoff, live and after full time. It owns what that page shows: scoreboard and subtitle,
  chronicle, ratings, the zone duel with per-team strengths, lineups, head-to-head history,
  lineup-editor visibility and the cup-only extras.

### Modified Capabilities

- `match-simulation`: post-match processing now stores per-minute instants and report flavor for
  cup ties and friendlies too, not only for league matches.
- `socios`: the "weather drawn once" rule extends from league matches to every match with a
  crowd (cup ties and friendlies).
- `world-cup`: a group match renders through the shared match viewer and carries its group and
  matchday label. The current wording says it renders through "the normal friendly page".

## Impact

- **Backend.**
  - New migration `resources/sql/init83_match_viewer_parity.sql`: a `cup_tie_instants` table,
    plus flavor columns on `cup_ties` and `friendly_matches`.
  - `engine/cups.go` (`simulateCupTie`), `engine/friendlies.go` (`simulateFriendly`) and
    `engine/stadium.go` (`creditFriendlyTicketRevenue` takes the weather as a parameter).
  - `common/cups.go` (`SetCupTiePlayed`, `CupTieDetails`, `GetCupTieDetails`) and
    `common/friendlies.go` (`FriendlyMatch`, `EnrichPlayedFriendly`, `SetFriendlyPlayed`).
  - `common/matchflavor.go`.
  - No new transactions, notifications or REST routes. Two existing JSON responses gain fields.
- **Frontend.**
  - New `features/match/MatchViewer.tsx` plus view-model adapters.
  - `MatchDetailsPage.tsx`, `CupTieDetailsPage.tsx` and `FriendlyMatchPage.tsx` shrink to
    adapters.
  - `ZoneComparisonRows.tsx` and `MatchMinuteZoneComparison.tsx`.
  - `api/types.ts`.
  - New i18n keys in `en.json` and `es.json` for the competition labels and the zone-duel help
    text.
