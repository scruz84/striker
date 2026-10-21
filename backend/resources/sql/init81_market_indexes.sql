-- Indexes for the transfer market's activity panel (GET /v1/transfers/stats, polled every 60 s by
-- every open market page) - see common.GetTransferMarketStats.
--
-- transfer_offer_log is append-only and never shrinks; the "offers in the last hour/day" counts
-- only need its newest rows, which this index finds without scanning the whole log.
CREATE INDEX IF NOT EXISTS transfer_offer_log_created_idx ON transfer_offer_log(created_at);

-- The biggest fee ever paid is read from player_transfers. The partial predicate lists the kinds
-- common.topTransferFeeKinds counts (listing, offer, free agent premium, buy option, permanent)
-- so the index stays small and the query's ORDER BY fee DESC LIMIT 1 is a single index lookup.
CREATE INDEX IF NOT EXISTS player_transfers_fee_idx ON player_transfers(fee DESC) WHERE kind IN (0, 1, 2, 4, 6);
