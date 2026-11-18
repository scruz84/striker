-- Cup ties and friendlies store what league matches store, so the one match viewer needs no
-- "this competition doesn't have X" branches (openspec change unify-match-viewer):
--   * cup_tie_instants: the per-minute zone instants of a cup tie (league has match_instants,
--     friendlies have friendly_instants), keyed by the tie id because cup ties have their own id
--     sequence.
--   * the report flavor columns (weather, attendance, referee and two assistants) that `matches`
--     already has, on cup_ties and friendly_matches.
-- Everything is additive and re-runnable. Matches played before this migration keep NULL flavor and
-- no instants, and render without the zone duel / flavor lines, as old league matches already do.
CREATE TABLE IF NOT EXISTS cup_tie_instants (
    tie_id bigint NOT NULL REFERENCES cup_ties(tie_id),
    minute int NOT NULL,
    home_goals int DEFAULT 0,
    away_goals int DEFAULT 0,
    home_qua varchar(500),
    away_qua varchar(500),
    midfield_diff double precision,
    homeattack_right_diff double precision,
    homeattack_left_diff double precision,
    homeattack_center_diff double precision,
    awayattack_right_diff double precision,
    awayattack_left_diff double precision,
    awayattack_center_diff double precision,
    PRIMARY KEY (tie_id, minute)
);
CREATE INDEX IF NOT EXISTS cup_tie_instants_idx ON cup_tie_instants(tie_id);

ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS weather varchar(20);
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS attendance integer;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS referee_name varchar(120);
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS assistant1_name varchar(120);
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS assistant2_name varchar(120);

ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS weather varchar(20);
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS attendance integer;
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS referee_name varchar(120);
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS assistant1_name varchar(120);
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS assistant2_name varchar(120);
