# Spec Delta

## MODIFIED Requirements

### Requirement: Club reset pending business
Resetting a club SHALL cancel its open transfer and loan listings, every pending transfer offer it made or received, and every pending friendly challenge it sent or received, and MUST release its hired trainer to the trainer market and clear its watchlist. Already scheduled matches and friendlies MUST still be played by the reset club.

#### Scenario: Offer for one of its players
- **WHEN** another club has a pending offer for a player of the reset club
- **THEN** the offer disappears and the player can be signed as a free agent instead

#### Scenario: Scheduled friendly
- **WHEN** the reset club has a friendly scheduled for tomorrow
- **THEN** the friendly is played by the reset club's new squad

#### Scenario: Watchlist
- **WHEN** a club that followed 12 players is reset
- **THEN** its watchlist is empty

### Requirement: Data export
A signed-in user SHALL be able to download, from the Account page, a JSON file with all personal data stored about them: profile (name, email, provider, country, language, created and last-active times, terms acceptance), their club summary, notification preferences, push subscription endpoints, groups, notifications and their club's watchlist.

#### Scenario: Download my data
- **WHEN** a user clicks "Download my data"
- **THEN** their browser saves `striker-my-data.json` containing their email and their notifications

#### Scenario: Watchlist in the export
- **WHEN** a user who follows 3 players downloads his data
- **THEN** the file lists those 3 players
