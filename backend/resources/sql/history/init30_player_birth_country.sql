-- The country a player was born in, independent of which team (and thus which country) they
-- currently play for - the transfer market has no country restriction, so a player generated
-- for one country's world can end up on a team anywhere. Set at creation time from the country
-- whose name pool generated the player (see common.doCreatePlayers); never changed afterward by
-- a transfer, loan, or release. Nullable since it isn't knowable for anyone created before this
-- column existed - backfilled below with each player's current team's country as the best
-- available guess (accurate for anyone who hasn't been transferred across a country boundary,
-- which covers effectively every pre-existing player, since a global transfer market is new).
ALTER TABLE players ADD COLUMN IF NOT EXISTS birth_country varchar(4) REFERENCES countries(name);

UPDATE players SET birth_country = t.country
FROM teams t
WHERE players.team_id = t.team_id AND players.birth_country IS NULL;
