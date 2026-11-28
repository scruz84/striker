package common

import (
	"database/sql"
	"errors"
	"time"
)

// RetiredPlayer is what is left of a player once he retires (openspec/specs/retired-players): the
// snapshot recordRetiredPlayerTx writes just before his players row is deleted. His awards,
// transfers, club history and match stats are kept under the same player id.
type RetiredPlayer struct {
	Id            int64        `json:"id"`
	FirstName     string       `json:"firstName"`
	LastName      string       `json:"lastName"`
	BirthCountry  string       `json:"birthCountry,omitempty"`
	Avatar        PlayerAvatar `json:"avatar"`
	BestPosition  int          `json:"bestPosition"`
	Age           int          `json:"age"`
	LastTeamId    *int64       `json:"lastTeamId,omitempty"`
	LastTeamName  string       `json:"lastTeamName,omitempty"`
	RetiredSeason *int         `json:"retiredSeason,omitempty"`
	RetiredAt     time.Time    `json:"retiredAt"`
}

// insertRetiredPlayerStmtSql snapshots the player from his own row: his club is the one he belongs
// to, or for a free agent the one he last played for (players.last_team_id), and the season is
// that club's country's current season. $2 is his best position (BestCorePosition, computed in Go
// from his skills). A second retirement of the same id is a no-op.
const insertRetiredPlayerStmtSql = "INSERT INTO retired_players " +
	"(player_id, first_name, last_name, birth_country, avatar, best_position, age, last_team_id, last_team_name, retired_season) " +
	"SELECT p.player_id, COALESCE(p.first_name, ''), COALESCE(p.last_name, ''), p.birth_country, p.avatar, $2, p.age, " +
	"t.team_id, t.name, c.current_season " +
	"FROM players p " +
	"LEFT JOIN teams t ON t.team_id = COALESCE(p.team_id, p.last_team_id) " +
	"LEFT JOIN countries c ON c.name = t.country " +
	"WHERE p.player_id = $1 " +
	"ON CONFLICT (player_id) DO NOTHING"

// recordRetiredPlayerTx writes the retirement snapshot of player inside tx, before his row is
// deleted in the same transaction.
func recordRetiredPlayerTx(tx *sql.Tx, player Player) error {
	_, err := tx.Exec(insertRetiredPlayerStmtSql, player.Id, BestCorePosition(player))
	return err
}

const getRetiredPlayerStmtSql = "SELECT player_id, first_name, last_name, COALESCE(birth_country, ''), COALESCE(avatar, ''), " +
	"best_position, age, last_team_id, COALESCE(last_team_name, ''), retired_season, retired_at " +
	"FROM retired_players WHERE player_id = $1"

// GetRetiredPlayer returns the retirement snapshot of playerId, or nil when no player with that id
// ever retired.
func GetRetiredPlayer(dbCtx *DatabaseContext, playerId int64) (*RetiredPlayer, error) {
	var r RetiredPlayer
	var avatar string
	var lastTeamId sql.NullInt64
	var retiredSeason sql.NullInt64
	err := dbCtx.Db.QueryRow(getRetiredPlayerStmtSql, playerId).Scan(&r.Id, &r.FirstName, &r.LastName, &r.BirthCountry, &avatar,
		&r.BestPosition, &r.Age, &lastTeamId, &r.LastTeamName, &retiredSeason, &r.RetiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if avatar != "" {
		r.Avatar = GetAvatarFromString(avatar)
	}
	if lastTeamId.Valid {
		id := lastTeamId.Int64
		r.LastTeamId = &id
	}
	if retiredSeason.Valid {
		season := int(retiredSeason.Int64)
		r.RetiredSeason = &season
	}
	return &r, nil
}

// RetiredPlayerDetails is the profile of a retired player (see handlers.PlayerDetailsHandler): the
// snapshot plus the record kept under his id. Retired is always set, which is how a client tells
// it from an active player's PlayerDetails.
type RetiredPlayerDetails struct {
	Retired RetiredPlayer `json:"retired"`
	PlayerCareerStats
	SeasonAwards PlayerSeasonAwards       `json:"seasonAwards"`
	SeasonStats  []PlayerSeasonStats      `json:"seasonStats"`
	BestXi       PlayerBestXiCounts       `json:"bestXi"`
	ClubHistory  []PlayerClubHistoryEntry `json:"clubHistory"`
	Transfers    []PlayerTransfer         `json:"transfers"`
}
