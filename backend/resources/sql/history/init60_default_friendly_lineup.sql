-- Lets a manager keep a separate standing lineup for friendly matches from their league/cup
-- default (teams.formation/default_lineup_positions) - a friendly with no saved per-match
-- override (see friendly_lineup/friendly_lineup_positions) now falls back to this instead, so a
-- manager can rest first-teamers or try out fringe players in friendlies without touching the
-- competitive default. common.GetLineupsForFriendly falls back further to the league/cup default
-- when this one hasn't been set yet, so nothing changes for a team that never customizes it.
--
-- Mirrors default_lineup_positions exactly; formation gets its own column on teams for the same
-- reason teams.formation already holds the league/cup one.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS friendly_formation smallint NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS default_friendly_lineup_positions (
    team_id bigint references teams(team_id),
    player_id bigint references players(player_id) on delete set null,
    position smallint not null,
    primary key (team_id, player_id)
);
