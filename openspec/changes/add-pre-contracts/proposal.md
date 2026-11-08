# Proposal

## Why

A contract's last season has no tension today. The owning club can renew at any moment at the
player's market wage, nobody else can approach the player, and a player the club forgets about
simply becomes a free agent at the rollover, where the first club to click "sign" gets him. There
is no reason to watch rivals' expiring contracts, no risk in leaving a renewal for later and no
way for a CPU club to plan next season's squad around a free signing. Real football has the
pre-contract ("Bosman"): in the last months of a contract, other clubs may agree a free move with
the player for the next season.

## What Changes

- **Last-season marker**: every player in the last season of his contract shows a public
  "last season" marker (his exact years and wage stay as they are today: years owner-only).
- **Pre-contract offers**: once his club's league has played half its season, any other club may
  offer a player in his last season a contract for next season (wage and years, no fee). The
  player only considers offers of at least 1.10 × his expected wage (he prefers to stay), keeps
  them pending for 72 hours from the first one and then signs the best (highest wage, then most
  seasons, then the earliest).
- **Right to counter-offer**: his club is told each offer's buyer, wage and years and when the
  player will decide, and is reminded 24 hours before. It can counter by renewing him: while
  offers are pending he stays for 10% less than the best rival offer (best wage ÷ 1.10, never
  below his usual expectation). A renewal, sale or loan cancels every pending offer. Once he signs
  a pre-contract, he can no longer be renewed, listed, loaned, offered for or sold, and he joins
  the new club at his country's next rollover with the agreed terms, no premium to his old club.
- **CPU buyers**: a Manager or Director scans the best players in their last season at each core
  position and makes pre-contract offers on those it would sign, scored like a free signing, at
  exactly 1.10 × the player's expected wage; at most 1 offer per pass and at most 2 signed
  pre-contracts at a time.
- **CPU sellers**: a Manager or Director that receives a pre-contract offer on a player it keeps
  counters on its next offer sweep by renewing him at the raised wage, up to 1.5 × his skill-based
  wage for a starter (2 × for a Director's key player) and 1.25 × for anyone else; beyond that, or
  for a player it would not keep, it tries to sell him (see rework-contract-terms "Selling
  contracts it will not renew").

Out of scope:
- Fees, release clauses or sell-on percentages on a pre-contract (see add-release-clauses).
- Pre-contracts for players on loan (the parent club's contract keeps ticking during a loan and
  never lapses mid-loan; they can be approached once back).
- An automatic transfer-window calendar; pre-contract offers follow the existing window flag.

Open questions for the owner:
- Half of the league season and 1.10 × the expected wage are first guesses; should the start be
  earlier (a third) to give clubs more time?
- Should a pre-contract also be allowed while the transfer window is closed (it is an agreement,
  not a move)? The default is no, for one consistent rule.
- Is "stays for 10% less than the best rival offer" the right home advantage? It could later
  depend on the player's intelligence (a Mastermind gives his club less of a discount).
- Should a rival be allowed to improve its offer after the owning club counters? The default is
  no: a renewal ends the race.

## Capabilities

### New Capabilities

- `pre-contracts`: offering, signing and completing pre-contracts, the last-season marker, the
  restrictions on a pre-contracted player and the notifications.

### Modified Capabilities

- `cpu-transfer-market`: new "Pre-contract targets" requirement.
- `cpu-team-management`: new "Answering pre-contract offers" requirement.

## Impact

- Schema: `resources/sql/init104_pre_contracts.sql` (`pre_contract_offers`).
- Backend: new `common/precontracts.go`; `common/contracts.go` (`AdvancePlayerContract` completes
  a signed pre-contract; renewal and release refuse a pre-contracted player); listing, loan and
  offer functions refuse him; `engine/tasks.go` (`OfferResponseJob` decides pending pre-contracts);
  `engine/aitransfers.go`, `engine/aiteams.go` (CPU buyer and seller); handlers and routes.
- New notification type `NOTIFICATION_PRE_CONTRACT` and player transfer kind
  `PLAYER_TRANSFER_PRE_CONTRACT`. No TRANSACTION_* reason: no money moves.
- Frontend: "Last season" chip, a pre-contract dialog, pre-contracts in the Offers tab, a
  "Leaving: …" chip in the squad list.
