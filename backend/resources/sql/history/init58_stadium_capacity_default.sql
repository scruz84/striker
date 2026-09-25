-- Raises the default stadium_capacity for newly created teams from 2000 to 10000. Editing
-- init.sql's CREATE TABLE text only affects a brand-new database - an already-existing teams
-- table keeps whatever DEFAULT it was created with regardless, since a column default is fixed
-- in the catalog at creation time, not re-read from the script. Naturally idempotent - re-running
-- this just reasserts the same default - so safe for EnsureSchema to reapply on every startup
-- along with every other numbered migration.
--
-- Existing teams' current stadium capacities are deliberately left untouched: this only changes
-- what a newly inserted team gets from here on.
ALTER TABLE teams ALTER COLUMN stadium_capacity SET DEFAULT 10000;
