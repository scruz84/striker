package common

// Formation constants (teams.formation). FormationBalanced is the shape SelectBestLineup
// already generates (GK, 4 defenders, 4 midfielders, 2 strikers) - it's the default for every
// new team, so choosing a different formation is an opt-in tactical decision, not a change
// forced on anyone.
const FormationBalanced = 0  // 4-4-2
const FormationDefensive = 1 // 5-3-2
const FormationAttacking = 2 // 3-4-3

func IsValidFormation(formation int) bool {
	return formation == FormationBalanced || formation == FormationDefensive || formation == FormationAttacking
}

// FormationPositionCounts returns how many players a formation needs at each tactical
// position - every formation sums to 11 (including the goalkeeper). Used both to validate a
// submitted lineup (see handlers.validateLineup) and, in principle, to build one automatically,
// though SelectBestLineup currently only ever builds FormationBalanced's shape.
func FormationPositionCounts(formation int) map[int]int {
	switch formation {
	case FormationDefensive:
		return map[int]int{
			GOALKEEPER:             1,
			DEFENSE:                3,
			LATERAL_DEFENSE_LEFT:   1,
			LATERAL_DEFENSE_RIGHT:  1,
			MIDFIELD:               1,
			LATERAL_MIDFIELD_LEFT:  1,
			LATERAL_MIDFIELD_RIGHT: 1,
			STRIKER:                2,
		}
	case FormationAttacking:
		return map[int]int{
			GOALKEEPER:             1,
			DEFENSE:                1,
			LATERAL_DEFENSE_LEFT:   1,
			LATERAL_DEFENSE_RIGHT:  1,
			MIDFIELD:               2,
			LATERAL_MIDFIELD_LEFT:  1,
			LATERAL_MIDFIELD_RIGHT: 1,
			STRIKER:                3,
		}
	default: // FormationBalanced
		return map[int]int{
			GOALKEEPER:             1,
			DEFENSE:                2,
			LATERAL_DEFENSE_LEFT:   1,
			LATERAL_DEFENSE_RIGHT:  1,
			MIDFIELD:               2,
			LATERAL_MIDFIELD_LEFT:  1,
			LATERAL_MIDFIELD_RIGHT: 1,
			STRIKER:                2,
		}
	}
}

const GetTeamFormationStmtSql = "SELECT formation FROM teams WHERE team_id = $1"
const SetTeamFormationStmtSql = "UPDATE teams SET formation = $1 WHERE team_id = $2"

func GetTeamFormation(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var formation int
	err := dbCtx.Db.QueryRow(GetTeamFormationStmtSql, teamId).Scan(&formation)
	return formation, err
}

func SetTeamFormation(dbCtx *DatabaseContext, teamId int64, formation int) error {
	_, err := dbCtx.Db.Exec(SetTeamFormationStmtSql, formation, teamId)
	return err
}
