# Spec Delta

## ADDED Requirements

### Requirement: Icon style
Icons SHALL use the outlined style. A filled glyph SHALL appear only for a selected or current state (navigation entries, selected chips) or where the glyph has no outlined form in use (status icons in alerts). Two-tone icons MUST NOT be used.

#### Scenario: Dashboard card headers
- **WHEN** a player opens the dashboard
- **THEN** the card header icons (next match, last result, top scorer, sponsor deal…) are outlined glyphs

#### Scenario: No two-tone icon left
- **WHEN** the frontend is built
- **THEN** no source file imports a two-tone icon, and the icon style test fails if one does
