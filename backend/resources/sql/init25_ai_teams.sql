-- How aggressively an unowned team manages itself each season/matchday (see
-- common.RunAITeamManagement) - meaningless once a team has a real owner (owner_id is set), so
-- it's simply left at whatever it was assigned. 0 = not yet assigned (backfilled at service
-- startup by common.BackfillAiLevels for any pre-existing unowned team); 1-3 = Caretaker/
-- Manager/Director of Football, see common.AiLevelCaretaker/AiLevelManager/AiLevelDirector.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS ai_level smallint NOT NULL DEFAULT 0;
