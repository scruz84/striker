-- The league page's recent-transfers panel (common.GetRecentLeagueTransfers) reads the newest
-- player_transfers rows first. Re-runnable; no data change.
CREATE INDEX IF NOT EXISTS player_transfers_created_at_idx ON player_transfers(created_at DESC);
