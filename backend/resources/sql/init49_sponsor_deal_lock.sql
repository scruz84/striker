-- Locks a team's sponsor deal choice for the remainder of the season it was made in (issue #70) -
-- previously a team could switch deals before every single match, defeating the point of a deal
-- being a season-long tradeoff rather than something to flip chasing whichever formula suits the
-- next result. 0 (no season is ever numbered 0 - see createSeason, which starts at 1) means
-- "never explicitly chosen", so a team's very first choice is always allowed regardless of the
-- current season number.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS sponsor_deal_season integer NOT NULL DEFAULT 0;
