# Proposal

## Why

CPU clubs have no answer to money trouble. A Manager or Director renews every expiring contract,
including bench veterans, and on a second try accepts whatever raise the player asks for. When a
club falls into debt it keeps shopping until the insolvency lockout (budget below −200,000) stops
it, and from then on nothing lists players to dig it back out: only surplus is ever sold. The
`striker.ai.insolvent_teams` gauge added by improve-cpu-ai-foundations makes this visible. This
change gives Managers and Directors the basic financial discipline a human manager has.

## What Changes

- **Financial crisis**: a CPU club is in crisis while its budget is below 0. A Manager or Director
  in crisis signs nobody (no transfer pass), renews only its current XI, and once per management
  pass lists its highest-wage player that can be sold (not a key player, an international or a
  recent signing; minimum depth kept) at 80% of the usual asking price, while it has fewer than 2
  open transfer listings. Any CPU club in crisis, Caretakers included, accepts offers from 80% of
  its usual threshold.
- **Selective renewals**: outside a crisis a Manager or Director renews a player with one season
  or less left only if he is in its current XI, younger than 30, or needed to keep his position at
  minimum depth. The second offer at the player's own terms is only made to a starter, or when
  those terms are at most 1.25 × his skill-based wage.
- **Releasing unsold veterans**: a Manager or Director not in crisis, with more owned players than
  the squad size (18), releases at most one player per pass who is 30 or older, has 2 or more
  seasons left and has been on the transfer or loan list for 14 days without a taker, when its
  budget covers 3 × the severance (10 × wage).

Out of scope:
- Any change for Caretakers beyond the offer discount (they stay reactive).
- A wage-bill ceiling for Managers' signings (the 50% budget reserve stays their only rule).
- Changing the severance, the insolvency line or any other economy constant.

Open questions for the owner:
- Is 0 the right crisis line, or should a club enter it earlier (for example when a forecast says
  it will be insolvent within N weeks)? The default is the simple, visible rule.
- Should a club in crisis also sell key players? The default is no: results would collapse.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-team-management`: "Contract renewals" becomes selective; new "CPU financial crisis",
  "Crisis offer discount" and "Releasing unsold veterans" requirements.
- `cpu-transfer-market`: "Transfer pass" is skipped in a crisis.

## Impact

- Backend: new `engine/aifinances.go`; `engine/aiteams.go` (renewals, offer answers, pass
  order); `common/transferscoring.go` (`OfferContext.Distressed`); `common/loans.go` (loan listing
  times of a team).
- No schema change, no new TRANSACTION_* reason (releases use `TRANSACTION_CONTRACT_RELEASE`
  through `common.ReleasePlayer`), no new notification.
