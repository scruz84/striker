package common

import (
	"sort"
	"strings"
	"time"
)

// Opponent preview (see openspec opponent-preview): what an owner sees about the club he is about
// to play, built only from data that is already public - results, kickoff lineups and stored match
// ratings of matches whose live replay has finished, and the league table.

const (
	// OpponentPreviewFormMatches is how many of the opponent's latest finished league/cup matches
	// feed the form strip, the likely XI and the in-form players.
	OpponentPreviewFormMatches = 5
	// OpponentPreviewInFormPlayers is how many in-form players the preview lists.
	OpponentPreviewInFormPlayers = 3
	// OpponentPreviewMinAppearances is how many rated appearances in the form window a player needs
	// to be listed as in form.
	OpponentPreviewMinAppearances = 3
)

// PreviewPlayer is one player of the opponent's likely XI: no skills, only who he is and where he
// lined up.
type PreviewPlayer struct {
	PlayerId int64  `json:"playerId"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// LikelyLineup is the formation and XI the opponent kicked off with in its latest finished league
// or cup match (Kind/MatchId name that match, so the UI can say where the XI comes from).
type LikelyLineup struct {
	Kind      string          `json:"kind"`
	MatchId   int64           `json:"matchId"`
	Formation int             `json:"formation"`
	Players   []PreviewPlayer `json:"players"`
}

// InFormPlayer is one of the opponent's best-rated players over the form window.
type InFormPlayer struct {
	PlayerId      int64   `json:"playerId"`
	Name          string  `json:"name"`
	Position      int     `json:"position"`
	AverageRating float64 `json:"averageRating"`
	Appearances   int     `json:"appearances"`
}

// OpponentPreviewTeam is the opponent's identity, league position and recent results.
type OpponentPreviewTeam struct {
	Id       int64              `json:"id"`
	Name     string             `json:"name"`
	Position int                `json:"position"`
	Form     []TeamMatchSummary `json:"form"`
}

// OpponentPreview is the public part of the preview; the projected zone duel is added by the
// handler, which owns the engine dependency.
type OpponentPreview struct {
	Opponent OpponentPreviewTeam `json:"opponent"`
	Likely   *LikelyLineup       `json:"likely"`
	InForm   []InFormPlayer      `json:"inForm"`
}

// recentFinishedCompetitive keeps the club's league and cup matches that are played and whose live
// replay has finished (match_date at or before cutoff - see LiveBroadcastCutoff), newest first,
// capped at limit. A friendly never counts as form.
func recentFinishedCompetitive(matches []TeamMatchSummary, cutoff time.Time, limit int) []TeamMatchSummary {
	finished := make([]TeamMatchSummary, 0, len(matches))
	for _, m := range matches {
		if m.Kind == MatchKindFriendly || !m.Played || m.MatchDate.After(cutoff) {
			continue
		}
		finished = append(finished, m)
	}
	sort.SliceStable(finished, func(i, j int) bool { return finished[i].MatchDate.After(finished[j].MatchDate) })
	if len(finished) > limit {
		finished = finished[:limit]
	}
	return finished
}

// ratingsKind maps a TeamMatchSummary kind to the SimulationKind the ratings are stored under.
func ratingsKind(kind string) string {
	if kind == MatchKindCup {
		return SimulationKindCup
	}
	return SimulationKindLeague
}

// aggregateInForm averages teamId's rated appearances over the given matches' ratings (newest
// match first) and returns the best limit players with at least minAppearances of them, best first.
// A player's position is the one he held in his most recent appearance. Ties break on more
// appearances, then lower id, so the result is deterministic.
func aggregateInForm(teamId int64, ratingsByMatch [][]PlayerRating, minAppearances int, limit int) []InFormPlayer {
	type tally struct {
		sum         float64
		appearances int
		position    int
	}
	tallies := make(map[int64]*tally)
	for _, ratings := range ratingsByMatch {
		for _, r := range ratings {
			if r.TeamId != teamId || !r.Rated {
				continue
			}
			t := tallies[r.PlayerId]
			if t == nil {
				// the first time a player shows up is his most recent match, since the input is
				// newest first
				t = &tally{position: r.Position}
				tallies[r.PlayerId] = t
			}
			t.sum += r.Rating
			t.appearances++
		}
	}

	result := make([]InFormPlayer, 0, len(tallies))
	for id, t := range tallies {
		if t.appearances < minAppearances {
			continue
		}
		result = append(result, InFormPlayer{
			PlayerId:      id,
			Position:      t.position,
			AverageRating: roundTenth(t.sum / float64(t.appearances)),
			Appearances:   t.appearances,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.AverageRating != b.AverageRating {
			return a.AverageRating > b.AverageRating
		}
		if a.Appearances != b.Appearances {
			return a.Appearances > b.Appearances
		}
		return a.PlayerId < b.PlayerId
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func previewPlayerName(p *Player) string {
	return strings.TrimSpace(p.FirstName + " " + p.LastName)
}

// likelyLineupFromSaved turns a stored kickoff lineup into the preview's XI, dropping players who
// are no longer in the club's squad. nil when nothing is left to show.
func likelyLineupFromSaved(kind string, matchId int64, lineup Lineup, squad map[int64]*Player) *LikelyLineup {
	players := make([]PreviewPlayer, 0, len(lineup.Positions))
	for _, p := range lineup.Positions {
		member, ok := squad[p.PlayerId]
		if !ok {
			continue
		}
		players = append(players, PreviewPlayer{PlayerId: p.PlayerId, Name: previewPlayerName(member), Position: p.Position})
	}
	if len(players) == 0 {
		return nil
	}
	return &LikelyLineup{Kind: kind, MatchId: matchId, Formation: lineup.Formation, Players: players}
}

// GetOpponentLikelyLineup is the opponent's kickoff lineup in the newest of form (newest first)
// that has a stored one, with players who left the club dropped; nil when none has.
func GetOpponentLikelyLineup(dbCtx *DatabaseContext, opponentId int64, form []TeamMatchSummary, squad map[int64]*Player) (*LikelyLineup, error) {
	for _, m := range form {
		var lineup Lineup
		var has bool
		var err error
		if m.Kind == MatchKindCup {
			lineup, has, err = GetCupTieLineup(dbCtx, m.MatchId, opponentId)
		} else {
			lineup, has, err = GetMatchLineupOverride(dbCtx, m.MatchId, opponentId)
		}
		if err != nil {
			return nil, err
		}
		if !has {
			continue
		}
		if likely := likelyLineupFromSaved(m.Kind, m.MatchId, lineup, squad); likely != nil {
			return likely, nil
		}
	}
	return nil, nil
}

// GetOpponentPreview builds the public part of the preview of opponentId: its last
// OpponentPreviewFormMatches finished league/cup results, its league position, the XI it kicked
// off with in its latest such match and its best-rated players over them. Nothing in it comes from
// a match that is still being replayed or from the opponent's saved lineup for a future match.
func GetOpponentPreview(dbCtx *DatabaseContext, opponentId int64) (*OpponentPreview, error) {
	team, err := GetTeamById(opponentId, dbCtx)
	if err != nil {
		return nil, err
	}
	position := 0
	if team.DivisionId != 0 {
		season, err := GetCurrentSeasonForLeague(dbCtx, team.DivisionId)
		if err != nil {
			return nil, err
		}
		if position, err = GetTeamClassificationPosition(dbCtx, team.DivisionId, season, opponentId); err != nil {
			return nil, err
		}
	}

	matches, err := GetTeamMatches(dbCtx, opponentId)
	if err != nil {
		return nil, err
	}
	form := recentFinishedCompetitive(matches, LiveBroadcastCutoff(), OpponentPreviewFormMatches)

	players, err := GetPlayersFromTeamId(dbCtx, opponentId)
	if err != nil {
		return nil, err
	}
	squad := make(map[int64]*Player, len(*players))
	for _, p := range *players {
		squad[p.Id] = p
	}

	preview := &OpponentPreview{
		Opponent: OpponentPreviewTeam{Id: opponentId, Name: team.TeamName, Position: position, Form: form},
		InForm:   make([]InFormPlayer, 0),
	}
	if len(form) == 0 {
		return preview, nil
	}

	if preview.Likely, err = GetOpponentLikelyLineup(dbCtx, opponentId, form, squad); err != nil {
		return nil, err
	}

	ratingsByMatch := make([][]PlayerRating, 0, len(form))
	for _, m := range form {
		ratings, err := GetMatchPlayerRatings(dbCtx, ratingsKind(m.Kind), m.MatchId)
		if err != nil {
			return nil, err
		}
		ratingsByMatch = append(ratingsByMatch, ratings)
	}
	for _, p := range aggregateInForm(opponentId, ratingsByMatch, OpponentPreviewMinAppearances, OpponentPreviewInFormPlayers) {
		// a player who left the club is no longer worth scouting
		member, ok := squad[p.PlayerId]
		if !ok {
			continue
		}
		p.Name = previewPlayerName(member)
		preview.InForm = append(preview.InForm, p)
	}
	return preview, nil
}
