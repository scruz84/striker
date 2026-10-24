package common

import "math/rand"

// matchWeatherConditions are the possible weather codes a match report can show - the frontend
// maps each to its own icon/label (see match.weather.* i18n keys), this side only needs to pick
// one and store it.
var matchWeatherConditions = []string{"sunny", "cloudy", "rain", "windy"}

// RandomWeather picks one of matchWeatherConditions - purely cosmetic flavor for a match report
// (issue: Hattrick-style match report), with no effect on the simulation itself.
func RandomWeather() string {
	return matchWeatherConditions[rand.Intn(len(matchWeatherConditions))]
}

// refereeNamePool is the set of country codes a referee's name can be drawn from - the same pool
// GetRandomName already draws players from (see nameTableSuffixByCountry), since a referee isn't
// meaningfully tied to either team's country.
var refereeNamePool = []string{"es", "en", "fr", "de", "it", "pt"}

// RandomPersonName picks a random country's name pool (see refereeNamePool) and returns one
// "First Last" name from it - used for a match's referee and assistant referees, which aren't
// tied to a specific player or team the way GetRandomName's other callers are.
func RandomPersonName(dbCtx *DatabaseContext) (string, error) {
	country := refereeNamePool[rand.Intn(len(refereeNamePool))]
	first, last, err := GetRandomName(dbCtx, country)
	if err != nil {
		return "", err
	}
	return first + " " + last, nil
}

// UpdateMatchFlavorStmtSql stores a match's report flavor - see UpdateMatchFlavor.
const UpdateMatchFlavorStmtSql = "UPDATE matches SET weather = $1, attendance = $2, referee_name = $3, " +
	"assistant1_name = $4, assistant2_name = $5 WHERE match_id = $6"

// UpdateMatchFlavor persists a just-simulated match's report flavor (weather, crowd size,
// referee/assistants) - generated once here rather than on every later read, so a match's report
// shows the same numbers/names every time it's viewed (see resources/sql/init40_match_flavor.sql).
// attendance is passed in rather than computed here so it can be the exact same figure already
// shown in the home team's ticket-revenue transaction (see engine.creditTicketRevenue) - drawing
// a second, independent random attendance here would let the two disagree.
func UpdateMatchFlavor(dbCtx *DatabaseContext, matchId int64, weather string, attendance int, refereeName string, assistant1Name string, assistant2Name string) error {
	_, err := dbCtx.Db.Exec(UpdateMatchFlavorStmtSql, weather, attendance, refereeName, assistant1Name, assistant2Name, matchId)
	return err
}

// UpdateCupTieFlavorStmtSql / UpdateFriendlyFlavorStmtSql are UpdateMatchFlavorStmtSql for the cup
// tie and friendly tables (see resources/sql/init83_match_viewer_parity.sql), so every competition
// stores the same report flavor and the match viewer needs no per-competition gaps.
const UpdateCupTieFlavorStmtSql = "UPDATE cup_ties SET weather = $1, attendance = $2, referee_name = $3, " +
	"assistant1_name = $4, assistant2_name = $5 WHERE tie_id = $6"
const UpdateFriendlyFlavorStmtSql = "UPDATE friendly_matches SET weather = $1, attendance = $2, referee_name = $3, " +
	"assistant1_name = $4, assistant2_name = $5 WHERE match_id = $6"

// UpdateCupTieFlavor persists a just-simulated cup tie's report flavor - see UpdateMatchFlavor.
func UpdateCupTieFlavor(dbCtx *DatabaseContext, tieId int64, weather string, attendance int, refereeName string, assistant1Name string, assistant2Name string) error {
	_, err := dbCtx.Db.Exec(UpdateCupTieFlavorStmtSql, weather, attendance, refereeName, assistant1Name, assistant2Name, tieId)
	return err
}

// UpdateFriendlyFlavor persists a just-simulated friendly's report flavor - see UpdateMatchFlavor.
func UpdateFriendlyFlavor(dbCtx *DatabaseContext, matchId int64, weather string, attendance int, refereeName string, assistant1Name string, assistant2Name string) error {
	_, err := dbCtx.Db.Exec(UpdateFriendlyFlavorStmtSql, weather, attendance, refereeName, assistant1Name, assistant2Name, matchId)
	return err
}
