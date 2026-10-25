# Proposal

## Why

The flag shown next to a player (squad list, transfer market, player profile) or a team (team page,
dashboard live-match card) comes from `frontend/src/utils/countryFlag.ts`, which maps a country code
to a flag **emoji**. Some of those flags are wrong, and the approach itself is unreliable:

- England (`en`), Scotland (`sco`) and Wales (`wal`) all render the United Kingdom flag 🇬🇧, so a
  Scottish player shows the Union Jack instead of the Saltire. They are three distinct countries in
  `common.KnownCountries`, each with its own national team, yet the UI cannot tell them apart.
- The code comment claims the UK flag is "supported everywhere". It is not: Windows ships no flag
  emoji at all, so every flag there degrades to two letters ("ES", "GB"). Kosovo (🇽🇰) also
  depends on the platform font.
- The emoji route cannot express the home-nation flags in a portable way (they are multi-codepoint
  tag sequences), which is why the mapping fell back to the UK flag in the first place.

## What Changes

- Audit all 63 country codes of `KnownCountries` against the flags shown, and correct every
  mismatch (known: `en`, `sco`, `wal`).
- Replace the emoji with image flags (SVG) rendered by one shared component, so every country,
  including England, Scotland, Wales and Kosovo, has its own correct flag on every platform.
- Use that component in all five places that show a flag today (player profile, squad list,
  transfer market player cell, team page header, dashboard suggested live match), keeping the
  localized country name as tooltip / accessible label.
- Show flags where none appear today (owner decision): every national team name (national teams
  page, World Cup, cup ties, team matches, call-ups) and every player name whose row has the
  player's birth country (lineup editors, ratings, market, recent transfers, player popover). Rows
  that carry no birth country today (search results, top scorers, latest evolutions) gain a
  `birthCountry` field in their API payload.
- Add a test that fails when a `KnownCountries` code has no flag, so a future country cannot ship
  without one.
- **BREAKING** (internal only): `countryFlag()` no longer returns an emoji string; callers switch
  to the component.

## Capabilities

### New Capabilities

- `country-flags`: which flag the UI shows for a country code (players' birth country, clubs' and
  national teams' country), including the home nations and Kosovo, and how it degrades when a code
  is unknown.

### Modified Capabilities

None.

## Impact

- Frontend: `frontend/src/utils/countryFlag.ts` (replaced by a flag component + code map), the five
  existing call sites, the new call sites listed in design.md, and new tests.
- Backend (additive JSON field only, no SQL migration): `birthCountry` on `ScorerStanding`
  (`common/topscorers.go`), `PlayerSearchResult` (`common/search.go`) and `TeamPlayerEvolution`
  (`common/playerprogression.go`).
- Dependency / assets: a flag SVG set (see design.md); no locale change.
- Out of scope: changing how a player's `birthCountry` is generated, per-club crests, and a flag on
  a club (non-national) team name in lists, since a club row carries no country today.

## Decisions taken

- Owner approved SVG flags instead of emoji, and flags in the places that show none today.
