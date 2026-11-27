# Proposal

## Why

Every CPU club of a level behaves identically - the same thresholds, the same reserve, the same
appetite - so a human quickly learns to read them all, and a club keeps the AI level it drew on
creation forever: a Caretaker promoted to the top division stays a Caretaker. Real clubs have a
character and adapt to the division they play in. Academy promotion also only ever fills a thin
position, so even a Director lets a gifted prospect wait in the academy.

## What Changes

- **AI level after promotion or relegation**: at the season rollover a CPU club that changes
  division draws a level with its new tier's weights (see "AI levels") and moves one step towards
  it (never two).
- **Club style**: every CPU club has a stable style, assigned once from its id - balanced (40%),
  youth (20%), frugal (20%) or ambitious (20%) - and stored, that adjusts a Manager's or Director's transfer profile:
  a youth club signs projects and lends youngsters at any level from Manager up; a frugal one keeps
  15 more points of budget reserve and protects its key players less (×0.5); an ambitious one keeps
  10 points less (never below 5%), lowers its minimum score by 1 (never below 1) and protects key
  players more (×0.5). A Caretaker ignores its style.
- **Prospects by potential**: a Director, or a youth-style Manager, that did not fill a thin
  position from its academy this pass promotes the prospect whose projected value at his best
  position (its noisy read of his ceiling, 3 seasons on) reaches its weakest starter's value there,
  under the usual once-a-week cap.

Out of scope:
- Editing the style (admin or otherwise); showing it to players.
- Levels changing for any reason other than a change of division.

Open questions for the owner:
- Should the style be visible (for example "known for its academy" on the club page)?
- Is one step per division change the right speed, or should a promoted Caretaker jump straight
  to its new tier's draw?

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-team-management`: new "AI level after a division change", "Club style" and "Prospects by
  potential" requirements.

## Impact

- Schema: `resources/sql/init102_team_ai_style.sql` adds the nullable `teams.ai_style` (NULL reads
  as balanced), filled at team creation and by a startup backfill.
- Backend: new `common/aistyle.go`; `common/aimanager.go` (`RandomAiLevelFrom`,
  `StepAiLevelToward`); `engine/season.go` (rollover moves); `engine/aiteams.go`,
  `engine/aitransfers.go`, `engine/ailoans.go` (profile per team); academy promotion in
  `engine/aiteams.go`.
