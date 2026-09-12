package common

import (
	"database/sql"
	"testing"
	"time"
)

func TestCanPromoteFromAcademyAt_AllowsWhenNeverPromoted(t *testing.T) {
	now := time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC) // a Wednesday
	canPromote, next := canPromoteFromAcademyAt(sql.NullTime{}, now)
	if !canPromote {
		t.Fatal("expected a team that has never promoted to be allowed to promote")
	}
	if !next.IsZero() {
		t.Fatalf("expected a zero nextAvailableAt when promotion is allowed, got %v", next)
	}
}

func TestCanPromoteFromAcademyAt_BlocksWithinTheSameWeekAsLastPromotion(t *testing.T) {
	// Monday 2024-01-01 through Sunday 2024-01-07 is one week (StartOfWeek(any of these) ==
	// 2024-01-01 00:00 UTC).
	lastPromotion := sql.NullTime{Valid: true, Time: time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)}
	now := time.Date(2024, 1, 5, 23, 59, 0, 0, time.UTC) // Friday, same week
	canPromote, next := canPromoteFromAcademyAt(lastPromotion, now)
	if canPromote {
		t.Fatal("expected a second promotion in the same week to be blocked")
	}
	want := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // next Monday
	if !next.Equal(want) {
		t.Fatalf("expected next available at %v, got %v", want, next)
	}
}

func TestCanPromoteFromAcademyAt_AllowsOnceTheFollowingWeekStarts(t *testing.T) {
	lastPromotion := sql.NullTime{Valid: true, Time: time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)}
	now := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // the following Monday, 00:00 UTC exactly
	canPromote, next := canPromoteFromAcademyAt(lastPromotion, now)
	if !canPromote {
		t.Fatal("expected promotion to be allowed again once the next week has started")
	}
	if !next.IsZero() {
		t.Fatalf("expected a zero nextAvailableAt when promotion is allowed, got %v", next)
	}
}
