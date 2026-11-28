package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// MatchResult is everything a simulated fixture stores in the one transaction that records it (see
// SaveLeagueMatchResultTx, SaveFriendlyResultTx and SaveCupTieResultTx).
type MatchResult struct {
	HomeGoals int
	AwayGoals int
	Events    []Event
	Instants  []MatchInstant
	// Effects are the post-match effects the replay's completion (or the recovery job) applies
	// later. Stored as pending in the same transaction; nil stores none.
	Effects *PostMatchPayload
	// EffectsDueNow makes the pending effects due immediately instead of when the live-broadcast
	// window has passed - for a default result, which has no replay to wait for.
	EffectsDueNow bool
	// PlayerRatings are the match's final per-player ratings (see ComputePlayerRatings), stored in
	// match_player_ratings in the same transaction. Nil stores none (a default result).
	PlayerRatings []PlayerRating
	// RivalryLevel is the level the match kicked off with (RIVALRY_*, see common/rivalries.go),
	// stored with a league match or cup tie so it always shows the level it was played at.
	RivalryLevel int
	// FirstHalfAddedTime/SecondHalfAddedTime are each half's added minutes (see
	// openspec/specs/stoppage-time), stored on the fixture; 0 for a half never reached and for a
	// default result.
	FirstHalfAddedTime  int
	SecondHalfAddedTime int
}

// PostMatchPayload holds what is needed to apply a match's spoiler-gated post-match effects
// (availability countdown and new injuries/suspensions, skill growth, result notification) after
// the live replay, even if the engine process that simulated the match is long gone. Squad players
// are not stored: they are re-read when the effects are applied.
type PostMatchPayload struct {
	Home            int64 `json:"home"`
	Away            int64 `json:"away"`
	HomeGoals       int   `json:"homeGoals"`
	AwayGoals       int   `json:"awayGoals"`
	WentToPenalties bool  `json:"wentToPenalties"`
	// SkipSkillGrowth is true for a default result: nobody played, so no skill, stamina or form
	// drift (which would otherwise treat the whole squad as benched).
	SkipSkillGrowth         bool                    `json:"skipSkillGrowth"`
	PendingUnavailabilities []PendingUnavailability `json:"pendingUnavailabilities"`
	HomeMinutesByPosition   map[int64]map[int]int   `json:"homeMinutesByPosition"`
	AwayMinutesByPosition   map[int64]map[int]int   `json:"awayMinutesByPosition"`
	// HomeLineup/AwayLineup and Events feed the per-player ratings that drive Form's post-match
	// update (see engine.matchRatingsForProgression) when Ratings is missing - a payload stored
	// by a build from before stored ratings.
	HomeLineup *Lineup `json:"homeLineup"`
	AwayLineup *Lineup `json:"awayLineup"`
	Events     []Event `json:"events"`
	// Ratings is every rated player's stored match rating (see RatingsById): unrated ("s.c.")
	// appearances are left out, so Form treats them as featuring without a performance
	// adjustment.
	Ratings map[int64]float64 `json:"ratings,omitempty"`
	// CupFinal marks a cup tie that was its cup's final - worth more rivalry points (see
	// engine.rivalryPointsForMatch).
	CupFinal bool `json:"cupFinal,omitempty"`
}

// StandingsDelta is the classification increment one side gets from a result.
type StandingsDelta struct {
	Points int
	Wins   int
	Draws  int
	Losses int
}

// StandingsDeltaFor returns the classification increment for a side that scored goalsFor and
// conceded goalsAgainst: 3 points for a win, 1 for a draw.
func StandingsDeltaFor(goalsFor int, goalsAgainst int) StandingsDelta {
	switch {
	case goalsFor > goalsAgainst:
		return StandingsDelta{Points: 3, Wins: 1}
	case goalsFor < goalsAgainst:
		return StandingsDelta{Losses: 1}
	default:
		return StandingsDelta{Points: 1, Draws: 1}
	}
}

const saveLeagueMatchPlayedStmtSql = "UPDATE matches SET played = true, in_simulation = false, home_goals = $1, away_goals = $2, rivalry_level = $4, " +
	"first_half_added_time = $5, second_half_added_time = $6 " +
	"WHERE match_id = $3 AND played = false RETURNING match_date"

const saveFriendlyPlayedStmtSql = "UPDATE friendly_matches SET played = true, in_simulation = false, home_goals = $1, away_goals = $2, " +
	"first_half_added_time = $4, second_half_added_time = $5 " +
	"WHERE match_id = $3 AND played = false RETURNING match_date"

const saveCupTiePlayedStmtSql = "UPDATE cup_ties SET played = true, in_simulation = false, home_goals = $1, away_goals = $2, " +
	"went_to_penalties = $3, winner_team_id = $4, rivalry_level = $6, first_half_added_time = $7, second_half_added_time = $8 " +
	"WHERE tie_id = $5 AND played = false RETURNING match_date"

const insertPendingPostMatchEffectsStmtSql = "INSERT INTO pending_post_match_effects (kind, match_id, due_at, payload) " +
	"VALUES ($1, $2, $3, $4) ON CONFLICT (kind, match_id) DO UPDATE SET due_at = EXCLUDED.due_at, payload = EXCLUDED.payload"

// SaveLeagueMatchResultTx records a simulated league match in one transaction: the result, its
// events and instants (replacing any left by an earlier failed attempt), both sides' standings,
// and the pending post-match effects. It only applies while the match is still unplayed and
// returns ErrAlreadyPlayed otherwise - two simulations of one match can both reach this point, and
// only the first may book anything. A failure anywhere stores nothing. Callers book money, report
// flavor and the live broadcast only after it returns nil.
func SaveLeagueMatchResultTx(dbCtx *DatabaseContext, matchId int64, divisionId int64, homeTeamId int64, awayTeamId int64, result MatchResult) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var matchDate time.Time
	if err := tx.QueryRow(saveLeagueMatchPlayedStmtSql, result.HomeGoals, result.AwayGoals, matchId, result.RivalryLevel,
		result.FirstHalfAddedTime, result.SecondHalfAddedTime).Scan(&matchDate); err != nil {
		return savePlayedError(err)
	}
	if err := replaceMatchEventsTx(tx, SimulationKindLeague, "match_events", "match_id", "match_instants", matchId, result, InsertMatchEventStmtSql, InsertMatchInstantStmtSql); err != nil {
		return err
	}

	// Standings are only ever incremented, so two matches of one league saving at once must not
	// interleave their read-modify-write with the repair that rebuilds a row (see
	// engine.RepairStandingsOnce) - hence one lock per league, taken only now that the match row
	// itself is claimed by this transaction.
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", NamespacedLockKey(LockNamespaceStandings, divisionId)); err != nil {
		return err
	}
	if err := updateClassificationTx(tx, divisionId, homeTeamId, result.HomeGoals, result.AwayGoals); err != nil {
		return err
	}
	if err := updateClassificationTx(tx, divisionId, awayTeamId, result.AwayGoals, result.HomeGoals); err != nil {
		return err
	}

	if err := insertPendingPostMatchEffectsTx(tx, SimulationKindLeague, matchId, matchDate, result); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveFriendlyResultTx is SaveLeagueMatchResultTx's friendly counterpart: the result, its events
// and instants, the World Cup group standings when the friendly is a group fixture, and the
// pending post-match effects, in one transaction guarded by played = false.
func SaveFriendlyResultTx(dbCtx *DatabaseContext, matchId int64, homeTeamId int64, awayTeamId int64, result MatchResult) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var matchDate time.Time
	if err := tx.QueryRow(saveFriendlyPlayedStmtSql, result.HomeGoals, result.AwayGoals, matchId,
		result.FirstHalfAddedTime, result.SecondHalfAddedTime).Scan(&matchDate); err != nil {
		return savePlayedError(err)
	}
	if err := replaceMatchEventsTx(tx, SimulationKindFriendly, "friendly_events", "match_id", "friendly_instants", matchId, result, InsertFriendlyEventStmtSql, InsertFriendlyInstantStmtSql); err != nil {
		return err
	}
	// A no-op for the overwhelming majority of friendlies, which aren't a World Cup group fixture.
	if err := UpdateWorldCupStandingsAfterMatchTx(tx, matchId, homeTeamId, awayTeamId, result.HomeGoals, result.AwayGoals); err != nil {
		return err
	}
	if err := insertPendingPostMatchEffectsTx(tx, SimulationKindFriendly, matchId, matchDate, result); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveCupTieResultTx is SaveLeagueMatchResultTx's cup counterpart: the result (with the shoot-out
// flag and the winner), its events and instants, and the pending post-match effects, in one
// transaction guarded by played = false.
func SaveCupTieResultTx(dbCtx *DatabaseContext, tieId int64, wentToPenalties bool, winnerTeamId int64, result MatchResult) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var matchDate sql.NullTime
	if err := tx.QueryRow(saveCupTiePlayedStmtSql, result.HomeGoals, result.AwayGoals, wentToPenalties, winnerTeamId, tieId, result.RivalryLevel,
		result.FirstHalfAddedTime, result.SecondHalfAddedTime).Scan(&matchDate); err != nil {
		return savePlayedError(err)
	}
	if err := replaceMatchEventsTx(tx, SimulationKindCup, "cup_tie_events", "tie_id", "cup_tie_instants", tieId, result, InsertCupTieEventStmtSql, InsertCupTieInstantStmtSql); err != nil {
		return err
	}
	kickoff := time.Now()
	if matchDate.Valid {
		kickoff = matchDate.Time
	}
	if err := insertPendingPostMatchEffectsTx(tx, SimulationKindCup, tieId, kickoff, result); err != nil {
		return err
	}
	return tx.Commit()
}

func savePlayedError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAlreadyPlayed
	}
	return err
}

// replaceMatchEventsTx deletes the events and instants an earlier failed attempt may have left
// for id, then inserts result's, returning every insert error. All three event tables take the
// same columns in the same order, the other side's players included (the defending keeper on a
// chance - see openspec match-simulation "Defending keeper on every chance").
func replaceMatchEventsTx(tx *sql.Tx, kind string, eventsTable string, idColumn string, instantsTable string, id int64, result MatchResult,
	insertEventSql string, insertInstantSql string) error {
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = $1", eventsTable, idColumn), id); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = $1", instantsTable, idColumn), id); err != nil {
		return err
	}
	if err := replaceMatchPlayerRatingsTx(tx, kind, id, result.PlayerRatings); err != nil {
		return err
	}
	if err := replacePlayerMatchStatsTx(tx, kind, id, result); err != nil {
		return err
	}

	insertEvent, err := tx.Prepare(insertEventSql)
	if err != nil {
		return err
	}
	defer insertEvent.Close()
	for _, e := range result.Events {
		teamPlayersJson, _ := json.Marshal(e.TeamOwnerPlayers)
		teamPlayerNamesJson, _ := json.Marshal(e.TeamOwnerPlayersNames)
		otherPlayersJson, _ := json.Marshal(e.OtherTeamPlayers)
		otherPlayerNamesJson, _ := json.Marshal(e.OtherTeamPlayersNames)
		_, err = insertEvent.Exec(id, e.EventType, e.TeamOwner, e.OtherTeam, e.Minute, teamPlayersJson,
			teamPlayerNamesJson, otherPlayersJson, otherPlayerNamesJson, e.HomeGoals, e.AwayGoals, e.SetPiece, e.Mentality,
			e.AddedMinute, e.AddedTime)
		if err != nil {
			return err
		}
	}

	insertInstant, err := tx.Prepare(insertInstantSql)
	if err != nil {
		return err
	}
	defer insertInstant.Close()
	for _, i := range result.Instants {
		homeQuaJson, _ := json.Marshal(i.HomeQua)
		awayQuaJson, _ := json.Marshal(i.AwayQua)
		if _, err := insertInstant.Exec(id, i.Minute, i.HomeGoals, i.AwayGoals, homeQuaJson, awayQuaJson, i.MidfieldDiff,
			i.HomeAttackRightDiff, i.HomeAttackLeftDiff, i.HomeAttackCenterDiff,
			i.AwayAttackRightDiff, i.AwayAttackLeftDiff, i.AwayAttackCenterDiff, encodePlayerContributions(i.PlayerContributions),
			i.AddedMinute); err != nil {
			return err
		}
	}
	return nil
}

func updateClassificationTx(tx *sql.Tx, divisionId int64, teamId int64, goalsFor int, goalsAgainst int) error {
	d := StandingsDeltaFor(goalsFor, goalsAgainst)
	_, err := tx.Exec(UpdateTeamClassificationStmtSql, 1, d.Points, d.Wins, d.Losses, d.Draws, goalsFor, goalsAgainst, divisionId, teamId)
	return err
}

func insertPendingPostMatchEffectsTx(tx *sql.Tx, kind string, id int64, kickoff time.Time, result MatchResult) error {
	if result.Effects == nil {
		return nil
	}
	payload, err := json.Marshal(result.Effects)
	if err != nil {
		return err
	}
	dueAt := kickoff.Add(LiveBroadcastWindow)
	if result.EffectsDueNow {
		dueAt = time.Now()
	}
	_, err = tx.Exec(insertPendingPostMatchEffectsStmtSql, kind, id, dueAt, payload)
	return err
}

const claimPostMatchEffectsStmtSql = "DELETE FROM pending_post_match_effects WHERE kind = $1 AND match_id = $2 RETURNING payload"

// ClaimPostMatchEffects takes a match's pending post-match effects, or returns nil when there are
// none (already applied by the replay's completion or by the recovery job, or a match saved before
// they were stored). Whoever deletes the row applies it, so the effects are applied at most once.
func ClaimPostMatchEffects(dbCtx *DatabaseContext, kind string, matchId int64) (*PostMatchPayload, error) {
	var raw []byte
	err := dbCtx.Db.QueryRow(claimPostMatchEffectsStmtSql, kind, matchId).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var payload PostMatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DuePostMatchEffects is one pending record claimed by ClaimDuePostMatchEffects.
type DuePostMatchEffects struct {
	Kind    string
	MatchId int64
	Payload PostMatchPayload
}

// MaxDuePostMatchEffectsPerPass bounds how many pending records one recovery pass takes.
const MaxDuePostMatchEffectsPerPass = 200

const claimDuePostMatchEffectsStmtSql = "DELETE FROM pending_post_match_effects WHERE (kind, match_id) IN (" +
	"SELECT kind, match_id FROM pending_post_match_effects WHERE due_at <= now() ORDER BY due_at LIMIT $1 FOR UPDATE SKIP LOCKED) " +
	"RETURNING kind, match_id, payload"

// ClaimDuePostMatchEffects takes up to limit pending records whose due_at has passed. FOR UPDATE
// SKIP LOCKED lets several engine replicas drain concurrently without taking the same record, and
// the delete makes the claim final: a record that cannot be decoded is dropped with an error
// rather than retried forever.
func ClaimDuePostMatchEffects(dbCtx *DatabaseContext, limit int) ([]DuePostMatchEffects, error) {
	rows, err := dbCtx.Db.Query(claimDuePostMatchEffectsStmtSql, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	due := make([]DuePostMatchEffects, 0)
	var decodeErr error
	for rows.Next() {
		var d DuePostMatchEffects
		var raw []byte
		if err := rows.Scan(&d.Kind, &d.MatchId, &raw); err != nil {
			return due, err
		}
		if err := json.Unmarshal(raw, &d.Payload); err != nil {
			decodeErr = fmt.Errorf("decoding pending post-match effects of %s %d: %w", d.Kind, d.MatchId, err)
			continue
		}
		due = append(due, d)
	}
	if err := rows.Err(); err != nil {
		return due, err
	}
	return due, decodeErr
}

// encodePlayerContributions is the player_contributions column value of one instant: its JSON, or
// NULL for an instant without contributions (a default result, or a test building instants by
// hand).
func encodePlayerContributions(contributions *InstantContributions) any {
	if contributions == nil {
		return nil
	}
	raw, err := json.Marshal(contributions)
	if err != nil {
		return nil
	}
	return string(raw)
}

// decodePlayerContributions reads an instant's player_contributions column back - nil for NULL
// (an instant saved before init89_match_player_ratings.sql) or unreadable JSON.
func decodePlayerContributions(raw sql.NullString) *InstantContributions {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	var contributions InstantContributions
	if err := json.Unmarshal([]byte(raw.String), &contributions); err != nil {
		return nil
	}
	return &contributions
}

const DeleteMatchPlayerRatingsStmtSql = "DELETE FROM match_player_ratings WHERE kind = $1 AND match_id = $2"

const InsertMatchPlayerRatingStmtSql = "INSERT INTO match_player_ratings (kind, match_id, player_id, team_id, rated, rating, " +
	"minutes, position, came_on, went_off, own_value, rival_value, zone_delta, goals_delta, cards_delta, injury_delta, result_delta) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)"

const GetMatchPlayerRatingsStmtSql = "SELECT player_id, team_id, rated, rating, minutes, position, came_on, went_off, own_value, " +
	"rival_value, zone_delta, goals_delta, cards_delta, injury_delta, result_delta " +
	"FROM match_player_ratings WHERE kind = $1 AND match_id = $2 ORDER BY player_id"

// replaceMatchPlayerRatingsTx replaces a fixture's stored ratings with ratings, inside the result
// transaction (see replaceMatchEventsTx) - so a retried save never leaves stale rows behind.
func replaceMatchPlayerRatingsTx(tx *sql.Tx, kind string, id int64, ratings []PlayerRating) error {
	if _, err := tx.Exec(DeleteMatchPlayerRatingsStmtSql, kind, id); err != nil {
		return err
	}
	if len(ratings) == 0 {
		return nil
	}
	insert, err := tx.Prepare(InsertMatchPlayerRatingStmtSql)
	if err != nil {
		return err
	}
	defer insert.Close()
	for _, r := range ratings {
		if _, err := insert.Exec(kind, id, r.PlayerId, r.TeamId, r.Rated, r.Rating, r.Minutes, r.Position, r.CameOn, r.WentOff,
			r.Own, r.Rival, r.Zone, r.Goals, r.Cards, r.Injury, r.Result); err != nil {
			return err
		}
	}
	return nil
}

// GetMatchPlayerRatings reads a fixture's stored ratings (kind is a SimulationKind*), sorted by
// player id - nil when none are stored (a match simulated before init89_match_player_ratings.sql,
// or a default result).
func GetMatchPlayerRatings(dbCtx *DatabaseContext, kind string, id int64) ([]PlayerRating, error) {
	rows, err := dbCtx.Db.Query(GetMatchPlayerRatingsStmtSql, kind, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ratings []PlayerRating
	for rows.Next() {
		var r PlayerRating
		var rating, own, rival sql.NullFloat64
		var cameOn, wentOff sql.NullInt64
		var zone, goals, cards, injury, result float64
		if err := rows.Scan(&r.PlayerId, &r.TeamId, &r.Rated, &rating, &r.Minutes, &r.Position, &cameOn, &wentOff,
			&own, &rival, &zone, &goals, &cards, &injury, &result); err != nil {
			return nil, err
		}
		// real columns read back as float32-precision doubles; snap them to the one-decimal values
		// that were written.
		r.Rating = roundTenth(rating.Float64)
		r.Zone, r.Goals, r.Cards, r.Injury, r.Result = roundTenth(zone), roundTenth(goals), roundTenth(cards), roundTenth(injury), roundTenth(result)
		if cameOn.Valid {
			v := int(cameOn.Int64)
			r.CameOn = &v
		}
		if wentOff.Valid {
			v := int(wentOff.Int64)
			r.WentOff = &v
		}
		if own.Valid {
			v := roundTenth(own.Float64)
			r.Own = &v
		}
		if rival.Valid {
			v := roundTenth(rival.Float64)
			r.Rival = &v
		}
		ratings = append(ratings, r)
	}
	return ratings, rows.Err()
}

// LoadPlayerRatings is every read path's source of a played fixture's ratings: the stored rows
// when there are any, otherwise ComputePlayerRatings over the fixture's starting lineups, instants
// and events at its final minute - the fallback for a match simulated before stored ratings
// (whose instants carry no contributions, so its zone component is 0). homeLineup/awayLineup must
// be the starting XIs, before any substitution is replayed onto them.
func LoadPlayerRatings(dbCtx *DatabaseContext, kind string, id int64, homeTeamId int64, awayTeamId int64,
	homeLineup *Lineup, awayLineup *Lineup, instants *[]MatchInstant, events *[]Event) []PlayerRating {
	stored, err := GetMatchPlayerRatings(dbCtx, kind, id)
	if err != nil {
		log.Errorf("error reading the stored ratings of %s %d: %v", kind, id, err)
	}
	if len(stored) > 0 {
		return stored
	}
	var instantList []MatchInstant
	if instants != nil {
		instantList = *instants
	}
	var eventList []Event
	if events != nil {
		eventList = *events
	}
	if eventList == nil && instantList == nil {
		return nil
	}
	return ComputePlayerRatings(homeTeamId, awayTeamId, homeLineup, awayLineup, instantList, eventList,
		MatchFinalMinute(instantList, eventList))
}

// addRatedPlayers makes sure every rated player appears in his side's players map, fetching the
// ones a match's squad lookup left out by id - a national team only loads its lineup's players, so
// whoever was substituted off (or came on) would otherwise have a rating but no name.
func addRatedPlayers(dbCtx *DatabaseContext, ratings []PlayerRating, homeTeamId int64, homePlayers map[int64]Player, awayPlayers map[int64]Player) {
	missing := make([]int64, 0)
	for _, r := range ratings {
		_, inHome := homePlayers[r.PlayerId]
		_, inAway := awayPlayers[r.PlayerId]
		if !inHome && !inAway {
			missing = append(missing, r.PlayerId)
		}
	}
	if len(missing) == 0 {
		return
	}
	extra, err := GetPlayersByIds(dbCtx, missing)
	if err != nil {
		log.Errorf("error reading rated players %v: %v", missing, err)
		return
	}
	teamOf := make(map[int64]int64, len(ratings))
	for _, r := range ratings {
		teamOf[r.PlayerId] = r.TeamId
	}
	for _, p := range *extra {
		if teamOf[p.Id] == homeTeamId {
			homePlayers[p.Id] = *p
		} else {
			awayPlayers[p.Id] = *p
		}
	}
}
