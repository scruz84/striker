# Spec Delta

## MODIFIED Requirements

### Requirement: Matchday gate
After a league match or cup tie the home side alone SHALL receive attendance × ticket price. Attendance MUST follow the socios model (see the socios spec) and never exceed stadium capacity. Ticket prices are integers from 1 to 39, default 10: 39 is the last price at which any socio still attends.

#### Scenario: Away side
- **WHEN** a league match is played
- **THEN** the away side receives no gate money

#### Scenario: Price outside the range
- **WHEN** an owner sets a ticket price of 0 or of 40 or more
- **THEN** the request fails with `ErrInvalidTicketPrice` (HTTP 400) and the stored price is unchanged

#### Scenario: Highest allowed price
- **WHEN** an owner sets a ticket price of 39
- **THEN** the price is stored and the next home match draws socios × 1/30 × the other factors

#### Scenario: Legacy price above the range
- **WHEN** the migration runs and a team's stored ticket price is 69
- **THEN** its price becomes 10, its next home match earns a gate, and a team already at 1–39 keeps its price

## ADDED Requirements

### Requirement: Ticket price guidance
The finance page SHALL show the allowed ticket price range (1–39) and a recommended price: the price that maximises expected gate revenue for the team's capacity, socios and recent league form, the same price a CPU Manager team would charge. A price outside the range MUST be blocked before it is sent.

#### Scenario: Members cannot fill the ground
- **WHEN** a team with 30,000 seats and 500 socios opens the price dialog
- **THEN** the recommended price shown is at most 25

#### Scenario: More members than seats
- **WHEN** a team with 1,000 seats and 3,000 socios opens the price dialog
- **THEN** the recommended price shown is above 10 and is the price that just fills the ground

#### Scenario: Out-of-range input
- **WHEN** the owner types 50 in the price dialog
- **THEN** the field shows the 1–39 range as an error and Save is disabled
