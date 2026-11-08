# Design

## Context

See proposal.md for the motivation. What exists today and constrains the approach:

- `frontend/src/theme.ts` builds both themes from `themeFromSourceColor(argbFromHex('#4CAF50'))`
  (legacy `Scheme`). `buildPalette` copies 22 roles into the MUI palette (augmented with
  `primaryContainer`, `onPrimaryContainer`, … `inversePrimary`, `rating`), with
  `background.default = background`, `background.paper = surface`.
- Elevation is faked: `MuiPaper.elevation1` drops the shadow and sets
  `mixHex(paper, primary, 0.05 | 0.10)`. `mixHex` is also used by `buildRatingScale`.
- `@material/material-color-utilities` 0.4.0 ships `DynamicScheme`, `TonalPalette`, `Hct`,
  `Variant` and the preset `SchemeTonalSpot`, whose getters include every surface-container role.
- Pages read roles through `theme.palette.*` only in a handful of places (`errorContainer` /
  `onErrorContainer` 3×, `primaryContainer` 2×, `outline` 2×, `palette.rating` through
  `ratingTierColor`). 68 `Paper`s are `variant="outlined"`, the rest use the default elevation.
- No unit test covers `theme.ts` today.

## Goals / Non-Goals

**Goals:**
- Same brand colours, plus the surface-container roles.
- Only `theme.ts` changes; every page picks the new surfaces up through MUI.
- Contrast is a tested property, not a review item.

**Non-Goals:**
- Component restyling (change `complete-m3-component-theme`).
- A custom role name scheme: the MUI palette keeps the M3 role names it already uses.

## Decisions

### 1. A `DynamicScheme` built from the legacy core palettes, not `SchemeTonalSpot`

Measured with the seed `#4CAF50`:

| role (light)         | legacy `Scheme` | `SchemeTonalSpot` | chosen |
|----------------------|-----------------|-------------------|--------|
| primary              | `#006e1c`       | `#3b6939`         | `#006e1c` |
| primaryContainer     | `#94f990`       | `#bcf0b4`         | `#94f990` |
| onPrimaryContainer   | `#002204`       | `#235024`         | `#005313` |
| surface              | `#fcfdf6`       | `#f7fbf1`         | `#f9faf4` |

`SchemeTonalSpot` caps primary chroma at 36, which turns the brand green into a muted olive.
The chosen scheme is `new DynamicScheme({ sourceColorHct, variant: Variant.TONAL_SPOT, isDark,
contrastLevel: 0, primaryPalette, secondaryPalette, tertiaryPalette, neutralPalette,
neutralVariantPalette })` with the palettes the legacy `CorePalette` uses: primary
`(hue, max(48, chroma))`, secondary `(hue, 16)`, tertiary `(hue + 60, 24)`, neutral `(hue, 4)`,
neutral variant `(hue, 8)`. Role tones (including the surface containers) then follow the current
M3 dynamic-colour rules while hues and chromas stay the app's.

*Alternatives considered:* `SchemeTonalSpot` (off-brand, above); `SchemeFidelity` /
`SchemeContent` (keep the seed's chroma but derive tertiary from a temperature analogue, which
changes the tertiary hue the app uses for its accent chips); keeping the legacy `Scheme` and
computing surface containers by hand from the neutral palette (duplicates the library's tone
tables, and drifts on the next M3 revision).

### 2. Palette mapping

- `background.default = surface`, `background.paper = surface`. A raised `Paper` gets
  `surfaceContainerLow` through the `MuiPaper.elevation1` override (shadow stays `none`), so an
  outlined `Paper` keeps the page colour, as the spec's surface hierarchy requires.
- New augmented palette fields: `surfaceDim`, `surfaceBright`, `surfaceContainerLowest`,
  `surfaceContainerLow`, `surfaceContainer`, `surfaceContainerHigh`, `surfaceContainerHighest`,
  `onSurface`. Existing fields keep their names and meaning; `surfaceVariant` is kept (M3
  deprecated it, but `MuiButton`'s disabled style reads it) and maps to
  `surfaceContainerHighest`, its documented replacement.
- `mixHex` stays, only for `buildRatingScale`. The rating scale is built from `outline` and
  `primary`, which do not change, so `palette.rating` stays identical.

### 3. A pinned colour test

`frontend/src/theme.test.ts` reads both themes and asserts:
- the exact hex values the spec's scenarios list (primary, primary container, the five surface
  containers, surface, outline variant);
- WCAG 2.x contrast ≥ 4.5:1 for every `on*` pair and `onSurface`/`onSurfaceVariant` over each
  surface level, and ≥ 3:1 for `outline` over `surface`, using a small local
  `contrastRatio(hexA, hexB)` helper (no new dependency).

Pinning hexes makes an accidental library bump that shifts tones fail loudly instead of silently
restyling the app.

## Risks / Trade-offs

- [Raised panels lose their green tint and read slightly greyer] → Called out as owner question 1
  in the proposal; the screenshots of the M3 plan show the result.
- [`on*Container` text gets lighter (tone 30 instead of 10)] → Contrast stays ≥ 7.2:1 for every
  pair; the live hero (team-dashboard spec) keeps the same role pair.
- [A future `@material/material-color-utilities` release changes tone tables] → The pinned-hex
  test fails, and the bump becomes a deliberate change.

## Migration Plan

Frontend-only and stateless: ships with the next frontend deploy, rolls back by reverting the
commit. The stored theme-mode preference (`localStorage`) is unaffected.
