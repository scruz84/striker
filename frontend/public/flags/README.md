# Country flags

4:3 SVG flags for every code in `backend/common/countries.go` `KnownCountries`, copied from the
[flag-icons](https://github.com/lipis/flag-icons) package (MIT, see `LICENSE`) v7.6.0 and named by
the game's own country code: `en` = `gb-eng`, `sco` = `gb-sct`, `wal` = `gb-wls`, every other file
is the ISO 3166-1 alpha-2 file of the same name. Served from `/flags/<code>.svg` (see
`src/components/CountryFlag.tsx`); a new country needs its file here.
