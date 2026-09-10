package engine

import (
	"errors"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// RunAITeamManagement drives every automatic action an unowned team's AI level entitles it to -
// called once per match for each side involved (see UpdateTeamFinancesAfterMatch) instead of a
// separate scheduler job, so AI upkeep happens exactly as often as game time actually advances.
// See common.AiLevelCaretaker/Manager/Director for what each tier adds on top of the one below
// it; a no-op for any team with a real owner.
func RunAITeamManagement(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for AI management: %s", teamId, err)
		return
	}
	if team.OwnerId > 0 || team.AiLevel < common.AiLevelCaretaker {
		return
	}

	refreshAiTeamLineup(dbCtx, teamId)
	manageRosterIfManager(dbCtx, team)
	// Level 3 (transfer shopping/stadium/trainer) actions are added in a later phase, gated on
	// team.AiLevel >= common.AiLevelDirector.
}

// contractRenewalThresholdYears is how few seasons a Manager-level AI team lets a player's
// contract run down to before proactively renewing it (see renewExpiringContracts) - a real
// owner wouldn't let a useful squad player walk for nothing at season end just from inattention.
const contractRenewalThresholdYears = 1

// surplusDepthMargin is how far a core position's depth (see common.AssessPositionNeeds) must
// exceed common.MinPositionDepth before releaseSurplusPlayer treats it as overstocked enough to
// trim - one full backup beyond the minimum, so a position sitting exactly at minimum depth is
// never touched.
const surplusDepthMargin = 1

// manageRosterIfManager runs every Manager-level (see common.AiLevelManager) roster/business
// action for an unowned team: keep valuable players under contract, trim genuinely surplus
// depth, fill a thin position from the free-agent market, and charge what the gate will bear.
// A no-op below Manager level - see RunAITeamManagement, the only caller.
func manageRosterIfManager(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel < common.AiLevelManager {
		return
	}

	players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d roster management: %s", team.Id, err)
		return
	}

	renewExpiringContracts(dbCtx, team.Id, *players)
	releaseSurplusPlayer(dbCtx, team.Id, *players)
	signFreeAgentForWeakestPosition(dbCtx, team.Id, *players)
	maximizeTicketPrice(dbCtx, team.Id)
}

// renewExpiringContracts extends every squad player's contract once it's down to
// contractRenewalThresholdYears seasons or fewer remaining - unconditionally, since a Manager
// tier team always prefers keeping a player on the books over losing them to free agency for
// nothing. ErrPlayerOnLoan (the parent club's contract, not this team's to renew) and
// ErrContractAtMax are expected outcomes, not failures, and are silently skipped.
func renewExpiringContracts(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	for _, p := range players {
		if p.ContractYearsRemaining > contractRenewalThresholdYears {
			continue
		}
		if err := common.RenewPlayerContract(dbCtx, p.Id, teamId); err != nil &&
			!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrContractAtMax) {
			log.Errorf("error renewing contract for AI-managed team %d player %d: %s", teamId, p.Id, err)
		}
	}
}

// releaseSurplusPlayer frees up one roster spot (and its wage) when a core position is
// comfortably overstocked (see surplusDepthMargin), releasing the weakest player at the single
// most overstocked position - at most one release per call, so a squad's depth erodes gradually
// rather than all at once. Never shrinks the squad below common.SquadSize, the size every team
// starts with. ErrPlayerOnLoan and ErrInsufficientBudget (can't afford the severance right now)
// are expected outcomes, not failures.
func releaseSurplusPlayer(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	if len(players) <= common.SquadSize {
		return
	}

	overstockedPosition := -1
	maxSurplus := surplusDepthMargin
	for _, need := range common.AssessPositionNeeds(players) {
		if surplus := need.Depth - common.MinPositionDepth[need.Position]; surplus > maxSurplus {
			maxSurplus = surplus
			overstockedPosition = need.Position
		}
	}
	if overstockedPosition == -1 {
		return
	}

	var weakest *common.Player
	weakestValue := 0
	for _, p := range players {
		if common.BestCorePosition(*p) != overstockedPosition {
			continue
		}
		if v := common.PositionValue(overstockedPosition, *p); weakest == nil || v < weakestValue {
			weakest, weakestValue = p, v
		}
	}
	if weakest == nil {
		return
	}

	if err := common.ReleasePlayer(dbCtx, weakest.Id, teamId); err != nil &&
		!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrInsufficientBudget) {
		log.Errorf("error releasing surplus player %d for AI-managed team %d: %s", weakest.Id, teamId, err)
	}
}

// signFreeAgentForWeakestPosition signs the best available free agent for a team's single
// neediest core position (see common.WeakestPositionNeed), but only when that position is
// actually thin - a squad that merely has below-average depth everywhere shouldn't go shopping.
// ErrTransferWindowClosed and ErrPlayerNotFree (someone else signed them first) are expected
// outcomes, not failures.
func signFreeAgentForWeakestPosition(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	weakest := common.WeakestPositionNeed(players)
	if weakest == nil || !weakest.Thin {
		return
	}

	freeAgents, err := common.GetFreeAgents(dbCtx)
	if err != nil {
		log.Errorf("error fetching free agents for AI-managed team %d: %s", teamId, err)
		return
	}
	var best *common.Player
	bestValue := 0
	for _, p := range freeAgents {
		if common.BestCorePosition(*p) != weakest.Position {
			continue
		}
		if v := common.PositionValue(weakest.Position, *p); best == nil || v > bestValue {
			best, bestValue = p, v
		}
	}
	if best == nil {
		return
	}

	if err := common.SignFreeAgent(dbCtx, best.Id, teamId); err != nil &&
		!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPlayerNotFree) {
		log.Errorf("error signing free agent %d for AI-managed team %d: %s", best.Id, teamId, err)
	}
}

// maximizeTicketPrice charges the maximum allowed ticket price - see TicketRevenue's own doc
// comment: attendance in this game's economy never falls as price rises, so there's no tradeoff
// for a rational owner to weigh, and a Manager-level AI team should behave like one.
func maximizeTicketPrice(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching stadium for AI-managed team %d ticket pricing: %s", teamId, err)
		return
	}
	if stadium.TicketPrice >= common.MaxTicketPrice {
		return
	}
	if err := common.SetTicketPrice(dbCtx, teamId, common.MaxTicketPrice); err != nil {
		log.Errorf("error setting ticket price for AI-managed team %d: %s", teamId, err)
	}
}

// refreshAiTeamLineup re-picks and persists an unowned team's best-XI default lineup from its
// current squad - the Caretaker-level baseline every AI-managed team gets, every AI level and up.
// Runs every match rather than only at season rollover (see autoManageLineupIfUnowned in
// progression.go, which still covers the retirement/academy-intake refresh at season end) so a
// mid-season squad change - a Level 2/3 signing or release, once those phases land - shows up in
// the default lineup right away instead of waiting for the next season.
func refreshAiTeamLineup(dbCtx *common.DatabaseContext, teamId int64) {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d lineup refresh: %s", teamId, err)
		return
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, common.FlattenLineup(common.SelectBestLineup(*players))); err != nil {
		log.Errorf("error refreshing lineup for AI-managed team %d: %s", teamId, err)
	}
}
