package common

import (
	"errors"
	"strings"
)

const (
	MinTeamNameLength = 2
	MaxTeamNameLength = 50
)

var (
	ErrInvalidTeamName    = errors.New("team name must be between 2 and 50 characters")
	ErrTeamNameAlreadySet = errors.New("team name has already been changed once")
)

const RenameTeamStmtSql = "UPDATE teams SET name = $1, name_changed = true " +
	"WHERE team_id = $2 AND owner_id = $3 AND name_changed = false"

// ValidateTeamName trims name and checks it's a reasonable length, returning the trimmed
// value. Used by RenameTeam so a name of all whitespace, or one that's empty/absurdly long,
// never reaches the database.
func ValidateTeamName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < MinTeamNameLength || len(trimmed) > MaxTeamNameLength {
		return "", ErrInvalidTeamName
	}
	return trimmed, nil
}

// RenameTeam sets teamId's name, but only once ever - the WHERE clause's name_changed = false
// makes this atomic against a race between two rapid rename requests for the same team: whichever
// commits first flips the flag, and the second's UPDATE simply matches zero rows. Returns
// ErrTeamNameAlreadySet if teamId isn't owned by ownerId or has already been renamed.
func RenameTeam(dbCtx *DatabaseContext, teamId int64, ownerId int64, newName string) error {
	trimmed, err := ValidateTeamName(newName)
	if err != nil {
		return err
	}
	result, err := dbCtx.Db.Exec(RenameTeamStmtSql, trimmed, teamId, ownerId)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTeamNameAlreadySet
	}
	return nil
}
