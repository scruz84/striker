-- Emergent rivalries (see openspec/specs/rivalries and common/rivalries.go): one score per
-- unordered pair of clubs of the same country (team_low_id < team_high_id), 0-40, grown by their
-- shared history and decayed at each season rollover. A match stores the level it kicked off with
-- (RIVALRY_NONE / _RIVAL / _CLASICO), so a played match always shows the level it was played at.
-- Re-runnable.
CREATE TABLE IF NOT EXISTS team_rivalries (
    team_low_id  bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    team_high_id bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    country      varchar(4) NOT NULL,
    score        real NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 40),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_low_id, team_high_id),
    CHECK (team_low_id < team_high_id)
);

CREATE INDEX IF NOT EXISTS team_rivalries_high_idx ON team_rivalries (team_high_id);
CREATE INDEX IF NOT EXISTS team_rivalries_country_idx ON team_rivalries (country);

ALTER TABLE matches ADD COLUMN IF NOT EXISTS rivalry_level smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS rivalry_level smallint NOT NULL DEFAULT 0;
