-- Team budgets, player wages, and a transaction ledger. See engine.deductWages and the
-- transfer market (handlers/transferHandlers.go) for what actually moves money.
--
-- Runs automatically after init3_player_unavailability.sql on a fresh database. On an
-- already-initialized one, apply it manually once - see backend/README.md.

ALTER TABLE teams
    ADD COLUMN IF NOT EXISTS budget bigint NOT NULL DEFAULT 400000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS wage int NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS team_transactions (
    transaction_id bigserial PRIMARY KEY,
    team_id        bigint NOT NULL REFERENCES teams(team_id),
    amount         bigint NOT NULL, -- positive = income, negative = expense
    reason         smallint NOT NULL,
    description    varchar(500),
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS team_transactions_idx1 ON team_transactions(team_id, created_at DESC);
