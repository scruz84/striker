-- A team's chosen sponsor deal (see common.SponsorDeal / engine.sponsorshipAmount) - a tradeoff
-- between a higher guaranteed per-match sponsorship payment (0, Steady) and a lower guaranteed
-- payment made up for by a bigger bonus on a win or draw (2, Performance), with 1 (Balanced) in
-- between. Defaults to Balanced so existing teams land on a reasonable middle ground rather than
-- the flattest, lowest-upside option.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS sponsor_deal SMALLINT NOT NULL DEFAULT 1;
