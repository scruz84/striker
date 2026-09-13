-- The youth academy pool ("cantera"): a handful of young prospects held separately from a
-- team's fieldable senior squad until a manager actually promotes one (see
-- common.PromoteFromAcademy). Distinct from the existing automatic academy backfill
-- (common.CreateAcademyPlayers, still unchanged) that keeps a team's squad size from falling
-- below common.SquadSize - a cantera prospect never counts toward that floor, and only joins the
-- senior squad, with a real wage and contract, once promoted.
--
-- last_academy_promotion_at enforces the one-promotion-per-real-calendar-week cap (see
-- common.CanPromoteFromAcademy) - a single timestamp is enough since at most one promotion can
-- ever be "the most recent" for a team.
ALTER TABLE players ADD COLUMN IF NOT EXISTS in_academy boolean NOT NULL DEFAULT false;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS last_academy_promotion_at timestamptz;
