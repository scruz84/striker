# Proposal

## Why

The per-player match ratings (the "Notas" tab, the live ratings panel, the lineup pitch chips,
the player's match history and Form drift) come from `common.ComputeMatchRatings`, a heuristic
that ignores how the player actually played. It is also wrong around substitutions: a finished
match's lineup has the substitutions applied on top of the starting XI, so a player who came off
disappears from the ratings (along with any goal or card he had), and the player who came on
takes the slot and gets the full result bonus after a few minutes. The rating also says nothing
about what the player gave to his zone, even though that zone value (the per-minute
`TeamQualification`) is what the engine compares to decide each zone duel and so the goals.

## What Changes

- **Everyone who featured gets a rating.** Starters, substitutes and players who were sent off
  or injured are all rated. A substitution no longer moves events or minutes from one player to
  another. Each rating carries the minutes played and the minute the player came on or went off.
- **New zone component.** Every simulated minute, the engine records for each player on the
  pitch his own contribution to the zone or zones his position feeds (the same fatigue-adjusted
  values `calculateQua` adds up) and the opponent's per-player value in the mirrored duel. The
  zone component of the rating is `3.0 × the average duel edge` over the player's minutes,
  capped at ±1.5. Edge = (own − rival) / (own + rival).
- **Result bonus scales with minutes:** ±0.3 × minutes / 90 instead of a flat ±0.3.
- **Form and the deterministic jitter leave the rating.** Form doesn't change how a player
  plays in the simulation. Feeding it into the rating, which in turn drives Form, was a loop.
  The jitter was fake variety, and the zone component now gives real variety.
- Goal +1.4, yellow −0.4, red −1.5, injury −0.2, base 6.0 and the [1, 10] clamp are unchanged.
- **Ratings are computed once, at simulation time, and stored** (a new `match_player_ratings`
  table, plus a per-minute `player_contributions` column on the three instants tables). They
  stop drifting as players' skills and Form change after the match.
- **The ratings view explains each number:** position, minutes (with in/out minute), the
  player's zone contribution against the rival's ("58 vs 45"), and a breakdown (zone, goals,
  cards, injury, result) on tap or hover.
- **Live and scrubbed ratings use the same formula.** They are rebuilt from the per-minute
  contributions in the instants plus the events and score up to that minute. Injury and red-card
  events stay withheld live. At the final minute, the scrubbed figures equal the stored ratings.
- Form drift reads the stored rating, so substitutes now get a performance adjustment too.
- **Under 10 minutes is "s.c."**: such a player keeps his record (minutes, contribution,
  breakdown) but shows no rating, and Form gets no performance adjustment from that match.
- **Matches simulated before this change** keep a fallback computed on read. It fixes the
  substitution bug (everyone who featured, events credited to whoever had them, result bonus by
  minutes) and has no zone component, because those matches have no per-minute contributions.

### Out of scope

- Changing how the engine itself computes zones, chances or goals. The zone component only reads
  values the simulation already uses.
- Assists, saves or any new event types.
- Backfilling zone components for matches simulated before this change.
- Player of the Season (`season-awards`), which keeps its own goal/card tally.

### Owner decisions

1. Zone weight 3.0 × edge, capped at ±1.5 (55 vs 40 gives +0.47; 70 vs 30 gives +1.2).
2. Form stays out of the rating.
3. A player with fewer than 10 minutes is shown as "s.c." (sin calificar): his record is stored
   with its breakdown, but he has no rating number, and Form gets only the +2 for featuring.
4. Per-minute contributions are stored on the instants (~40-50 KB per match).

## Capabilities

### New Capabilities

- `match-ratings`: how a player's match rating is computed (base, zone component, events,
  result), who gets one, when it is computed and stored, and the fallback for matches simulated
  before it existed.

### Modified Capabilities

- `match-viewer`: the ratings view lists everyone who featured with minutes, zone contribution
  and breakdown. The "Live spoiler withholding" requirement is corrected to match the code: live
  ratings are shown, computed without injury and red-card events.
- `player-development`: Form drift reads the stored match rating, and a substitute's rating counts
  like a starter's.

## Impact

- Backend: new `common/matchratings.go` logic (zone contributions, rating formula, legacy
  fallback); the engine records contributions each minute (`engine/initial_simulation.go`,
  `engine/simulation.go`) and computes ratings at the end of `runMatchSimulation`; the three
  `Save*ResultTx` functions write ratings and contributions; read paths (`WeekMatchById`,
  `GetCupTieDetails`, `EnrichPlayedFriendly`, `common/playermatches.go`) read stored ratings;
  `engine/progression.go` reads ratings from the post-match payload; the live broadcast carries
  each instant's contributions.
- Migration `backend/resources/sql/init89_match_player_ratings.sql`.
- API: `WeekMatch`, `CupTieDetails`, `FriendlyMatch` gain `playerRatings` (detailed list);
  `ratings` (id → number) stays for existing consumers. `MatchInstant` gains
  `playerContributions`.
- Frontend: `utils/liveMatchRatings.ts` (rebuilt on contributions), `features/match/
  TeamRatingsColumn.tsx`, `MatchMinuteRatings.tsx`, `MatchRatingsTab.tsx`, `MatchPitchChip.tsx`,
  `api/types.ts`, `i18n/locales/{en,es}.json`.
- Specs: new `match-ratings`; `match-viewer` and `player-development` updated, including their
  Code maps.
