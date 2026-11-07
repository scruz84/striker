-- Regional (international) club cup entries (see openspec/specs/international-cup and
-- engine.StartRegionalCupsIfDue): which domestic season's tier-1 champion each country sent to each
-- edition. A country is "fresh" for the next edition only once it has a champion from a newer
-- season than the newest one recorded here.
--
-- Re-runnable: the table and index are IF NOT EXISTS and the backfill skips rows already present.

CREATE TABLE IF NOT EXISTS international_cup_entries (
    cup_id  bigint NOT NULL REFERENCES cups(cup_id) ON DELETE CASCADE,
    country varchar(4) NOT NULL,
    season  int NOT NULL,
    team_id bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    PRIMARY KEY (cup_id, country)
);

CREATE INDEX IF NOT EXISTS international_cup_entries_country_idx ON international_cup_entries (country, season);

-- Editions created before this table existed: each round-1 entrant of a regional cup (the World
-- Cup knockout, region 'world', has national teams instead) is recorded with its country and the
-- newest season it won a tier-1 league - the champion that country sent.
INSERT INTO international_cup_entries (cup_id, country, season, team_id)
SELECT DISTINCT ON (c.cup_id, l.country) c.cup_id, l.country, sh.season, sh.team_id
FROM cups c
JOIN cup_ties ct ON ct.cup_id = c.cup_id AND ct.round = 1
JOIN season_history sh ON sh.team_id IN (ct.home_team_id, ct.away_team_id) AND sh.final_position = 1
JOIN leagues l ON l.league_id = sh.league_id AND l.tier = 1
WHERE c.region IS NOT NULL AND c.region <> 'world'
ORDER BY c.cup_id, l.country, sh.season DESC
ON CONFLICT (cup_id, country) DO NOTHING;
