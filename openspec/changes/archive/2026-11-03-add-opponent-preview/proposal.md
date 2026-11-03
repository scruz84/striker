# Proposal

## Why

When a manager sets his lineup he sees who he plays and when (`LineupEditorPage` subtitle), and
the match page shows the head-to-head (`HeadToHeadCard`), but nothing about how the opponent
plays now: its form, the shape and XI it has been using, its in-form players, or where on the
pitch his own XI wins or loses the duel. All of that is already public (results, stored kickoff
lineups, stored match ratings) or computable with the engine's own zone ratings. A preview next to
the lineup editor turns the lineup from a routine into a decision, and pairs naturally with team
mentality and conditional orders.

## What Changes

- An **opponent preview** card for a club's next match (in the lineup editor) and for any upcoming
  fixture the owner's club plays (in the match page's scheduled view, next to his lineup editor):
  - the opponent's last 5 competitive results and league position;
  - its **likely shape**: the formation and XI it kicked off with in its latest competitive match;
  - its **in-form players**: the 3 best average match ratings over its last 5 competitive matches
    (at least 3 appearances);
  - a **projected zone duel**: the owner's currently saved lineup for that fixture against the
    opponent's likely XI, rated with the engine's own zone ratings at minute 1 (home advantage
    and mentality included when those changes have shipped), shown per zone like the viewer's
    zone duel.
- The projection refreshes when the owner saves a different lineup.
- Only the participating owner sees the card, and only data that is already public is used for
  the opponent: never its saved lineup for the upcoming match.

Out of scope:
- The opponent's current injuries and suspensions (other clubs' availability is not public today;
  see the open question).
- Predicting the result or win probabilities.
- Previews for fixtures the owner's club does not play.

Open questions for the owner:
- Should other clubs' injured and suspended players become public so the preview can strike them
  from the likely XI? Today only a club's own squad shows availability.
- Should the projected duel also be offered to the opponent's view of the same fixture (the
  default: yes, each participant sees his own projection)?

## Capabilities

### New Capabilities
- `opponent-preview`: the preview card's contents, data sources, privacy and where it is shown.

### Modified Capabilities
- `match-viewer`: "Viewer phases" — the scheduled view shows the preview to a participating owner.

## Impact

- Backend: new `common/opponentpreview.go` (form, latest kickoff lineup, in-form players) and an
  engine-side pure projection that reuses `calculateQua` / `blendLine` (exported through a small
  `engine.ProjectZoneDuel` helper the service calls); endpoint
  `GET /v1/teams/preview/:kind/:id` (`kind` = league, cup or friendly).
- Frontend: `OpponentPreviewCard` (reusing `ZoneComparisonRows`), placed in `LineupEditorPage`
  and the scheduled view of `MatchViewer`; en/es strings.
- No migration, no notification.
