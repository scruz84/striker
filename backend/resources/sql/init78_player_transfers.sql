-- One row per player move between clubs (or to/from free agency), written in the same
-- transaction as the ledger rows and budget changes of each flow - see common.RecordPlayerTransfer
-- and its callers (ExecuteTransfer, AcceptOffer, SignFreeAgent, AcceptLoan, ExerciseLoanBuyOption,
-- ReleasePlayer, AdvancePlayerContract). Until now a player's "transfer history" was reconstructed
-- from team_transactions, which holds two free-text English rows per transfer, no structured
-- origin/destination, and nothing at all for a free agent signed without a premium.
--
-- from_team_id is NULL for a free agent signed with no previous club to pay; to_team_id is NULL
-- when the player is released. kind is one of common.PLAYER_TRANSFER_*. fee is the price paid
-- between clubs (0 when none changed hands).
CREATE TABLE IF NOT EXISTS player_transfers (
    transfer_id  bigserial PRIMARY KEY,
    player_id    bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id bigint REFERENCES teams(team_id) ON DELETE SET NULL,
    to_team_id   bigint REFERENCES teams(team_id) ON DELETE SET NULL,
    fee          bigint NOT NULL DEFAULT 0,
    kind         smallint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_player_transfers_player_id ON player_transfers(player_id, created_at DESC);

-- Backfills the history that predates this table. Only runs while the table is empty, so it is a
-- no-op once any transfer has been recorded (by this backfill or by live code) and safe to
-- re-apply at every startup. Sources, all matched on player_id + created_at (every flow inserts
-- both ledger rows and the club-history row inside one transaction, so they share a timestamp):
--   1. Paired TRANSFER_OUT (reason 1) / TRANSFER_IN (reason 0) ledger rows: bought listings,
--      accepted offers, buy options and free agent premiums. The old ledger can't tell a bought
--      listing from an accepted offer, so those become kind 6 (PLAYER_TRANSFER_PERMANENT).
--   2. Paired LOAN_FEE (reason 13) rows: loans that carried a fee (a free loan left no ledger row).
--   3. CONTRACT_RELEASE (reason 8) rows: early releases, with no destination club.
--   4. player_club_history rows that no row above explains and that aren't the player's first
--      club (that one is the initial squad): free agents signed without a premium, with no origin.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM player_transfers) THEN
        INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, created_at)
        SELECT o.player_id, i.team_id, o.team_id, -o.amount,
               CASE WHEN o.description LIKE 'Exercised buy option%' THEN 4
                    WHEN o.description LIKE 'Paid free agent transfer premium%' THEN 2
                    ELSE 6 END,
               o.created_at
        FROM team_transactions o
        JOIN team_transactions i ON i.player_id = o.player_id AND i.created_at = o.created_at
                                AND i.amount = -o.amount AND i.reason = 0
        WHERE o.reason = 1 AND o.player_id IS NOT NULL;

        INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, created_at)
        SELECT o.player_id, i.team_id, o.team_id, -o.amount, 3, o.created_at
        FROM team_transactions o
        JOIN team_transactions i ON i.player_id = o.player_id AND i.created_at = o.created_at
                                AND i.amount = -o.amount AND i.reason = 13
        WHERE o.reason = 13 AND o.amount < 0 AND o.player_id IS NOT NULL;

        INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, created_at)
        SELECT player_id, team_id, NULL, 0, 5, created_at
        FROM team_transactions
        WHERE reason = 8 AND player_id IS NOT NULL;

        INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, created_at)
        SELECT h.player_id, NULL, h.team_id, 0, 2, h.joined_at
        FROM (
            SELECT player_id, team_id, joined_at,
                   row_number() OVER (PARTITION BY player_id ORDER BY joined_at, history_id) AS rn
            FROM player_club_history
        ) h
        WHERE h.rn > 1
          AND NOT EXISTS (
              SELECT 1 FROM player_transfers pt
              WHERE pt.player_id = h.player_id AND pt.to_team_id = h.team_id AND pt.created_at = h.joined_at
          );
    END IF;
END
$$;
