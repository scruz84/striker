# Spec Delta

## ADDED Requirements

### Requirement: Release clauses in CPU renewals
When it renews a player, a Manager SHALL set a release clause of 2 × its valuation for a non-starter and none for a starter; a Director 4 × its valuation unless he is a key player or an international; a Caretaker none. A clause below the minimum is raised to it, and the wage offered is the discounted expected wage.

#### Scenario: Manager bench renewal
- **WHEN** a Manager renews a bench player it values at 300,000, his market value on a 3-season contract, with an expected wage of 1,500
- **THEN** it sets a 600,000 clause and offers 1,350

#### Scenario: Director key player
- **WHEN** a Director renews its key striker
- **THEN** it sets no clause and pays his full expected wage
