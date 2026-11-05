package common

import "testing"

// GetAiLevelStats against a real Postgres (STRIKER_TEST_DB_DSN): an insolvent Director and an owned
// club are counted in their buckets, national teams and clubs awaiting a reset are left out. The
// shared database holds other tests' clubs, so counts are compared before and after.
func TestGetAiLevelStats_CountsEachBucket(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	bucket := func(stats []AiLevelStats, level string) AiLevelStats {
		for _, s := range stats {
			if s.Level == level {
				return s
			}
		}
		t.Fatalf("no %s bucket in %+v", level, stats)
		return AiLevelStats{}
	}
	before, err := GetAiLevelStats(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ('al', 1) ON CONFLICT (name) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('al', 1, 'AL1', 1) RETURNING league_id`)
	owner := scan(`INSERT INTO users (name, provider, email) VALUES ('al-owner', 'dev', 'al-owner-' || extract(epoch from now())::text || '@example.test') RETURNING user_id`)
	scan(`INSERT INTO teams (name, country, league_id, budget, ai_level) VALUES ('Broke Director', 'al', $1, -250000, 3) RETURNING team_id`, league)
	scan(`INSERT INTO teams (name, country, league_id, budget, ai_level, owner_id) VALUES ('Owned', 'al', $1, 100, 3, $2) RETURNING team_id`, league, owner)
	scan(`INSERT INTO teams (name, country, league_id, budget, ai_level, reset_pending_since) VALUES ('Resetting', 'al', $1, 100, 0, now()) RETURNING team_id`, league)

	after, err := GetAiLevelStats(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if got := bucket(after, AiLevelBucketDirector).Teams - bucket(before, AiLevelBucketDirector).Teams; got != 1 {
		t.Errorf("%d more Directors, want 1", got)
	}
	if got := bucket(after, AiLevelBucketDirector).Insolvent - bucket(before, AiLevelBucketDirector).Insolvent; got != 1 {
		t.Errorf("%d more insolvent Directors, want 1", got)
	}
	if got := bucket(after, AiLevelBucketHuman).Teams - bucket(before, AiLevelBucketHuman).Teams; got != 1 {
		t.Errorf("%d more human clubs, want the owned one", got)
	}
}
