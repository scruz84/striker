package common

import (
	"errors"
	"time"
)

// LineupLockMinutes is how close to a match's scheduled kickoff a manager may still edit that
// match's lineup (league match, cup tie, or friendly - see the Set*LineupHandler functions in
// package handlers). League matches and friendlies start being read into simulation up to 15
// minutes early (see GetSimulableMatchesStmtSql/GetSimulableFriendliesStmtSql), so this gives a
// 5-minute margin rather than cutting it exactly at the simulation's own read time.
const LineupLockMinutes = 20

// ErrLineupLocked is returned by a Set*LineupHandler once IsLineupLocked reports true for the
// match/tie/friendly being edited.
var ErrLineupLocked = errors.New("lineup can no longer be changed this close to kickoff")

// IsLineupLocked reports whether matchDate is within LineupLockMinutes of now, or has already
// passed - either way, too late for a lineup change to still matter.
func IsLineupLocked(matchDate time.Time) bool {
	return time.Until(matchDate) < LineupLockMinutes*time.Minute
}
