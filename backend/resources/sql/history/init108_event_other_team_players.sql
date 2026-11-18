-- The defending goalkeeper on every chance (openspec match-simulation "Defending keeper on every
-- chance"): league match events already store the other side's players; cup tie and friendly
-- events get the same two columns so they can store the keeper too. Nullable like match_events'
-- own columns, so a row written before this migration (or by an older replica during a rolling
-- deploy) reads as "no keeper". No backfill. Re-runnable.
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS other_team_players varchar(200);
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS other_team_player_names varchar(1000);
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS other_team_players varchar(200);
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS other_team_player_names varchar(1000);
