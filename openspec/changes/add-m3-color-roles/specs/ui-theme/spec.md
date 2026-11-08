# Spec Delta

## Purpose

The client's visual design system: the Material 3 colour roles every screen draws from, the
contrast each role pair guarantees in light and dark mode, and the surface each kind of container
sits on.

## ADDED Requirements

### Requirement: Colour roles from one seed
Every theme colour SHALL be a Material 3 colour role derived from the seed `#4CAF50`, in a light and a dark scheme. The roles SHALL include `surfaceDim`, `surfaceBright` and five surface-container levels (`Lowest`, `Low`, base, `High`, `Highest`). Domain colours that are not roles (medals, pitch stripes, avatar inks, club crests) are exempt.

#### Scenario: Brand colour unchanged
- **WHEN** the app renders in light mode
- **THEN** the primary role is `#006e1c` and the primary container `#94f990`, as before this change

#### Scenario: Brand colour unchanged in dark mode
- **WHEN** the app renders in dark mode
- **THEN** the primary role is `#78dc77` and the primary container `#005313`

#### Scenario: Surface container levels in light mode
- **WHEN** the app renders in light mode
- **THEN** the surface containers are `#ffffff`, `#f3f4ee`, `#eeeee8`, `#e8e9e2` and `#e2e3dd` from lowest to highest

#### Scenario: Surface container levels in dark mode
- **WHEN** the app renders in dark mode
- **THEN** the surface containers are `#0c0f0c`, `#1a1c19`, `#1e201d`, `#282b27` and `#333531` from lowest to highest

### Requirement: Contrast of role pairs
In both modes, text in an `on*` role over its paired role (primary, secondary, tertiary, error and their containers, and `onSurface`/`onSurfaceVariant` over every surface level) MUST have a contrast ratio of at least 4.5:1. The `outline` role over `surface` MUST reach at least 3:1.

#### Scenario: Text on a container
- **WHEN** text in `onErrorContainer` sits on `errorContainer` in light mode
- **THEN** the contrast ratio is at least 4.5:1 (it is 7.24:1 with `#93000a` on `#ffdad6`)

#### Scenario: Secondary text on the highest surface
- **WHEN** text in `onSurfaceVariant` sits on `surfaceContainerHighest` in light mode
- **THEN** the contrast ratio is at least 4.5:1 (it is 7.20:1)

#### Scenario: Outline on the page
- **WHEN** an outlined control sits on the page background in light mode
- **THEN** its outline reaches at least 3:1 against it (it is 4.28:1)

### Requirement: Surface hierarchy
The page background SHALL use the `surface` role. A raised panel SHALL use `surfaceContainerLow` and an outlined panel SHALL use `surface` with an `outlineVariant` border. Containers MUST NOT be tinted by blending the primary colour into them.

#### Scenario: Raised panel in light mode
- **WHEN** a raised panel is shown in light mode
- **THEN** its background is `#f3f4ee` on a `#f9faf4` page, with no green blended in

#### Scenario: Raised panel in dark mode
- **WHEN** the same panel is shown in dark mode
- **THEN** its background is `#1a1c19` on a `#121411` page

#### Scenario: Outlined panel
- **WHEN** an outlined panel is shown in light mode
- **THEN** its background matches the page and its border is `#c2c9bd`
