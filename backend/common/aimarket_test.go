package common

import (
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"
)

// positionSkillSQL is generated from SlotWeights: each expression names exactly the slot's
// non-zero weights, and its integer arithmetic (the same in Postgres as in Go for these
// non-negative values) gives PositionValue for a fixed player.
func TestPositionSkillSQL_MatchesPositionValue(t *testing.T) {
	p := Player{GoalKeeper: 33, Defense: 71, Lateral: 58, Pass: 44, Midfield: 67, Dribbling: 52, Shot: 79}
	values := skillValues(p)
	for _, position := range CorePositions {
		expr, ok := positionSkillSQL[position]
		if !ok {
			t.Fatalf("no market expression for core position %d", position)
		}
		for i, w := range SlotWeights[position] {
			term := skillColumns[i] + " * " + strconv.Itoa(w)
			if named := strings.Contains(expr, term); named != (w != 0) {
				t.Errorf("position %d: expression %q naming %q is %v, want %v", position, expr, term, named, w != 0)
			}
		}
		substituted := expr
		for i, column := range skillColumns {
			substituted = strings.ReplaceAll(substituted, column, strconv.Itoa(values[i]))
		}
		tv, err := types.Eval(token.NewFileSet(), nil, token.NoPos, substituted)
		if err != nil {
			t.Fatalf("position %d: evaluating %q: %s", position, substituted, err)
		}
		if got := tv.Value.String(); got != strconv.Itoa(PositionValue(position, p)) {
			t.Errorf("position %d: SQL gives %s, PositionValue %d", position, got, PositionValue(position, p))
		}
	}
}
