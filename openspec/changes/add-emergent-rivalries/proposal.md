# Proposal

## Why

Every pairing in the game feels the same: a team's 30th meeting with a club that knocked it out of
the cup twice plays like a first meeting with a stranger. Real derbies are the matches fans
remember, but the world has no geography below the country (`common/regions.go` only groups
countries by continent), so rivalries cannot come from proximity. They can come from history:
the game already stores every meeting (`common/headtohead.go`), cup eliminations, title races,
red cards and transfers. A rivalry that grows out of what managers actually lived through works in
all 211 countries with no new data, and gives two human managers who keep crossing paths a real
grudge match.

## What Changes

- A new per-pair **rivalry score** (two teams of the same country, stored once per pair) that
  grows with "heated" history: close league results, thrashings, red cards, cup eliminations
  (more for a final), a title race (champion and runner-up of a division) and the sale of a
  key player between the two clubs. Events between two human-owned clubs count ×1.5.
- The score **decays ×0.6 at each season rollover** and is capped at 40, so old feuds fade unless
  they are fed.
- Two levels derived from the score: **Rival** (≥ 8) and **Clásico** (≥ 16).
- Effects of a league match or cup tie between rivals:
  - a bigger home crowd (rival factor ×1.10 for Rival, ×1.20 for Clásico);
  - higher stakes for the fans (the fanbase result-drift roll is 30% / 45% instead of 15%);
  - a tenser match (card incident probability 3.6% / 4.2% per minute instead of 3%).
- A new notification (`NOTIFICATION_RIVALRY`) when a pair reaches Rival or Clásico, applied only
  after the match's live replay finishes.
- UI: a "Rivals" section on the team page (top 3 with level and head-to-head), a Rival / Clásico
  chip on fixture rows, the dashboard's next-match hero and the match viewer scoreboard.

Out of scope:
- Geographic or "same town" derbies (option B of the discussion), and any invented geography.
- Rivalries between clubs of different countries or involving national teams; friendlies neither
  feed nor are affected by a rivalry.
- Any effect on the simulation besides card frequency (no "derby boost" to team strength).

Open questions for the owner:
- When a club is released and reset to a CPU club (`user-accounts`), should its rivalries be kept
  (club history, the default here) or wiped with the owner's other history?
- Should a manager be able to see the exact score, or only the level? The default here shows the
  level and a 0–40 bar; the score itself is not secret since it is built from public results.

## Capabilities

### New Capabilities
- `rivalries`: how a rivalry between two clubs is scored from their shared history, how it decays,
  its levels, its notification and where it is shown.

### Modified Capabilities
- `socios`: "Attendance factors" — the rival factor is multiplied by the rivalry level.
- `fanbase`: "Result drift" — the drift probability is higher in a match between rivals.
- `match-simulation`: "Discipline" — the per-minute card probability is higher between rivals.

## Impact

- Backend: new migration and `common/rivalries.go` (score storage, levels, events); updates from
  `engine/postmatch.go` (after the replay), `engine/season.go` (title race, decay) and the
  transfer paths (`common/transfers.go`, `common/offers.go`) for key-player sales;
  `engine/stadium.go`, `engine/fanbase.go`, `engine/discipline_simulation.go` read the level.
- API: `GET /v1/teams/team/:id/rivalries`; `rivalry` level added to `TeamMatchSummary` and to the
  match viewer payload.
- Frontend: team page section, fixture chip, dashboard hero chip, viewer subtitle; en/es strings;
  Spanish notification template.
- Economy: slightly larger gates in rival matches; checked with the economy simulation.
