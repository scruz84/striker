package common

import (
	"errors"
	"testing"
)

func TestIsUniqueViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"pgx unique constraint", errors.New(`ERROR: duplicate key value violates unique constraint "transfer_listings_player_id_key"`), true},
		{"generic unique wording", errors.New("UNIQUE constraint failed"), true},
		{"unrelated error", errors.New("connection refused"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isUniqueViolation(tc.err); got != tc.want {
				t.Errorf("isUniqueViolation(%q) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
