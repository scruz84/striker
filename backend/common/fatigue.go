package common

// FatigueMaxPenalty is the largest fraction a player's effective core skills can be reduced by
// from in-match fatigue alone - reached only by a Stamina-0 player who has played every minute of
// the match so far. A Stamina-99 player is never fatigued at all, whatever the minute (see
// FatigueMultiplier).
const FatigueMaxPenalty = 0.30

// FatigueMatchLength is the match length (in minutes) FatigueMultiplier normalizes elapsed
// playing time against - matches the engine's own endMinute.
const FatigueMatchLength = 90

// FatigueMultiplier scales down linearly with both elapsed playing time (as a fraction of
// FatigueMatchLength) and how little Stamina (0-99) the player has, capped so extra time logged
// beyond FatigueMatchLength (there isn't any - matches run exactly 90 minutes - but this keeps the
// formula well-defined regardless) never pushes the penalty past FatigueMaxPenalty. It lives in
// common so the engine's StaminaSimulation and the CPU lineup planner (see PlanAiMatchLineup) read
// the exact same formula.
func FatigueMultiplier(stamina int, minutesPlayed int) float64 {
	return FatigueMultiplierAt(stamina, float64(minutesPlayed))
}

// FatigueMultiplierAt is FatigueMultiplier for a fractional minute count. The multiplier is linear
// in minutes played, so a player's average multiplier over a stretch of play is the value at the
// stretch's midpoint - which is what the CPU lineup planner uses to value a substitution.
func FatigueMultiplierAt(stamina int, minutesPlayed float64) float64 {
	return FatigueMultiplierWithPenalty(stamina, minutesPlayed, FatigueMaxPenalty)
}

// FatigueMultiplierWithPenalty is FatigueMultiplierAt with the side's own maximum penalty, which
// its mentality sets (see FatigueMaxPenaltyFor): an Attacking side tires faster, a Defensive one
// slower.
func FatigueMultiplierWithPenalty(stamina int, minutesPlayed float64, maxPenalty float64) float64 {
	if minutesPlayed > FatigueMatchLength {
		minutesPlayed = FatigueMatchLength
	}
	elapsedFraction := minutesPlayed / float64(FatigueMatchLength)
	staminaDeficit := float64(99-stamina) / 99
	return 1 - maxPenalty*elapsedFraction*staminaDeficit
}
