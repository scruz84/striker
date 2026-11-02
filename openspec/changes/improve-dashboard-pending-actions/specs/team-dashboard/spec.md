# Spec Delta

## ADDED Requirements

### Requirement: Pending panel
The owner's dashboard SHALL show a Pending panel listing every pending item (see "Pending items") as one row with an icon, a text and one action, ordered error, warning, info, success, then by the order of "Pending items". With no pending item the panel MUST NOT be shown.

#### Scenario: Nothing pending
- **WHEN** an owner has nothing pending
- **THEN** the dashboard shows no Pending panel

#### Scenario: Ordering
- **WHEN** an owner is insolvent, has a received offer and has no trainer
- **THEN** the panel lists insolvency first, then the missing trainer, then the offer

### Requirement: Pending items
The panel SHALL include, when they apply: insolvent (error); insolvency within 4 weeks by the cash-flow forecast (warning); unavailable starters in the next fixture's lineup, with their projected replacements (warning); contracts with 1 season or less left (warning); no trainer hired (warning); ticket price 3 or more away from the recommended price (info); offers received (info); friendly challenges received (info); an academy prospect ready (success).

#### Scenario: Injured starter for the next match
- **WHEN** the lineup for the next league match has an injured left-back whose projected replacement is B
- **THEN** the panel says one starter of that match is unavailable, names B as his replacement and links to the lineup editor

#### Scenario: Injured reserve
- **WHEN** only a player outside the next fixture's XI is injured
- **THEN** no unavailable-starters item is shown

#### Scenario: Forecast
- **WHEN** the cash-flow forecast reaches −maxDebt in 3 weeks
- **THEN** a warning item links to the finance page

#### Scenario: Ticket price
- **WHEN** the ticket price is 15 and the recommended price is 11
- **THEN** an info item shows both prices and opens the ticket price dialog

#### Scenario: No trainer
- **WHEN** the club has no trainer
- **THEN** a warning item links to the coaching tab of the transfer market

### Requirement: Snoozing pending items
Every item except an error SHALL offer "Remind me tomorrow", which hides it in that browser until the next local midnight. A snoozed item MUST come back earlier if its content changes (for example a second offer arrives).

#### Scenario: Snoozed offers
- **WHEN** an owner snoozes "1 offer received" and later a second offer arrives the same day
- **THEN** the item "2 offers received" is shown again

#### Scenario: Insolvency
- **WHEN** an owner is insolvent
- **THEN** the insolvency item offers no snooze

### Requirement: Pending panel on phones
On a phone-width screen the panel SHALL show at most the 3 first items and a "Show all (N)" toggle when there are more.

#### Scenario: Five items on a phone
- **WHEN** an owner with 5 pending items opens the dashboard on a phone
- **THEN** 3 items and "Show all (5)" are shown, and the toggle reveals the other 2
