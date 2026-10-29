# transfer-comparables Specification

## Purpose

Gives a manager making a transfer offer or listing one of their own players a reference price:
recent completed transfers of players similar to the target, built only from information that is
public for every player.

## Requirements

### Requirement: Transfer snapshot
Every recorded player transfer SHALL store the moved player's best core position (goalkeeper, defense, midfield or striker), age and average skill (0-99) as they were at the moment of the transfer. Transfers recorded before this capability existed have no snapshot and MUST never be returned as comparables.

#### Scenario: Snapshot frozen at sale
- **WHEN** a 24-year-old striker with average skill 61 is bought, and later turns 25 and reaches average skill 66
- **THEN** that transfer still compares as a 24-year-old striker with average skill 61

#### Scenario: Old transfer without snapshot
- **WHEN** a transfer row predates the snapshot
- **THEN** it never appears in any comparables list

### Requirement: Eligible transfers
A transfer SHALL be eligible as a comparable only if it is a bought listing, an accepted offer or an exercised loan buy option, its fee is above 0, it has a snapshot, it happened within the last 60 days, and it is not a transfer of the target player himself. Loans, free-agent signings (and their premiums) and releases MUST never be comparables.

#### Scenario: Free-agent premium ignored
- **WHEN** a free agent was signed paying a 40,000 premium to his last club 3 days ago
- **THEN** that signing is not a comparable

#### Scenario: Too old
- **WHEN** a matching transfer happened 61 days ago
- **THEN** it is not a comparable

#### Scenario: Target's own past transfer
- **WHEN** the target player himself was sold 10 days ago
- **THEN** that sale is not listed among his comparables

### Requirement: Similarity bands
A comparable SHALL share the target's current best core position and fall within a band of his current age and average skill. The narrow band is used first; only if it yields fewer than 3 transfers, the wide band is used instead and the result is flagged as broadened.

#### Scenario: Narrow band
- **WHEN** the target is a 26-year-old midfielder with average skill 58
- **THEN** comparables are midfielders aged 24-28 at sale with average skill 54-62 at sale (age ±2, average ±4)

#### Scenario: Broadened search
- **WHEN** the narrow band finds only 2 transfers for that midfielder
- **THEN** the wide band is used, midfielders aged 22-30 with average skill 50-66 (age ±4, average ±8), and the result is flagged broadened

#### Scenario: Different position
- **WHEN** a 26-year-old defender with average skill 58 was sold yesterday
- **THEN** he is never a comparable for that midfielder

### Requirement: Ordering and size
Comparables SHALL be returned newest first, ties broken by the higher transfer id, capped at 8 entries.

#### Scenario: More than 8 matches
- **WHEN** 12 transfers match the band
- **THEN** only the 8 most recent are returned

### Requirement: Public fields only
Each comparable SHALL expose only: the player's id and name, the snapshot position, age and average skill, the fee, the transfer date, the transfer kind, and the selling and buying clubs (id and name). It MUST NOT expose the player's individual skill ratings, traits, stamina, form, current wage or contract, at sale or now.

#### Scenario: Comparable owned by another human
- **WHEN** a comparable now plays for another human's club and is not listed
- **THEN** the response contains no skill rating, trait, stamina, form, wage or contract figure for him

### Requirement: Summary figures
Alongside the list, the response SHALL give the number of comparables and the median, lowest and highest fee among them. With an even count the median is the mean of the two middle fees, rounded down to a whole unit. With no comparables the summary figures are absent.

#### Scenario: Median of four
- **WHEN** the comparables' fees are 100,000, 120,000, 150,000 and 400,000
- **THEN** the count is 4, the median 135,000, the lowest 100,000 and the highest 400,000

### Requirement: Offer dialog section
The transfer offer dialog SHALL show a "Recent comparable transfers" section below the fee field while the offer has not been sent: the summary line, then one row per comparable (name linking to the player page, position, age, average skill, fee, from → to club, date). Broadened results and an empty result MUST each show an explanatory note.

#### Scenario: No comparables
- **WHEN** the endpoint returns no comparables
- **THEN** the section reads "No similar players have been transferred in the last 60 days" and the offer can still be sent

#### Scenario: Broadened note
- **WHEN** the result is flagged broadened
- **THEN** the section notes that the search was widened because few close matches exist

#### Scenario: Offer sent
- **WHEN** the selling club receives the offer (the player accepted the personal terms)
- **THEN** the section is hidden along with the fee and terms fields

#### Scenario: Failure does not block bidding
- **WHEN** loading the comparables fails
- **THEN** the section shows a short error and the offer form keeps working

### Requirement: Comparables endpoint
`GET /v1/teams/player/:id/transfer-comparables` SHALL require a signed-in user and return the target's reference values (position, age, average skill), the broadened flag, the summary and the comparables. An unknown player id MUST return 404 and a non-numeric id 400.

#### Scenario: Unknown player
- **WHEN** the endpoint is called for a player id that does not exist
- **THEN** it responds 404

## Code map

- `backend/resources/sql/init88_player_transfer_snapshot.sql` (snapshot columns, partial index).
- `backend/common/playertransfers.go` (`RecordPlayerTransfer` snapshot, `GetTransferComparables`,
  `getTransferComparablesStmtSql`, `TransferComparable`, `TransferComparablesResult`,
  `SummarizeComparables`, `Comparables*` band / window / limit constants).
- `backend/handlers/playerHandlers.go` (`TransferComparablesHandler`),
  `backend/service/fmservice.go` (route `GET /v1/teams/player/:id/transfer-comparables`).
- Tests: `backend/common/transfer_comparables_test.go` (summary, statement literals, no private JSON
  field), `transfer_comparables_integration_test.go` (every spec scenario, snapshot at the move,
  `TestTransferComparables_UsesIndex`), `backend/handlers/transferComparablesHandler_test.go`.
- `frontend/src/components/TransferComparables.tsx` (`errorText` prop), `TransferOfferDialog.tsx`,
  `ListPlayerForTransferDialog.tsx`; `frontend/src/features/transfers/ListPlayerDialog.tsx`;
  `frontend/src/api/hooks.ts` (`useTransferComparables`), `api/types.ts`; strings under
  `transfers.offerDialog.comparables.*` in `i18n/locales/{en,es}.json`.
