-- The kickoff time the challenger optionally proposed when sending a friendly challenge (see
-- common.ChallengeToFriendly) - null means they left it up to the opponent, the same as before
-- this column existed. The opponent still has final say when accepting
-- (common.AcceptFriendlyChallenge/AcceptFriendlyChallengeHandler); this is only ever used as a
-- suggested default, never enforced.
ALTER TABLE friendly_challenges ADD COLUMN IF NOT EXISTS proposed_kickoff timestamptz;
