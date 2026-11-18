-- One counter-offer round on a pending transfer offer (see openspec/specs/transfer-negotiation and
-- common.CounterOffer): the selling club may answer with a higher fee, which then waits for the
-- buyer to accept or decline until it expires (common.CounterOfferLifetime after countered_at).
-- Both columns stay NULL on an offer that was never countered. Re-runnable.
ALTER TABLE transfer_offers ADD COLUMN IF NOT EXISTS counter_amount bigint NULL;
ALTER TABLE transfer_offers ADD COLUMN IF NOT EXISTS countered_at timestamptz NULL;

-- The periodic expiry (common.ExpireCounterOffers) only ever looks at countered offers.
CREATE INDEX IF NOT EXISTS transfer_offers_countered_at_idx ON transfer_offers (countered_at) WHERE countered_at IS NOT NULL;
