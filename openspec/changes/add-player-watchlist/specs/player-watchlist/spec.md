# Spec Delta

## Purpose

Lets a manager follow players of other clubs, see their market status in one place and be told
when one becomes available or moves, without anyone else learning whom he follows.

## ADDED Requirements

### Requirement: Following players
A club's owner SHALL be able to follow and unfollow any player who is not in his own squad, up to 30 followed players. Following an already followed player, or unfollowing one who is not followed, MUST succeed without change.

#### Scenario: Follow
- **WHEN** a manager follows a striker of another club
- **THEN** the striker appears in his watchlist

#### Scenario: Limit
- **WHEN** a manager who follows 30 players tries to follow another
- **THEN** the request fails with a limit error and the watchlist is unchanged

#### Scenario: Own player
- **WHEN** a manager tries to follow a player of his own squad
- **THEN** the request fails with a validation error

### Requirement: Watchlist view
The transfer market SHALL have a Watchlist tab listing every followed player with his club (or "free agent"), his transfer listing price, his loan listing terms and the actions available to the owner for that status: make an offer, buy, take on loan or sign. A followed player who joins the owner's club MUST leave the watchlist.

#### Scenario: Listed player
- **WHEN** a followed player is transfer-listed at 500,000
- **THEN** his row shows "Transfer listed · 500,000" and a Buy action

#### Scenario: Signed by the follower
- **WHEN** the follower buys a followed player
- **THEN** the player is removed from his watchlist

### Requirement: Watchlist notifications
Each follower SHALL get one `NOTIFICATION_WATCHLIST` when a followed player is transfer-listed, loan-listed, becomes a free agent or moves to another club (transfer, accepted offer, loan or buy option). A follower MUST NOT be notified of an event caused by his own club.

#### Scenario: Player listed
- **WHEN** a club lists a player that two other managers follow
- **THEN** each of them gets a watchlist notification linking to the player

#### Scenario: Player released
- **WHEN** a followed player is released and becomes a free agent
- **THEN** his followers are told he can be signed

#### Scenario: Own club's sale
- **WHEN** a manager follows a player and then buys him
- **THEN** that manager gets no watchlist notification for the move

### Requirement: Watchlist privacy
Who follows a player SHALL be visible only to the follower. No API response, notification or page MUST reveal to a club, or to anyone else, which clubs follow a player or how many.

#### Scenario: Selling club
- **WHEN** a club lists a player followed by 5 managers
- **THEN** nothing it can see tells it that the player is followed
