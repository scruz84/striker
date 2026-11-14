# Proposal

## Why

A country's pyramid is 1/4/16 groups. Every season 2 teams go down from the top division, but
the league page paints all 4 tier-2 group leaders as "promotion", and only the best 2 of them
actually go up. The same mismatch repeats between tiers 2 and 3. Players read the table as
"2 go down, 4 go up". The owner wants a pyramid where the zones on the table are literally true,
plus a promotion playoff that gives the end of the season more at stake.

## What Changes

- **New pyramid rule.** Every tier has exactly twice as many groups as the tier above it. The new
  default structure is 1/2/4/8/16 groups of 10 teams (310 teams per country, up from 210).
  `ValidateWorldConfig` rejects any structure that breaks the rule, so the admin world editor
  can only save valid pyramids.
- **New movement rule between adjacent tiers**, derived from the structure and no longer
  admin-configured:
  - Lower-tier group winners (1st) go up directly.
  - The bottom 2 of every upper-tier group go down directly.
  - Lower-tier runners-up (2nd) play a single-match **promotion playoff**, on the day after the
    last league matchday at the upper tier's league kickoff time (14:00 for I–II, 14:15 for II–III,
    and so on), against the teams
    3rd and 4th from bottom of the upper tier. The worst of those upper-tier teams faces the
    best runner-up. The match is at the upper-tier team's ground, with penalties on a draw. The
    winner plays in the upper tier next season and the loser in the lower tier.
- **New end-of-season phase.** Once every league match of a country has been played and
  broadcast, the engine creates that country's playoff ties. The season rolls over only after
  every playoff tie has been played and broadcast.
- **Restructure at rollover.** When a country's leagues do not match the saved world structure,
  the next rollover reshapes them.
  - This season's promotion and relegation run first.
  - Then every team is ranked down the pyramid and the new tiers are filled top-down. Teams that
    do not fit drop to the tier below.
  - The bottom tier is topped up with newly generated CPU teams.
  - Leagues that no longer exist are archived rather than deleted, so their history stays
    readable. No team is deleted.
  - For an existing world this happens once, at each country's next rollover. A migration moves
    a world still on the old default 1/4/16 to the new default.
- **Staggered league kickoffs.** Each country has a league kickoff time on its local clock,
  defaulting to 14:00. Tier 1 kicks off at that time and each lower tier 15 minutes later:
  14:15, 14:30, 14:45 and 15:00. Kickoffs are dated in the country's time zone, so they no
  longer drift by an hour when the clocks change. Leagues the restructure creates follow the rule
  with no extra setup. The per-league override remains for exceptions.
- **BREAKING (admin):** the "promotion/relegation slots per group" admin setting is removed.
  Slots are now derived from the structure. The `PUT` endpoint for that setting is removed.
- **League page.**
  - The table shows three zones: direct promotion, playoff, and direct relegation.
  - A "Playoff" section lists the country's playoff ties for this tier while they exist.
  - Past-season tables mark teams that went up or down through the playoff.
- **Notifications** (en/es):
  - Playoff qualification, sent with the opponent and date.
  - Promotion or relegation through the playoff.
  - A move caused by the restructure.
- **CPU season objectives** count the playoff places as part of the promotion or relegation fight.

## Capabilities

### New Capabilities
- `league-pyramid`: covers the shape of a country's league pyramid and how teams move between
  tiers at the end of a season. That includes the structure rule, direct promotion and relegation,
  the promotion playoff, the rollover gate and restructuring an existing country.

### Modified Capabilities
- `league-page`: the standings table gets three zones (promotion, playoff, relegation) taken
  from the pyramid, plus a playoff section and playoff marks on past seasons.
- `match-scheduling`: a league's kickoff time becomes the country's local kickoff time plus 15
  minutes per tier, unless it has an override. The country kickoff becomes an admin setting that
  re-dates unplayed matches when it changes.
- `team-finances`: prize money and merchandising state that tiers below the third use tier-3 pools,
  and that a playoff tie pays no prize of its own.
- `engine-replicas`: the season rollover now also creates the playoff once, waits for it, and
  restructures leagues, all under the same per-country lock.

## Impact

- **Backend engine:**
  - `engine/promotion.go` is rewritten. It computes direct moves and playoff pairings from the
    structure.
  - `engine/season.go` adds the playoff phase, the gate, the restructure step and notifications.
  - `engine/cups.go` must skip playoff "cups" in `AdvanceCups` and in the round prize.
  - Team and league creation moves out of `handlers/initializer.go` into `common`, so the engine
    can generate teams.
- **Backend common:**
  - `worldconfig.go` gets the new default and validation.
  - `promotionconfig.go` is replaced by slot derivation.
  - `database.go` has every league listing filter out archived leagues, and its zone slots are
    replaced.
  - `cups.go` gains a cup kind, and `GetLatestCupForCountry` ignores playoffs.
  - `seasonobjective.go` and `seasonhistory.go` change.
  - `notificationtranslations.go` gets 4 new templates.
- **API:**
  - `DivisionDetails` gains `playoffSlots` and a `playoff` tie list.
  - Season history lines gain `viaPlayoff`.
  - The admin overview drops the promotion/relegation fields.
  - `/v1/admin/promotion-relegation` is removed.
- **Scheduling:**
  - `engine/engine.go` and `engine/reschedule.go` date fixtures in the country's location.
  - `common/leagueschedule.go` gets the effective schedule per league.
  - `common/timezones.go` gets the country kickoff.
  - Add-country stops writing per-league overrides.
- **SQL:** new migration `init110_league_pyramid_playoffs.sql`.
- **Frontend:**
  - `DivisionDetailsPage.tsx` gets the zones, the playoff section and past-season marks.
  - In admin, `RulesTab.tsx` loses its form and `WorldTab.tsx` shows the pyramid-rule
    validation and a "pending restructure" note.
  - Locales `en.json` and `es.json` change.
- **Economy:** tiers 4 and 5 fall back to the tier-3 prize, wage, sponsorship and merchandising
  values that already apply to "any tier below the third". There are 100 more CPU teams per country.
- **Interaction with in-flight changes:** `add-cpu-season-context` owns the CPU season objective
  spec. This change only feeds it the new slot counts.

## Out of scope

- Two-legged playoffs (aggregate score, away legs). The playoff is one match.
- Re-tuning the tier 4/5 economy. Those tiers keep the existing "below tier 3" values until an
  economy pass checks them with `ECON_SIM`.
- Structures with any ratio other than 2 between tiers, or different team counts per group
  within one tier.
- Merging or deleting existing teams. A restructure only moves teams and adds CPU teams.

## Owner decisions

- The playoff pays no prize money of its own, only gate and sponsorship, so no new
  `TRANSACTION_*` reason is added.
- The playoff is played the day after the last league matchday, at the upper tier's league
  kickoff time.
- Tier 1 kicks off at 14:00 local time and every lower tier 15 minutes later.
- When the world is restructured, surplus teams drop and a new bottom tier is filled with
  generated CPU teams.
