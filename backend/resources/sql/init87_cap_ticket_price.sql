-- Cap legacy ticket prices (see openspec fix-empty-gate-high-ticket-price).
--
-- Since rebalance-economy-socios the crowd is socios x priceFactor x ..., and priceFactor is 0 from
-- a price of 40 (engine.SociosPriceZeroAt). Owners could still set up to 100 (and 69 was the best
-- price under the old model), so a team left on such a price draws nobody and earns no gate.
-- common.MaxTicketPrice is now 39; reset every stored price above it to the default 10. Re-runnable:
-- a second run matches no rows.
UPDATE teams SET ticket_price = 10 WHERE ticket_price > 39;
