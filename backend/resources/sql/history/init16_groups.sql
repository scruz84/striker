-- Friend groups: a lightweight social leaderboard among teams the group's members pick from
-- the existing shared world - NOT isolated private leagues (their own teams/fixtures). That
-- would need generating a whole separate league on demand, a much bigger change; this is the
-- scoped-down version - see common/groups.go.
CREATE TABLE IF NOT EXISTS groups (
    group_id    bigserial PRIMARY KEY,
    name        varchar(100) NOT NULL,
    owner_id    bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    invite_code varchar(20) NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id  bigint NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    user_id   bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS group_members_idx1 ON group_members(user_id);
