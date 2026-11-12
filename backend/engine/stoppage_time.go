package engine

import (
	"time"

	"github.com/scruz84/striker/common"
)

// Stoppage time (openspec/specs/stoppage-time): when a half's regulation minutes end, the time
// lost in them to its stoppages is added back at the end of that half. Each constant is the time
// one stoppage costs; the base is the time lost to ordinary restarts (throw-ins, goal kicks...).
const (
	StoppageBaseFirstHalf  = 30 * time.Second
	StoppageBaseSecondHalf = 60 * time.Second
	StoppageGoal           = 40 * time.Second
	StoppageSubstitution   = 30 * time.Second // injury replacements included
	StoppageInjury         = 60 * time.Second
	StoppageYellow         = 15 * time.Second
	StoppageRed            = 40 * time.Second // a second yellow is stored as one RED_CARD event
	StoppagePenalty        = 45 * time.Second // awarded, on top of the goal when it is scored
)

// MinAddedMinutes and MaxAddedFirstHalf/MaxAddedSecondHalf bound a half's added time. The second
// half's maximum is what bounds the longest live replay (see MaxBroadcastDuration), which must stay
// well inside common.LiveBroadcastWindow.
const (
	MinAddedMinutes    = 1
	MaxAddedFirstHalf  = 5
	MaxAddedSecondHalf = 5
)

// firstHalfLastMinute and secondHalfLastMinute are the clock minutes a half's added time hangs
// off: 45+1..45+N and 90+1..90+N.
const (
	firstHalfLastMinute  = 45
	secondHalfLastMinute = 90
)

// timeLost adds up what the regulation minutes fromMinute..toMinute of a half lost to stoppages:
// base plus each goal, substitution, injury, card and penalty recorded in them. Events already in
// added time (AddedMinute > 0) never count, so added time is fixed when regulation ends.
func timeLost(events []common.Event, fromMinute int, toMinute int, base time.Duration) time.Duration {
	lost := base
	for _, e := range events {
		if e.AddedMinute != 0 || e.Minute < fromMinute || e.Minute > toMinute {
			continue
		}
		switch e.EventType {
		case common.GOAL:
			lost += StoppageGoal
		case common.SUBSTITUTION:
			lost += StoppageSubstitution
		case common.INJURY:
			lost += StoppageInjury
		case common.YELLOW_CARD:
			lost += StoppageYellow
		case common.RED_CARD:
			lost += StoppageRed
		}
		if (e.EventType == common.GOAL || e.EventType == common.NO_GOAL) && e.SetPiece == common.SET_PIECE_PENALTY {
			lost += StoppagePenalty
		}
	}
	return lost
}

// addedMinutesFor turns a half's time lost into whole added minutes: rounded half up, then clamped
// to MinAddedMinutes..maxMinutes.
func addedMinutesFor(events []common.Event, fromMinute int, toMinute int, base time.Duration, maxMinutes int) int {
	lost := timeLost(events, fromMinute, toMinute, base)
	minutes := int((lost + 30*time.Second) / time.Minute)
	if minutes < MinAddedMinutes {
		return MinAddedMinutes
	}
	if minutes > maxMinutes {
		return maxMinutes
	}
	return minutes
}

// firstHalfAddedMinutes and secondHalfAddedMinutes are addedMinutesFor over each half's regulation
// minutes (1-45 and 46-90).
func firstHalfAddedMinutes(events []common.Event) int {
	return addedMinutesFor(events, 1, firstHalfLastMinute, StoppageBaseFirstHalf, MaxAddedFirstHalf)
}

func secondHalfAddedMinutes(events []common.Event) int {
	return addedMinutesFor(events, firstHalfLastMinute+1, secondHalfLastMinute, StoppageBaseSecondHalf, MaxAddedSecondHalf)
}
