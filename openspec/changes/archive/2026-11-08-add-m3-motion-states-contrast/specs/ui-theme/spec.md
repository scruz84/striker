# Spec Delta

## ADDED Requirements

### Requirement: Motion
Animated changes SHALL use M3 easing and durations: standard `cubic-bezier(0.2, 0, 0, 1)` for changes in place, emphasized decelerate `cubic-bezier(0.05, 0.7, 0.1, 1)` over 400 ms for surfaces entering, and emphasized accelerate `cubic-bezier(0.3, 0, 0.8, 0.15)` over 200 ms for surfaces leaving.

#### Scenario: Dialog opens
- **WHEN** a dialog, drawer or bottom sheet opens
- **THEN** it enters over 400 ms on the emphasized decelerate curve, and leaves over 200 ms on the emphasized accelerate curve

### Requirement: Reduced motion
When the system asks for reduced motion, every transition and animation SHALL complete at once.

#### Scenario: Reduce motion on
- **WHEN** a player with "reduce motion" on opens the navigation drawer or watches a LIVE chip
- **THEN** the drawer appears without sliding and the chip does not pulse

### Requirement: State layers
Interactive elements SHALL show M3 state layers over their content colour: 8 % on hover, 10 % on focus and on press. Inactive icons SHALL use `onSurfaceVariant`.

#### Scenario: Hovering a list item
- **WHEN** the pointer rests on a navigation drawer item in light mode
- **THEN** its background is `#1a1c19` at 8 % opacity

### Requirement: High contrast
When the system asks for more contrast, the app SHALL switch to M3's high-contrast scheme from the same seed, in light and dark mode. In that scheme every `on*` role over its paired role, and `onSurface` / `onSurfaceVariant` over every surface level, MUST reach 7:1, and the `outline` role MUST reach 4.5:1 against the page.

#### Scenario: Increase contrast in light mode
- **WHEN** a player with "increase contrast" on opens the app in light mode
- **THEN** the primary role is `#003509` and secondary text (`onSurfaceVariant`) is `#000000`

#### Scenario: Increase contrast in dark mode
- **WHEN** the same player switches to dark mode
- **THEN** the primary role is `#c5ffbc` and every text pair reaches at least 7:1

#### Scenario: No preference
- **WHEN** the system has no contrast preference
- **THEN** the standard scheme of the `Colour roles from one seed` requirement applies
