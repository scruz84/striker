-- Players who left a club stayed in the lineups it had saved for matches still to come (issue
-- #399): a sold player started for his old club. common.removePlayerFromTeamLineupsTx now takes
-- him out when he leaves, and the kickoff refill replaces anyone in a lineup who is not in the
-- squad. This removes what an older build left behind, from the standing lineups and from the
-- lineups of matches not played or picked up by a simulation yet. Played matches keep their
-- lineups: they record who played.
--
-- friendly_lineup_substitutions was created by init59, which was folded into history/ without
-- reaching init.sql, so a database created from the consolidated baseline lacks it (saving a
-- friendly lineup with substitutions fails there). It is created here when missing, with the
-- shape init59 gave it.
--
-- Re-runnable: CREATE TABLE IF NOT EXISTS, and each DELETE/UPDATE only matches stale rows.

CREATE TABLE IF NOT EXISTS friendly_lineup_substitutions (
    match_id      bigint NOT NULL REFERENCES friendly_matches(match_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (match_id, team_id, slot)
);

-- Standing lineups of clubs (a national team's players belong to their clubs, so national teams
-- are left alone).
DELETE FROM default_lineup_positions d USING players p
WHERE p.player_id = d.player_id AND p.team_id IS DISTINCT FROM d.team_id
  AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.team_id = d.team_id AND t.is_national_team);
DELETE FROM default_friendly_lineup_positions d USING players p
WHERE p.player_id = d.player_id AND p.team_id IS DISTINCT FROM d.team_id
  AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.team_id = d.team_id AND t.is_national_team);

-- Lineups of league matches, cup ties and friendlies still to be played.
DELETE FROM lineup_positions lp USING lineup l, matches m, players p
WHERE lp.lineup_id = l.lineup_id AND m.match_id = l.match_id AND p.player_id = lp.player_id
  AND m.played = false AND m.in_simulation = false AND p.team_id IS DISTINCT FROM l.team_id
  AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.team_id = l.team_id AND t.is_national_team);
DELETE FROM cup_tie_lineup_positions lp USING cup_tie_lineup l, cup_ties ct, players p
WHERE lp.lineup_id = l.lineup_id AND ct.tie_id = l.tie_id AND p.player_id = lp.player_id
  AND ct.played = false AND ct.in_simulation = false AND p.team_id IS DISTINCT FROM l.team_id
  AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.team_id = l.team_id AND t.is_national_team);
DELETE FROM friendly_lineup_positions lp USING friendly_lineup l, friendly_matches f, players p
WHERE lp.lineup_id = l.lineup_id AND f.match_id = l.match_id AND p.player_id = lp.player_id
  AND f.played = false AND f.in_simulation = false AND p.team_id IS DISTINCT FROM l.team_id
  AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.team_id = l.team_id AND t.is_national_team);
