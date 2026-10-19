package common

import (
	"math"
	"math/rand"
	"time"
)

// TransferFeeWageMultiplier is how many times a player's own market-rate wage (see
// PlayerWageFromSkills) their base transfer value is worth - the yardstick every CPU team uses
// to price a player, both when listing one and when judging a fee (see MarketValue).
const TransferFeeWageMultiplier = 20

// MinSquadSizeAfterSale is the smallest owned senior squad a CPU team will keep once a player
// leaves - a couple under SquadSize, since every team starts at exactly SquadSize and the academy
// only tops back up to that (see engine's listSurplusPlayer, which shares this floor).
const MinSquadSizeAfterSale = SquadSize - 2

// MaxAiSquadSize is the largest owned senior squad a CPU team grows to by buying: it never has an
// acquisition quota, but a squad that keeps improving still can't grow without bound - past this,
// it has to sell before it signs.
const MaxAiSquadSize = SquadSize + 4

// AiMinHoldPeriod is how long a CPU team keeps a player it just signed before it will list him as
// surplus - without it a signing that happens to be the weakest at an overstocked position gets
// listed the moment he arrives, and bounces between clubs at the same fee.
const AiMinHoldPeriod = 14 * 24 * time.Hour

const (
	// ProjectionHorizonSeasons is how far ahead a CPU team looks when judging a young player.
	ProjectionHorizonSeasons = 3
	// projectionSkillGainPerSeason is the rough number of points a young player's skills climb
	// per season while below their (hidden) ceiling - an approximation of the per-match drift
	// engine.updatePlayerSkillsAfterMatch applies, since the real rate depends on playing time.
	projectionSkillGainPerSeason = 4
	// ProjectionMaxAge: a player younger than this is a candidate for projection.
	ProjectionMaxAge = 24
	// projectionValueWeight is how much of a young player's projected extra value is already
	// paid for in their price today.
	projectionValueWeight = 0.5

	// thinPositionBonus is added to a target's score when it fills a position the squad is short
	// at (fewer than MinPositionDepth players).
	thinPositionBonus = 4.0
	// valueForMoneyWeight/valueForMoneyCap turn "price vs. what the team thinks he's worth" into
	// score points: a bargain earns up to the cap, an overpay costs up to the cap.
	valueForMoneyWeight = 4.0
	valueForMoneyCap    = 4.0
	// MaxProjects is how many squad places a team dedicates to young players it signed for
	// their future, and maxProjectBudgetShare the share of the budget a single one may cost.
	MaxProjects           = 3
	maxProjectBudgetShare = 0.25
	// WageRunwayRounds is how many rounds of the new payroll a team that watches its wage bill
	// (see AiTransferProfile.ConsidersWageBill) must still be able to pay after a signing.
	WageRunwayRounds = 20

	// internationalBonusBase/internationalBonusPerCap are the score a profile that values
	// internationals (a Director) gives a candidate in a national-team pool, and for each of his
	// caps up to internationalCapsCap.
	internationalBonusBase   = 2.0
	internationalBonusPerCap = 0.1

	// keyPlayerStrengthDrop is how many best-XI points losing a starter must cost for him to
	// count as irreplaceable even when the position keeps its minimum depth.
	keyPlayerStrengthDrop = 6
)

// ageValueFactor scales a player's transfer value by age: a veteran is worth progressively less
// since his decline is near, a very young one a little more.
func ageValueFactor(age int) float64 {
	switch {
	case age < 23:
		return 1.2
	case age < 28:
		return 1.0
	case age < 31:
		return 0.85
	case age < 34:
		return 0.6
	default:
		return 0.4
	}
}

// MarketValue is the base transfer value of a player - his market wage times
// TransferFeeWageMultiplier, scaled by age. It is the price a CPU team asks when listing him and
// the starting point of every valuation; EvaluateIncomingOffer and ScoreTarget perturb it per
// AiTransferProfile.
func MarketValue(p Player) int64 {
	wage := PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot)
	return int64(math.Round(float64(wage*TransferFeeWageMultiplier) * ageValueFactor(p.Age)))
}

// internationalCapsPremiumUnit/internationalCapsCap shape how much more a national-team player
// is worth with caps: the profile's InternationalPremium for a pool member with none, up to twice
// that at internationalCapsCap caps.
const internationalCapsCap = 20

// internationalPremium is the fraction a CPU team adds to what it asks for a national-team
// player: the profile's base premium, growing with his caps up to double. 0 for a profile
// without one, and for a player who isn't an international.
func internationalPremium(profile AiTransferProfile, international bool, caps int) float64 {
	if !international || profile.InternationalPremium <= 0 {
		return 0
	}
	if caps > internationalCapsCap {
		caps = internationalCapsCap
	}
	if caps < 0 {
		caps = 0
	}
	return profile.InternationalPremium * (1 + float64(caps)/internationalCapsCap)
}

// ListingAskingPrice is what a CPU team asks for a player it puts on the transfer list: his
// market value, plus the profile's premium for a national-team player (more with caps).
func ListingAskingPrice(p Player, profile AiTransferProfile, international bool, caps int) int64 {
	value := float64(MarketValue(p))
	value *= 1 + internationalPremium(profile, international, caps)
	return int64(math.Round(value))
}

func clampSkill(v float64) int {
	r := int(math.Round(v))
	if r < 1 {
		return 1
	}
	if r > 99 {
		return 99
	}
	return r
}

// noiseFactor draws a multiplicative error in [1-amplitude, 1+amplitude].
func noiseFactor(rng *rand.Rand, amplitude float64) float64 {
	if amplitude == 0 {
		return 1
	}
	return 1 + (rng.Float64()*2-1)*amplitude
}

// perceivePlayer returns a copy of p whose seven core skills are scaled by one shared error
// factor - how good a team with this much valuation noise believes he is.
func perceivePlayer(p Player, noise float64, rng *rand.Rand) Player {
	if noise == 0 {
		return p
	}
	factor := noiseFactor(rng, noise)
	p.GoalKeeper = clampSkill(float64(p.GoalKeeper) * factor)
	p.Defense = clampSkill(float64(p.Defense) * factor)
	p.Lateral = clampSkill(float64(p.Lateral) * factor)
	p.Pass = clampSkill(float64(p.Pass) * factor)
	p.Midfield = clampSkill(float64(p.Midfield) * factor)
	p.Dribbling = clampSkill(float64(p.Dribbling) * factor)
	p.Shot = clampSkill(float64(p.Shot) * factor)
	return p
}

// youthGrowthFactor is how much of the full projected growth a player of this age still has
// ahead of him.
func youthGrowthFactor(age int) float64 {
	switch {
	case age <= 20:
		return 1
	case age <= 22:
		return 0.8
	default:
		return 0.5
	}
}

// ProjectPlayer estimates what a player under ProjectionMaxAge will be after
// ProjectionHorizonSeasons seasons, as the team evaluating him sees it. Each skill climbs by the
// youth growth rate but never past the team's *estimate* of his ceiling: the real hidden ceiling
// (caps, nil when unknown) is read with the profile's PotentialNoise, so a Manager misjudges a
// prospect's headroom badly and a Director only slightly. With no ceiling at all the team assumes
// a typical ten-point headroom. A profile that doesn't estimate potential, or a player past the
// age limit, projects to himself.
func ProjectPlayer(p Player, caps *PlayerSkillCaps, profile AiTransferProfile, rng *rand.Rand) Player {
	if !profile.EstimatesPotential || p.Age >= ProjectionMaxAge {
		return p
	}
	growth := float64(projectionSkillGainPerSeason*ProjectionHorizonSeasons) * youthGrowthFactor(p.Age)
	project := func(skill, ceiling int) int {
		headroom := float64(10)
		if caps != nil {
			headroom = float64(ceiling - skill)
		}
		if headroom < 0 {
			headroom = 0
		}
		estimated := headroom * noiseFactor(rng, profile.PotentialNoise)
		gain := math.Min(growth, estimated)
		if gain < 0 {
			gain = 0
		}
		return clampSkill(float64(skill) + gain)
	}
	var c PlayerSkillCaps
	if caps != nil {
		c = *caps
	}
	p.GoalKeeper = project(p.GoalKeeper, c.GoalKeeperMax)
	p.Defense = project(p.Defense, c.DefenseMax)
	p.Lateral = project(p.Lateral, c.LateralMax)
	p.Pass = project(p.Pass, c.PassMax)
	p.Midfield = project(p.Midfield, c.MidfieldMax)
	p.Dribbling = project(p.Dribbling, c.DribblingMax)
	p.Shot = project(p.Shot, c.ShotMax)
	return p
}

// PerceivedValue is what a team with this profile believes a player is worth: his market value
// recomputed from skills seen through the profile's valuation noise, plus - for a profile that
// estimates potential - half of the extra value he is projected to gain by growing. A Caretaker
// therefore prices a teenage wonderkid like any other teenager, and a Director gets him almost
// right.
func PerceivedValue(p Player, caps *PlayerSkillCaps, profile AiTransferProfile, rng *rand.Rand) int64 {
	seen := perceivePlayer(p, profile.ValuationNoise, rng)
	value := float64(MarketValue(seen))
	if profile.EstimatesPotential && p.Age < ProjectionMaxAge {
		projected := ProjectPlayer(seen, caps, profile, rng)
		if extra := float64(MarketValue(projected)) - value; extra > 0 {
			value += extra * projectionValueWeight
		}
	}
	return int64(math.Round(value))
}

// positionDepthOf counts the players in squad whose best core position is position.
func positionDepthOf(squad []*Player, position int) int {
	depth := 0
	for _, p := range squad {
		if BestCorePosition(*p) == position {
			depth++
		}
	}
	return depth
}

func squadWithout(squad []*Player, playerId int64) []*Player {
	kept := make([]*Player, 0, len(squad))
	for _, p := range squad {
		if p.Id != playerId {
			kept = append(kept, p)
		}
	}
	return kept
}

// IsKeyPlayer reports whether selling playerId would leave the squad lame: he starts in the best
// XI and either his position drops below MinPositionDepth without him or the XI loses at least
// keyPlayerStrengthDrop points of strength. A bench player is never a key player.
func IsKeyPlayer(squad []*Player, playerId int64) bool {
	var player *Player
	for _, p := range squad {
		if p.Id == playerId {
			player = p
			break
		}
	}
	if player == nil {
		return false
	}
	starts := false
	for _, ids := range SelectBestLineup(squad) {
		for _, id := range ids {
			if id == playerId {
				starts = true
			}
		}
	}
	if !starts {
		return false
	}
	rest := squadWithout(squad, playerId)
	position := BestCorePosition(*player)
	if positionDepthOf(rest, position) < MinPositionDepth[position] {
		return true
	}
	return SquadStrength(squad)-SquadStrength(rest) >= keyPlayerStrengthDrop
}

// Reasons EvaluateIncomingOffer gives for its decision.
const (
	OfferReasonAccepted      = "accepted"
	OfferReasonBelowValue    = "below_value"
	OfferReasonKeyPlayer     = "key_player"
	OfferReasonSquadTooSmall = "squad_too_small"
	OfferReasonThinPosition  = "thin_position"
	OfferReasonInternational = "international"
)

// OfferContext is everything EvaluateIncomingOffer needs to know about an offer on one of a CPU
// team's own players. Squad is the team's owned senior squad (loaned-in players excluded) and
// includes Player; Caps is his hidden ceiling (nil if unknown); International says whether he is
// in his country's national-team pool.
type OfferContext struct {
	Squad         []*Player
	Player        Player
	Caps          *PlayerSkillCaps
	Amount        int64
	International bool
	// InternationalCaps is his international caps count, which raises the premium.
	InternationalCaps int
}

// OfferDecision is a CPU team's answer to an offer: Threshold is the lowest fee it would have
// taken for this player in this squad, Reason one of the OfferReason* constants.
type OfferDecision struct {
	Accept    bool
	Threshold int64
	Reason    string
}

// EvaluateIncomingOffer decides whether a CPU team with the given profile sells a player for an
// offer. It replaces the old flat "20x wage" rule: the bar is the team's perceived value of the
// player (so a noisy Caretaker undervalues or overvalues him at random), raised by the profile
// when selling would hurt - a key starter (ProtectsKeyPlayers) costs KeyPlayerPriceMultiplier
// times as much, a national-team player the InternationalPremium on top (up to double with caps). Selling that would leave
// the squad under MinSquadSizeAfterSale, or leaving his position under MinPositionDepth, is refused
// outright - at every level.
func EvaluateIncomingOffer(profile AiTransferProfile, ctx OfferContext, rng *rand.Rand) OfferDecision {
	threshold := float64(PerceivedValue(ctx.Player, ctx.Caps, profile, rng))
	reason := OfferReasonBelowValue

	if len(ctx.Squad)-1 < MinSquadSizeAfterSale {
		return OfferDecision{Threshold: int64(math.Round(threshold)), Reason: OfferReasonSquadTooSmall}
	}
	if position := BestCorePosition(ctx.Player); positionDepthOf(ctx.Squad, position)-1 < MinPositionDepth[position] {
		return OfferDecision{Threshold: int64(math.Round(threshold)), Reason: OfferReasonThinPosition}
	}
	if premium := internationalPremium(profile, ctx.International, ctx.InternationalCaps); premium > 0 {
		threshold *= 1 + premium
		reason = OfferReasonInternational
	}
	if profile.ProtectsKeyPlayers && IsKeyPlayer(ctx.Squad, ctx.Player.Id) {
		threshold *= profile.KeyPlayerPriceMultiplier
		reason = OfferReasonKeyPlayer
	}

	decision := OfferDecision{Threshold: int64(math.Round(threshold)), Reason: reason}
	if float64(ctx.Amount) >= threshold {
		decision.Accept = true
		decision.Reason = OfferReasonAccepted
	}
	return decision
}

// TargetContext is the situation a CPU team scores one candidate signing in. Squad is the team's
// owned senior squad (loaned-in excluded); Price is the fee it would pay (the asking price of a
// listing, the offer it would make for an unlisted player, the premium for a free agent); Budget
// and WageBill are the team's current figures; ProjectsInUse is how many places it already
// dedicates to young prospects. Caps is the candidate's hidden ceiling, which the scorer only
// ever reads through the profile's noise.
type TargetContext struct {
	Squad  []*Player
	Target Player
	Caps   *PlayerSkillCaps
	Price  int64
	// BidMultiple, when positive, prices the signing as a bid for a player nobody has listed:
	// Price is then the team's own valuation of him times BidMultiple (and Price is ignored).
	BidMultiple float64
	Budget      int64
	WageBill    int64
	// WageFactor is the team's division wage factor (see DivisionWageFactor): WageBill already
	// includes it, so the candidate's stored wage must be scaled by it too to be comparable. 0
	// means 1.0 (no factor).
	WageFactor    float64
	ProjectsInUse int
	// International is whether the candidate is in his country's national-team pool, and
	// InternationalCaps his caps count - both only matter to a profile that values them.
	International     bool
	InternationalCaps int
}

// TargetScore is ScoreTarget's verdict, broken into the parts that make it up.
type TargetScore struct {
	// Total is what the team compares against AiTransferProfile.MinScore; meaningless unless
	// Affordable.
	Total float64
	// Improvement is how many points the candidate adds to the best XI as the team sees him now
	// (plus the bonus for filling a thin position); Projection how many more he is expected to
	// add after ProjectionHorizonSeasons seasons (0 unless the profile signs projects);
	// ValueForMoney the bargain/overpay adjustment; Reputation the bonus for being in a national
	// team's pool, which grows with his caps (profiles that value internationals only).
	Improvement   float64
	Projection    float64
	ValueForMoney float64
	Reputation    float64
	// Value is the team's perceived worth of the candidate - what it would stop at when raising
	// a bid. Price is what the signing was scored at (the context's, or the bid it implied).
	Value int64
	Price int64
	// Affordable is false when the signing would leave the budget under the profile's reserve
	// (or, for a profile that watches payroll, short of WageRunwayRounds of the new wage bill);
	// Unaffordable verdicts carry no Total. BudgetShort says that was for lack of money (as
	// opposed to a reserve or payroll limit that no sale of this size would lift) - what a
	// Director that sells to buy looks for.
	Affordable  bool
	BudgetShort bool
	// Project is true when the signing is justified by growth rather than by the present.
	Project bool
}

// consolidation is the share of the squad's best XI aged 25 or more - how settled the starting
// eleven is. A team with a settled eleven can afford to invest in the future; one still sorting
// its starters out should not.
func consolidation(squad []*Player) float64 {
	byId := make(map[int64]*Player, len(squad))
	for _, p := range squad {
		byId[p.Id] = p
	}
	total, settled := 0, 0
	for _, ids := range SelectBestLineup(squad) {
		for _, id := range ids {
			if p, ok := byId[id]; ok {
				total++
				if p.Age >= 25 {
					settled++
				}
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(settled) / float64(total)
}

// ScoreTarget values a candidate signing for a CPU team. Pure: the same inputs and the same rng
// state always give the same answer, so it can be tested without a database. It combines how
// much the best XI improves (SquadStrength with the candidate, as the team perceives him through
// its valuation noise) with a bonus for filling a thin position, a projection of what a young
// player will add in ProjectionHorizonSeasons seasons (profile.SignsProjects only, weighted more
// for a team with settled starters), whether the price is a bargain or an overpay against the
// team's own valuation, and whether the budget (and, for a Director, the payroll) can take it.
func ScoreTarget(profile AiTransferProfile, ctx TargetContext, rng *rand.Rand) TargetScore {
	score := TargetScore{}
	score.Value = PerceivedValue(ctx.Target, ctx.Caps, profile, rng)
	score.Price = ctx.Price
	if ctx.BidMultiple > 0 {
		score.Price = int64(math.Round(float64(score.Value) * ctx.BidMultiple))
	}

	remaining := ctx.Budget - score.Price
	if remaining < 0 || float64(remaining) < float64(ctx.Budget)*profile.BudgetReserve {
		score.BudgetShort = true
		return score
	}
	if profile.ConsidersWageBill {
		newWage := int64(ctx.Target.Wage)
		if ctx.Target.Wage == 0 {
			newWage = int64(PlayerWageFromSkills(ctx.Target.GoalKeeper, ctx.Target.Defense, ctx.Target.Lateral,
				ctx.Target.Pass, ctx.Target.Midfield, ctx.Target.Dribbling, ctx.Target.Shot))
		}
		if ctx.WageFactor > 0 {
			newWage = ApplyWageFactor(newWage, ctx.WageFactor)
		}
		newBill := ctx.WageBill + newWage
		if remaining < newBill*WageRunwayRounds {
			score.BudgetShort = true
			return score
		}
	}
	score.Affordable = true

	seen := perceivePlayer(ctx.Target, profile.ValuationNoise, rng)
	current := SquadStrength(ctx.Squad)
	withTarget := SquadStrength(append(append([]*Player(nil), ctx.Squad...), &seen))
	score.Improvement = float64(withTarget - current)
	if position := BestCorePosition(seen); positionDepthOf(ctx.Squad, position) < MinPositionDepth[position] {
		score.Improvement += thinPositionBonus
	}

	if profile.SignsProjects && ctx.Target.Age < ProjectionMaxAge && ctx.ProjectsInUse < MaxProjects &&
		float64(score.Price) <= float64(ctx.Budget)*maxProjectBudgetShare {
		projected := ProjectPlayer(seen, ctx.Caps, profile, rng)
		withProjected := SquadStrength(append(append([]*Player(nil), ctx.Squad...), &projected))
		if gain := withProjected - withTarget; gain > 0 {
			weight := 0.3 + 0.7*consolidation(ctx.Squad)
			score.Projection = float64(gain) * weight
			score.Project = score.Improvement < float64(thinPositionBonus)
		}
	}

	if score.Price == 0 {
		score.ValueForMoney = valueForMoneyCap
	} else if score.Value > 0 {
		ratio := float64(score.Price) / float64(score.Value)
		score.ValueForMoney = math.Max(-valueForMoneyCap, math.Min(valueForMoneyCap, (1-ratio)*valueForMoneyWeight))
	}

	if ctx.International && profile.InternationalPremium > 0 {
		caps := ctx.InternationalCaps
		if caps > internationalCapsCap {
			caps = internationalCapsCap
		}
		score.Reputation = internationalBonusBase + internationalBonusPerCap*float64(caps)
	}

	score.Total = score.Improvement + score.Projection + score.ValueForMoney + score.Reputation
	return score
}
