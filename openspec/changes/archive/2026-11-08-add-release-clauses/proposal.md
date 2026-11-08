# Proposal

## Why

Once a player is under contract, only his club decides whether he leaves: a buyer can make an
offer, the seller can reject it forever, and a renewal is a single number (the wage) with no
trade-off. Release clauses ("cláusula de rescisión") add a real decision to every contract: a
club can save wages by accepting a clause the player likes, at the risk that any club pays it and
takes him without asking. They also give buyers, human and CPU, a way to reach a player whose
club will not sell.

## What Changes

- **Optional clause**: a renewal, a free-agent signing, a transfer offer's personal terms and a
  pre-contract may carry a release clause of at least 1.5 × the player's market value on a
  3-season contract. The clause is public.
- **Wage discount**: a player accepts 10% less than his expected wage with a clause of at most 2 ×
  that value, 5% less with one of at most 4 ×, and no discount above.
- **Paying the clause**: while the window is open, any solvent club with the budget may pay a
  player's clause, 14 days or more after it was set, with personal terms he accepts; the player
  moves at once, without the selling club's consent, and the fee goes to it.
- **CPU sellers**: a Manager sets a clause of 2 × its valuation when it renews a non-starter (10%
  cheaper wage); a Director sets 4 × its valuation on players who are neither key players nor
  internationals (5% cheaper wage); a Caretaker never sets one.
- **CPU buyers**: a Director treats a player whose clause is at most 1.5 × its valuation as
  buyable at the clause, like a listed player, at most once per pass; no CPU club pays a clause on
  a human club's player if a CPU club paid one on that club in the last 7 days.

Out of scope:
- Sell-on percentages, instalments and player swaps.
- Clauses on loans or loan buy options.
- Renegotiating a clause without a renewal.

Open questions for the owner:
- Are 1.5 × (minimum), 2 × / 4 × (discount bands) and 10% / 5% the right figures? The discount
  must be worth a real risk; tune with `ECON_SIM=1`.
- Should a human be able to protect a player from CPU clause payments entirely? The default is
  the 7-day rate limit only: the human chose the clause.
- This change assumes add-pre-contracts for the pre-contract clause; without it, that part is
  dropped.

## Capabilities

### New Capabilities

- `release-clauses`: setting a clause, the wage discount, paying it, its visibility and
  notifications.

### Modified Capabilities

- `cpu-transfer-market`: new "Paying release clauses" requirement.
- `cpu-team-management`: new "Release clauses in CPU renewals" requirement.

## Impact

- Schema: `resources/sql/init105_release_clauses.sql` (`players.release_clause`,
  `players.release_clause_set_at`, `transfer_offers.offered_release_clause`,
  `pre_contract_offers.release_clause`).
- Backend: new `common/releaseclauses.go` (`PayReleaseClause`, discount and minimum); renewal,
  free-agent, offer and pre-contract functions accept an optional clause; `engine/aiteams.go`
  (CPU renewals), `engine/aitransfers.go` (`candidateReleaseClause`); handlers and routes.
- Player transfer kind `PLAYER_TRANSFER_RELEASE_CLAUSE`; the money uses `TRANSACTION_TRANSFER_OUT`
  / `TRANSACTION_TRANSFER_IN` (no new reason). Notifications use `NOTIFICATION_TRANSFER`.
- Frontend: a clause field in the contract terms, the clause on the player page and market rows,
  a "Pay release clause" dialog.
