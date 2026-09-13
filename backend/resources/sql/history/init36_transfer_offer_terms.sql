-- A transfer offer now carries the personal terms (wage/contract length) the buyer is proposing
-- to the player, alongside the fee proposed to the selling club - see
-- common.MakeTransferOffer/AcceptOffer. Defaulted rather than NOT NULL so any offer already
-- pending when this migration runs (made under the old fee-only shape) still loads; such an
-- offer simply can't be accepted until re-made, since 0/0 would fail AcceptOffer's own validation
-- if it were ever read back as real terms - in practice, this only matters on an upgrade with
-- offers already in flight.
ALTER TABLE transfer_offers ADD COLUMN IF NOT EXISTS offered_wage integer NOT NULL DEFAULT 0;
ALTER TABLE transfer_offers ADD COLUMN IF NOT EXISTS offered_years integer NOT NULL DEFAULT 0;
