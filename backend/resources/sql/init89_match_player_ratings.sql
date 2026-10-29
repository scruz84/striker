-- Stored per-player match ratings (see openspec fidelity-match-ratings / match-ratings).
--
-- match_player_ratings holds one row per player who featured in a simulated league match, cup tie
-- or friendly, written in the same transaction as the result (common.replaceMatchEventsTx). kind is
-- a common.SimulationKind* value and match_id is that kind's own id (matches.match_id,
-- cup_ties.tie_id or friendly_matches.match_id - three independent sequences, hence no foreign
-- key, same as pending_post_match_effects). rated is false for an appearance under 10 minutes
-- ("s.c."): the row keeps its breakdown but the rating isn't shown or used for Form.
--
-- player_contributions on the three instants tables is that minute's per-player zone contribution
-- ({"<playerId>": {"p": position, "o": own value, "r": rival value}}), from which the live view and
-- the minute scrubber rebuild the ratings as of any minute. Rows saved before this migration keep
-- NULL: their ratings fall back to common.ComputeLegacyPlayerRatings, with no zone component.
--
-- Re-runnable: CREATE TABLE / ADD COLUMN IF NOT EXISTS.
CREATE TABLE IF NOT EXISTS match_player_ratings (
    kind         text    NOT NULL,
    match_id     bigint  NOT NULL,
    player_id    bigint  NOT NULL,
    team_id      bigint  NOT NULL,
    rated        boolean NOT NULL,
    rating       real    NOT NULL,
    minutes      integer NOT NULL,
    position     integer NOT NULL,
    came_on      integer,
    went_off     integer,
    own_value    real,
    rival_value  real,
    zone_delta   real    NOT NULL,
    goals_delta  real    NOT NULL,
    cards_delta  real    NOT NULL,
    injury_delta real    NOT NULL,
    result_delta real    NOT NULL,
    PRIMARY KEY (kind, match_id, player_id)
);

ALTER TABLE match_instants ADD COLUMN IF NOT EXISTS player_contributions jsonb;
ALTER TABLE cup_tie_instants ADD COLUMN IF NOT EXISTS player_contributions jsonb;
ALTER TABLE friendly_instants ADD COLUMN IF NOT EXISTS player_contributions jsonb;
