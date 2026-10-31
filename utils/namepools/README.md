# Name pools

One directory per country code (every known country except `es`, whose ~29k-name dump predates
this layout): `first.txt` (at least 300 distinct male given names) and `last.txt` (at least 500
distinct surnames), one name per line, UTF-8, Latin script (romanized for Japan, Korea, Greece,
the Arab world, ...).

`python3 utils/gen_name_pools.py` validates them and writes
`backend/resources/sql/namepools/init91_name_pool_<code>.sql`, the idempotent migration the backend
applies at startup (see the `country-catalog` OpenSpec spec). Edit the text files, run the
generator, commit both. `--check` fails if a generated file is stale.

## Where the names came from

`utils/build_name_pool_sources.py` documents and reproduces the first build (the header describes
the method: the pools the repository already had, the hand-written seeds in
`utils/name_pool_seeds.py`, the MIT-licensed `names-dataset` package for the countries it covers,
and blends of culturally close countries for the rest). The lists are a starting point, not
ground truth:

- Countries without a dataset of their own (most small nations and territories) are blends of
  neighbours or of the cultures their footballers come from, so two neighbours can share many
  names. Each country takes a different hash-selected subset of its donors.
- Migrant-heavy countries still carry some names of foreign origin; that is realistic for a few
  (Russia, Gulf states) and noise for others. Spot-check a country before relying on it and trim
  it by hand.
- Vietnam and Thailand are composed from syllables, and the Korean surname list pads the ~100
  real surnames with surname + clan-seat pairs, because those languages do not have 500 distinct
  surnames to draw from.
- Female surname forms (Slavic `-ova`, Icelandic `-dóttir`, ...) are dropped: the draw is for
  male footballers.
