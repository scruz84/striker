# Proposal

## Why

Every match rates every player, but those ratings are only seen on the match page. A team of the
week gives each matchday a highlight, a reason to open the league page after a round, and a small
reward for a manager whose players shine.

## What Changes

- **Team of the week**: once every match of a league matchday has finished its live replay, the
  league picks a 4-4-2 of its best rated players (goalkeeper, defence, midfield, attack by the
  position each played most) and stores it.
- **Team of the season**: at the rollover, alongside the Golden Boot and Player of the Season, the
  same 4-4-2 by average rating among players with at least 36 minutes per match.
- **Notification** (`NOTIFICATION_AWARD`) to each owner whose players made it, listing them, with a
  link that opens that round on the division page.
- **Division page**: a team of the week card on a pitch, browsable by matchday and to the team of the
  season; a past season shows its team of the season.
- **Player profile**: how many teams of the week and of the season he made.
- New endpoint `GET /v1/teams/division/:id/best-xi?season=&week=`.

Out of scope:
- Other formations, and best XIs of cups or friendlies.

No open question for the owner.
