# Spec Delta

## ADDED Requirements

### Requirement: Listing dialog section
Every dialog that puts one of the manager's own players on the transfer market SHALL show the same "Recent comparable transfers" section below the asking-price field, for the player being listed. It has the same summary line, rows, empty note and broadened note as the offer dialog. When the dialog lets the manager pick the player, the section MUST follow the current pick and stay hidden while no player is picked.

#### Scenario: Listing from the player page
- **WHEN** the owner opens "List for transfer" for his 26-year-old midfielder with average skill 58
- **THEN** below the asking price the section lists recent sales of midfielders aged 24-28 with average skill 54-62, or the wide band flagged broadened if fewer than 3 exist

#### Scenario: Picking in the market dialog
- **WHEN** the owner opens "List a player" on the transfer market
- **THEN** no comparables section shows until he picks a player, and picking another player replaces it with that player's comparables

#### Scenario: Nothing comparable
- **WHEN** the endpoint returns no comparables for the player being listed
- **THEN** the section reads "No similar players have been transferred in the last 60 days" and the player can still be listed

### Requirement: Listing never blocked by comparables
A failure or a slow load of the comparables SHALL NOT stop the manager from listing the player: the asking-price field and the list button keep working. A failed load MUST show a short note saying the player can still be listed. The comparables are requested only while a listing dialog is open with a player chosen.

#### Scenario: Failure while listing
- **WHEN** loading the comparables fails in a listing dialog
- **THEN** the section reads "Could not load comparable transfers. You can still list the player." and listing at the typed price succeeds

#### Scenario: Closed dialog
- **WHEN** the owner views his squad without opening any listing dialog
- **THEN** no comparables request is sent for his players
