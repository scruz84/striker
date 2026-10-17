-- Freezes, for every already-played league match that has no explicit lineup override yet, the
-- same lineup common.GetLineupsForMatch resolves for it today (that team's own
-- default_lineup_positions - see GetLineupsForMatchStmtSql's fallback) into lineup/lineup_positions
-- as that match's own permanent record.
--
-- Before engine.persistResolvedLineupSnapshot started saving this at simulation time, a played
-- match with no override just kept re-reading default_lineup_positions fresh on every view - once
-- a transfer/retirement (see common.DeletePlayerLineupPositionsStmtSql) or a changed
-- default removed one of a past match's players from that table, the match's "Lineups" tab
-- degraded or went fully blank, with no way back since nothing had ever recorded who actually
-- played. Running this once pins down whatever is still resolvable today before more of it decays
-- the same way; a team whose default lineup has already drained to nothing by the time this runs
-- has nothing left here to freeze, so matches already blank stay blank - this only protects what
-- hasn't been lost yet. Safe to re-run: ON CONFLICT/NOT EXISTS make every INSERT here a no-op once
-- applied.
INSERT INTO lineup (match_id, team_id, formation)
SELECT m.match_id, t.team_id, t.formation
FROM matches m
JOIN teams t ON t.team_id IN (m.home_team_id, m.away_team_id)
WHERE m.played
ON CONFLICT (match_id, team_id) DO NOTHING;

INSERT INTO lineup_positions (lineup_id, player_id, position)
SELECT l.lineup_id, dlp.player_id, dlp.position
FROM lineup l
JOIN default_lineup_positions dlp ON dlp.team_id = l.team_id
WHERE NOT EXISTS (SELECT 1 FROM lineup_positions lp WHERE lp.lineup_id = l.lineup_id);
