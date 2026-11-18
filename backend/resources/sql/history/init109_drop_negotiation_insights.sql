-- negotiation_insights (the former init106_negotiation_insights.sql) recorded which clubs had
-- negotiated with a player so his negotiation tier showed on his page. That tier is now private
-- (openspec/specs/player-negotiation): no manager ever sees it, so nothing records or reads the
-- table any more and it is dropped. A brand-new database never creates it (init106 is gone);
-- there this migration is a harmless no-op.
DROP TABLE IF EXISTS negotiation_insights;
