package engine

import (
	"math/rand"
	"sort"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// restructureTarget is the pyramid a country's leagues are reshaped into at rollover (see
// restructureIfNeeded) - the saved world structure. A variable so tests that lay out their own tiny
// pyramids can switch restructuring off (see TestMain); nil or empty means "never restructure".
var restructureTarget = func(dbCtx *common.DatabaseContext) []common.TierConfig {
	return common.GetWorldConfig(dbCtx)
}

// RankedTeam is one team as a restructure orders it: the tier it plays in after the season's
// moves, then its final position, points, goal difference and goals scored (see
// openspec/specs/league-pyramid "Restructure ordering"). A team with no final standing (e.g. one
// created since) has Position 0 and sorts after every ranked team of its tier.
type RankedTeam struct {
	TeamId      int64
	Tier        int
	Position    int
	Points      int
	GoalDiff    int
	GoalsScored int
}

// GroupSlot names one group of a target structure.
type GroupSlot struct {
	Tier  int
	Group int
}

// RestructurePlan is where every team goes, plus how many new CPU teams each short group needs.
type RestructurePlan struct {
	Placement map[int64]GroupSlot
	Missing   map[GroupSlot]int
}

func rankedBefore(a, b RankedTeam) bool {
	if a.Tier != b.Tier {
		return a.Tier < b.Tier
	}
	aRanked, bRanked := a.Position > 0, b.Position > 0
	if aRanked != bRanked {
		return aRanked
	}
	if a.Position != b.Position {
		return a.Position < b.Position
	}
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	if a.GoalDiff != b.GoalDiff {
		return a.GoalDiff > b.GoalDiff
	}
	if a.GoalsScored != b.GoalsScored {
		return a.GoalsScored > b.GoalsScored
	}
	return a.TeamId < b.TeamId
}

// snakeGroup is the 1-based group the k-th (0-based) team of a tier with groups groups is dealt
// to: 1..G, then G..1, and so on.
func snakeGroup(k, groups int) int {
	round, idx := k/groups, k%groups
	if round%2 == 0 {
		return idx + 1
	}
	return groups - idx
}

// PlanRestructure reshapes teams into target (see openspec/specs/league-pyramid "Restructure
// ordering" and "Restructure fills groups and tops up"): it orders every team, cuts the list into
// tiers top down by groups x teams per group, deals each tier's teams to its groups in snake
// order, and counts the places left empty, which are all in the bottom tier when the structure
// has more places than teams. Teams beyond the structure's places (never the case after
// ValidateWorldConfigFitsTeams) are dealt into the bottom tier's groups anyway rather than
// dropped. Pure.
func PlanRestructure(teams []RankedTeam, target []common.TierConfig) RestructurePlan {
	tiers := append([]common.TierConfig(nil), target...)
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Tier < tiers[j].Tier })
	ordered := append([]RankedTeam(nil), teams...)
	sort.SliceStable(ordered, func(i, j int) bool { return rankedBefore(ordered[i], ordered[j]) })

	plan := RestructurePlan{Placement: make(map[int64]GroupSlot, len(ordered)), Missing: make(map[GroupSlot]int)}
	if len(tiers) == 0 {
		return plan
	}
	next := 0
	for i, tc := range tiers {
		places := tc.Groups * tc.TeamsPerLeague
		end := next + places
		if i == len(tiers)-1 || end > len(ordered) {
			// The bottom tier takes everyone left, and a short list ends early.
			end = len(ordered)
		}
		filled := make(map[int]int, tc.Groups)
		for k := next; k < end; k++ {
			group := snakeGroup(k-next, tc.Groups)
			plan.Placement[ordered[k].TeamId] = GroupSlot{Tier: tc.Tier, Group: group}
			filled[group]++
		}
		for g := 1; g <= tc.Groups; g++ {
			if short := tc.TeamsPerLeague - filled[g]; short > 0 {
				plan.Missing[GroupSlot{Tier: tc.Tier, Group: g}] = short
			}
		}
		next = end
	}
	return plan
}

// restructureIfNeeded reshapes country's active leagues into the saved world structure when they
// differ from it (see openspec/specs/league-pyramid "Restructure detection"), right after the
// season's promotion/relegation moves and before the new season's cup and fixtures. It runs under
// the country's rollover lock. finalStandings are the season's final tables (before any move); the
// season's moves are already applied to the teams it reads.
//
// It returns the moves the new season's starting order should be built from (see
// ComputeSeasonSeeds), relative to the league each team finished the season in: every team whose
// final league differs from it, promoted or relegated by tier, lateral when it only changed group.
// It returns nil when the country was not restructured, so the caller keeps seasonMoves.
func restructureIfNeeded(dbCtx *common.DatabaseContext, country string, season int, finalStandings []LeagueStandings) []LeagueMove {
	target := restructureTarget(dbCtx)
	if len(target) == 0 {
		return nil
	}
	if err := common.ValidateWorldConfig(target); err != nil {
		log.Errorf("not restructuring %s: the saved world structure is invalid: %s", country, err)
		return nil
	}
	shape, err := common.GetCountryPyramidShape(dbCtx, country)
	if err != nil {
		log.Errorf("error reading %s's league pyramid before a restructure: %s", country, err)
		return nil
	}
	if shape.MatchesConfig(target) {
		return nil
	}
	log.Infof("restructuring %s's leagues into %d tiers (%d places)", country, len(target), common.WorldConfigPlaces(target))
	return applyRestructure(dbCtx, country, season, shape, target, finalStandings)
}

func applyRestructure(dbCtx *common.DatabaseContext, country string, season int, shape common.PyramidShape, target []common.TierConfig,
	finalStandings []LeagueStandings) []LeagueMove {
	tierOfLeague := make(map[int64]int, len(shape.Leagues))
	nameOfLeague := make(map[int64]string, len(shape.Leagues))
	leagueOfSlot := make(map[GroupSlot]int64)
	for _, l := range shape.Leagues {
		tierOfLeague[l.LeagueId] = l.Tier
		nameOfLeague[l.LeagueId] = l.Name
		slot := GroupSlot{Tier: l.Tier, Group: l.GroupNumber}
		if _, taken := leagueOfSlot[slot]; !taken {
			leagueOfSlot[slot] = l.LeagueId
		}
	}

	// Each team's final standing and the league it finished the season in.
	type finish struct {
		position int
		standing common.TeamStanding
		leagueId int64
	}
	finishes := make(map[int64]finish)
	for _, league := range finalStandings {
		for i, st := range league.Standings {
			finishes[st.TeamId] = finish{position: i + 1, standing: st, leagueId: league.LeagueId}
		}
	}

	// Every team as it stands now, after the season's moves.
	currentLeague := make(map[int64]int64)
	teams := make([]RankedTeam, 0)
	for _, l := range shape.Leagues {
		for _, teamId := range common.GetDivisionTeams(dbCtx, l.LeagueId) {
			currentLeague[teamId] = l.LeagueId
			rt := RankedTeam{TeamId: teamId, Tier: l.Tier}
			if f, ok := finishes[teamId]; ok {
				rt.Position, rt.Points, rt.GoalDiff, rt.GoalsScored = f.position, f.standing.Points, f.standing.GoalDiff, f.standing.GoalsScored
			}
			teams = append(teams, rt)
		}
	}
	plan := PlanRestructure(teams, target)

	// Reuse every league whose tier and group still exist; create the rest.
	tierName := make(map[int]string, len(target))
	for _, tc := range target {
		tierName[tc.Tier] = tc.Name
		for g := 1; g <= tc.Groups; g++ {
			slot := GroupSlot{Tier: tc.Tier, Group: g}
			if _, ok := leagueOfSlot[slot]; ok {
				continue
			}
			leagueId, err := common.CreateLeague(dbCtx, country, tc.Name, tc.Tier, g)
			if err != nil {
				log.Errorf("error creating league %s for %s's restructure: %s", common.LeagueName(tc.Name, g), country, err)
				continue
			}
			leagueOfSlot[slot] = leagueId
			tierOfLeague[leagueId] = tc.Tier
			nameOfLeague[leagueId] = common.LeagueName(tc.Name, g)
		}
	}

	finalLeague := make(map[int64]int64, len(currentLeague))
	for teamId, from := range currentLeague {
		finalLeague[teamId] = from
		slot, ok := plan.Placement[teamId]
		if !ok {
			continue
		}
		to, ok := leagueOfSlot[slot]
		if !ok || to == from {
			continue
		}
		if err := common.UpdateTeamLeague(dbCtx, teamId, to); err != nil {
			log.Errorf("error moving team %d to league %d in %s's restructure: %s", teamId, to, country, err)
			continue
		}
		if err := common.UpdateClassificationLeague(dbCtx, teamId, to); err != nil {
			log.Errorf("error repointing classification for team %d to league %d: %s", teamId, to, err)
		}
		finalLeague[teamId] = to
		if tierOfLeague[to] != tierOfLeague[from] {
			notifyRestructureMoved(dbCtx, teamId, nameOfLeague[to], tierOfLeague[to] < tierOfLeague[from])
			redrawAiLevelAfterMove(dbCtx, teamId, to, rand.New(rand.NewSource(time.Now().UnixNano()+teamId)))
		}
	}

	// Top the short groups up with new CPU teams.
	created := 0
	for slot, missing := range plan.Missing {
		leagueId, ok := leagueOfSlot[slot]
		if !ok {
			continue
		}
		for i := 0; i < missing; i++ {
			teamId, err := common.CreateTeam(dbCtx, country, leagueId, slot.Tier)
			if err != nil {
				log.Errorf("error creating a CPU team in league %d for %s's restructure: %s", leagueId, country, err)
				continue
			}
			common.InsertTeamsToClassification(dbCtx, leagueId, season, []int64{teamId})
			created++
		}
	}
	if created > 0 {
		log.Infof("restructure of %s created %d CPU teams", country, created)
	}

	// Archive every league the structure no longer has.
	targetLeagues := make(map[int64]bool, len(leagueOfSlot))
	for _, tc := range target {
		for g := 1; g <= tc.Groups; g++ {
			if id, ok := leagueOfSlot[GroupSlot{Tier: tc.Tier, Group: g}]; ok {
				targetLeagues[id] = true
			}
		}
	}
	for _, l := range shape.Leagues {
		if targetLeagues[l.LeagueId] {
			continue
		}
		if err := common.ArchiveLeagueIfEmpty(dbCtx, l.LeagueId, season); err != nil {
			log.Errorf("error archiving league %d after %s's restructure: %s", l.LeagueId, country, err)
		}
	}

	// The starting order's moves, relative to where each team finished the season.
	moves := make([]LeagueMove, 0)
	for teamId, to := range finalLeague {
		f, ok := finishes[teamId]
		if !ok || f.leagueId == to {
			continue
		}
		fromTier, toTier := tierOfLeague[f.leagueId], tierOfLeague[to]
		moves = append(moves, LeagueMove{
			TeamId:      teamId,
			NewLeagueId: to,
			Promoted:    toTier < fromTier,
			Lateral:     toTier == fromTier,
		})
	}
	sort.Slice(moves, func(i, j int) bool { return moves[i].TeamId < moves[j].TeamId })
	return moves
}

// notifyRestructureMoved tells teamId's owner which league it plays in next season because of a
// restructure (see openspec/specs/league-pyramid "Restructure notifications").
func notifyRestructureMoved(dbCtx *common.DatabaseContext, teamId int64, leagueName string, up bool) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error getting team %d for its restructure notification: %s", teamId, err)
		return
	}
	notificationType := common.NOTIFICATION_RELEGATION
	if up {
		notificationType = common.NOTIFICATION_PROMOTION
	}
	msg := common.NotificationMessage{
		Text:   team.TeamName + " will play next season in " + leagueName + " because of the league restructure.",
		Key:    "notifications.messages.restructureMoved",
		Params: map[string]any{"team": team.TeamName, "league": leagueName},
	}
	if err := common.NotifyTeamOwner(dbCtx, teamId, notificationType, msg); err != nil {
		log.Errorf("error notifying team %d of its restructure move: %s", teamId, err)
	}
}
