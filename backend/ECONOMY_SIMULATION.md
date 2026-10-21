# Economy audit: simulation and rebalancing plan

Written after the suspicion that "there is more income than expenses" in the game. This documents
what a simulation of the *current* code says, why, and a plan to fix it within these constraints
from the game's owner:

- Stadium attendance must depend on the club's **socios** (members), the team's **form**, the
  **opponent** and the **weather**.
- Stadiums must **still be expandable**, and the stadium itself stays as it is: existing
  capacities, expansion cost, maintenance and season tickets are not touched.
- Beyond attendance, the only other lever is **wages**. Sponsorship, prize money and
  merchandising keep their current values.

The simulator lives in `engine/economysim_test.go` (skipped unless `ECON_SIM` is set) so every
number below can be reproduced and every constant re-tuned:

```
cd backend
ECON_SIM=1 go test ./engine -run TestEconomySimulation -v -count=1
# optional: ECON_SIM_SCENARIOS=baseline,B-socios+tier-wages  ECON_SIM_SEASONS=40  ECON_SIM_SEED=2
```

## TL;DR

- **The suspicion is right, and by a wide margin.** With the code as shipped, the total money in
  the world grows ~23x in 18 seasons and ~80x in 36 (84M -> 1.9B -> 6.8B), and the median team's
  budget goes from 400k to 2.4M to 4.2M. It is not an "early game" effect that fades: it gets worse
  over time.
- **Causes:**
  1. **Attendance scales with whatever stadium you build.** Revenue is `capacity x occupancy x
     price` with no limit on how many people want to come. An expansion costs ~20 per seat and an
     extra seat earns ~19 per home match at the AI's price: it pays for itself in about one home
     match. A Director-level CPU team (60% of tier 1) or a human who expands whenever possible
     reaches 24,500 seats after 18 seasons and 48,500 after 36.
  2. **The "optimal" ticket price is 69, 7x the default.** `demandRate` is so flat that charging 69
     keeps 27% of the stadium, so every Manager+ CPU team earns 3.7x the gate of a team on the
     default price of 10, for free.
  3. **Tier income grows 5x per tier but costs are flat.** Wages depend only on skills, which are
     the same in every division, while sponsorship is 8,000 / 4,000 / 1,500 per match and prize
     pools 500k / 200k / 50k. A hands-off tier-1 team earns 4.7x its expenses, tier 2 2.2x, tier 3
     ~1.06x (the only tier the original balancing got right).
  4. **No real sinks.** 97% of every team's spending is wages.
- **Proposal (scenario `B-socios+tier-wages`):** attendance driven by socios x price x form x
  opponent x weather (capped by capacity), socios growing slowly with fanbase and league position,
  a CPU rule to expand only when the ground keeps selling out, and a **division wage factor**
  (x4.0 / x2.1 / x1.1 for tiers 1 / 2 / 3). Result: total money levels off at ~2x the start,
  hands-off teams sit at ~1.3 / 1.16 / 1.0 income-to-expense, good management is worth +10-17%,
  and big tier-1 clubs grow their grounds to 1,600-2,000 seats because their socios demand it.

## How the economy works today (the facts the simulation rests on)

Everything below is read straight from the code; `engine/economysim_test.go` calls the real
functions wherever it can (its baseline calls `CalculateAttendance` itself).

| Flow | Rule | Where |
|---|---|---|
| Ticket revenue (home only) | `attendance x price`; `attendance = capacity x clamp(demandRate(price) x fanbaseFactor + jitter, 0.1, 1.0)`, `demandRate = 0.5 - 0.35 x (price-10)/90` | `engine/stadium.go` |
| Sponsorship (both sides, every league *and cup* match) | base 8,000 / 4,000 / 1,500 by tier x deal multiplier (win/draw bonus) x fanbase factor 0.8-1.2 | `engine/sponsorship.go` |
| Prize money (season end) | pool 500k / 200k / 50k split 2x..0 by position; cup round x 5,000; champion +100k | `engine/prizemoney.go` |
| Season tickets + merchandising (season end) | capacity x 5 x fanbase factor; merch pools 100k / 40k / 10k | `engine/seasonincome.go` |
| Wages (per round) | `sum(7 skills) x 5/7` per player, 18 players -> ~4,050/round at avg skill 45; trainer `skill x 8/7` | `common/fmcommon.go`, `engine/wages.go` |
| Stadium maintenance (per round) | `capacity / 10` = 100 at 1,000 seats | `engine/stadium.go` |
| Stadium expansion | cost = `capacity x 10 x seats/500`, +500 seats, 7-day build, no maximum | `common/stadium.go` |
| Weather | `sunny/cloudy/rain/windy`, drawn at random *after* attendance, purely cosmetic | `common/matchflavor.go`, `engine/simulation.go` |
| Transfers / loans / free agents | zero-sum between teams (free agents cost only their wage) | `common/transfers.go` etc. |

CPU behaviour that matters (`engine/aiteams.go`): Manager and Director teams set the ticket price to
`optimalTicketPrice()` (= **69**, confirmed by running it); Director teams expand the stadium by 500
seats whenever `budget >= 3 x cost`, hire the best coach, and buy listed players.

Per-team reference numbers at the default 1,000-seat stadium (fanbase 50): gate per home match
**5,020 at price 10** and **18,776 at price 69**; wage bill 4,050/round + 100 maintenance = ~75k per
18-round season; sponsorship per match (balanced deal, ~1.3x with win bonus) ~2,000 / ~5,200 /
~10,400 for tiers 3 / 2 / 1.

## Simulation design (and its limits)

The simulator builds the default world (1 + 4 + 16 leagues of 10 teams, 210 teams), plays double
round-robin seasons (18 rounds), the tier-1 cup, fanbase drift, season-end prize/season-ticket/
merchandising income, wage growth from skill progression (+0.4%/season, +3%/season with a coach,
capped at +25% by skill ceilings), and the daily CPU decisions (price, stadium expansion with the
real 3x budget cushion and 7-day build, coach hiring). In every league two teams are **reference
humans**: *passive* (default price, never expands) and *optimal* (best price, expands whenever the
rules allow, hires a coach). The other eight use the shipped AI-level mix.

Not modelled: promotion/relegation (teams stay in their tier; fanbase still drifts to the tier
target), the transfer market (zero-sum, so irrelevant to total money), match results beyond a
strength-weighted Poisson, injuries, loans. Ratios are cadence-independent; only stadium construction
(7 real days) depends on the schedule, and the simulation assumes the default **daily** rounds
(~19 seasons per real year). Tier-1 rows are single teams per class and therefore noisy (their
ratio moves with where they finish); the world totals are stable across seeds 1-3.

## Diagnosis: the baseline

Per team, per season, average of seasons 2-40 (S1 = first season in brackets). "Ratio" = income /
expenses, where expenses include stadium expansions.

| Team | Tier 1 | Tier 2 | Tier 3 |
|---|---|---|---|
| Human, passive (price 10, no expansion) | 4.71 (S1 5.01) | 2.15 (S1 2.20) | 1.06 (S1 1.25) |
| CPU Caretaker | 3.87 | 2.10 | 1.02 |
| CPU Manager (best price) | 6.13 | 3.59 | 2.45 |
| CPU Director (best price + expands) | 7.54 | 5.51 | 4.74 |
| Human, optimal | 7.98 | 5.56 | 4.73 |

| After season | Total money (start 84M) | Median team budget | Director stadium |
|---|---|---|---|
| 18 (~1 real year) | 1.94B (23x) | 2.4M | 24,500 seats |
| 36 | 6.84B (81x) | 4.2M | 48,500 seats |

Is it "normal at the start because players are worse"? Only a little: in season 1 a passive tier-3
team shows 1.25 against a steady-state 1.06, because fanbase starts at 50 and slides towards the
tier-3 target of 30. Wage growth from player development (0.4-3%/season, capped near +25%) is far
too small to ever close a 2-7x gap: the imbalance is structural.

## The proposal

### 1. Attendance driven by socios, form, opponent and weather

```
attendance = min(capacity, socios x price x form x opponent x weather x noise)

price    = clamp(1 - (ticketPrice - 10) / 30, 0, 1.3)      1.0 at the default price, 0 at 40
form     = 0.8 + 0.4 x (league points in the last 5) / 15   0.8 after five defeats .. 1.2 after five wins
opponent = 0.85 + 0.30 x opponentFanbase / 99               x (1 + 0.1 x cup round) in cup ties
weather  = sunny 1.05, cloudy 1.00, windy 0.92, rain 0.82   (the existing weather codes)
noise    = 1 +/- 10%
```

- **Socios** is a new per-team number (`teams.socios`). Every team starts with **500** (the same
  crowd a default team gets today). At each season rollover it moves 30% of the way towards
  `2.2 x fanbase^1.5`, scaled +10% for the league winner down to -10% for the last team.
  Superlinear on purpose: a tier-1 club whose fanbase climbs towards its tier target of 80 heads
  for ~1,400-1,700 socios and outgrows the 1,000-seat ground, while a tier-3 club (target 30) settles
  around 360-500 and never needs to expand. Socios is what grows demand; seats don't.
- The price factor only covers the price itself, and attendance goes to 0 at 40, so there is no
  floor to exploit: the revenue-maximising price is ~20 when the ground isn't full, or the price
  that just fills it when demand exceeds capacity.
- Weather becomes real: draw it **before** attendance and store the same value in the match report
  (today `updateMatchReportFlavor` draws it afterwards, as decoration).
- Fanbase keeps driving sponsorship/merchandising/season tickets as now; its effect on crowds goes
  through socios instead of `fanbaseAttendanceFactor`.

### 2. Expansion stays possible, and becomes a real decision

`UpgradeStadium`, its cost, maintenance and season tickets are unchanged, and nothing is done to
existing stadiums. What changes is *why* you would expand: a ground only pays for more seats when
socios x the factors above keep exceeding capacity. The CPU's `upgradeStadiumIfAffordable` gets
one extra condition: expand only when the ground has sold out in at least ~40% of recent home
matches (attendance == capacity in the last N `matches.attendance` rows; no new column needed),
instead of whenever it has 3x the cost in the bank. In the simulation, tier-1 Directors and
optimising humans grow to 1,600-2,000 seats; tier 2 and 3 clubs stay at 1,000.

Existing worlds keep their stadiums as they are. A club that already built 30,000 seats will see
crowds limited by its socios, but it does not go broke: season tickets (30,000 x 5 = 150k/season)
still exceed maintenance (3,000/round = 54k/season). Season tickets paying for empty seats is a
small leftover faucet (+3.2 per empty seat per season, against an expansion cost of 20+ per seat),
too slow to matter and outside the agreed scope.

### 3. Wages: a division factor

Apply a factor to the wage bill by the division the club currently plays in, at payment time in
`GetTeamWageBill` (and the trainer's wage):

| Tier | Factor |
|---|---|
| 1 | 4.0 |
| 2 | 2.1 |
| 3 | 1.1 |

- Story: players at a top-flight club earn more than the same players in the third tier (the
  usual promotion/relegation clauses). Promotion roughly doubles both income and wage bill; relegation
  halves both.
- Applying it at payment time needs no migration and follows promotion/relegation automatically.
  The stored, skill-derived wage stays the *base* wage used for offers, renewals and the AI's
  transfer valuation (`minAcceptableTransferFeeWageMultiplier`), so none of that logic changes. The
  UI should show the effective wage (base x factor) wherever a wage bill is shown.
- **Option C (variant `C-socios+tier-wages+win-bonus`)**: a smaller factor (x3.3 / x1.8 / x1.05) plus
  a squad win bonus per league win (9,000 / 3,500 / 300). Same overall balance, but wage costs then
  follow results the way sponsorship bonuses already do, so a promoted side that keeps losing is
  hit less hard. Slightly more to build and to explain.

## Results

Per team, per season (average of seasons 2-40), seed 1. Ratio = income / expenses.

| Scenario | Passive T1 / T2 / T3 | Manager T1 / T2 / T3 | Total money @18 / @36 | Tier-1 Director seats |
|---|---|---|---|---|
| baseline | 4.71 / 2.15 / 1.06 | 6.13 / 3.59 / 2.45 | 1,943M / 6,841M | 54,000 |
| A - socios demand only | 5.14 / 2.41 / 1.10 | 5.18 / 2.57 / 1.29 | 288M / 439M | 1,500 |
| **B - socios + division wages** | **1.30 / 1.16 / 1.00** | **1.32 / 1.24 / 1.17** | **157M / 169M** | **1,600** |
| C - socios + smaller wage factor + win bonus | 1.26 / 1.14 / 1.03 | 1.31 / 1.24 / 1.20 | 161M / 180M | 1,500 |

- **A alone** kills the stadium printing press (6.8B -> 439M) and keeps expansion meaningful, but
  tiers 1-2 stay far too rich: their surplus comes from sponsorship and prize money, which the
  attendance model doesn't touch.
- **B** levels the world off at ~2x the starting money (157M -> 169M over the last 18 seasons, i.e.
  ~0.4% per season), with hands-off teams near break-even in tier 3 and a modest surplus higher
  up. Seeds 2 and 3 give the same picture: 161M / 177M and 161M / 176M, passive tier 3 at
  0.99 / 0.95, tier 2 at 1.00 / 1.17, tier 1 at 1.27 / 1.45.
- Good management (best ticket price) is worth +10-17% over passive, never multiples. The
  "optimal" human with a coach shows a lower ratio in tiers 2-3 because its coach makes the squad
  better and therefore dearer; the simulator doesn't credit the extra wins.

## Implementation plan

1. **Socios** (`resources/sql/init78_socios.sql`): `ALTER TABLE teams ADD COLUMN socios integer NOT
   NULL DEFAULT 500`; for existing teams set it straight to the target for their current fanbase
   (`round(2.2 * fanbase ^ 1.5)`) so they don't ramp up from 500. Drift it at rollover next to
   `awardSeasonTicketsAndMerchandising` (it already has the final standings), and show it on the
   stadium/finances page with a notification when it changes at season end.
2. **Attendance** (`engine/stadium.go`): new `CalculateAttendance(capacity, price, socios, form,
   opponentFanbase, cupRound, weather, r)`; form from the last five league results (one query on
   `matches`); weather drawn once in `simulateMatch` and passed both here and to
   `UpdateMatchFlavor`. Friendlies keep their split gate but use the same attendance function.
3. **Price**: rewrite `optimalTicketPrice` to scan prices against the new demand (capacity, socios,
   form), so the Manager-level AI keeps charging what the gate will bear.
4. **CPU expansion**: add the sell-out condition to `upgradeStadiumIfAffordable`.
5. **Wages**: `DivisionWageFactor` map in `common/fmcommon.go`, applied in `GetTeamWageBill`, the
   trainer wage and `GetTeamLoanWageContribution`; expose the effective wage to the frontend.
6. **Docs/tests**: update `GAME_MECHANICS.md` (Economy section), `engine/stadium_test.go`
   (attendance, price), a test that attendance never exceeds capacity or socios x 1.3 x 1.2 x 1.15 x
   1.05 x 1.1 (times the cup-round factor), and keep `economysim_test.go` in sync as the acceptance check.

Ship 1-4 and 5 together: attendance alone (scenario A) still leaves tiers 1-2 printing money, and
the wage factor alone would push every club below 1 until crowds stop scaling with capacity.

### Acceptance criteria

- Hands-off team: tier 3 ~1.0, tier 2 ~1.1-1.2, tier 1 ~1.3 income-to-expense (steady state).
- Good management beats hands-off by 10-25%, never by multiples.
- Total money within ~2x of the start and growing < 1% per season after season ~20.
- Expansion still pays for clubs whose socios outgrow their ground, and only for them.

## Open decisions

1. **B or C** for wages (flat division factor, or a smaller factor plus win bonuses).
2. **Existing budgets**: with balanced flows they stop growing, but what was already earned
   stays (some CPU clubs hold millions). Leave it, or a one-off admin adjustment.
3. **Socios curve**: `2.2 x fanbase^1.5` gives top clubs ~1,400-1,700 socios. A bigger coefficient
   makes expansion more common (and tier-1 richer, which the wage factor would then need to absorb).

## Validating against production data

Run these before changing anything and again afterwards. They only read.

```sql
-- Money supply and spread by tier
SELECT l.tier, count(*) AS teams, sum(t.budget) AS total, avg(t.budget)::bigint AS avg,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY t.budget) AS median, max(t.budget) AS max
FROM teams t JOIN leagues l ON l.league_id = t.league_id GROUP BY l.tier ORDER BY l.tier;

-- Stadium sizes and ticket prices (look for a spike at 69 and capacities far above 1000)
SELECT l.tier, t.ai_level, count(*) AS teams, avg(t.stadium_capacity)::int AS avg_capacity,
       max(t.stadium_capacity) AS max_capacity, avg(t.ticket_price)::numeric(6,1) AS avg_price
FROM teams t JOIN leagues l ON l.league_id = t.league_id GROUP BY l.tier, t.ai_level ORDER BY 1, 2;

-- Income vs expense by reason over the last 30 days (reason codes: see TRANSACTION_* in
-- common/fmconstants.go: 2 wages, 4 ticket, 5 prize, 6 sponsorship, 7 stadium upgrade, ...)
SELECT l.tier, tt.reason, sum(tt.amount) AS net, count(*) AS rows
FROM team_transactions tt
JOIN teams t ON t.team_id = tt.team_id JOIN leagues l ON l.league_id = t.league_id
WHERE tt.created_at > now() - interval '30 days' AND tt.reason NOT IN (0, 1) -- exclude transfers
GROUP BY l.tier, tt.reason ORDER BY l.tier, tt.reason;
```

If the real numbers disagree with the simulation (especially if real CPU stadiums are *not* growing),
re-check the assumptions listed under "Simulation design" - the most likely culprits are the AI
daily tick not running for some teams, or a non-default match schedule changing seasons per year.
