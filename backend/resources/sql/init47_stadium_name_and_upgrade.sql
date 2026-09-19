-- A stadium's name (see common.RandomStadiumName) - null for a team created before this column
-- existed, lazily backfilled by common.GetStadium the first time it's read, same as
-- team_club_history style lazy backfills elsewhere.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS stadium_name varchar(200);

-- A stadium capacity upgrade (see common.UpgradeStadium) now takes common.StadiumUpgradeDuration
-- to complete rather than applying instantly - stadium_upgrade_target is the capacity it will
-- have once stadium_upgrade_completes_at passes; both null means no upgrade is in progress.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS stadium_upgrade_target integer;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS stadium_upgrade_completes_at timestamptz;
