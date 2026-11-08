# Spec Delta

## MODIFIED Requirements

### Requirement: Player demands
A player SHALL only consider a pre-contract offer whose wage is at least his move premium × his expected wage - 1.05, 1.075, 1.10, 1.15 or 1.20 for a Naive, Casual, Aware, Shrewd or Mastermind player - and whose length is at most his age's maximum term; an offer below that MUST be refused at once, reporting the wage he would accept.

#### Scenario: Below the premium
- **WHEN** an Aware player's expected wage is 2,000 and the offer is 2,100
- **THEN** no offer is created and he reports 2,200

#### Scenario: Mastermind
- **WHEN** a Mastermind player's expected wage is 2,000
- **THEN** he only considers offers of 2,400 or more

### Requirement: Right to counter-offer
Before the player decides, his club SHALL be told each offer's buyer, wage and years and the decision time, and MAY counter by renewing him. While offers are pending, his renewal expectation is at least the best pending wage ÷ his move premium, rounded up. A renewal, sale or loan cancels every pending offer and tells each buyer, without the renewal's terms.

#### Scenario: Matching a rival
- **WHEN** an Aware player's expected wage is 2,000 and the best pending offer is 3,300
- **THEN** his club must offer at least 3,000 to renew him

#### Scenario: Renewed in time
- **WHEN** his club renews him at 3,000, 48 hours after the first offer
- **THEN** every pending offer is cancelled and each buyer is told he stayed

#### Scenario: Better offer during the wait
- **WHEN** a second club offers 3,520 after the first offered 3,300
- **THEN** his club is told, the bar to keep him rises to 3,200 and the decision time does not move

#### Scenario: No counter
- **WHEN** his club does not renew him within the 72 hours
- **THEN** he signs the best pending offer
