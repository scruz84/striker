package common

import (
	"math"
	"sort"
)

// Per-player match rating constants (see openspec/specs/match-ratings). A rating is the base, plus
// the zone component (how the player's own zone contribution compared with the rival's in the
// mirrored duel, minute by minute - see PlayerContribution), plus his own goals/cards/injury, plus
// his team's result scaled by the minutes he played. frontend/src/utils/liveMatchRatings.ts is a
// line-by-line port of ComputePlayerRatings and must stay in sync with it - both are pinned by the
// shared fixture frontend/src/utils/testdata/match_ratings_parity.json.
const (
	matchRatingBase        = 6.0
	matchRatingMin         = 1.0
	matchRatingMax         = 10.0
	matchRatingGoal        = 1.4
	matchRatingYellowCard  = -0.4
	matchRatingRedCard     = -1.5
	matchRatingInjury      = -0.2
	matchRatingResultBonus = 0.3
	// matchRatingZoneWeight scales a player's mean duel edge ((own - rival) / (own + rival), so
	// -1..1) into the zone component, capped at +-matchRatingZoneCap: 55 vs 40 gives +0.47.
	matchRatingZoneWeight = 3.0
	matchRatingZoneCap    = 1.5
	// MatchRatingMinMinutes is the fewest minutes a player needs to be rated; below it he is
	// "s.c." (sin calificar) - see PlayerRating.Rated.
	MatchRatingMinMinutes = 10
	// matchRatingFullMatch is the minute count the result component is scaled against.
	matchRatingFullMatch = 90
)

// PlayerRating is one player's rating for one match, with its breakdown: Rating is always
// Base + Zone + Goals + Cards + Injury + Result (each already rounded to one decimal), clamped to
// [1, 10]. Rated is false for an appearance under MatchRatingMinMinutes ("s.c."): the record
// keeps its breakdown, but the rating is neither shown nor used for Form.
type PlayerRating struct {
	PlayerId int64 `json:"playerId"`
	TeamId   int64 `json:"teamId"`
	Rated    bool  `json:"rated"`
	// Rating is computed even when Rated is false, so the breakdown still adds up.
	Rating float64 `json:"rating"`
	// Minutes is how many simulated minutes listed the player on the pitch (see
	// MatchInstant.PlayerContributions), or, for a match without contributions, the minutes from
	// kickoff or his coming on until the final minute or his leaving.
	Minutes int `json:"minutes"`
	// Position is the tactical position he spent the most minutes in (his starting or inherited
	// slot for a match without contributions).
	Position int `json:"position"`
	// CameOn/WentOff are the minutes of the substitution that brought him on, and of the
	// substitution, red card or injury that took him off - nil when it didn't happen.
	CameOn  *int `json:"cameOn,omitempty"`
	WentOff *int `json:"wentOff,omitempty"`
	// Own/Rival are his mean zone contribution and the rival's, to one decimal - nil for a match
	// without contributions.
	Own    *float64 `json:"own,omitempty"`
	Rival  *float64 `json:"rival,omitempty"`
	Zone   float64  `json:"zone"`
	Goals  float64  `json:"goals"`
	Cards  float64  `json:"cards"`
	Injury float64  `json:"injury"`
	Result float64  `json:"result"`
}

// roundTenth rounds half away from zero to one decimal (liveMatchRatings.ts's roundTenth matches it
// exactly, including for negative halves). A negative zero comes back as 0, so it never reads
// "-0.0".
func roundTenth(v float64) float64 {
	rounded := math.Round(v*10) / 10
	if rounded == 0 {
		return 0
	}
	return rounded
}

// snapContribution turns a stored float32 contribution (already one decimal) into the float64 a
// JSON reader gets for the same text, so Go and the frontend do the same arithmetic.
func snapContribution(v float32) float64 {
	return roundTenth(float64(v))
}

type ratingParticipant struct {
	teamId          int64
	position        int
	positionMinutes map[int]int
	// positionOrder lists the positions in the order he first played them, so ties in
	// mostPlayedPosition go to the earlier one deterministically.
	positionOrder   []int
	firstPosition   int
	minutes         int
	edgeSum         float64
	ownSum          float64
	rivalSum        float64
	hasContribution bool
	cameOn          *int
	wentOff         *int
	goals           int
	yellows         int
	reds            int
	injuries        int
}

// ComputePlayerRatings rates every player who featured in a match, as of upToMinute: only instants
// and events at or before that minute count, the result component uses the score of the last goal
// (or abandonment) up to then, and the "s.c." threshold is the smaller of MatchRatingMinMinutes and
// the minutes elapsed so far. Called with the match's last minute, it gives the final ratings the
// engine stores (see MatchResult.PlayerRatings).
//
// When the instants up to upToMinute carry PlayerContributions, those decide who was on the pitch,
// for how long, in which position and with what zone component. Otherwise (a match simulated
// before init89_match_player_ratings.sql) the starting lineups and the substitution, red-card and
// injury events do, with a zone component of 0. Events are credited to the players they name in
// both cases, so a substitution never moves a goal, a card or minutes to someone else.
//
// Returns nil when nothing can be rated (no contributions and a missing lineup). The result is
// sorted by player id.
func ComputePlayerRatings(homeTeamId int64, awayTeamId int64, homeLineup *Lineup, awayLineup *Lineup,
	instants []MatchInstant, events []Event, upToMinute int) []PlayerRating {
	participants := make(map[int64]*ratingParticipant)
	participant := func(playerId int64, teamId int64) *ratingParticipant {
		p, ok := participants[playerId]
		if !ok {
			p = &ratingParticipant{teamId: teamId, positionMinutes: make(map[int]int), firstPosition: -1}
			participants[playerId] = p
		}
		return p
	}

	elapsed := 0
	withContributions := false
	for _, instant := range instants {
		if instant.Minute <= upToMinute && instant.PlayerContributions != nil {
			withContributions = true
			break
		}
	}

	if withContributions {
		for _, instant := range instants {
			if instant.Minute > upToMinute || instant.PlayerContributions == nil {
				continue
			}
			elapsed++
			for _, side := range []struct {
				teamId        int64
				contributions map[int64]PlayerContribution
			}{{homeTeamId, instant.PlayerContributions.Home}, {awayTeamId, instant.PlayerContributions.Away}} {
				for playerId, c := range side.contributions {
					p := participant(playerId, side.teamId)
					own, rival := snapContribution(c.Own), snapContribution(c.Rival)
					edge := 0.0
					if own+rival > 0 {
						edge = (own - rival) / (own + rival)
					}
					p.minutes++
					p.edgeSum += edge
					p.ownSum += own
					p.rivalSum += rival
					p.hasContribution = true
					if p.firstPosition < 0 {
						p.firstPosition = c.Position
					}
					if _, seen := p.positionMinutes[c.Position]; !seen {
						p.positionOrder = append(p.positionOrder, c.Position)
					}
					p.positionMinutes[c.Position]++
				}
			}
		}
	} else {
		if homeLineup == nil || awayLineup == nil {
			return nil
		}
		for _, lineup := range []*Lineup{homeLineup, awayLineup} {
			for _, pos := range lineup.Positions {
				p := participant(pos.PlayerId, lineup.TeamId)
				p.firstPosition = pos.Position
			}
		}
		elapsed = upToMinute
		for _, event := range events {
			if event.Minute <= upToMinute && event.EventType == MATCH_ABANDONED && event.Minute < elapsed {
				elapsed = event.Minute
			}
		}
		if elapsed > matchRatingFullMatch {
			elapsed = matchRatingFullMatch
		}
	}

	homeGoals, awayGoals := 0, 0
	for _, event := range events {
		if event.Minute > upToMinute {
			continue
		}
		switch event.EventType {
		case GOAL, MATCH_ABANDONED:
			homeGoals, awayGoals = event.HomeGoals, event.AwayGoals
		case SUBSTITUTION:
			if len(event.TeamOwnerPlayers) != 2 {
				continue
			}
			out := participant(event.TeamOwnerPlayers[0], event.TeamOwner)
			in := participant(event.TeamOwnerPlayers[1], event.TeamOwner)
			minute := event.Minute
			if out.wentOff == nil {
				out.wentOff = &minute
			}
			if in.cameOn == nil {
				in.cameOn = &minute
			}
			if in.firstPosition < 0 {
				in.firstPosition = out.firstPosition
			}
		}
		var credit func(p *ratingParticipant)
		switch event.EventType {
		case GOAL:
			credit = func(p *ratingParticipant) { p.goals++ }
		case YELLOW_CARD:
			credit = func(p *ratingParticipant) { p.yellows++ }
		case RED_CARD:
			credit = func(p *ratingParticipant) { p.reds++ }
		case INJURY:
			credit = func(p *ratingParticipant) { p.injuries++ }
		default:
			continue
		}
		for _, playerId := range event.TeamOwnerPlayers {
			p := participant(playerId, event.TeamOwner)
			credit(p)
			if (event.EventType == RED_CARD || event.EventType == INJURY) && p.wentOff == nil {
				minute := event.Minute
				p.wentOff = &minute
			}
		}
	}

	if !withContributions {
		for _, p := range participants {
			start, end := 0, elapsed
			if p.cameOn != nil {
				start = *p.cameOn
			}
			if p.wentOff != nil && *p.wentOff < end {
				end = *p.wentOff
			}
			if end > start {
				p.minutes = end - start
			}
		}
	}

	threshold := MatchRatingMinMinutes
	if elapsed < threshold {
		threshold = elapsed
	}

	ratings := make([]PlayerRating, 0, len(participants))
	for playerId, p := range participants {
		r := PlayerRating{
			PlayerId: playerId,
			TeamId:   p.teamId,
			Minutes:  p.minutes,
			Position: mostPlayedPosition(p),
			CameOn:   p.cameOn,
			WentOff:  p.wentOff,
			Goals:    roundTenth(float64(p.goals) * matchRatingGoal),
			Cards:    roundTenth(float64(p.yellows)*matchRatingYellowCard + float64(p.reds)*matchRatingRedCard),
			Injury:   roundTenth(float64(p.injuries) * matchRatingInjury),
		}
		if p.hasContribution {
			own := roundTenth(p.ownSum / float64(p.minutes))
			rival := roundTenth(p.rivalSum / float64(p.minutes))
			r.Own, r.Rival = &own, &rival
			zone := matchRatingZoneWeight * p.edgeSum / float64(p.minutes)
			r.Zone = roundTenth(math.Max(-matchRatingZoneCap, math.Min(matchRatingZoneCap, zone)))
		}
		minutes := math.Min(float64(p.minutes), matchRatingFullMatch)
		isHome := p.teamId == homeTeamId
		switch {
		case (isHome && homeGoals > awayGoals) || (!isHome && awayGoals > homeGoals):
			r.Result = roundTenth(matchRatingResultBonus * minutes / matchRatingFullMatch)
		case (isHome && awayGoals > homeGoals) || (!isHome && homeGoals > awayGoals):
			r.Result = roundTenth(-matchRatingResultBonus * minutes / matchRatingFullMatch)
		}
		rating := roundTenth(matchRatingBase + r.Zone + r.Goals + r.Cards + r.Injury + r.Result)
		r.Rating = math.Max(matchRatingMin, math.Min(matchRatingMax, rating))
		r.Rated = p.minutes >= threshold
		ratings = append(ratings, r)
	}
	sort.Slice(ratings, func(i, j int) bool { return ratings[i].PlayerId < ratings[j].PlayerId })
	return ratings
}

// mostPlayedPosition is the position a player spent the most minutes in, ties going to the one he
// played first; for a match without contributions, the slot he started in or inherited.
func mostPlayedPosition(p *ratingParticipant) int {
	best, bestMinutes := p.firstPosition, 0
	for _, position := range p.positionOrder {
		if minutes := p.positionMinutes[position]; minutes > bestMinutes {
			best, bestMinutes = position, minutes
		}
	}
	if best < 0 {
		return 0
	}
	return best
}

// MatchFinalMinute is the minute ComputePlayerRatings is called with for a finished match: the
// last instant's or event's minute, at least a full 90 (the same rule the frontend's
// finalMatchMinute uses for its scrubber).
func MatchFinalMinute(instants []MatchInstant, events []Event) int {
	final := matchRatingFullMatch
	for _, instant := range instants {
		if instant.Minute > final {
			final = instant.Minute
		}
	}
	for _, event := range events {
		if event.Minute > final {
			final = event.Minute
		}
	}
	return final
}

// RatingsById is the id -> rating map WeekMatch.Ratings and friends expose, holding only rated
// players (an "s.c." appearance has no rating). Nil for nil ratings.
func RatingsById(ratings []PlayerRating) *map[int64]float64 {
	if ratings == nil {
		return nil
	}
	byId := make(map[int64]float64, len(ratings))
	for _, r := range ratings {
		if r.Rated {
			byId[r.PlayerId] = r.Rating
		}
	}
	return &byId
}
