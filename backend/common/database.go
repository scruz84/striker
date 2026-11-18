package common

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"
)

const ForUpdate = "FOR UPDATE"

// current_season lives on `countries` (one counter per country - see
// init75_season_per_country.sql) rather than on the old global `season` singleton, so each
// country can advance its own season independently (see engine.AdvanceSeasonsIfComplete).
const GetCurrentSeasonForCountryStmtSql = "SELECT current_season FROM countries WHERE name = $1"
const SetCurrentSeasonForCountryStmtSql = "UPDATE countries SET current_season = $1 WHERE name = $2"
const GetCurrentSeasonForLeagueStmtSql = "SELECT c.current_season FROM leagues l JOIN countries c ON c.name = l.country WHERE l.league_id = $1"
const GetCurrentSeasonForTeamStmtSql = "SELECT c.current_season FROM teams t JOIN countries c ON c.name = t.country WHERE t.team_id = $1"
const GetCountryForLeagueStmtSql = "SELECT country FROM leagues WHERE league_id = $1"

const GetAllLeagueIdsStmtSql = "SELECT league_id FROM leagues WHERE archived_season IS NULL"

// GetSeasonMatchCountsStmtSql/CountSeasonMatchesStillBroadcastingStmtSql are scoped to one
// country's own leagues (see GetSeasonMatchCounts/CountSeasonMatchesStillBroadcasting) - a
// season is now per-country, so "is season N complete" has to mean "complete for this one
// country", not every match in the whole world.
const GetSeasonMatchCountsStmtSql = "SELECT count(*) FILTER (WHERE m.played = false), count(*) FROM matches m " +
	"JOIN leagues l ON l.league_id = m.league_id WHERE l.country = $1 AND m.season = $2"

const CountSeasonMatchesStillBroadcastingStmtSql = "SELECT count(*) FROM matches m JOIN leagues l ON l.league_id = m.league_id " +
	"WHERE l.country = $1 AND m.season = $2 AND m.played = true AND m.match_date > $3"

const ResetClassificationForSeasonStmtSql = "UPDATE classification SET season = $1, matches = 0, points = 0, wins = 0, " +
	"losses = 0, draws = 0, goals_scored = 0, goals_received = 0 WHERE league_id = $2"

// Prefers the highest division (lowest tier number) that still has an unclaimed team, only
// spilling into the next division down once the current one is full of human owners - a new
// manager should land in the top flight while there's room, not in a random division. Within
// whichever tier ends up supplying the pick, the specific team is still random.
const GetFreeTeamStmtSql = "SELECT t.team_id FROM teams t JOIN leagues l ON t.league_id = l.league_id " +
	"WHERE t.owner_id IS NULL AND t.reset_pending_since IS NULL AND t.dissolved_season IS NULL AND t.country = $1 ORDER BY l.tier ASC, random() LIMIT 1"

// For each of the two teams, returns EITHER all of that team's saved override rows for this match
// (lineup/lineup_positions) OR, if it has none, all of its default rows (default_lineup_positions
// joined to teams for the standing formation) - never a mix of the two. This has to pick one side
// wholesale rather than coalescing row-by-row: a naive per-team LEFT JOIN between the two position
// sets (matched only on team_id, since neither side aligns row-for-row with the other) would be a
// cross join once a team actually has 11 override rows alongside its 11 default rows - 121
// results instead of 11, each position duplicated against every other. That was never triggered
// before this override existed (lineup/lineup_positions were always empty for a real match), so
// this UNION ALL form (mirroring the Go-level hasOverride fallback GetFriendlyLineup/
// GetCupTieLineup already use) is required now that SetLineupForMatch actually writes rows here.
var GetLineupsForMatchStmtSql = "select c.team_id, d.player_id, d.\"position\", c.formation, c.mentality, " + rolesSelectSql("c", matchPlayedSql("c")) + " " +
	"from lineup c join lineup_positions d on (c.lineup_id = d.lineup_id) where c.team_id in ($1, $2) and c.match_id = $3 " +
	"union all " +
	"select a.team_id, a.player_id, a.\"position\", t.formation, t.mentality, " + rolesSelectSql("r", "false") + " " +
	"from default_lineup_positions a join teams t on t.team_id = a.team_id " +
	"left join default_lineup_roles r on r.team_id = a.team_id and r.friendly = false " +
	"where a.team_id in ($1, $2) and a.team_id not in (select c.team_id from lineup c where c.team_id in ($1, $2) and c.match_id = $3)"

// standingsOrderSql is the ordering of every league table (see openspec league-page "Starting
// order breaks exact ties"): points, goal difference, goals scored, wins, draws, then the season's
// starting order (classification.seed, unseeded rows last) and the team id, so the order is total
// and a new season reads in its starting order on matchday 0. Columns of classification, unqualified.
const standingsOrderSql = "points DESC, goals_scored - goals_received DESC, goals_scored DESC, wins DESC, draws DESC, " +
	"seed ASC NULLS LAST, team_id ASC"

const GetDivisionDetailsStmtSql = "SELECT a.team_id, b.name, a.matches, a.points, a.wins, a.losses, a.draws, a.goals_scored, a.goals_received, a.seed " +
	" FROM classification a JOIN teams b ON a.team_id = b.team_id " +
	"WHERE a.league_id = $1 and a.season = $2 ORDER BY a.points DESC, a.goals_scored-a.goals_received DESC, a.goals_scored DESC, a.wins DESC, a.draws DESC, " +
	"a.seed ASC NULLS LAST, a.team_id ASC"

// GetTeamClassificationPositionStmtSql ranks every team in a league/season with the same
// ordering GetDivisionDetailsStmtSql uses (standingsOrderSql), then picks out the requested team's
// rank - used by GetTeamDetailsById so a team's page can show where it actually sits without the
// caller fetching (and ranking) the whole division.
const GetTeamClassificationPositionStmtSql = "SELECT position FROM (" +
	"SELECT team_id, ROW_NUMBER() OVER (ORDER BY " + standingsOrderSql + ") AS position " +
	"FROM classification WHERE league_id = $1 AND season = $2" +
	") ranked WHERE team_id = $3"

const GetDivisionNameStmtSql = "SELECT name from leagues WHERE league_id = $1"

const GetDivisionTeamsStmtSql = "SELECT team_id from teams WHERE league_id = $1"

// The owner's name is deliberately not selected: the public team API must not expose the real
// (Google) name of a club's owner - see the privacy-and-legal spec.
const GetTeamStmtByIdSql = "SELECT a.owner_id, a.country, a.league_id, b.name league_name, a.name team_name, a.ai_level, a.ai_style " +
	"FROM teams a JOIN leagues b ON (a.league_id = b.league_id) " +
	"WHERE a.team_id = $1"

const GetTeamDetailsStmtByOwnerSql = "SELECT team_id, country, league_id, name, name_changed FROM teams WHERE owner_id = $1"
const GetTeamDetailsStmtSql = "SELECT owner_id, country, league_id, name FROM teams WHERE team_id = $1"
const GetTeamIdsFromLeagueSql = "SELECT team_id FROM teams where league_id = $1"
const GetTeamOwnerStmtSql = "SELECT owner_id FROM teams WHERE team_id = $1"

const GetPlayerFullNameStmtSql = "SELECT first_name, last_name FROM players WHERE player_id = $1"

// Excludes academy prospects (p.in_academy) - a team's fieldable squad is only ever its senior
// players; a prospect only joins this list once actually promoted (see PromoteFromAcademy). See
// GetCantera for the separate query that lists a team's current academy pool instead.
const GetPlayersFromTeamStmtSql = "SELECT player_id, p.team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"COALESCE(shirt_number, 0), stamina, form " +
	"FROM teams t JOIN players p ON t.team_id = p.team_id " +
	"WHERE t.team_id = $1 AND p.in_academy = false"

// GetSimulableMatchesStmtSql deliberately has no lower bound on match_date (unlike the upper
// bound, which keeps this from ever grabbing something scheduled further out than the next 15
// minutes) - same reasoning as common.GetSimulableFriendliesStmtSql's identical widening: a match
// whose 15-minute pre-kickoff window was missed entirely (a restart or deploy spanning its exact
// claim window) must still be picked up on the very next tick once match_date is back in the
// past, not left stranded at played = false, in_simulation = false forever - a state GetStuckMatches
// can never see either, since that only covers a row already claimed (in_simulation = true) mid-crash.
// A single such permanently stuck league match blocks its whole country's season from ever
// advancing (see engine.AdvanceSeasonsIfComplete, which requires every match in a country's
// season to be played before rolling that one country over) - the most likely explanation for a
// season that never seems to finish.
//
// No `season = ...` filter: unlike before current_season moved onto `countries` (one counter per
// country, see init75_season_per_country.sql), this query no longer looks up any one season
// number at all. played = false already implies "belongs to its league's country's current
// season" on its own - a country's season only advances once every one of its own matches is
// played, and no new (future-season) fixtures exist until that happens - so filtering by season
// here would be redundant even if there were still a single season number to filter by.
const GetSimulableMatchesStmtSql = "select match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, " +
	"home_goals, away_goals from matches where match_date < now()+interval '15 minutes' and played is false and in_simulation is false " +
	"and " + simulationEligibleSql + " order by league_id"

const GetSimulableMatchesStmtForUpdateSql = GetSimulableMatchesStmtSql + " " + ForUpdate

const GetUserIdByEmailStmtSql = "SELECT user_id FROM users WHERE email = $1"

// COALESCEd because a user who hasn't picked a country yet (see RegisterPendingUserStmtSql) has
// no row in `teams` to join and a NULL country - scanning those straight into Go's non-nullable
// string fields would error, where the empty-string/'0' sentinels let GetUserDetails represent
// "no team assigned yet" instead.
const GetUserStmtSql = "SELECT u.name userName, COALESCE(u.country, '') as country, u.provider, u.email, " +
	"COALESCE(t.team_id::text, '0') as teamId, COALESCE(t.name, '') as teamName, COALESCE(t.league_id::text, '0') as leagueId, u.is_admin, " +
	"u.terms_version, COALESCE(u.released_team_name, '') " +
	"from users u left join teams t on (u.user_id = t.owner_id) where u.user_id = $1"

const GetUserIsAdminStmtSql = "SELECT is_admin FROM users WHERE user_id = $1"

const SetUserAdminStmtSql = "UPDATE users SET is_admin = $1 WHERE user_id = $2"

const GetWeekMatchesStmtSql = "select match_id, w.league_id, w.season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" w.weather, w.attendance, w.referee_name, w.assistant1_name, w.assistant2_name," +
	" t0.name as home_team_name, t1.name as away_team_name, t0.stadium_name as home_stadium_name " +
	"FROM (SELECT match_id, league_id, season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" weather, attendance, referee_name, assistant1_name, assistant2_name " +
	"FROM matches where league_id = $1 and week_number = $2 and season = $3) w " +
	"join teams t0 on (w.home_team_id = t0.team_id) join teams t1 on (w.away_team_id = t1.team_id)"

const GetWeekMatchesByIdStmtSql = "select match_id, w.league_id, w.season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" w.weather, w.attendance, w.referee_name, w.assistant1_name, w.assistant2_name," +
	" t0.name as home_team_name, t1.name as away_team_name, t0.stadium_name as home_stadium_name " +
	"FROM (SELECT match_id, league_id, season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" weather, attendance, referee_name, assistant1_name, assistant2_name " +
	"FROM matches where match_id = $1) w " +
	"join teams t0 on (w.home_team_id = t0.team_id) join teams t1 on (w.away_team_id = t1.team_id)"

const GetWeekMatchesEventsByIdStmtSql = "SELECT event_id, match_id, event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals, set_piece, mentality, added_minute, added_time " +
	"FROM match_events WHERE match_id = $1 order by minute asc, added_minute asc, event_id asc"

const GetWeekMatchesInstantsByIdStmtSql = "SELECT match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff, player_contributions, added_minute" +
	" FROM match_instants WHERE match_id = $1 order by minute asc, added_minute asc"

const InsertTeamToClassificationStmtSql = "INSERT INTO classification (league_id, season, team_id) values ($1, $2, $3)"

const InsertMatch = "INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id) " +
	"VALUES ($1, $2, $3, $4, $5, $6)"
const InsertNewPlayerStmtSql = "INSERT INTO players (team_id, first_name, last_name, birth_day, " +
	"goal_keeper, defense, \"lateral\", pass, midfield, dribbling, shot, header, quick," +
	" creative, technical, avatar, wage, age, contract_years_remaining, birth_country, in_academy, shirt_number, stamina, form, " +
	"goal_keeper_max, defense_max, lateral_max, pass_max, midfield_max, dribbling_max, shot_max, intelligence) " +
	"values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, " +
	"$25, $26, $27, $28, $29, $30, $31, $32) " +
	"RETURNING player_id"

const InsertDefaultLineupPositionsStmtSql = "INSERT INTO default_lineup_positions (team_id, player_id, position) values ($1, $2, $3)"
const GetDefaultLineupStmtSql = "SELECT player_id, position FROM default_lineup_positions WHERE team_id = $1"
const DeleteDefaultLineupPositionsStmtSql = "DELETE FROM default_lineup_positions WHERE team_id = $1"

const InsertDefaultFriendlyLineupPositionsStmtSql = "INSERT INTO default_friendly_lineup_positions (team_id, player_id, position) values ($1, $2, $3)"
const GetDefaultFriendlyLineupStmtSql = "SELECT player_id, position FROM default_friendly_lineup_positions WHERE team_id = $1"
const DeleteDefaultFriendlyLineupPositionsStmtSql = "DELETE FROM default_friendly_lineup_positions WHERE team_id = $1"

const InsertMatchEventStmtSql = "INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals, set_piece, mentality, added_minute, added_time) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"

const InsertMatchInstantStmtSql = "" +
	"INSERT INTO match_instants(match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff, player_contributions, added_minute) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"

// RegisterPendingUserStmtSql registers a brand-new sign-in without a country - used the first
// time someone logs in (Google or dev), before they've picked which country to play in (see
// handlers.SelectCountryHandler). Leaves `country` NULL (the column allows it) rather than
// guessing one, and deliberately doesn't touch `teams` at all - AssignTeam only runs once a
// country is chosen.
const RegisterPendingUserStmtSql = "INSERT INTO users (name, provider, email) VALUES ($1, $2, $3) returning user_id"
const UnAssignPlayersStmtSql = "UPDATE players SET team_id = NULL, shirt_number = NULL WHERE team_id = $1"

// Resets name_changed to false alongside the fresh name - a newly assigned owner gets their own
// one-time rename, regardless of whether a previous owner had already used theirs.
// ai_level = 0 clears whatever CPU difficulty tier this team carried while unowned (see
// common.AiLevelCaretaker/Manager/Director) - Team.AiLevel's own doc comment already promises 0
// for an owned team, but leaving the old value in place here let it leak back out via GetTeamById
// as soon as the new owner viewed their own squad page (see AssignTeam).
//
// Everything else a previous owner (human or CPU) could have changed is reset to that column's
// table default too - budget, stadium_capacity, ticket_price, formation and sponsor_deal/
// sponsor_deal_season via SQL's DEFAULT keyword (so this stays in sync with the CREATE TABLE
// teams defaults in resources/sql/init.sql without duplicating the numbers here), and
// stadium_upgrade_target/stadium_upgrade_completes_at/last_academy_promotion_at explicitly to
// NULL since those have no meaningful default besides "nothing in progress yet". A fresh
// stadium_name is handed in alongside the fresh team name so the new owner isn't stuck with
// whatever the previous owner (or CPU) had named it. Without this, GetFreeTeamStmtSql handing a
// previously-owned team to a new manager would start them with someone else's leftover budget,
// stadium size/upgrade, ticket price, formation and sponsor deal instead of a true fresh start.
const UpdateTeamDetailsStmtSql = "UPDATE teams SET name = $1, owner_id = $2, name_changed = false, ai_level = 0, " +
	"stadium_name = $3, budget = DEFAULT, stadium_capacity = DEFAULT, ticket_price = DEFAULT, " +
	"formation = DEFAULT, sponsor_deal = DEFAULT, sponsor_deal_season = DEFAULT, " +
	"stadium_upgrade_target = NULL, stadium_upgrade_completes_at = NULL, last_academy_promotion_at = NULL " +
	"WHERE team_id = $4"

const UpdateTeamClassificationStmtSql = "UPDATE classification set matches = matches+$1, points = points+$2, wins = wins+$3, " +
	"losses = losses+$4, draws = draws+$5, goals_scored = goals_scored+$6, goals_received = goals_received+$7 " +
	"where league_id = $8 and team_id = $9"

const UpdateMatchSimulateStmtSql = "UPDATE matches SET in_simulation = $1, in_simulation_since = now(), " + simulationClaimSetSql + " WHERE "

const GetLeaguesWithTierStmtSql = "SELECT league_id, country, tier FROM leagues WHERE archived_season IS NULL ORDER BY country, tier, group_number"

const GetRankedClassificationStmtSql = "SELECT team_id, points, goals_scored - goals_received, goals_scored " +
	"FROM classification WHERE league_id = $1 ORDER BY " + standingsOrderSql

// GetFinalClassificationStmtSql is a league's full table lines in GetRankedClassificationStmtSql's
// order, read at rollover so season_history keeps each team's whole final line.
const GetFinalClassificationStmtSql = "SELECT team_id, matches, points, wins, draws, losses, goals_scored, goals_received " +
	"FROM classification WHERE league_id = $1 ORDER BY " + standingsOrderSql

const clearLeagueSeedsStmtSql = "UPDATE classification SET seed = NULL WHERE league_id = $1"

const UpdateClassificationSeedStmtSql = "UPDATE classification SET seed = $1 WHERE league_id = $2 AND team_id = $3"

const UpdateTeamLeagueStmtSql = "UPDATE teams SET league_id = $1 WHERE team_id = $2"

const UpdateClassificationLeagueStmtSql = "UPDATE classification SET league_id = $1 WHERE team_id = $2"

const UpsertPlayerUnavailabilityStmtSql = "INSERT INTO player_unavailability (player_id, reason, matches_remaining) " +
	"VALUES ($1, $2, $3) " +
	"ON CONFLICT (player_id) DO UPDATE SET " +
	"reason = CASE WHEN EXCLUDED.matches_remaining > player_unavailability.matches_remaining THEN EXCLUDED.reason ELSE player_unavailability.reason END, " +
	"matches_remaining = GREATEST(player_unavailability.matches_remaining, EXCLUDED.matches_remaining)"

const GetUnavailablePlayerIdsStmtSql = "SELECT pu.player_id FROM player_unavailability pu " +
	"JOIN players p ON p.player_id = pu.player_id WHERE p.team_id = $1 AND pu.matches_remaining > 0"

// GetUnavailablePlayerIdsForFriendlyStmtSql excludes players serving a red-card suspension:
// suspensions only bar official/league/cup matches, not friendlies. Injuries still apply.
var GetUnavailablePlayerIdsForFriendlyStmtSql = "SELECT pu.player_id FROM player_unavailability pu " +
	"JOIN players p ON p.player_id = pu.player_id WHERE p.team_id = $1 AND pu.matches_remaining > 0 " +
	"AND pu.reason != " + strconv.Itoa(UNAVAILABLE_SUSPENSION)

const GetTeamUnavailabilityStmtSql = "SELECT pu.player_id, pu.reason, pu.matches_remaining FROM player_unavailability pu " +
	"JOIN players p ON p.player_id = pu.player_id WHERE p.team_id = $1 AND pu.matches_remaining > 0 ORDER BY pu.player_id"

const DecrementUnavailabilityForTeamStmtSql = "UPDATE player_unavailability pu SET matches_remaining = matches_remaining - 1 " +
	"FROM players p WHERE p.player_id = pu.player_id AND p.team_id = $1"

const DeleteExpiredUnavailabilityStmtSql = "DELETE FROM player_unavailability WHERE matches_remaining <= 0"

const GetTeamBudgetStmtSql = "SELECT budget FROM teams WHERE team_id = $1"
const UpdateTeamBudgetStmtSql = "UPDATE teams SET budget = budget + $1 WHERE team_id = $2"
const UpdateAllTeamsBudgetStmtSql = "UPDATE teams SET budget = budget + $1 WHERE dissolved_season IS NULL"

// InsertTeamTransactionForAllTeamsStmtSql logs the same mass adjustment against every team's own
// ledger, one row per team_id, so a mass injection isn't invisible to any team's owner.
const InsertTeamTransactionForAllTeamsStmtSql = "INSERT INTO team_transactions (team_id, amount, reason, description, player_id) " +
	"SELECT team_id, $1, $2, $3, NULL FROM teams WHERE dissolved_season IS NULL"

// player_id is nullable - only a transaction about one specific player (a transfer, a release)
// sets it; team-level transactions (wages, ticket revenue, prize money, sponsorship, stadium
// upgrades, admin adjustments) pass nil. See GetPlayerTransactions.
const InsertTeamTransactionStmtSql = "INSERT INTO team_transactions (team_id, amount, reason, description, player_id) VALUES ($1, $2, $3, $4, $5)"

// insertLedgerRow writes one team_transactions row inside tx and buffers its amount in flows, to
// be exported by flows.Commit(ctx) once tx.Commit() has succeeded (see LedgerFlows). Every ledger
// row is written through here (or insertLedgerRowForAllTeams) so striker.economy.flow cannot miss
// a flow; a source test fails if InsertTeamTransactionStmtSql is used anywhere else. playerId is
// nil for team-level transactions.
func insertLedgerRow(tx *sql.Tx, flows *LedgerFlows, teamId int64, amount int64, reason int, description string, playerId any) error {
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, amount, reason, description, playerId); err != nil {
		return err
	}
	flows.Add(reason, amount)
	return nil
}

// insertLedgerRowForAllTeams is insertLedgerRow for a mass adjustment: one row per team, `teams`
// being how many teams the UPDATE affected.
func insertLedgerRowForAllTeams(tx *sql.Tx, flows *LedgerFlows, teams int64, amount int64, reason int, description string) error {
	if _, err := tx.Exec(InsertTeamTransactionForAllTeamsStmtSql, amount, reason, description); err != nil {
		return err
	}
	flows.AddTimes(reason, amount, teams)
	return nil
}

const GetTeamTransactionsStmtSql = "SELECT transaction_id, team_id, amount, reason, description, created_at, player_id " +
	"FROM team_transactions WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2"

// Academy prospects are excluded - they draw no wage until actually promoted (see
// PromoteFromAcademy), same reasoning as GetPlayersFromTeamStmtSql. A player currently on loan to
// teamId only charges teamId the slice of their wage the lender isn't covering (see
// player_loans.wage_contribution_percent, set via common.CreateLoanListing/AcceptLoan) - the
// lender is charged the covered remainder separately (see GetTeamLoanWageContribution and
// engine.deductWages), so the two sides always add up to exactly the player's full wage.
const GetTeamWageBillStmtSql = "SELECT COALESCE(SUM(p.wage * (100 - COALESCE(pl.wage_contribution_percent, 0)) / 100), 0) " +
	"FROM players p LEFT JOIN player_loans pl ON pl.player_id = p.player_id AND pl.to_team_id = p.team_id " +
	"WHERE p.team_id = $1 AND p.in_academy = false"

// nameTableSuffixByCountry maps a known country code to its names_<suffix>/lastnames_<suffix>
// table pair (see resources/sql/init.sql and the per-country resources/sql/namepools/ files). Table names can't be parameterized like ordinary values, so
// GetRandomName looks the suffix up here - an allowlist derived from KnownCountries - rather than
// ever building SQL from the country string directly. Derived at init so it can't drift from the
// catalog; a code that isn't 2-3 lowercase letters panics at startup, keeping the SQL-safety
// property the hand-kept map used to give (see nameTableSuffix).
var nameTableSuffixByCountry = func() map[string]string {
	suffixes := make(map[string]string, len(KnownCountries))
	for _, country := range KnownCountries {
		suffixes[country.Code] = nameTableSuffix(country.Code)
	}
	return suffixes
}()

var nameTableSuffixPattern = regexp.MustCompile(`^[a-z]{2,3}$`)

// nameTableSuffix returns the table-name suffix for a KnownCountries code - the code itself - and
// panics when it isn't 2-3 lowercase ASCII letters, since the suffix is concatenated into SQL.
func nameTableSuffix(code string) string {
	if !nameTableSuffixPattern.MatchString(code) {
		panic(fmt.Sprintf("common: country code %q cannot be used as a name table suffix (want ^[a-z]{2,3}$)", code))
	}
	return code
}

// GetRandomName picks a random first/last name from the given country's name pool. Falls back
// to Spanish ("es") names for an unrecognized country code rather than failing - a missing/typo'd
// country should never block player generation.
func GetRandomName(dbCtx *DatabaseContext, country string) (string, string, error) {
	suffix, ok := nameTableSuffixByCountry[country]
	if !ok {
		suffix = "es"
	}
	stmtNames, err := dbCtx.Db.Prepare("select name from names_" + suffix + " order by random() limit 1")
	if err != nil {
		return "", "", err
	}
	defer stmtNames.Close()
	stmtLastNames, err := dbCtx.Db.Prepare("select lastname from lastnames_" + suffix + " order by random() limit 1")
	if err != nil {
		return "", "", err
	}
	defer stmtLastNames.Close()
	nameRow := stmtNames.QueryRow()
	lastNameRow := stmtLastNames.QueryRow()
	var name string
	var lastName string
	err = nameRow.Scan(&name)
	if err != nil {
		return "", "", err
	}
	err = lastNameRow.Scan(&lastName)
	if err != nil {
		return "", "", err
	}
	return name, lastName, nil
}

// SquadSize is how many players a freshly created team starts with.
const SquadSize = 18

// SeniorSkillMin/Max and SeniorAgeMin/Max are the ranges used for a new team's starting
// squad. AcademySkillMin/Max and AcademyAgeMin/Max (see CreateAcademyPlayers) are deliberately
// lower/younger - raw prospects, not the finished article.
const SeniorSkillMin, SeniorSkillMax = 30, 60
const SeniorAgeMin, SeniorAgeMax = 18, 32

// StaminaGenerationMin/Max bounds a freshly generated player's starting Stamina - applied
// uniformly by doCreatePlayers/InjectFreeAgents regardless of senior/academy/free-agent, since
// physical conditioning (unlike the seven footballing skills) isn't meant to track a prospect's
// lower starting quality.
const StaminaGenerationMin, StaminaGenerationMax = 50, 90

// DefaultForm is the neutral starting Form every freshly generated player gets, whatever their
// skills or age - see engine.updateTeamPlayersAfterMatch for how it moves from there.
const DefaultForm = 50

func CreatePlayers(dbCtx *DatabaseContext, teamId int64, country string) {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if errInsert != nil {
		log.Printf("error inserting new players %s", errInsert)
		return
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, dbCtx.Db, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, country, false)
}

func GetTeamDetails(dbCtx *DatabaseContext, stmt *sql.Stmt, teamId int64) (*Team, error) {
	var userId = int64(-1)
	var country = ""
	var name = ""
	var leagueId = int64(-1)
	err := stmt.QueryRow(teamId).Scan(&userId, &country, &leagueId, &name)
	if err != nil {
		return nil, err
	}
	divName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	return &Team{Id: teamId,
		TeamName:     name,
		OwnerId:      userId,
		Country:      country,
		DivisionId:   leagueId,
		DivisionName: divName}, nil
}

// shirtNumberQueryer is satisfied by both *sql.DB and *sql.Tx, so nextFreeShirtNumber(s) work
// either standalone or as part of a caller's larger transaction.
type shirtNumberQueryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// NextFreeShirtNumbersStmtSql finds the `count` lowest positive integers not already assigned to
// one of $1's players - generate_series only needs to run up to (players already numbered) +
// count, since among that many consecutive candidates at most (players already numbered) can be
// taken, guaranteeing at least `count` free ones turn up.
const NextFreeShirtNumbersStmtSql = "SELECT n FROM generate_series(1, " +
	"(SELECT COUNT(*) FROM players WHERE team_id = $1 AND shirt_number IS NOT NULL) + $2) AS n " +
	"WHERE n NOT IN (SELECT shirt_number FROM players WHERE team_id = $1 AND shirt_number IS NOT NULL) " +
	"ORDER BY n LIMIT $2"

// nextFreeShirtNumbers returns the `count` lowest shirt numbers not currently used by any of
// teamId's players (issue #42) - every code path that adds one or more players to a team's
// fieldable senior roster (initial squad creation, an academy backfill, an academy promotion,
// signing a free agent, a transfer or loan taking effect) uses this so numbers stay gap-free
// within a squad and never collide with a teammate's. The unique (team_id, shirt_number) index
// (see resources/sql/history/init39_player_shirt_number.sql) is the backstop against two concurrent
// callers computing the same number for the same team before either commits - a real
// possibility this doesn't itself guard against, but one this app's usage pattern (one manager
// acting on their own team at a time) makes rare enough not to warrant an explicit lock for a
// "fixed for now" feature.
func nextFreeShirtNumbers(q shirtNumberQueryer, teamId int64, count int) ([]int, error) {
	rows, err := q.Query(NextFreeShirtNumbersStmtSql, teamId, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	numbers := make([]int, 0, count)
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		numbers = append(numbers, n)
	}
	return numbers, rows.Err()
}

// nextFreeShirtNumber is nextFreeShirtNumbers for the single-player case (signing, a promotion, a
// transfer or loan taking effect).
func nextFreeShirtNumber(q shirtNumberQueryer, teamId int64) (int, error) {
	numbers, err := nextFreeShirtNumbers(q, teamId, 1)
	if err != nil {
		return 0, err
	}
	if len(numbers) == 0 {
		return 0, errors.New("no free shirt number found")
	}
	return numbers[0], nil
}

// doCreatePlayers inserts `count` random players for a team, with skills uniformly drawn from
// [minSkill, maxSkill] and age from [minAge, maxAge]. Used for a new team's starting squad, for
// automatic youth academy backfills (see CreateAcademyPlayers), and for cantera prospects (see
// createCanteraProspects in academy.go) - all three just vary the ranges and inAcademy. An
// academy prospect (inAcademy true) draws no wage and has no contract yet - both are only
// assigned once actually promoted (see PromoteFromAcademy), the same way a free agent's are only
// assigned once signed; shirt numbers follow the same rule; inAcademy players are created with
// none. historyEx runs on the same connection/transaction as insertStmt (dbCtx.Db or a tx, per
// caller) - a fresh senior-squad player (inAcademy false) gets a player_club_history row for
// teamId; a cantera prospect doesn't, since they aren't on the senior squad yet (see
// RecordPlayerClubHistory).
func doCreatePlayers(dbCtx *DatabaseContext, teamId int64, insertStmt *sql.Stmt, historyEx sqlExecutor, count int, minSkill int, maxSkill int, minAge int, maxAge int, country string, inAcademy bool) {
	var shirtNumbers []int
	if !inAcademy {
		var err error
		shirtNumbers, err = nextFreeShirtNumbers(dbCtx.Db, teamId, count)
		if err != nil {
			log.Errorf("error assigning shirt numbers for team %d: %s", teamId, err)
		}
	}
	for i := 0; i < count; i++ {
		firstName, lastName, _ := GetPlayerName(dbCtx, country)
		avatar := RandomAvatarAsString()
		goalKeeper := minSkill + rand.Intn(maxSkill-minSkill+1)
		defense := minSkill + rand.Intn(maxSkill-minSkill+1)
		lateral := minSkill + rand.Intn(maxSkill-minSkill+1)
		pass := minSkill + rand.Intn(maxSkill-minSkill+1)
		midfield := minSkill + rand.Intn(maxSkill-minSkill+1)
		dribbling := minSkill + rand.Intn(maxSkill-minSkill+1)
		shot := minSkill + rand.Intn(maxSkill-minSkill+1)
		stamina := StaminaGenerationMin + rand.Intn(StaminaGenerationMax-StaminaGenerationMin+1)
		age := minAge + rand.Intn(maxAge-minAge+1)
		wage := 0
		contractYears := 0
		if !inAcademy {
			wage = PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
			contractYears = RandomContractYears()
		}
		var shirtNumber sql.NullInt64
		if i < len(shirtNumbers) {
			shirtNumber = sql.NullInt64{Int64: int64(shirtNumbers[i]), Valid: true}
		}
		goalKeeperMax := rollSkillCap(goalKeeper)
		defenseMax := rollSkillCap(defense)
		lateralMax := rollSkillCap(lateral)
		passMax := rollSkillCap(pass)
		midfieldMax := rollSkillCap(midfield)
		dribblingMax := rollSkillCap(dribbling)
		shotMax := rollSkillCap(shot)
		var newPlayerId int64
		err := insertStmt.QueryRow(&teamId, &firstName, &lastName, time.Now().UnixMilli(),
			goalKeeper,
			defense,
			lateral,
			pass,
			midfield,
			dribbling,
			shot,
			false,
			false,
			false,
			false,
			&avatar,
			wage,
			age,
			contractYears,
			&country,
			inAcademy,
			shirtNumber,
			stamina,
			DefaultForm,
			goalKeeperMax,
			defenseMax,
			lateralMax,
			passMax,
			midfieldMax,
			dribblingMax,
			shotMax,
			RandomIntelligence()).Scan(&newPlayerId)
		if err != nil {
			log.Errorf("error creating player for team %d: %s", teamId, err)
			continue
		}
		if !inAcademy {
			if err := RecordPlayerClubHistory(historyEx, newPlayerId, teamId); err != nil {
				log.Errorf("error recording club history for new player %d team %d: %s", newPlayerId, teamId, err)
			}
		}
	}
}

// AssignTeam assigns a team to a user on a country
func AssignTeam(dbCtx *DatabaseContext, userId int64, country string) (*Team, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// A session can outlive its account (the JWT stays valid until it expires): refuse to hand a
	// club to a user who no longer exists, and hold the row so a concurrent deletion waits for
	// this claim to finish instead of leaving a club owned by a vanished id.
	var lockedUserId int64
	if err := tx.QueryRow(LockUserStmtSql, userId).Scan(&lockedUserId); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	//Get a free team
	var teamId = int64(-1)
	err = tx.QueryRow(GetFreeTeamStmtSql, &country).Scan(&teamId)
	if err != nil {
		return nil, err
	}

	//Un-assign current players
	if _, err = tx.Exec(UnAssignPlayersStmtSql, &teamId); err != nil {
		return nil, err
	}

	//Create new players
	insertStmt, err2 := tx.Prepare(InsertNewPlayerStmtSql)
	if err2 != nil {
		return nil, err2
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, tx, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, country, false)

	//Update team details
	ustmt, err := tx.Prepare(UpdateTeamDetailsStmtSql)
	if err != nil {
		return nil, err
	}
	defer ustmt.Close()
	teamName := RandomTeamName(country)
	stadiumName := RandomStadiumName(country)
	_, err = ustmt.Exec(teamName, userId, stadiumName, teamId)
	if err != nil {
		return nil, err
	}

	// The "your club was released" notice has been delivered once they pick a new club.
	if _, err := tx.Exec(ClearReleasedTeamNameStmtSql, userId); err != nil {
		return nil, err
	}

	//Get team details
	teamStmt, err := tx.Prepare(GetTeamDetailsStmtSql)
	if err != nil {
		return nil, err
	}
	defer teamStmt.Close()
	team, err := GetTeamDetails(dbCtx, teamStmt, teamId)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// The squad just created above is brand new, but any default lineup this team already had
	// (from whoever/whatever fielded it before - a CPU-managed team, or a previous owner) still
	// points at the players just orphaned by UnAssignPlayersStmtSql, not this one. Seed a
	// sensible lineup now rather than leaving the new owner's first matches fielding
	// stale/missing players (silently resolved to zero-skill ghosts by the simulation) until
	// they visit the lineup editor themselves. Best-effort: the team and its players already
	// exist regardless of whether this succeeds.
	if players, err := GetPlayersFromTeamId(dbCtx, teamId); err != nil {
		log.Errorf("error fetching new squad for team %d to seed its lineup: %s", teamId, err)
	} else if err := SetDefaultLineup(dbCtx, teamId, FlattenLineup(SelectBestLineup(*players))); err != nil {
		log.Errorf("error seeding default lineup for team %d: %s", teamId, err)
	}

	return team, nil
}

func GetLineupsForMatch(dbCtx *DatabaseContext, matchId int64, teamIds *[]int64) (*map[int64]*Lineup, error) {
	lineups := make(map[int64]*Lineup, 0)

	lineups[(*teamIds)[0]] = &Lineup{
		MatchId:       matchId,
		TeamId:        (*teamIds)[0],
		Positions:     make([]LineupPosition, 0),
		Substitutions: make([]LineupSubstitution, 0),
	}
	lineups[(*teamIds)[1]] = &Lineup{
		MatchId:       matchId,
		TeamId:        (*teamIds)[1],
		Positions:     make([]LineupPosition, 0),
		Substitutions: make([]LineupSubstitution, 0),
	}

	rows, err := dbCtx.Db.Query(GetLineupsForMatchStmtSql, (*teamIds)[0], (*teamIds)[1], matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teamId int64
	var playerId int64
	var position int
	var formation int
	var mentality int
	var roles roleScan
	for rows.Next() {
		err := rows.Scan(append([]any{&teamId, &playerId, &position, &formation, &mentality}, roles.targets()...)...)
		if err != nil {
			return nil, err
		}
		lineup := lineups[teamId]
		lineup.Formation = formation
		lineup.Mentality = mentality
		lineup.LineupRoles = roles.roles()
		lineup.Positions = append(lineup.Positions, LineupPosition{
			PlayerId: playerId,
			Position: position,
		})
	}

	return &lineups, nil
}

const InsertMatchLineupStmtSql = "INSERT INTO lineup (match_id, team_id, formation, mentality, captain_id, penalty_taker_id, free_kick_taker_id) VALUES ($1, $2, $3, $4, $5, $6, $7) " +
	"ON CONFLICT (match_id, team_id) DO UPDATE SET formation = EXCLUDED.formation, mentality = EXCLUDED.mentality, " +
	"captain_id = EXCLUDED.captain_id, penalty_taker_id = EXCLUDED.penalty_taker_id, free_kick_taker_id = EXCLUDED.free_kick_taker_id RETURNING lineup_id"
const DeleteMatchLineupPositionsStmtSql = "DELETE FROM lineup_positions WHERE lineup_id = $1"
const InsertMatchLineupPositionStmtSql = "INSERT INTO lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)"

// SetLineupForMatch replaces teamId's lineup override for one league match, along with any
// substitutions planned specifically for it (see lineup_substitutions) - the same
// upsert-then-replace-positions-and-substitutions shape as SetFriendlyLineup, guarded by the same
// non-blocking per-(matchId, teamId) advisory lock. This is what finally puts the
// lineup/lineup_positions tables to use: GetLineupsForMatch already reads them with a fallback to
// default_lineup_positions, so a row inserted here is picked up by both simulation and the
// played-match "Lineups" tab (WeekMatchById) with no further changes needed anywhere else.
func SetLineupForMatch(dbCtx *DatabaseContext, matchId int64, teamId int64, formation int, mentality int, roles LineupRoles, positions []LineupPosition, substitutions []LineupSubstitution, orders []LineupOrder) error {
	if err := ValidateMentality(mentality); err != nil {
		return err
	}
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1, $2)", matchId, teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	var lineupId int64
	if err := tx.QueryRow(InsertMatchLineupStmtSql, append([]any{matchId, teamId, formation, mentality}, roleValues(roles)...)...).Scan(&lineupId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteMatchLineupPositionsStmtSql, lineupId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertMatchLineupPositionStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	for _, p := range positions {
		if _, err := insertStmt.Exec(lineupId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(DeleteMatchLineupSubstitutionsStmtSql, matchId, teamId); err != nil {
		return err
	}
	insertSubStmt, err := tx.Prepare(InsertMatchLineupSubstitutionStmtSql)
	if err != nil {
		return err
	}
	defer insertSubStmt.Close()
	for i, s := range substitutions {
		if _, err := insertSubStmt.Exec(matchId, teamId, i+1, s.PlayerOutId, s.PlayerInId, s.Minute); err != nil {
			return err
		}
	}

	if err := replaceOrders(tx, DeleteMatchLineupOrdersStmtSql, InsertMatchLineupOrderStmtSql, []any{matchId, teamId}, orders); err != nil {
		return err
	}

	return tx.Commit()
}

var GetMatchLineupOverrideStmtSql = "SELECT a.lineup_id, a.formation, a.mentality, " + rolesSelectSql("a", matchPlayedSql("a")) + " FROM lineup a WHERE a.match_id = $1 AND a.team_id = $2"

const GetMatchLineupOverridePositionsStmtSql = "SELECT player_id, position FROM lineup_positions WHERE lineup_id = $1"

// GetMatchLineupOverride returns teamId's saved lineup override for one league match, including
// any substitutions planned specifically for it (see GetMatchLineupSubstitutions) - the
// GetFriendlyLineup equivalent for lineup/lineup_positions. hasOverride is false (with a zero
// Lineup) when the team hasn't saved one for this match yet, so callers like the lineup editor
// can fall back to showing the team's current default lineup as a starting point.
func GetMatchLineupOverride(dbCtx *DatabaseContext, matchId int64, teamId int64) (Lineup, bool, error) {
	lineup := Lineup{MatchId: matchId, TeamId: teamId, Positions: make([]LineupPosition, 0), Substitutions: make([]LineupSubstitution, 0)}

	var lineupId int64
	var roles roleScan
	err := dbCtx.Db.QueryRow(GetMatchLineupOverrideStmtSql, matchId, teamId).Scan(append([]any{&lineupId, &lineup.Formation, &lineup.Mentality}, roles.targets()...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return lineup, false, nil
	}
	if err != nil {
		return Lineup{}, false, err
	}
	lineup.LineupRoles = roles.roles()

	rows, err := dbCtx.Db.Query(GetMatchLineupOverridePositionsStmtSql, lineupId)
	if err != nil {
		return Lineup{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var p LineupPosition
		if err := rows.Scan(&p.PlayerId, &p.Position); err != nil {
			return Lineup{}, false, err
		}
		lineup.Positions = append(lineup.Positions, p)
	}

	substitutions, err := GetMatchLineupSubstitutions(dbCtx, matchId, teamId)
	if err != nil {
		return Lineup{}, false, err
	}
	lineup.Substitutions = substitutions

	return lineup, true, nil
}

func GetSimulableMatches(dbCtx *DatabaseContext) (*[]WeekMatch, error) {
	matches := make([]WeekMatch, 0)
	var matchId int64
	var divisionId int64
	var season int
	var weekNumber int
	var matchDate time.Time
	var home int64
	var away int64
	var homeGoals int
	var awayGoals int

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		log.Errorf("error starting transaction for simulable matches: %s", err)
		return nil, err
	}
	rows, err := tx.Query(GetSimulableMatchesStmtForUpdateSql)
	if err != nil {
		log.Errorf("error getting matches for simulation: %s", err)
		_ = tx.Rollback()
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&matchId, &divisionId, &season, &weekNumber, &matchDate, &home, &away, &homeGoals, &awayGoals)
		if err != nil {
			log.Errorf("error getting match for simulation: %s", err)
		} else {
			matches = append(matches, WeekMatch{
				MatchId:    matchId,
				DivisionId: divisionId,
				Season:     season,
				WeekNumber: weekNumber,
				MatchDate:  matchDate,
				Home:       home,
				Away:       away,
				HomeGoals:  homeGoals,
				AwayGoals:  awayGoals,
			})
		}
	}
	if len(matches) > 0 {
		var updateMatches = UpdateMatchSimulateStmtSql
		var inClause = "match_id in ("
		var first = true
		for _, m := range matches {
			if !first {
				inClause += ", "
			}
			inClause += strconv.FormatInt(m.MatchId, 10)
			first = false
		}
		inClause += ")"
		updateMatches += inClause
		if _, err := tx.Exec(updateMatches, true); err != nil {
			log.Errorf("error claiming matches for simulation: %s", err)
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("error committing the claim of matches for simulation: %s", err)
		return nil, err
	}
	return &matches, nil
}

// StuckSimulationAge is how long a match or friendly must have been claimed (in_simulation_since)
// before GetStuckMatches/GetStuckFriendlies treat it as abandoned. A live simulation pass finishes
// a claimed row within seconds; a large backlog within a couple of minutes. Without this age, the
// recovery job could re-simulate a row a live pass had claimed the same second - kickoffs on the
// hour are claimed 15 minutes early, exactly when the 15-minute recovery job fires - and play it
// twice.
const StuckSimulationAge = 10 * time.Minute

// GetStuckMatchesStmtSql re-claims every league match claimed (in_simulation = true) at least $1
// seconds ago (or before claims were timestamped) but never marked played, restamping
// in_simulation_since so a second replica running the same statement at the same instant finds
// nothing - see GetStuckMatches.
const GetStuckMatchesStmtSql = "WITH stuck AS (UPDATE matches SET in_simulation_since = now(), " + simulationClaimSetSql + " " +
	"WHERE played is false and in_simulation is true and " + simulationAttemptsLeftSql + " " +
	"and (in_simulation_since is null or in_simulation_since <= now() - ($1 * interval '1 second')) " +
	"RETURNING match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals) " +
	"select match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
	"from stuck order by league_id"

// GetStuckMatches re-claims and returns every league match a simulation pass claimed via
// GetSimulableMatches (setting in_simulation = true) at least StuckSimulationAge ago but never
// finished simulating - most likely because the process crashed mid-run. The caller is expected
// to re-simulate them right away (see engine.FinishStuckMatches), since GetSimulableMatches' own
// "and in_simulation is false" filter means the normal scheduler will never pick these back up on
// its own. The re-claim is one statement, so when several --engine replicas run the recovery at
// the same instant each stuck match goes to exactly one of them.
func GetStuckMatches(dbCtx *DatabaseContext) (*[]WeekMatch, error) {
	rows, err := dbCtx.Db.Query(GetStuckMatchesStmtSql, StuckSimulationAge.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]WeekMatch, 0)
	for rows.Next() {
		var m WeekMatch
		if err := rows.Scan(&m.MatchId, &m.DivisionId, &m.Season, &m.WeekNumber, &m.MatchDate,
			&m.Home, &m.Away, &m.HomeGoals, &m.AwayGoals); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return &matches, nil
}

// GetElapsedWeeksForCountry returns how many rounds country's existing leagues have already
// completed this season - the least-advanced league's played-round count. Used when a new
// division is added to an already-running country mid-season (issue #68) to know how many of its
// own weeks to immediately simulate so it doesn't lag behind everyone else. A brand new league
// (no matches yet) never contributes a row here, so it can't skew its own answer. Returns 0 if
// the country has no other league with a played match yet (e.g. it's the very start of a
// season) - the natural "nothing to catch up on" case.
func GetElapsedWeeksForCountry(dbCtx *DatabaseContext, country string, season int) (int, error) {
	var elapsed sql.NullInt64
	err := dbCtx.Db.QueryRow(
		"SELECT MIN(played_weeks) FROM ("+
			"SELECT l.league_id, COUNT(DISTINCT m.week_number) AS played_weeks "+
			"FROM leagues l JOIN matches m ON m.league_id = l.league_id AND m.season = $2 AND m.played = true "+
			"WHERE l.country = $1 GROUP BY l.league_id"+
			") t", country, season).Scan(&elapsed)
	if err != nil {
		return 0, err
	}
	if !elapsed.Valid {
		return 0, nil
	}
	return int(elapsed.Int64), nil
}

// GetMatchesForLeaguesUpToWeek returns every unplayed match among leagueIds in season with
// week_number <= maxWeek, marking them in_simulation like GetSimulableMatches does - so the
// regular per-minute scheduler (which only looks at each match's own kickoff window) can't also
// pick one of these up mid-backfill. Used to catch a newly created division up to what the rest
// of its country has already played (issue #68) by force-simulating its past-due weeks right
// away instead of waiting for their (still mostly future-dated) kickoff times.
func GetMatchesForLeaguesUpToWeek(dbCtx *DatabaseContext, leagueIds []int64, season int, maxWeek int) (*[]WeekMatch, error) {
	matches := make([]WeekMatch, 0)
	if len(leagueIds) == 0 || maxWeek <= 0 {
		return &matches, nil
	}

	leagueInClause := int64SliceToInClause(leagueIds)

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	query := "select match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
		"from matches where league_id in " + leagueInClause + " and season = $1 and week_number <= $2 " +
		"and played is false and in_simulation is false and " + simulationAttemptsLeftSql + " order by week_number, league_id"
	rows, err := tx.Query(query, season, maxWeek)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	var matchId, divisionId int64
	var seasonScan, weekNumber int
	var matchDate time.Time
	var home, away int64
	var homeGoals, awayGoals int
	for rows.Next() {
		if err := rows.Scan(&matchId, &divisionId, &seasonScan, &weekNumber, &matchDate, &home, &away, &homeGoals, &awayGoals); err != nil {
			log.Errorf("error scanning backfill match: %s", err)
			continue
		}
		matches = append(matches, WeekMatch{
			MatchId: matchId, DivisionId: divisionId, Season: seasonScan, WeekNumber: weekNumber,
			MatchDate: matchDate, Home: home, Away: away, HomeGoals: homeGoals, AwayGoals: awayGoals,
		})
	}
	rows.Close()

	if len(matches) > 0 {
		matchIds := make([]int64, len(matches))
		for i, m := range matches {
			matchIds[i] = m.MatchId
		}
		if _, err := tx.Exec("UPDATE matches SET in_simulation = true, in_simulation_since = now(), " + simulationClaimSetSql + " WHERE match_id IN " + int64SliceToInClause(matchIds)); err != nil {
			log.Errorf("error marking backfill matches in_simulation: %s", err)
		}
	}
	return &matches, tx.Commit()
}

// int64SliceToInClause renders ids as a literal "(1, 2, 3)" SQL IN-clause - safe here because
// every caller passes server-generated ids (league/match primary keys), never user input.
func int64SliceToInClause(ids []int64) string {
	clause := "("
	for i, id := range ids {
		if i > 0 {
			clause += ", "
		}
		clause += strconv.FormatInt(id, 10)
	}
	return clause + ")"
}

func GetUserDetails(userId int64, dbCtx *DatabaseContext) (*LoggedUser, error) {
	var name string
	var country string
	var provider string
	var email string
	var teamId string
	var teamName string
	var leagueId string
	var isAdmin bool
	var termsVersion sql.NullInt64
	var releasedTeamName string
	err := dbCtx.Db.QueryRow(GetUserStmtSql, userId).Scan(&name, &country, &provider, &email, &teamId, &teamName, &leagueId, &isAdmin,
		&termsVersion, &releasedTeamName)
	if err != nil {
		return nil, err
	}
	return &LoggedUser{
		Id:                      strconv.FormatInt(userId, 10),
		Name:                    name,
		Provider:                provider,
		Email:                   email,
		Country:                 country,
		TeamId:                  teamId,
		TeamName:                teamName,
		TeamLeague:              leagueId,
		IsAdmin:                 isAdmin,
		PendingCountrySelection: teamId == "0",
		TermsAccepted:           termsVersion.Valid && int(termsVersion.Int64) >= CurrentTermsVersion,
		TermsVersion:            CurrentTermsVersion,
		ReleasedTeamName:        releasedTeamName,
	}, nil
}

// SetUserCountry records the country a user picked during onboarding (see
// handlers.SelectCountryHandler) - called once, right before AssignTeam hands them a team in
// that country.
func SetUserCountry(dbCtx *DatabaseContext, userId int64, country string) error {
	_, err := dbCtx.Db.Exec("UPDATE users SET country = $1 WHERE user_id = $2", country, userId)
	return err
}

// SetUserLanguage records a user's chosen UI language (see frontend LanguageSwitcher.tsx) so
// external notification delivery (email/push) can render in that language too - see
// deliverExternalNotifications.
func SetUserLanguage(dbCtx *DatabaseContext, userId int64, language string) error {
	_, err := dbCtx.Db.Exec("UPDATE users SET language = $1 WHERE user_id = $2", language, userId)
	return err
}

// GetUserLanguage returns the user's stored language preference, or "" if none is recorded -
// callers should fall back to English in that case.
func GetUserLanguage(dbCtx *DatabaseContext, userId int64) (string, error) {
	var language sql.NullString
	err := dbCtx.Db.QueryRow("SELECT language FROM users WHERE user_id = $1", userId).Scan(&language)
	if err != nil {
		return "", err
	}
	return language.String, nil
}

// IsUserAdmin is a lightweight admin check (no join) for the RequireAdmin middleware - it's on
// the hot path of every admin request, so it skips GetUserDetails' team join entirely.
func IsUserAdmin(dbCtx *DatabaseContext, userId int64) (bool, error) {
	var isAdmin bool
	err := dbCtx.Db.QueryRow(GetUserIsAdminStmtSql, userId).Scan(&isAdmin)
	return isAdmin, err
}

// SetUserAdmin grants or revokes admin access - see IsConfiguredAdminEmail for the only way
// it's currently called (auto-promotion at login time; there's no admin-managing-admins UI).
func SetUserAdmin(dbCtx *DatabaseContext, userId int64, isAdmin bool) error {
	_, err := dbCtx.Db.Exec(SetUserAdminStmtSql, isAdmin, userId)
	return err
}

func GetUserIdByEmail(email string, dbCtx *DatabaseContext) (int64, error) {
	var userId = int64(-1)
	err := dbCtx.Db.QueryRow(GetUserIdByEmailStmtSql, &email).Scan(&userId)
	return userId, err
}

func GetTeamByOwner(userId int64, dbCtx *DatabaseContext) (*Team, error) {
	var teamId int64
	var country string
	var leagueId int64
	var name string
	var nameChanged bool
	err := dbCtx.Db.QueryRow(GetTeamDetailsStmtByOwnerSql, userId).Scan(&teamId, &country, &leagueId, &name, &nameChanged)
	if err != nil {
		return nil, err
	}
	divisionName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	return &Team{
		Id:           teamId,
		OwnerId:      userId,
		Country:      country,
		TeamName:     name,
		DivisionId:   leagueId,
		DivisionName: divisionName,
		NameChanged:  nameChanged,
	}, nil
}

func GetTeamById(teamId int64, dbCtx *DatabaseContext) (*Team, error) {
	var userId sql.NullInt64
	var country string
	var leagueId int64
	var leagueName string
	var name string
	var aiLevel int
	var aiStyle sql.NullInt64
	err := dbCtx.Db.QueryRow(GetTeamStmtByIdSql, teamId).Scan(&userId, &country, &leagueId, &leagueName, &name, &aiLevel, &aiStyle)
	if err != nil {
		return nil, err
	}
	return &Team{
		Id:           teamId,
		OwnerId:      userId.Int64,
		Country:      country,
		TeamName:     name,
		DivisionId:   leagueId,
		DivisionName: leagueName,
		AiLevel:      aiLevel,
		AiStyle:      AiStyleFromCode(aiStyle),
	}, nil
}

// GetTeamClassificationPosition reports where teamId currently sits in leagueId's classification
// for season (1-based, same ordering as GetDivisionDetails' table), or 0 if the team has no
// classification row yet (e.g. just moved into the league and not yet backfilled - see
// InsertTeamToClassificationStmtSql).
func GetTeamClassificationPosition(dbCtx *DatabaseContext, leagueId int64, season int, teamId int64) (int, error) {
	var position int
	err := dbCtx.Db.QueryRow(GetTeamClassificationPositionStmtSql, leagueId, season, teamId).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return position, nil
}

const GetTeamClassificationPointsStmtSql = "SELECT points FROM classification WHERE team_id = $1"

// GetTeamClassificationPoints returns teamId's current points in whichever league it plays in
// right now (classification is keyed on team_id alone - a team only ever has one row, in its
// current league - see the classification table's own primary key), or 0 for a team with no
// classification row at all: a national team (used as a cup-seeding signal by engine.bracket.go
// for a World Cup knockout bracket, which has no club-league standing to speak of) or a club team
// that hasn't been backfilled into one yet.
func GetTeamClassificationPoints(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var points int
	err := dbCtx.Db.QueryRow(GetTeamClassificationPointsStmtSql, teamId).Scan(&points)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return points, nil
}

func GetDivisionDetails(dbCtx *DatabaseContext, leagueId int64) (*DivisionDetails, error) {
	leagueName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	currentSeason, err := GetCurrentSeasonForLeague(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}

	rows, err := dbCtx.Db.Query(GetDivisionDetailsStmtSql, leagueId, currentSeason)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Classification
	positions := make([]ClassificationPosition, TeamsPerLeague)
	teamPosition := 1
	for rows.Next() {
		var teamId int64
		var teamName string
		var matches int
		var points int
		var wins int
		var losses int
		var draws int
		var goalsScored int
		var goalsReceived int
		var seed sql.NullInt64

		rows.Scan(&teamId, &teamName, &matches, &points, &wins, &losses, &draws, &goalsScored, &goalsReceived, &seed)
		var seedPtr *int
		if seed.Valid {
			v := int(seed.Int64)
			seedPtr = &v
		}
		positions[teamPosition-1] = ClassificationPosition{
			Position:      teamPosition,
			TeamId:        teamId,
			TeamName:      teamName,
			Points:        points,
			Matches:       matches,
			Wins:          wins,
			Draws:         draws,
			Losses:        losses,
			GoalsScored:   goalsScored,
			GoalsReceived: goalsReceived,
			Seed:          seedPtr,
		}
		teamPosition++
	}

	var currentWeek int
	currentWeek = positions[0].Matches
	//Last week matches
	var previousWeekMatches *[]WeekMatch
	var nextWeekMatches *[]WeekMatch
	if currentWeek >= 1 {
		previousWeekMatches, err = WeekMatches(dbCtx, leagueId, currentSeason, currentWeek)
	}
	//Next week matches
	nextWeekMatches, err = WeekMatches(dbCtx, leagueId, currentSeason, currentWeek+1)

	if nextWeekMatches == nil {
		aux := make([]WeekMatch, 0)
		nextWeekMatches = &aux
	}
	if previousWeekMatches == nil {
		aux := make([]WeekMatch, 0)
		previousWeekMatches = &aux
	}

	zones, err := GetLeagueZones(dbCtx, leagueId)
	if err != nil {
		// A lookup failure just means "no zones": the table is still correct without the markers.
		log.Errorf("error reading the zones of league %d: %s", leagueId, err)
		zones = Zones{}
	}
	playoff, err := GetLeaguePlayoffTies(dbCtx, leagueId)
	if err != nil {
		log.Errorf("error reading the playoff ties of league %d: %s", leagueId, err)
		playoff = make([]CupTie, 0)
	}

	return &DivisionDetails{
		Id:     leagueId,
		Season: currentSeason,
		Name:   leagueName,
		Classification: Classification{
			Positions: positions,
		},
		PreviousWeekMatches: *previousWeekMatches,
		NextWeekMatches:     *nextWeekMatches,
		Zones:               zones,
		Playoff:             playoff,
	}, nil
}

func WeekMatches(dbCtx *DatabaseContext, leagueId int64, season int, weekNumber int) (*[]WeekMatch, error) {
	rows, err := dbCtx.Db.Query(GetWeekMatchesStmtSql, leagueId, weekNumber, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]WeekMatch, 0)
	for rows.Next() {
		matches = append(matches, *readWeekMatch(rows))
	}
	return &matches, nil
}

func readWeekMatch(rows *sql.Rows) *WeekMatch {
	var matchId int64
	var leagueId int64
	var season int
	var played bool
	var inSimulation bool
	var weekNumber int
	var matchDate time.Time
	var homeTeamId int64
	var awayTeamId int64
	var homeGoals int
	var awayGoals int
	var homeTeamName string
	var awayTeamName string
	var homeStadiumName sql.NullString
	var weather sql.NullString
	var attendance sql.NullInt64
	var refereeName sql.NullString
	var assistant1Name sql.NullString
	var assistant2Name sql.NullString
	rows.Scan(&matchId, &leagueId, &season, &played, &inSimulation, &weekNumber, &matchDate, &homeTeamId, &awayTeamId, &homeGoals, &awayGoals,
		&weather, &attendance, &refereeName, &assistant1Name, &assistant2Name,
		&homeTeamName, &awayTeamName, &homeStadiumName)
	return &WeekMatch{
		MatchId:         matchId,
		DivisionId:      leagueId,
		Season:          season,
		WeekNumber:      weekNumber,
		MatchDate:       matchDate,
		Played:          played,
		InSimulation:    inSimulation,
		Home:            homeTeamId,
		HomeName:        homeTeamName,
		HomeStadiumName: homeStadiumName.String,
		Away:            awayTeamId,
		AwayName:        awayTeamName,
		HomeGoals:       homeGoals,
		AwayGoals:       awayGoals,
		Weather:         weather.String,
		Attendance:      int(attendance.Int64),
		RefereeName:     refereeName.String,
		Assistant1Name:  assistant1Name.String,
		Assistant2Name:  assistant2Name.String,
	}
}

func GetDivisionName(dbCtx *DatabaseContext, leagueId int64) (string, error) {
	divs, err := dbCtx.Db.Query(GetDivisionNameStmtSql, leagueId)
	if err != nil {
		return "", err
	}
	defer divs.Close()
	var divisionName string
	if divs.Next() {
		divs.Scan(&divisionName)
	}
	if divisionName == "" {
		return "", errors.New("league " + strconv.FormatInt(leagueId, 10) + " not found")
	}
	return divisionName, nil
}

func GetFullTeamDetailsByUser(userId int64, isOwner bool, dbCtx *DatabaseContext) (*TeamDetails, error) {
	team, errTeam := GetTeamByOwner(userId, dbCtx)
	if errTeam != nil {
		return nil, errTeam
	}

	players, err2 := GetPlayersFromTeamId(dbCtx, team.Id)
	if err2 != nil {
		return nil, err2
	}

	unavailability, err3 := GetTeamUnavailability(dbCtx, team.Id)
	if err3 != nil {
		return nil, err3
	}

	transferListed, err4 := GetTransferListedPlayerIdsForTeam(dbCtx, team.Id)
	if err4 != nil {
		return nil, err4
	}
	transferListedIds := make([]int64, 0, len(transferListed))
	for playerId := range transferListed {
		transferListedIds = append(transferListedIds, playerId)
	}

	// The dashboard's Afición tile reads this straight off /v1/user/team - leaving it out made
	// every manager's own fanbase render as 0 while their finances page showed the real value.
	fanbase, err5 := GetFanbase(dbCtx, team.Id)
	if err5 != nil {
		return nil, err5
	}

	loanedOut, err6 := GetLoanedOutPlayers(dbCtx, team.Id)
	if err6 != nil {
		return nil, err6
	}
	loanedIn, err7 := GetLoanedInPlayers(dbCtx, team.Id)
	if err7 != nil {
		return nil, err7
	}

	return &TeamDetails{
		Team:                    team,
		Players:                 *players,
		Fanbase:                 fanbase,
		Unavailability:          unavailability,
		TransferListedPlayerIds: transferListedIds,
		LoanedOut:               loanedOut,
		LoanedIn:                loanedIn,
	}, nil
}

func WeekMatchById(dbCtx *DatabaseContext, matchId int64) (*WeekMatch, error) {
	rows, err := dbCtx.Db.Query(GetWeekMatchesByIdStmtSql, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	match := readWeekMatch(rows)
	rows.Close()
	match.Rivalry = GetFixtureRivalry(dbCtx, MatchKindLeague, matchId)
	match.Events = GetMatchEvents(dbCtx, matchId)
	match.MatchInstants = GetMatchInstants(dbCtx, matchId)
	// The starting XIs, kept before any substitution is replayed below - the ratings fallback for
	// a match simulated before stored ratings needs to know who started (see LoadPlayerRatings).
	var startingHome, startingAway *Lineup
	lineups, err := GetLineupsForMatch(dbCtx, matchId, &[]int64{match.Home, match.Away})
	if err != nil {
		log.Errorf("error getting line ups for match: %v", err)
	} else {
		match.HomeLineup = (*lineups)[match.Home]
		match.AwayLineup = (*lineups)[match.Away]
		// The set-piece takers are the owning manager's business; only the captain's armband is public.
		RedactLineupTakers(match.HomeLineup)
		RedactLineupTakers(match.AwayLineup)
		startingHome, startingAway = CopyLineup(match.HomeLineup), CopyLineup(match.AwayLineup)
		// GetLineupsForMatch only ever returns the XI submitted before kickoff - replay every
		// substitution this match actually had on top of it, so a finished match's report shows
		// who was really on the pitch (see ApplySubstitutionToLineup's own doc comment).
		if match.Events != nil {
			for _, event := range *match.Events {
				ApplySubstitutionToLineup(match.HomeLineup, event)
				ApplySubstitutionToLineup(match.AwayLineup, event)
			}
		}
	}
	homePlayers, err := GetPlayersForMatchTeam(dbCtx, match.Home, match.HomeLineup)
	if err != nil {
		log.Errorf("error getting home team players: %v", err)
	} else {
		pl := playerListToMap(homePlayers)
		match.HomePlayers = &pl
	}
	awayPlayers, err := GetPlayersForMatchTeam(dbCtx, match.Away, match.AwayLineup)
	if err != nil {
		log.Errorf("error getting away team players: %v", err)
	} else {
		pl := playerListToMap(awayPlayers)
		match.AwayPlayers = &pl
	}
	// Only meaningful once the match has actually happened - GetLineupsForMatch above can return
	// a not-yet-played match's default/pre-selected lineup too (see its own doc comment), which
	// the ratings alone can't tell apart from a genuinely played one.
	if match.Played {
		match.PlayerRatings = LoadPlayerRatings(dbCtx, SimulationKindLeague, matchId, match.Home, match.Away,
			startingHome, startingAway, match.MatchInstants, match.Events)
		match.Ratings = RatingsById(match.PlayerRatings)
		if match.HomePlayers != nil && match.AwayPlayers != nil {
			addRatedPlayers(dbCtx, match.PlayerRatings, match.Home, *match.HomePlayers, *match.AwayPlayers)
		}
	}

	return match, nil
}

func playerListToMap(players *[]*Player) map[int64]Player {
	m := make(map[int64]Player)
	for _, player := range *players {
		m[player.Id] = *player
	}
	return m
}

func GetMatchEvents(dbCtx *DatabaseContext, matchId int64) *[]Event {
	rows, err := dbCtx.Db.Query(GetWeekMatchesEventsByIdStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		var eventId int64
		var matchId int64
		var eventType int
		var teamOwner int64
		var otherTeam int64
		var minute int
		var teamPlayers string
		var teamPlayerNames string
		var otherTeamPlayers string
		var otherTeamPlayerNames string
		var homeGoals int
		var awayGoals int
		var setPiece int
		var mentality sql.NullInt64
		var addedMinute int
		var addedTime sql.NullInt64
		rows.Scan(&eventId, &matchId, &eventType, &teamOwner, &otherTeam, &minute, &teamPlayers, &teamPlayerNames,
			&otherTeamPlayers, &otherTeamPlayerNames, &homeGoals, &awayGoals, &setPiece, &mentality, &addedMinute, &addedTime)

		event := Event{
			EventType:             eventType,
			Minute:                minute,
			TeamOwner:             teamOwner,
			TeamOwnerPlayers:      nil,
			TeamOwnerPlayersNames: nil,
			OtherTeam:             otherTeam,
			OtherTeamPlayers:      nil,
			OtherTeamPlayersNames: nil,
			HomeGoals:             homeGoals,
			AwayGoals:             awayGoals,
			SetPiece:              setPiece,
			Mentality:             nullableInt(mentality),
			AddedMinute:           addedMinute,
			AddedTime:             nullableInt(addedTime),
		}
		if teamPlayers != "" {
			players := make([]int64, 0)
			_ = json.Unmarshal([]byte(teamPlayers), &players)
			event.TeamOwnerPlayers = players
		}
		if teamPlayerNames != "" {
			players := make([]string, 0)
			_ = json.Unmarshal([]byte(teamPlayerNames), &players)
			event.TeamOwnerPlayersNames = players
		}
		if otherTeamPlayers != "" {
			players := make([]int64, 0)
			_ = json.Unmarshal([]byte(otherTeamPlayers), &players)
			event.OtherTeamPlayers = players
		}
		if otherTeamPlayerNames != "" {
			players := make([]string, 0)
			_ = json.Unmarshal([]byte(otherTeamPlayerNames), &players)
			event.OtherTeamPlayersNames = players
		}
		events = append(events, event)
	}
	return &events
}

func GetMatchInstants(dbCtx *DatabaseContext, matchId int64) *[]MatchInstant {
	rows, err := dbCtx.Db.Query(GetWeekMatchesInstantsByIdStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()
	instants := make([]MatchInstant, 0)
	for rows.Next() {
		var matchId int64
		var minute int
		var homeGoals int
		var awayGoals int
		var homeQua string
		var awayQua string
		var midFieldDiff float32
		var homeAttackRightDiff float32
		var homeAttackLeftDiff float32
		var homeAttackCenterDiff float32
		var awayAttackRightDiff float32
		var awayAttackLeftDiff float32
		var awayAttackCenterDiff float32
		var contributions sql.NullString
		var addedMinute int
		rows.Scan(&matchId, &minute, &homeGoals, &awayGoals, &homeQua, &awayQua, &midFieldDiff,
			&homeAttackRightDiff, &homeAttackLeftDiff, &homeAttackCenterDiff,
			&awayAttackRightDiff, &awayAttackLeftDiff, &awayAttackCenterDiff, &contributions, &addedMinute)

		homeTeamQua := TeamQualification{}
		awayTeamQua := TeamQualification{}
		if homeQua != "" {
			_ = json.Unmarshal([]byte(homeQua), &homeTeamQua)
		}
		if awayQua != "" {
			_ = json.Unmarshal([]byte(awayQua), &awayTeamQua)
		}
		instants = append(instants, MatchInstant{
			MatchId:              matchId,
			Minute:               minute,
			AddedMinute:          addedMinute,
			HomeGoals:            homeGoals,
			AwayGoals:            awayGoals,
			HomeQua:              homeTeamQua,
			AwayQua:              awayTeamQua,
			MidfieldDiff:         midFieldDiff,
			HomeAttackRightDiff:  homeAttackRightDiff,
			HomeAttackLeftDiff:   homeAttackLeftDiff,
			HomeAttackCenterDiff: homeAttackCenterDiff,
			AwayAttackRightDiff:  awayAttackRightDiff,
			AwayAttackLeftDiff:   awayAttackLeftDiff,
			AwayAttackCenterDiff: awayAttackCenterDiff,
			PlayerContributions:  decodePlayerContributions(contributions),
		})
	}
	return &instants
}

func GetTeamDetailsById(teamId int64, isOwner bool, dbCtx *DatabaseContext) (*TeamDetails, error) {
	country, isNationalTeam, errNational := GetNationalTeamCountry(dbCtx, teamId)
	if errNational != nil {
		return nil, errNational
	}
	if isNationalTeam {
		return getNationalTeamDetails(dbCtx, teamId, country)
	}
	if dissolved, err := getDissolvedTeamDetails(dbCtx, teamId); err != nil || dissolved != nil {
		return dissolved, err
	}

	team, errTeam := GetTeamById(teamId, dbCtx)
	if errTeam != nil {
		return nil, errTeam
	}
	// AiLevel is an internal difficulty parameter for engine.RunAITeamManagement (see
	// engine/aiteams.go), never a manager-facing fact about any team, owned or not (#19) - clear
	// it here rather than in GetTeamById itself, since that function is also how the engine reads
	// a team's real level to decide how to manage it.
	team.AiLevel = 0

	// GetTeamById never computes a team's league standing (it's also the engine's read path, which
	// has no use for it) - filled in here instead, same current-season classification
	// GetDivisionDetails ranks the whole league table from, so a team's page shows the real
	// position instead of always reading 0.
	currentSeason, err := GetCurrentSeasonForLeague(dbCtx, team.DivisionId)
	if err != nil {
		return nil, err
	}
	position, err := GetTeamClassificationPosition(dbCtx, team.DivisionId, currentSeason, teamId)
	if err != nil {
		return nil, err
	}
	team.ClassificationPosition = position

	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	// AverageSkill (unlike the underlying skill fields RedactSkills zeroes below) is meant to stay
	// visible on someone else's rostered player too, the same rough-scouting exception
	// handlers.PlayerDetailsHandler already makes for a single player's page - so it has to be
	// computed from the real skills before any redaction happens, own team or not.
	for _, p := range *players {
		p.AverageSkill = AverageSkill(*p)
	}

	// A squad's contract terms and detailed skill breakdown are the owning manager's business, not
	// any visitor's - see Player.RedactFinancials/RedactSkills. Only reached for someone else's
	// team; GetFullTeamDetailsByUser (the caller's own team) always passes isOwner true. A player
	// this team has put up for transfer is the one exception to the skill redaction - a
	// prospective buyer needs to see what they'd actually be signing (a rostered player is never
	// a free agent, so that exemption never applies here, unlike in RedactSkills' other callers).
	// Wage is a second, narrower exception (restored right after RedactFinancials zeroes it): it
	// stays visible on someone else's rostered player too, mirroring handlers.PlayerDetailsHandler,
	// so a visitor gets a rough sense of what signing this team's squad would cost.
	if !isOwner {
		listedPlayerIds, err := GetTransferListedPlayerIdsForTeam(dbCtx, teamId)
		if err != nil {
			return nil, err
		}
		for _, p := range *players {
			wage := p.Wage
			p.RedactFinancials()
			p.Wage = wage
			p.RedactSkills(listedPlayerIds[p.Id])
		}
	}

	stadium, err := GetStadium(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	lastHomeMatch, err := GetLastHomeMatch(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	fanbase, err := GetFanbase(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	trainer, err := GetTeamTrainer(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	loanedOut, err := GetLoanedOutPlayers(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	loanedIn, err := GetLoanedInPlayers(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	// A loaned-out player is someone else's rostered player now, so a visitor gets the same
	// redaction as for the club's own squad (wage stays visible, see above).
	if !isOwner {
		for i := range loanedOut {
			p := &loanedOut[i].Player
			wage := p.Wage
			p.RedactFinancials()
			p.Wage = wage
			p.RedactSkills(false)
		}
	}

	return &TeamDetails{
		Team:    team,
		Players: *players,
		Fanbase: fanbase,
		Stadium: &StadiumSummary{
			Name:          stadium.Name,
			Capacity:      stadium.Capacity,
			LastHomeMatch: lastHomeMatch,
		},
		Trainer:   trainer,
		LoanedOut: loanedOut,
		LoanedIn:  loanedIn,
	}, nil
}

const GetNationalTeamNameStmtSql = "SELECT name FROM teams WHERE team_id = $1 AND is_national_team = true"

// getNationalTeamDetails is GetTeamDetailsById's national-team counterpart (issue: a national
// team's page should show its recent matches, players, and coach) - a national team has no real
// league standing, stadium, or persistent roster of its own (see common/nationalteam.go), so
// GetTeamById's division JOIN would fail for it (league_id stays at -1) and GetPlayersFromTeamId
// would come back empty (its actual players never leave their real clubs' rosters). Its "squad" is
// shown instead as the country's current pool of eligible players by birth country (see
// GetPlayersByBirthCountry) - the same pool a weekly friendly's lineup is picked from - redacted
// the same way any other team's squad is for a visitor (GetTeamDetailsById's own !isOwner branch).
// There's deliberately no coach: a national team has no manager to hire one (see
// openspec/specs/national-teams), so the frontend simply never shows a
// CoachCard for it, the same way it already doesn't for any team that isn't the caller's own.
func getNationalTeamDetails(dbCtx *DatabaseContext, teamId int64, country string) (*TeamDetails, error) {
	var name string
	if err := dbCtx.Db.QueryRow(GetNationalTeamNameStmtSql, teamId).Scan(&name); err != nil {
		return nil, err
	}

	players, err := GetPlayersByBirthCountry(dbCtx, country)
	if err != nil {
		return nil, err
	}
	for _, p := range players {
		p.AverageSkill = AverageSkill(*p)
		wage := p.Wage
		p.RedactFinancials()
		p.Wage = wage
		p.RedactSkills(false)
	}

	return &TeamDetails{
		Team: &Team{
			Id:             teamId,
			Country:        country,
			TeamName:       name,
			IsNationalTeam: true,
		},
		Players:   players,
		LoanedOut: []LoanedOutPlayer{},
		LoanedIn:  []LoanedInPlayer{},
	}, nil
}

func GetTeamOwner(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	rows, err := dbCtx.Db.Query(GetTeamOwnerStmtSql, teamId)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	var userId = int64(-1)
	if rows.Next() {
		rows.Scan(&userId)
	}
	return userId, nil
}

func GetCurrentSeasonForCountry(dbCtx *DatabaseContext, country string) (int, error) {
	var season int
	err := dbCtx.Db.QueryRow(GetCurrentSeasonForCountryStmtSql, country).Scan(&season)
	return season, err
}

func SetCurrentSeasonForCountry(dbCtx *DatabaseContext, country string, season int) error {
	_, err := dbCtx.Db.Exec(SetCurrentSeasonForCountryStmtSql, season, country)
	return err
}

// GetCurrentSeasonForLeague resolves a league's own country's current season - for a caller
// that has a leagueId but not the bare country code (see GetDivisionDetails,
// engine.RescheduleUnplayedMatches).
func GetCurrentSeasonForLeague(dbCtx *DatabaseContext, leagueId int64) (int, error) {
	var season int
	err := dbCtx.Db.QueryRow(GetCurrentSeasonForLeagueStmtSql, leagueId).Scan(&season)
	return season, err
}

// GetCountryForLeague resolves a league's own country code - used by callers that only have a
// leagueId in scope but need the country itself, not just its current season (see
// engine.GenerateMatchesForLeagues).
func GetCountryForLeague(dbCtx *DatabaseContext, leagueId int64) (string, error) {
	var country string
	err := dbCtx.Db.QueryRow(GetCountryForLeagueStmtSql, leagueId).Scan(&country)
	return country, err
}

// GetCurrentSeasonForTeam resolves a team's own country's current season straight from
// teams.country, without needing its league_id - works the same for a national team
// (league_id = -1, but still a real teams.country) as for a club team. See AcceptLoan,
// SetSponsorDeal/IsSponsorDealLocked, engine.updateSkillsAfterPlayedMatch.
func GetCurrentSeasonForTeam(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var season int
	err := dbCtx.Db.QueryRow(GetCurrentSeasonForTeamStmtSql, teamId).Scan(&season)
	return season, err
}

const GetTransferWindowOpenStmtSql = "SELECT transfer_window_open FROM season"
const SetTransferWindowOpenStmtSql = "UPDATE season SET transfer_window_open = $1"

// IsTransferWindowOpen reports whether the transfer/loan markets are currently open - a single
// global, admin-toggled flag (see handlers.SetTransferWindowHandler) rather than anything
// derived from the season schedule, since fixtures are generated per-league on independent
// clocks and there's no single "week of the season" to hang an automatic window on. Fails open
// (returns true) on a lookup error, so a DB hiccup can't silently freeze the whole market.
func IsTransferWindowOpen(dbCtx *DatabaseContext) bool {
	var open bool
	if err := dbCtx.Db.QueryRow(GetTransferWindowOpenStmtSql).Scan(&open); err != nil {
		return true
	}
	return open
}

func SetTransferWindowOpen(dbCtx *DatabaseContext, open bool) error {
	_, err := dbCtx.Db.Exec(SetTransferWindowOpenStmtSql, open)
	return err
}

// international_cup_season/last_international_cup_check (added by
// init75_season_per_country.sql) drive the decoupled World Cup cadence (see
// engine.StartWorldCupIfDue) - unlike a domestic season, these can no
// longer be tied to any one country's own rollover, so they get their own counter and their own
// "when did we last check" timestamp instead.
// $1 is the cadence in seconds (time.Duration.Seconds()) - built into an interval via
// `* interval '1 second'` rather than relying on Postgres parsing Go's own Duration.String()
// format, which isn't guaranteed to round-trip through interval's input parser.
const ClaimInternationalCupSeasonStmtSql = "UPDATE season SET international_cup_season = international_cup_season + 1, " +
	"last_international_cup_check = now() WHERE last_international_cup_check IS NULL OR " +
	"last_international_cup_check <= now() - ($1 * interval '1 second') RETURNING international_cup_season"

// ClaimInternationalCupSeason increments the decoupled international-cup season counter and
// stamps last_international_cup_check with now(), but only if at least cadence has passed since
// the last check (or none has ever run yet), returning the new season number and true. It returns
// false when the round is not due. The due check and the increment are one statement, so when
// several --engine replicas fire the job at the same instant exactly one of them claims the round:
// the others wait on the winner's row lock and then see the stamp it just wrote. Checking and
// incrementing in two statements let both replicas start a round, giving two sets of
// international cups (and two World Cups) in one go.
func ClaimInternationalCupSeason(dbCtx *DatabaseContext, cadence time.Duration) (int, bool, error) {
	var season int
	err := dbCtx.Db.QueryRow(ClaimInternationalCupSeasonStmtSql, cadence.Seconds()).Scan(&season)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return season, true, nil
}

func GetAllLeagueIds(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetAllLeagueIdsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagueIds := make([]int64, 0)
	for rows.Next() {
		var leagueId int64
		if err := rows.Scan(&leagueId); err != nil {
			return nil, err
		}
		leagueIds = append(leagueIds, leagueId)
	}
	return leagueIds, nil
}

// GetSeasonMatchCounts returns how many matches exist for a season and how many of those
// are still unplayed, so callers can tell "season complete" (total > 0, unplayed == 0) apart
// from "season never started" (total == 0).
func GetSeasonMatchCounts(dbCtx *DatabaseContext, country string, season int) (total int, unplayed int, err error) {
	err = dbCtx.Db.QueryRow(GetSeasonMatchCountsStmtSql, country, season).Scan(&unplayed, &total)
	return total, unplayed, err
}

// CountSeasonMatchesStillBroadcasting returns how many already-simulated matches of a season
// are still inside their live-broadcast window (see LiveBroadcastWindow): played = true, but
// match_date is more recent than LiveBroadcastCutoff. A season whose last match is still being
// broadcast is not safe to close yet - see engine.AdvanceSeasonsIfComplete.
func CountSeasonMatchesStillBroadcasting(dbCtx *DatabaseContext, country string, season int) (count int, err error) {
	err = dbCtx.Db.QueryRow(CountSeasonMatchesStillBroadcastingStmtSql, country, season, LiveBroadcastCutoff()).Scan(&count)
	return count, err
}

// ResetClassificationForSeason rolls a league's existing classification rows into a new
// season. Note: the classification table's primary key is (league_id, team_id) - it has no
// season column in its key, so only the *current* season's standings can ever be stored per
// team; there is no historical per-season archive. Resetting in place (rather than inserting
// a new row per season) is what that constraint allows without a schema migration.
func ResetClassificationForSeason(dbCtx *DatabaseContext, leagueId int64, season int) error {
	_, err := dbCtx.Db.Exec(ResetClassificationForSeasonStmtSql, season, leagueId)
	return err
}

// GetLeaguesWithTier lists every league's country and tier (1 = top division), used to build
// each country's promotion/relegation pyramid.
func GetLeaguesWithTier(dbCtx *DatabaseContext) ([]LeagueTierInfo, error) {
	rows, err := dbCtx.Db.Query(GetLeaguesWithTierStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagues := make([]LeagueTierInfo, 0)
	for rows.Next() {
		var info LeagueTierInfo
		if err := rows.Scan(&info.LeagueId, &info.Country, &info.Tier); err != nil {
			return nil, err
		}
		leagues = append(leagues, info)
	}
	return leagues, nil
}

const GetLeaguesWithTierForCountryStmtSql = "SELECT league_id, country, tier FROM leagues WHERE country = $1 AND archived_season IS NULL ORDER BY tier, group_number"

// GetLeaguesWithTierForCountry is GetLeaguesWithTier scoped to one country - used to build just
// that country's own promotion/relegation pyramid when it rolls its season over on its own (see
// engine.applyPromotionRelegation, engine.AdvanceSeasonsIfComplete).
func GetLeaguesWithTierForCountry(dbCtx *DatabaseContext, country string) ([]LeagueTierInfo, error) {
	rows, err := dbCtx.Db.Query(GetLeaguesWithTierForCountryStmtSql, country)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagues := make([]LeagueTierInfo, 0)
	for rows.Next() {
		var info LeagueTierInfo
		if err := rows.Scan(&info.LeagueId, &info.Country, &info.Tier); err != nil {
			return nil, err
		}
		leagues = append(leagues, info)
	}
	return leagues, nil
}

const GetTeamTierStmtSql = "SELECT l.tier FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.team_id = $1"

// GetTeamTier returns the tier (1 = top division) of the league a team currently plays in -
// used to scale tier-dependent per-match income like sponsorship (see engine.creditSponsorship).
func GetTeamTier(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var tier int
	err := dbCtx.Db.QueryRow(GetTeamTierStmtSql, teamId).Scan(&tier)
	return tier, err
}

// GetRankedClassification returns a league's teams ordered best to worst (standingsOrderSql, the
// same ordering GetDivisionDetailsStmtSql uses).
func GetRankedClassification(dbCtx *DatabaseContext, leagueId int64) ([]TeamStanding, error) {
	rows, err := dbCtx.Db.Query(GetRankedClassificationStmtSql, leagueId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	standings := make([]TeamStanding, 0)
	for rows.Next() {
		var s TeamStanding
		if err := rows.Scan(&s.TeamId, &s.Points, &s.GoalDiff, &s.GoalsScored); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}
	return standings, nil
}

// UpdateTeamLeague reassigns a team to a different league (promotion/relegation).
func UpdateTeamLeague(dbCtx *DatabaseContext, teamId int64, newLeagueId int64) error {
	_, err := dbCtx.Db.Exec(UpdateTeamLeagueStmtSql, newLeagueId, teamId)
	return err
}

// GetFinalClassification returns a league's full table lines best to worst (standingsOrderSql),
// with Position set to the 1-based rank. TeamName is left empty.
func GetFinalClassification(dbCtx *DatabaseContext, leagueId int64) ([]ClassificationPosition, error) {
	rows, err := dbCtx.Db.Query(GetFinalClassificationStmtSql, leagueId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]ClassificationPosition, 0)
	for rows.Next() {
		var l ClassificationPosition
		if err := rows.Scan(&l.TeamId, &l.Matches, &l.Points, &l.Wins, &l.Draws, &l.Losses, &l.GoalsScored, &l.GoalsReceived); err != nil {
			return nil, err
		}
		l.Position = len(lines) + 1
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// SetLeagueSeeds replaces a league's starting order (classification.seed) with seeds, team id to
// seed, in one transaction: the league's other rows are left without a seed, so they sort after
// every seeded team. Rerunning it with the same seeds writes the same values.
func SetLeagueSeeds(dbCtx *DatabaseContext, leagueId int64, seeds map[int64]int) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(clearLeagueSeedsStmtSql, leagueId); err != nil {
		return err
	}
	for teamId, seed := range seeds {
		if _, err := tx.Exec(UpdateClassificationSeedStmtSql, seed, leagueId, teamId); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateClassificationLeague repoints a team's existing classification row at its new league
// after a promotion/relegation move, so the season reset that follows (which operates
// per-league) picks it up correctly.
func UpdateClassificationLeague(dbCtx *DatabaseContext, teamId int64, newLeagueId int64) error {
	_, err := dbCtx.Db.Exec(UpdateClassificationLeagueStmtSql, newLeagueId, teamId)
	return err
}

// AddPlayerUnavailability records a player as unavailable (suspended or injured) for their
// next `matches` matches. If the player is already unavailable, this only ever extends the
// duration (never shortens an existing suspension/injury) and keeps whichever reason belongs
// to the longer outstanding duration.
func AddPlayerUnavailability(dbCtx *DatabaseContext, playerId int64, reason int, matches int) error {
	_, err := dbCtx.Db.Exec(UpsertPlayerUnavailabilityStmtSql, playerId, reason, matches)
	return err
}

// GetUnavailablePlayerIds returns the set of a team's players currently serving a suspension
// or injury.
func GetUnavailablePlayerIds(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	return queryUnavailablePlayerIds(dbCtx, GetUnavailablePlayerIdsStmtSql, teamId)
}

// GetUnavailablePlayerIdsForFriendly returns the set of a team's players unavailable for a
// friendly match: injured players still count, but red-card suspensions don't, since a
// suspension only bars official league/cup matches.
func GetUnavailablePlayerIdsForFriendly(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	return queryUnavailablePlayerIds(dbCtx, GetUnavailablePlayerIdsForFriendlyStmtSql, teamId)
}

func queryUnavailablePlayerIds(dbCtx *DatabaseContext, stmt string, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(stmt, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	unavailable := make(map[int64]bool)
	for rows.Next() {
		var playerId int64
		if err := rows.Scan(&playerId); err != nil {
			return nil, err
		}
		unavailable[playerId] = true
	}
	return unavailable, nil
}

// GetTeamUnavailability returns every one of a team's players currently serving a suspension or
// injury, with its reason and how many matches it still has to run.
func GetTeamUnavailability(dbCtx *DatabaseContext, teamId int64) ([]PlayerUnavailability, error) {
	rows, err := dbCtx.Db.Query(GetTeamUnavailabilityStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unavailability []PlayerUnavailability
	for rows.Next() {
		var u PlayerUnavailability
		if err := rows.Scan(&u.PlayerId, &u.Reason, &u.MatchesRemaining); err != nil {
			return nil, err
		}
		unavailability = append(unavailability, u)
	}
	return unavailability, rows.Err()
}

// DecrementUnavailabilityForTeam counts a played match against every currently-unavailable
// player on a team, and clears anyone whose suspension/injury has now run out.
func DecrementUnavailabilityForTeam(dbCtx *DatabaseContext, teamId int64) error {
	if _, err := dbCtx.Db.Exec(DecrementUnavailabilityForTeamStmtSql, teamId); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(DeleteExpiredUnavailabilityStmtSql)
	return err
}

func GetTeamBudget(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	var budget int64
	err := dbCtx.Db.QueryRow(GetTeamBudgetStmtSql, teamId).Scan(&budget)
	return budget, err
}

// ErrTeamInsolvent is returned by any spend-side action (buying, signing, borrowing) a team
// attempts while IsTeamSolvent reports false - see MaxDebt.
var ErrTeamInsolvent = errors.New("team budget is too far in debt for new spending")

// IsTeamSolvent reports whether teamId's budget is still above -MaxDebt, i.e. whether it's
// allowed to take on new spending (a transfer buy, a free-agent signing, borrowing a player on
// loan) - see the callers of ErrTeamInsolvent for exactly which actions this gates. Selling a
// player, releasing one, or cancelling a listing are never gated by this: those only ever help a
// team's budget, and are exactly how an insolvent team is meant to dig back out. Fails open
// (reports solvent) on a lookup error, the same convention IsTransferWindowOpen uses, so a DB
// hiccup can't silently freeze the whole market.
func IsTeamSolvent(dbCtx *DatabaseContext, teamId int64) bool {
	budget, err := GetTeamBudget(dbCtx, teamId)
	if err != nil {
		return true
	}
	return budget >= -MaxDebt
}

// AdjustTeamBudget changes a team's budget by `amount` (negative for an expense) and records
// it in the ledger, atomically.
func AdjustTeamBudget(dbCtx *DatabaseContext, teamId int64, amount int64, reason int, description string) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, amount, teamId); err != nil {
		return err
	}
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, teamId, amount, reason, description, nil); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	flows.Commit(context.Background())
	return nil
}

// AdjustAllTeamsBudget changes every team's budget by `amount` (negative for a deduction) and
// records the adjustment in each team's own ledger, atomically. It returns how many teams were
// affected.
func AdjustAllTeamsBudget(dbCtx *DatabaseContext, amount int64, reason int, description string) (int64, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(UpdateAllTeamsBudgetStmtSql, amount)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	var flows LedgerFlows
	if err := insertLedgerRowForAllTeams(tx, &flows, affected, amount, reason, description); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	flows.Commit(context.Background())
	return affected, nil
}

func GetTeamTransactions(dbCtx *DatabaseContext, teamId int64, limit int) ([]TeamTransaction, error) {
	rows, err := dbCtx.Db.Query(GetTeamTransactionsStmtSql, teamId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]TeamTransaction, 0)
	for rows.Next() {
		var t TeamTransaction
		var description sql.NullString
		var playerId sql.NullInt64
		if err := rows.Scan(&t.Id, &t.TeamId, &t.Amount, &t.Reason, &description, &t.CreatedAt, &playerId); err != nil {
			return nil, err
		}
		t.Description = description.String
		if playerId.Valid {
			t.PlayerId = &playerId.Int64
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

// GetTeamWageBill returns what a team actually pays in player wages per round: the combined
// stored (base) wage of every player on its roster, net of any slice a lender covers, scaled by
// its division's wage factor (see DivisionWageFactor).
func GetTeamWageBill(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	var total int64
	if err := dbCtx.Db.QueryRow(GetTeamWageBillStmtSql, teamId).Scan(&total); err != nil {
		return 0, err
	}
	factor, err := GetTeamWageFactor(dbCtx, teamId)
	if err != nil {
		return 0, err
	}
	return ApplyWageFactor(total, factor), nil
}

func GetPlayerFullName(dbCtx *DatabaseContext, playerId int64) (string, error) {
	var firstName string
	var lastName string
	row, err := dbCtx.Db.Query(GetPlayerFullNameStmtSql, playerId)
	if err != nil {
		return "", err
	}
	defer row.Close()
	if row.Next() {
		row.Scan(&firstName, &lastName)
	}
	return firstName + " " + lastName, nil
}

func GetPlayersFromTeamId(dbCtx *DatabaseContext, teamId int64) (*[]*Player, error) {
	rows, err := dbCtx.Db.Query(GetPlayersFromTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	players := make([]*Player, 0)
	var playerId int64
	var firstName string
	var lastName string
	var birthDay int64
	var goalKeeper int
	var defense int
	var lateral int
	var pass int
	var midfield int
	var dribbling int
	var shot int
	var header bool
	var quick bool
	var creative bool
	var technical bool
	var wage int
	var age int
	var avatar string
	var contractYearsRemaining int
	var birthCountry sql.NullString
	var shirtNumber int
	var stamina int
	var form int
	for rows.Next() {
		err := rows.Scan(&playerId, &teamId, &firstName, &lastName, &birthDay, &goalKeeper, &defense, &lateral, &pass,
			&midfield, &dribbling, &shot, &header, &quick, &creative, &technical, &wage, &age, &avatar, &contractYearsRemaining, &birthCountry, &shirtNumber, &stamina, &form)
		if err != nil {
			log.Errorf("error scanning player row: %s", err)
		} else {
			players = append(players, &Player{
				Id:                     playerId,
				TeamId:                 teamId,
				FirstName:              firstName,
				LastName:               lastName,
				BirthDay:               time.UnixMilli(birthDay),
				GoalKeeper:             goalKeeper,
				Defense:                defense,
				Lateral:                lateral,
				Pass:                   pass,
				Midfield:               midfield,
				Dribbling:              dribbling,
				Shot:                   shot,
				Header:                 header,
				Quick:                  quick,
				Creative:               creative,
				Technical:              technical,
				Wage:                   wage,
				Age:                    age,
				Avatar:                 GetAvatarFromString(avatar),
				ContractYearsRemaining: contractYearsRemaining,
				BirthCountry:           birthCountry.String,
				ShirtNumber:            shirtNumber,
				Stamina:                stamina,
				Form:                   form,
			})
		}
	}
	return &players, nil
}

const GetPlayersByIdsStmtSql = "SELECT player_id, COALESCE(team_id, 0), first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"COALESCE(shirt_number, 0), stamina, form " +
	"FROM players WHERE player_id = ANY($1)"

// GetPlayersByIds batch-fetches a specific set of players by id, regardless of which team any of
// them currently belongs to - unlike GetPlayersFromTeamId, which filters by team_id and so would
// come back empty for a national team's called-up players (they stay on their own real clubs'
// rosters rather than ever moving onto the national team's own team_id, see
// SetNationalTeamLineup). Used wherever a national team's lineup needs resolving to full player
// records - see GetPlayersForMatchTeam.
func GetPlayersByIds(dbCtx *DatabaseContext, playerIds []int64) (*[]*Player, error) {
	players := make([]*Player, 0, len(playerIds))
	if len(playerIds) == 0 {
		return &players, nil
	}

	rows, err := dbCtx.Db.Query(GetPlayersByIdsStmtSql, playerIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playerId, teamId int64
	var firstName, lastName string
	var birthDay int64
	var goalKeeper, defense, lateral, pass, midfield, dribbling, shot int
	var header, quick, creative, technical bool
	var wage, age int
	var avatar string
	var contractYearsRemaining int
	var birthCountry sql.NullString
	var shirtNumber, stamina, form int
	for rows.Next() {
		if err := rows.Scan(&playerId, &teamId, &firstName, &lastName, &birthDay, &goalKeeper, &defense, &lateral, &pass,
			&midfield, &dribbling, &shot, &header, &quick, &creative, &technical, &wage, &age, &avatar,
			&contractYearsRemaining, &birthCountry, &shirtNumber, &stamina, &form); err != nil {
			return nil, err
		}
		players = append(players, &Player{
			Id:                     playerId,
			TeamId:                 teamId,
			FirstName:              firstName,
			LastName:               lastName,
			BirthDay:               time.UnixMilli(birthDay),
			GoalKeeper:             goalKeeper,
			Defense:                defense,
			Lateral:                lateral,
			Pass:                   pass,
			Midfield:               midfield,
			Dribbling:              dribbling,
			Shot:                   shot,
			Header:                 header,
			Quick:                  quick,
			Creative:               creative,
			Technical:              technical,
			Wage:                   wage,
			Age:                    age,
			Avatar:                 GetAvatarFromString(avatar),
			ContractYearsRemaining: contractYearsRemaining,
			BirthCountry:           birthCountry.String,
			ShirtNumber:            shirtNumber,
			Stamina:                stamina,
			Form:                   form,
		})
	}
	return &players, rows.Err()
}

// GetAllTeamIdsStmtSql excludes a national team (see init71_national_teams.sql) - every current
// caller (RunAITeamManagement's and RunAiTeamDailyActivity's club-AI loops, RespondToAllPendingOffers,
// PayAllTeamsWagesNow, TopUpTrainerMarket, and ProgressPlayers' season-end rollover) means "every
// real, manageable club team," not a country's virtual representative squad - a national team has
// no owner, no trainer, no wage bill, and (by design) no real roster of its own to top up from the
// academy (its "squad" is picked fresh each call-up from GetPlayersByBirthCountry). Before this
// exclusion, ProgressPlayers' per-team academy top-up saw a national team's always-empty player
// count and generated it a full academy squad every season-end rollover - fake players that, having
// a real team_id and birth_country matching the national team's own country, were then themselves
// eligible to be called up alongside (or instead of) genuinely eligible club players (see
// GetPlayersByBirthCountryStmtSql's "team_id IS NOT NULL" filter). The other callers merely logged
// a harmless "no rows" error from GetTeamById (a national team has league_id = -1, so GetTeamById's
// own league join finds nothing) and skipped it - no corruption there, just log noise.
const GetAllTeamIdsStmtSql = "SELECT team_id FROM teams WHERE is_national_team = false AND dissolved_season IS NULL"

// Excludes academy prospects - topUpAcademy uses this to decide how far below SquadSize a
// team's senior squad has fallen, and a prospect sitting unpromoted in the cantera shouldn't
// count toward that floor (see GetPlayersFromTeamStmtSql for the same reasoning).
const CountTeamPlayersStmtSql = "SELECT count(*) FROM players WHERE team_id = $1 AND in_academy = false"
const UpdatePlayerProgressionStmtSql = "UPDATE players SET age = $1, goal_keeper = $2, defense = $3, \"lateral\" = $4, " +
	"pass = $5, midfield = $6, dribbling = $7, shot = $8, wage = $9 WHERE player_id = $10"
const DeletePlayerStmtSql = "DELETE FROM players WHERE player_id = $1"

const AcademySkillMin, AcademySkillMax = 15, 40
const AcademyAgeMin, AcademyAgeMax = 17, 19

// rollSkillCap picks a player's lifetime ceiling for one skill, given the starting value just
// rolled for it (see doCreatePlayers/InjectFreeAgents) - most players (70%) get a modest 5-15
// point gap above their starting value, a smaller share (25%) a real room-to-grow gap of 15-30,
// and a rare few (5%) a wonderkid-sized 30-45 gap, mirroring how few real players ever develop
// into world-class talents. Never below the starting value, never above the universal 99 ceiling
// (see driftSkill) - in practice a player whose starting roll is already close to 99 simply ends
// up with little or no headroom left, same as in real football.
func rollSkillCap(startingSkill int) int {
	var gap int
	switch roll := rand.Float64(); {
	case roll < 0.70:
		gap = 5 + rand.Intn(11) // 5..15
	case roll < 0.95:
		gap = 15 + rand.Intn(16) // 15..30
	default:
		gap = 30 + rand.Intn(16) // 30..45
	}
	max := startingSkill + gap
	if max > 99 {
		max = 99
	}
	return max
}

func GetAllTeamIds(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetAllTeamIdsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teamIds := make([]int64, 0)
	for rows.Next() {
		var teamId int64
		if err := rows.Scan(&teamId); err != nil {
			return nil, err
		}
		teamIds = append(teamIds, teamId)
	}
	return teamIds, nil
}

// GetTeamIdsForCountryStmtSql excludes national teams for the same reason GetAllTeamIdsStmtSql
// does (see its own doc comment) - a national team has no real roster of its own to age/top up.
const GetTeamIdsForCountryStmtSql = "SELECT team_id FROM teams WHERE country = $1 AND is_national_team = false AND dissolved_season IS NULL"

// GetTeamIdsForCountry lists every real, manageable club team belonging to country - used to
// scope season-end rollover work (player aging/academy top-up, expiring loans) to just the one
// country that's actually rolling over, since current_season now advances per country rather
// than globally (see engine.AdvanceSeasonsIfComplete).
func GetTeamIdsForCountry(dbCtx *DatabaseContext, country string) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetTeamIdsForCountryStmtSql, country)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teamIds := make([]int64, 0)
	for rows.Next() {
		var teamId int64
		if err := rows.Scan(&teamId); err != nil {
			return nil, err
		}
		teamIds = append(teamIds, teamId)
	}
	return teamIds, nil
}

func CountTeamPlayers(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(CountTeamPlayersStmtSql, teamId).Scan(&count)
	return count, err
}

// UpdatePlayerProgression persists a player's new age and skills after a season's aging/
// attribute drift (see engine/progression.go), recomputing wage from the new skills.
func UpdatePlayerProgression(dbCtx *DatabaseContext, playerId int64, age, goalKeeper, defense, lateral, pass, midfield, dribbling, shot int) error {
	wage := PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
	_, err := dbCtx.Db.Exec(UpdatePlayerProgressionStmtSql, age, goalKeeper, defense, lateral, pass, midfield, dribbling, shot, wage, playerId)
	return err
}

// PlayerSkillCaps is a player's individually-rolled lifetime ceiling for each of the seven core
// skills (see rollSkillCap) - deliberately kept out of the Player struct and never serialized to
// the frontend: a manager sees a player's current skills but never their hidden potential, same
// as scouting in most real football management games. The only reader is engine/progression.go's
// match-by-match skill drift (see GetPlayerSkillCaps), which clamps growth at these values
// instead of the universal [1, 99] range every player used to share.
type PlayerSkillCaps struct {
	GoalKeeperMax int
	DefenseMax    int
	LateralMax    int
	PassMax       int
	MidfieldMax   int
	DribblingMax  int
	ShotMax       int
}

const GetPlayerSkillCapsStmtSql = "SELECT player_id, goal_keeper_max, defense_max, lateral_max, pass_max, " +
	"midfield_max, dribbling_max, shot_max FROM players WHERE player_id = ANY($1)"

// GetPlayerSkillCaps batch-fetches the skill ceilings for a set of players, keyed by player_id -
// one query per match's worth of participants rather than one per player, since
// updateTeamPlayersAfterMatch needs every featured player's caps at once. A player id with no
// matching row simply has no entry in the returned map; callers fall back to the universal 99 cap
// (see driftSkill) rather than failing, so a player created before this feature existed (and
// still mid-backfill) never gets stuck unable to grow.
func GetPlayerSkillCaps(dbCtx *DatabaseContext, playerIds []int64) (map[int64]PlayerSkillCaps, error) {
	caps := make(map[int64]PlayerSkillCaps, len(playerIds))
	if len(playerIds) == 0 {
		return caps, nil
	}
	rows, err := dbCtx.Db.Query(GetPlayerSkillCapsStmtSql, playerIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var playerId int64
		var c PlayerSkillCaps
		if err := rows.Scan(&playerId, &c.GoalKeeperMax, &c.DefenseMax, &c.LateralMax, &c.PassMax,
			&c.MidfieldMax, &c.DribblingMax, &c.ShotMax); err != nil {
			return nil, err
		}
		caps[playerId] = c
	}
	return caps, rows.Err()
}

const UpdatePlayerStaminaAndFormStmtSql = "UPDATE players SET stamina = $1, form = $2 WHERE player_id = $3"

// UpdatePlayerStaminaAndForm persists a player's post-match Stamina/Form drift (see
// engine.updateTeamPlayersAfterMatch) - kept separate from UpdatePlayerProgression rather than
// folded into it, since these two attributes drift on every match a player features (or doesn't
// feature) in, independently of whether any of the seven core skills happened to drift that match.
func UpdatePlayerStaminaAndForm(dbCtx *DatabaseContext, playerId int64, stamina, form int) error {
	_, err := dbCtx.Db.Exec(UpdatePlayerStaminaAndFormStmtSql, stamina, form, playerId)
	return err
}

const GetPlayersDueBirthdayStmtSql = "SELECT player_id, birth_day, age FROM players WHERE birth_day <= $1"

// GetPlayersDueBirthday returns every player (rostered, free agent, or academy prospect alike -
// this deliberately doesn't filter by team_id or in_academy) whose birth_day anchor is at or
// before cutoff, i.e. at least engine.BirthdayCycleDays have passed since their last birthday -
// see engine.ProcessBirthdays, the only caller. Only Id, BirthDay and Age are populated; every
// other Player field is left at its zero value since the caller has no use for them.
func GetPlayersDueBirthday(dbCtx *DatabaseContext, cutoff time.Time) ([]*Player, error) {
	rows, err := dbCtx.Db.Query(GetPlayersDueBirthdayStmtSql, cutoff.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]*Player, 0)
	for rows.Next() {
		var playerId int64
		var birthDay int64
		var age int
		if err := rows.Scan(&playerId, &birthDay, &age); err != nil {
			return nil, err
		}
		players = append(players, &Player{Id: playerId, BirthDay: time.UnixMilli(birthDay), Age: age})
	}
	return players, nil
}

const UpdatePlayerBirthdayStmtSql = "UPDATE players SET age = $1, birth_day = $2 WHERE player_id = $3"

// UpdatePlayerBirthday persists a player's new age and birth_day anchor after
// engine.ProcessBirthdays fires one or more of their birthdays.
func UpdatePlayerBirthday(dbCtx *DatabaseContext, playerId int64, age int, birthDay time.Time) error {
	_, err := dbCtx.Db.Exec(UpdatePlayerBirthdayStmtSql, age, birthDay.UnixMilli(), playerId)
	return err
}

// RetirePlayer removes a player from the game. player_unavailability and transfer_listings
// both cascade-delete on the player_id FK, so those clean up automatically. But
// default_lineup_positions.player_id only goes to NULL on delete (see init.sql) - leaving that
// row behind would break GetDefaultLineup, which scans player_id into a non-nullable int64 -
// so that one is deleted explicitly first, the same way the transfer market already does for a
// sold player (see DeletePlayerLineupPositionsStmtSql).
func RetirePlayer(dbCtx *DatabaseContext, playerId int64) error {
	if _, err := dbCtx.Db.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(DeletePlayerStmtSql, playerId)
	return err
}

// CreateAcademyPlayers generates `count` young prospects (lower skills, younger age than a
// starting senior squad - see AcademySkillMin/Max, AcademyAgeMin/Max) directly onto a team's
// roster, to backfill a squad thinned out by retirements. trainerSkill (0 if the team has no
// head coach) raises the top of the prospect skill range - see academySkillMax - so investing
// in coaching pays off in the pipeline too, not just in match-ready squad development.
func CreateAcademyPlayers(dbCtx *DatabaseContext, teamId int64, count int, trainerSkill int) error {
	if count <= 0 {
		return nil
	}
	country, err := GetTeamCountry(dbCtx, teamId)
	if err != nil {
		return err
	}
	insertStmt, err := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, dbCtx.Db, count, AcademySkillMin, academySkillMax(trainerSkill), AcademyAgeMin, AcademyAgeMax, country, false)
	return nil
}

// academySkillMax raises the top of the academy prospect range by up to a quarter of a hired
// trainer's skill (0-99 -> +0-24), leaving the floor untouched - a team with no trainer (skill
// 0) gets exactly the historical AcademySkillMax.
func academySkillMax(trainerSkill int) int {
	return AcademySkillMax + trainerSkill/4
}

const GetTeamCountryStmtSql = "SELECT country FROM teams WHERE team_id = $1"

// GetTeamCountry looks up which country a team belongs to - needed wherever player/team name
// generation (see GetRandomName) has only a team id in scope, not the country itself.
func GetTeamCountry(dbCtx *DatabaseContext, teamId int64) (string, error) {
	var country string
	err := dbCtx.Db.QueryRow(GetTeamCountryStmtSql, teamId).Scan(&country)
	return country, err
}

// InsertMatchesTx inserts weekMatches using the caller's already-open transaction, so the
// insert happens inside the same transaction as whatever concurrency guard (e.g. the
// pg_try_advisory_xact_lock in engine.generateMatchesForLeagueIds) protects the "has this
// league already been generated" check that led to it - without that, two concurrent callers
// could both pass the check before either one commits its inserts.
func InsertMatchesTx(tx *sql.Tx, weekMatches [][]*WeekMatch) error {
	stmt, err := tx.Prepare(InsertMatch)
	if err != nil {
		log.Errorf("error loading insert match statement: %s", err)
		return err
	}
	defer stmt.Close()
	for weekIdx, _ := range weekMatches {
		for _, match := range weekMatches[weekIdx] {
			_, err := stmt.Exec(match.DivisionId, match.Season, match.WeekNumber, match.MatchDate, match.Home, match.Away)
			if err != nil {
				log.Errorf("error inserting match: %s", err)
			}
		}
	}
	return nil
}

// GetDefaultLineup returns a team's standing lineup (used for any match without a
// match-specific override, see GetLineupsForMatch).
func GetDefaultLineup(dbCtx *DatabaseContext, teamId int64) (*Lineup, error) {
	rows, err := dbCtx.Db.Query(GetDefaultLineupStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]LineupPosition, 0)
	for rows.Next() {
		var playerId int64
		var position int
		if err := rows.Scan(&playerId, &position); err != nil {
			return nil, err
		}
		positions = append(positions, LineupPosition{PlayerId: playerId, Position: position})
	}

	formation, err := GetTeamFormation(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	substitutions, err := GetDefaultLineupSubstitutions(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	mentality, err := GetTeamMentality(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	roles, err := GetDefaultLineupRoles(dbCtx, teamId, false)
	if err != nil {
		return nil, err
	}

	return &Lineup{TeamId: teamId, Formation: formation, Positions: positions, Substitutions: substitutions, Mentality: mentality, LineupRoles: roles}, nil
}

// SetDefaultLineup atomically replaces a team's default lineup. Callers must have already
// validated that every position is filled exactly once and every player belongs to the team.
func SetDefaultLineup(dbCtx *DatabaseContext, teamId int64, positions []LineupPosition) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Guards against two concurrent calls for the same team racing the delete-then-insert below
	// into a duplicate-key violation on default_lineup_positions_pkey (e.g. a league match and a
	// cup tie for the same AI-managed team landing in the same simulation pass - see
	// engine.RunAITeamManagement). Deliberately non-blocking (pg_try_, not pg_advisory_xact_lock)
	// so a losing caller never ties up a pooled connection waiting - if another call for this
	// team is already in flight, this one simply no-ops, since both would compute the identical
	// deterministic lineup from the same squad anyway.
	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1)", teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	if _, err := tx.Exec(DeleteDefaultLineupPositionsStmtSql, teamId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertDefaultLineupPositionsStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, p := range positions {
		if _, err := insertStmt.Exec(teamId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetDefaultFriendlyLineup returns a team's standing friendly-only lineup (used as the fallback
// for any friendly without a per-match override, see GetLineupsForFriendly) - kept separate from
// GetDefaultLineup's league/cup default so a manager can field a weaker/experimental XI in
// friendlies without touching their competitive default. Positions come back empty (never an
// error) when the manager hasn't set one up yet; callers fall back to the league/cup default in
// that case. Carries the friendly default's own scheduled substitutions (see
// GetDefaultFriendlyLineupSubstitutions), independent of the league/cup default's.
func GetDefaultFriendlyLineup(dbCtx *DatabaseContext, teamId int64) (*Lineup, error) {
	rows, err := dbCtx.Db.Query(GetDefaultFriendlyLineupStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]LineupPosition, 0)
	for rows.Next() {
		var playerId int64
		var position int
		if err := rows.Scan(&playerId, &position); err != nil {
			return nil, err
		}
		positions = append(positions, LineupPosition{PlayerId: playerId, Position: position})
	}

	formation, err := GetTeamFriendlyFormation(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	substitutions, err := GetDefaultFriendlyLineupSubstitutions(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	mentality, err := GetTeamFriendlyMentality(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	roles, err := GetDefaultLineupRoles(dbCtx, teamId, true)
	if err != nil {
		return nil, err
	}

	return &Lineup{TeamId: teamId, Formation: formation, Positions: positions, Substitutions: substitutions, Mentality: mentality, LineupRoles: roles}, nil
}

// SetDefaultFriendlyLineup atomically replaces a team's standing friendly-only lineup. Callers
// must have already validated that every position is filled exactly once and every player
// belongs to the team (see common.ValidateLineup). Mirrors SetDefaultLineup, including its
// non-blocking per-team advisory lock against a concurrent duplicate-key race.
func SetDefaultFriendlyLineup(dbCtx *DatabaseContext, teamId int64, positions []LineupPosition) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Same non-blocking per-team guard SetDefaultLineup uses, on a distinct lock key (the
	// two-argument form, teamId plus an arbitrary second key) so a concurrent save of this team's
	// league/cup default never contends with - or falsely no-ops - a save of its friendly default.
	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1, $2)", teamId, 1).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	if _, err := tx.Exec(DeleteDefaultFriendlyLineupPositionsStmtSql, teamId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertDefaultFriendlyLineupPositionsStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, p := range positions {
		if _, err := insertStmt.Exec(teamId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func InsertLineup(dbCtx *DatabaseContext, teamId int64, lineup map[int][]int64, asDefault bool) error {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertDefaultLineupPositionsStmtSql)
	if errInsert != nil {
		log.Errorf("error inserting new default lineup %s", errInsert)
		return errInsert
	}
	defer insertStmt.Close()
	for position, players := range lineup {
		for _, player := range players {
			_, err := insertStmt.Exec(teamId, player, position)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func InsertTeamsToClassification(dbCtx *DatabaseContext, leagueId int64, season int, teams []int64) {
	stmt, err := dbCtx.Db.Prepare(InsertTeamToClassificationStmtSql)
	if err != nil {
		log.Errorf("error preparing insert team to classification: %s", err)
		return
	}
	defer stmt.Close()
	for _, t := range teams {
		if _, err := stmt.Exec(leagueId, season, t); err != nil {
			log.Errorf("error inserting team %d to classification: %s", t, err)
		}
	}
}

func GetDivisionTeams(dbCtx *DatabaseContext, leagueId int64) []int64 {
	teams := make([]int64, 0)
	rows, err := dbCtx.Db.Query(GetDivisionTeamsStmtSql, leagueId)
	if err != nil {
		log.Errorf("error getting division teams for league %d: %s", leagueId, err)
		return teams
	}
	defer rows.Close()
	for rows.Next() {
		var t int64
		rows.Scan(&t)
		teams = append(teams, t)
	}
	return teams
}

func CreateDatabaseContext() (*DatabaseContext, error) {
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	databaseDetails := DatabaseDetails{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DbName:   os.Getenv("DB_DATABASE"),
		SslMode:  "disable",
	}

	dbCtx, err := OpenDatabase(databaseDetails)
	return dbCtx, err
}
