-- Recalculates every existing player and trainer's stored wage with the corrected
-- common.PlayerWageFromSkills/common.TrainerWageFromSkill formulas (see common/fmcommon.go and
-- common/trainers.go): WagePerSkillPoint/TrainerWagePerSkillPoint used to be flat per-skill-point
-- rates (5 and 8) calibrated for a wage bill charged once per real calendar week. Since
-- engine.PayMatchdayWages started charging the full bill once per round instead - with the default
-- league schedule playing a round every day - that left every already-existing player and trainer's
-- stored wage column up to 7x too high for how often it's now actually deducted (a brand-new
-- player or trainer gets the corrected rate automatically, from common.doCreatePlayers/
-- GenerateTrainerCandidates onward; this migration is what fixes everyone created before that
-- change). Pure functions of each row's own skill/skill columns, not of the previous wage value, so
-- re-running this is naturally idempotent - the WHERE clauses just skip rows already at the
-- corrected rate on every re-run after the first.
UPDATE players SET
    wage = (goal_keeper + defense + "lateral" + pass + midfield + dribbling + shot) * 5 / 7
WHERE wage <> (goal_keeper + defense + "lateral" + pass + midfield + dribbling + shot) * 5 / 7;

UPDATE trainers SET
    wage = skill * 8 / 7
WHERE wage <> skill * 8 / 7;
