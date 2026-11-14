-- Market rows left behind by a player who already left the club they name (issue #397): before the
-- fix, releasing a player or letting his contract expire kept his transfer listing, loan listing
-- and pending offers, so a later buyer paid the old club. common.clearPlayerMarketTx now removes
-- them when he leaves, and ExecuteTransfer/AcceptLoan refuse a listing whose player is no longer
-- the seller's or lender's. This removes the stale rows already stored.
--
-- Re-runnable: each DELETE only matches rows that are still stale.

DELETE FROM transfer_listings tl
WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.player_id = tl.player_id AND p.team_id = tl.selling_team_id);

DELETE FROM loan_listings ll
WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.player_id = ll.player_id AND p.team_id = ll.lending_team_id);

DELETE FROM transfer_offers o
WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.player_id = o.player_id AND p.team_id = o.to_team_id);
