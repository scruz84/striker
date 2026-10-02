-- Lets the challenger pick which side of the fixture their team takes when sending a friendly
-- challenge (see common.ChallengeToFriendly) - true (the default) keeps the challenger at home,
-- preserving the only behavior that existed before this column for every challenge already
-- pending or already auto-accepted.
ALTER TABLE friendly_challenges ADD COLUMN IF NOT EXISTS challenger_is_home boolean NOT NULL DEFAULT true;
