package common

import (
	"database/sql"
	"errors"
	"fmt"
)

// Lineup roles (see openspec/specs/match-simulation "Lineup roles"): every lineup may name a
// captain, a penalty taker and a free-kick taker. Each is optional - nil means "automatic" - and is
// only ever a hint: a named player who is not on the pitch when it matters simply falls back to the
// automatic rule, so no match can break because of a role.

// CaptainCardCancelProbability is the chance a card drawn for a team-mate of an on-pitch captain is
// cancelled (see openspec/specs/match-simulation "Captain's calming effect").
const CaptainCardCancelProbability = 0.10

// LineupRoles are a lineup's three optional roles. Embedded in Lineup, so its JSON fields sit
// next to the formation and positions.
type LineupRoles struct {
	CaptainId       *int64 `json:"captainId,omitempty"`
	PenaltyTakerId  *int64 `json:"penaltyTakerId,omitempty"`
	FreeKickTakerId *int64 `json:"freeKickTakerId,omitempty"`
}

// RedactTakers drops the set-piece takers, keeping the captain - for every lineup shown to someone
// other than its owner (the live and finished match views): the armband is public from kickoff
// on, the takers never are.
func (r *LineupRoles) RedactTakers() {
	r.PenaltyTakerId = nil
	r.FreeKickTakerId = nil
}

// RedactLineupTakers is RedactTakers on a lineup that may be nil.
func RedactLineupTakers(lineup *Lineup) {
	if lineup != nil {
		lineup.RedactTakers()
	}
}

// IsCaptain reports whether playerId is the named captain.
func (r LineupRoles) IsCaptain(playerId int64) bool {
	return r.CaptainId != nil && *r.CaptainId == playerId
}

// ValidateRoles checks a lineup's roles against the XI saved with them (positions) and the squad:
// each named player must be one of the starters or a squad player on the bench, and neither taker
// may be a goalkeeper - neither the lineup's goalkeeper nor a squad player whose best position is
// goalkeeper (BestCorePosition), mirroring the automatic rule, which only ever picks outfielders.
func ValidateRoles(positions []LineupPosition, roles LineupRoles, teamPlayers *[]*Player) error {
	startingPosition := make(map[int64]int, len(positions))
	for _, pos := range positions {
		startingPosition[pos.PlayerId] = pos.Position
	}
	roster := make(map[int64]*Player, len(*teamPlayers))
	for _, p := range *teamPlayers {
		roster[p.Id] = p
	}
	check := func(name string, id *int64, taker bool) error {
		if id == nil {
			return nil
		}
		player, ok := roster[*id]
		if !ok {
			return fmt.Errorf("%s: player %d does not belong to this team", name, *id)
		}
		if !taker {
			return nil
		}
		if position, starts := startingPosition[*id]; starts && position == GOALKEEPER {
			return fmt.Errorf("%s: player %d is the goalkeeper", name, *id)
		}
		if _, starts := startingPosition[*id]; !starts && BestCorePosition(*player) == GOALKEEPER {
			return fmt.Errorf("%s: player %d is a goalkeeper", name, *id)
		}
		return nil
	}
	if err := check("captain", roles.CaptainId, false); err != nil {
		return err
	}
	if err := check("penalty taker", roles.PenaltyTakerId, true); err != nil {
		return err
	}
	return check("free-kick taker", roles.FreeKickTakerId, true)
}

// rolesSelectSql selects a lineup row's three role columns (table alias a). Until the fixture is
// played (played, an SQL condition), each reads back as NULL when its player no longer belongs to
// the lineup's team - a sold or released player's role is "automatic" from then on (openspec
// match-simulation "Taker sold"); a played fixture's kickoff snapshot keeps its roles as they were,
// so the captain a match was played with still wears the armband in its report.
func rolesSelectSql(a string, played string) string {
	col := func(c string) string {
		return "CASE WHEN (" + played + ") OR EXISTS (SELECT 1 FROM players rp WHERE rp.player_id = " + a + "." + c + " AND rp.team_id = " + a + ".team_id) THEN " + a + "." + c + " END"
	}
	return col("captain_id") + ", " + col("penalty_taker_id") + ", " + col("free_kick_taker_id")
}

// roleScan holds the three nullable role columns while a row is scanned.
type roleScan struct {
	captain, penalty, freeKick sql.NullInt64
}

func (s *roleScan) targets() []any {
	return []any{&s.captain, &s.penalty, &s.freeKick}
}

func (s *roleScan) roles() LineupRoles {
	ptr := func(v sql.NullInt64) *int64 {
		if !v.Valid {
			return nil
		}
		id := v.Int64
		return &id
	}
	return LineupRoles{CaptainId: ptr(s.captain), PenaltyTakerId: ptr(s.penalty), FreeKickTakerId: ptr(s.freeKick)}
}

// roleValues are the roles as nullable column values, in rolesSelectSql's order.
func roleValues(r LineupRoles) []any {
	val := func(id *int64) any {
		if id == nil {
			return nil
		}
		return *id
	}
	return []any{val(r.CaptainId), val(r.PenaltyTakerId), val(r.FreeKickTakerId)}
}

// Played-fixture conditions for rolesSelectSql, by lineup table alias.
func matchPlayedSql(a string) string {
	return "EXISTS (SELECT 1 FROM matches pm WHERE pm.match_id = " + a + ".match_id AND pm.played)"
}

func cupTiePlayedSql(a string) string {
	return "EXISTS (SELECT 1 FROM cup_ties pt WHERE pt.tie_id = " + a + ".tie_id AND pt.played)"
}

func friendlyPlayedSql(a string) string {
	return "EXISTS (SELECT 1 FROM friendly_matches pf WHERE pf.match_id = " + a + ".match_id AND pf.played)"
}

var GetDefaultLineupRolesStmtSql = "SELECT " + rolesSelectSql("a", "false") + " FROM default_lineup_roles a WHERE a.team_id = $1 AND a.friendly = $2"

const SetDefaultLineupRolesStmtSql = "INSERT INTO default_lineup_roles (team_id, friendly, captain_id, penalty_taker_id, free_kick_taker_id) " +
	"VALUES ($1, $2, $3, $4, $5) ON CONFLICT (team_id, friendly) DO UPDATE SET captain_id = EXCLUDED.captain_id, " +
	"penalty_taker_id = EXCLUDED.penalty_taker_id, free_kick_taker_id = EXCLUDED.free_kick_taker_id"

// GetDefaultLineupRoles reads the roles of one of a team's standing defaults (friendly picks the
// friendly default over the league/cup one); a team that never saved any has all three automatic.
func GetDefaultLineupRoles(dbCtx *DatabaseContext, teamId int64, friendly bool) (LineupRoles, error) {
	var s roleScan
	err := dbCtx.Db.QueryRow(GetDefaultLineupRolesStmtSql, teamId, friendly).Scan(s.targets()...)
	if errors.Is(err, sql.ErrNoRows) {
		return LineupRoles{}, nil
	}
	if err != nil {
		return LineupRoles{}, err
	}
	return s.roles(), nil
}

// SetDefaultLineupRoles replaces the roles of one of a team's standing defaults - stored apart
// from its positions, the same way its formation and mentality are (see SetTeamMentality).
func SetDefaultLineupRoles(dbCtx *DatabaseContext, teamId int64, friendly bool, roles LineupRoles) error {
	_, err := dbCtx.Db.Exec(SetDefaultLineupRolesStmtSql, append([]any{teamId, friendly}, roleValues(roles)...)...)
	return err
}
