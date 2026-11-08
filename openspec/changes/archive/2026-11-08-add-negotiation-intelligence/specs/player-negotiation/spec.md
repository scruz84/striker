# Spec Delta

## Purpose

How a player's hidden negotiation intelligence shapes what he asks, and how a club learns it.

## ADDED Requirements

### Requirement: Skill trend in negotiations
The skill trend a player reacts to when negotiating SHALL be the average change of his seven core skills over his last 5 progression runs, rounded half away from zero; a player with no run has a trend of 0.

#### Scenario: One good match
- **WHEN** a player's last 5 runs changed his skills by +10, 0, 0, 0 and 0
- **THEN** his trend is +2

#### Scenario: Fewer runs
- **WHEN** a player has 2 runs of +3 and +4
- **THEN** his trend is +4 (3.5 rounded)

### Requirement: Negotiation insight
A club SHALL see a player's negotiation tier (Naive, Casual, Aware, Shrewd, Mastermind) on his page once it has negotiated with him - a renewal, a free-agent signing, a transfer offer, a pre-contract offer or a clause payment that he judged, accepted or not - and MUST NOT see it before, nor see another club's insight.

#### Scenario: After a refused offer
- **WHEN** a manager's offer is refused on personal terms by a Shrewd player
- **THEN** his page shows "Negotiator: Shrewd" to that manager

#### Scenario: Another club
- **WHEN** a manager who never negotiated with him opens the same page
- **THEN** no negotiator row is shown
