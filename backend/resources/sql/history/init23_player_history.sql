-- Ties a team_transactions row to the specific player it concerns (bought/sold in a transfer,
-- released early) - NULL for team-level transactions (wages, ticket revenue, prize money,
-- sponsorship, stadium upgrades, admin adjustments) that aren't about any one player. Lets a
-- player's page show their transfer history instead of just each team's own ledger. See
-- common.GetPlayerTransactions and handlers.PlayerDetailsHandler.
ALTER TABLE team_transactions ADD COLUMN IF NOT EXISTS player_id bigint REFERENCES players(player_id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS team_transactions_player_idx ON team_transactions(player_id) WHERE player_id IS NOT NULL;
