-- Snapshot of the moved player on every player_transfers row (see openspec add-transfer-comparables).
--
-- position / age / average_skill are the player's best core position (common.BestCorePosition), age
-- and average skill (common.AverageSkill) at the moment of the transfer, written by
-- common.RecordPlayerTransfer. The transfer offer dialog compares a target with recent sales of
-- similar players (common.GetTransferComparables) using only these three public figures, so a
-- comparable is judged by what the player was when sold and his current private ratings never leak
-- through the matching.
--
-- Rows recorded before this migration keep NULL: their past skills are unknown, so they are never
-- comparables (the partial index and the query both exclude them). No back-fill.
--
-- The partial index serves common.getTransferComparablesStmtSql: equality on position, then the
-- ORDER BY (created_at DESC, transfer_id DESC) so the planner walks it in order and stops after the
-- LIMIT, with no sort. The predicate keeps loans, releases, free-agent signings, free moves and
-- pre-snapshot rows out of the index; its kinds (0 listing, 1 offer, 4 buy option) and fee > 0 must
-- be spelled as literals in the query for the planner to use it.
--
-- Re-runnable: ADD COLUMN / CREATE INDEX IF NOT EXISTS.
ALTER TABLE player_transfers ADD COLUMN IF NOT EXISTS position smallint;
ALTER TABLE player_transfers ADD COLUMN IF NOT EXISTS age smallint;
ALTER TABLE player_transfers ADD COLUMN IF NOT EXISTS average_skill smallint;

CREATE INDEX IF NOT EXISTS player_transfers_comparables_idx
    ON player_transfers (position, created_at DESC, transfer_id DESC)
    WHERE kind IN (0, 1, 4) AND fee > 0 AND average_skill IS NOT NULL;
