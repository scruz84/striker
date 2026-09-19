-- Per-minute zone-by-zone team-strength snapshots for a played friendly (see
-- common.MatchInstant / engine.runMatchSimulation) - the friendly equivalent of match_instants,
-- kept in its own table for the same reason friendly_lineup is separate from lineup:
-- friendly_matches has its own id sequence, so writing here to match_instants would either
-- violate its foreign key to matches(match_id) or silently collide with an unrelated league
-- match that happens to share the same id.
CREATE TABLE IF NOT EXISTS friendly_instants (
    match_id bigint not null references friendly_matches(match_id),
    minute int not null,
    home_goals int default 0,
    away_goals int default 0,
    home_qua varchar(500),
    away_qua varchar(500),
    midfield_diff double precision,
    homeattack_right_diff double precision,
    homeattack_left_diff double precision,
    homeattack_center_diff double precision,
    awayattack_right_diff double precision,
    awayattack_left_diff double precision,
    awayattack_center_diff double precision,
    primary key (match_id, minute)
);

CREATE INDEX IF NOT EXISTS friendly_instants_idx ON friendly_instants(match_id);
