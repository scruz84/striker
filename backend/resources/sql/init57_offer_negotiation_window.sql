-- Admin-configurable minimum time (in minutes) a received transfer offer must sit pending
-- before the receiving team is allowed to accept it - see
-- common.GetOfferNegotiationWindowMinutes/common.AcceptOffer. Replaces the previous hardcoded
-- two-hour OfferNegotiationWindow constant; 120 keeps the same default behavior. Zero turns the
-- window off entirely (an offer becomes acceptable the instant it's made).
ALTER TABLE season ADD COLUMN IF NOT EXISTS offer_negotiation_window_minutes smallint NOT NULL DEFAULT 120;
