package common

import (
	"testing"
)

// TestRetirePlayer_KeepsHisRecord pins openspec/specs/retired-players: a retiring player leaves a
// snapshot, and his awards, transfers and club history outlive his players row.
func TestRetirePlayer_KeepsHisRecord(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	country := unusedTestCountryCode(t, db, 'r')
	seedTransfersCountry(t, db, country)
	if _, err := db.Exec(`UPDATE countries SET current_season = 7 WHERE name = $1`, country); err != nil {
		t.Fatalf("setting season: %s", err)
	}
	from := seedTransfersTeam(t, db, "Retire From "+country, country)
	club := seedTransfersTeam(t, db, "Retire Club "+country, country)
	playerId := seedTransfersPlayer(t, db, club, country)
	if _, err := db.Exec(`UPDATE players SET age = 38, shot = 90 WHERE player_id = $1`, playerId); err != nil {
		t.Fatalf("aging player: %s", err)
	}

	if err := RecordPlayerClubHistory(db, playerId, club); err != nil {
		t.Fatalf("club history: %s", err)
	}
	if _, err := db.Exec(InsertPlayerTransferStmtSql, playerId, from, club, 1000, PLAYER_TRANSFER_OFFER, STRIKER, 30, 60); err != nil {
		t.Fatalf("transfer: %s", err)
	}
	var leagueId int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, club).Scan(&leagueId); err != nil {
		t.Fatalf("league: %s", err)
	}
	if err := RecordPlayerSeasonAward(dbCtx, leagueId, 6, AwardGoldenBoot, playerId, 21); err != nil {
		t.Fatalf("award: %s", err)
	}

	if err := RetirePlayer(dbCtx, playerId); err != nil {
		t.Fatalf("RetirePlayer: %s", err)
	}

	if _, err := GetPlayerById(dbCtx, playerId); err == nil {
		t.Fatalf("player %d still in players after retiring", playerId)
	}
	retired, err := GetRetiredPlayer(dbCtx, playerId)
	if err != nil || retired == nil {
		t.Fatalf("GetRetiredPlayer = %v, %v", retired, err)
	}
	if retired.FirstName != "Test" || retired.LastName != "Player" || retired.Age != 38 || retired.BirthCountry != country {
		t.Errorf("snapshot identity = %+v", retired)
	}
	if retired.BestPosition != STRIKER {
		t.Errorf("best position = %d, want striker %d", retired.BestPosition, STRIKER)
	}
	if retired.LastTeamId == nil || *retired.LastTeamId != club || retired.LastTeamName != "Retire Club "+country {
		t.Errorf("last club = %v %q, want %d", retired.LastTeamId, retired.LastTeamName, club)
	}
	if retired.RetiredSeason == nil || *retired.RetiredSeason != 7 {
		t.Errorf("retired season = %v, want 7", retired.RetiredSeason)
	}

	awards, err := GetPlayerSeasonAwards(dbCtx, playerId)
	if err != nil || awards.GoldenBoots != 1 {
		t.Errorf("awards after retiring = %+v, %v; want 1 Golden Boot", awards, err)
	}
	transfers, err := GetPlayerTransfers(dbCtx, playerId, 10)
	if err != nil || len(transfers) != 1 {
		t.Errorf("transfers after retiring = %d, %v; want 1", len(transfers), err)
	}
	history, err := GetPlayerClubHistory(dbCtx, playerId)
	if err != nil || len(history) != 1 {
		t.Errorf("club history after retiring = %d, %v; want 1", len(history), err)
	}
}

// TestRetireFreeAgentAsTrainer_KeepsHisRecord: a free agent who retires into coaching is
// snapshotted with the club he last played for.
func TestRetireFreeAgentAsTrainer_KeepsHisRecord(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	country := unusedTestCountryCode(t, db, 'r')
	seedTransfersCountry(t, db, country)
	club := seedTransfersTeam(t, db, "Former Club "+country, country)
	playerId := seedTransfersPlayer(t, db, club, country)
	if _, err := db.Exec(`UPDATE players SET team_id = NULL, last_team_id = $2 WHERE player_id = $1`, playerId, club); err != nil {
		t.Fatalf("releasing player: %s", err)
	}
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		t.Fatalf("GetPlayerById: %s", err)
	}
	if err := RetireFreeAgentAsTrainer(dbCtx, player); err != nil {
		t.Fatalf("RetireFreeAgentAsTrainer: %s", err)
	}
	retired, err := GetRetiredPlayer(dbCtx, playerId)
	if err != nil || retired == nil {
		t.Fatalf("GetRetiredPlayer = %v, %v", retired, err)
	}
	if retired.LastTeamId == nil || *retired.LastTeamId != club || retired.RetiredSeason == nil || *retired.RetiredSeason != 1 {
		t.Errorf("snapshot = %+v, want last club %d in season 1", retired, club)
	}

	none, err := GetRetiredPlayer(dbCtx, -1)
	if err != nil || none != nil {
		t.Errorf("GetRetiredPlayer(-1) = %v, %v; want nil", none, err)
	}
}
