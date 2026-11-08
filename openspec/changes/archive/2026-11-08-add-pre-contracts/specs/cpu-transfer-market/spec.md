# Spec Delta

## ADDED Requirements

### Requirement: Pre-contract targets
A Manager or Director SHALL also scan the best players in their last season at each core position (a Director 10 per position, a Manager 5) and score each as a free signing at his offered wage. It MAY make one pre-contract offer per pass, at exactly 1.10 × his expected wage, to a candidate clearing its minimum score, with at most 2 signed pre-contracts, which count toward its squad size cap.

#### Scenario: Director plans next season
- **WHEN** a Director finds a striker in his last season whose expected wage is 3,000 and who scores above its minimum
- **THEN** it offers him a pre-contract at 3,300 and his club is notified

#### Scenario: Two signed already
- **WHEN** a Manager already has 2 signed pre-contracts
- **THEN** it makes no pre-contract offer

#### Scenario: Pile-up on a human's player
- **WHEN** three CPU clubs have pending pre-contract offers on a human's player
- **THEN** no other CPU club offers him one
