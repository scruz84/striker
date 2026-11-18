-- National teams (issue: "the national team must also play friendlies"): every active country
-- gets one unowned, financially-inert `teams` row flagged is_national_team, standing in for a
-- real virtual squad of that country's best eligible players by birth country rather than any
-- one club - see common/nationalteam.go. Deliberately NOT given a real league_id (stays at the
-- column's own default -1) or a non-zero ai_level (stays at the column's own default 0): that
-- combination already makes every existing "assign an unowned team to a new user"
-- (GetFreeTeamStmtSql) and "is this team AI-managed" (isAiManagedTeam/BackfillAiLevelsStmtSql)
-- query skip it for free, via the exact same JOIN-to-leagues/ai_level checks those already run -
-- no change needed to either. budget/stadium_capacity are pinned to 0 so the one per-match money
-- flow friendlies ever trigger (gate receipts, split from the home side's own stadium) is always
-- zero for a national-team fixture; stadium_name is left NULL so it never surfaces in the
-- stadium search either. See common/search.go's SearchTeamsStmtSql for the one listing that
-- *does* need an explicit exclusion, since it has no league to naturally filter through.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS is_national_team boolean NOT NULL DEFAULT false;

-- At most one national team per country - guards the rare race between the once-at-startup
-- backfill (common.BackfillNationalTeams) and an admin activating that same country at the exact
-- same moment (common.CreateNationalTeam already retries via GetNationalTeamId on a violation).
CREATE UNIQUE INDEX IF NOT EXISTS teams_national_team_country_idx ON teams(country) WHERE is_national_team;
