-- Indexes for the simulation backlog gauges (striker.matches.overdue / striker.matches.in_simulation,
-- see common.CountMatchBacklog), which the engine re-runs on every metrics export tick (60 s) on
-- every replica.
--
-- matches, cup_ties and friendly_matches only ever need their unplayed rows for that query, and a
-- row leaves the partial index the moment it is played, so each index stays at a few hundred
-- entries however much match history piles up.
CREATE INDEX IF NOT EXISTS matches_unplayed_date_idx ON matches(match_date) WHERE played = false;
CREATE INDEX IF NOT EXISTS friendly_matches_unplayed_date_idx ON friendly_matches(match_date) WHERE played = false;
CREATE INDEX IF NOT EXISTS cup_ties_unplayed_date_idx ON cup_ties(match_date) WHERE played = false;
