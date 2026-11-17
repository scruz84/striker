package engine

import (
	"math/rand"
	"sort"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// restructureTarget is the pyramid a country's leagues are reshaped into at rollover (see
// restructureIfNeeded) - its effective structure: its own, or else the global one. A variable so
// tests that lay out their own tiny pyramids can switch restructuring off (see TestMain); nil or
// empty means "never restructure".
var restructureTarget = func(dbCtx *common.DatabaseContext, country string) []common.TierConfig {
	return common.EffectiveStructure(dbCtx, country)
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
	// Protected is a team the restructure never dissolves: a user's club, or one still due to
	// play a cup tie (see openspec/specs/league-pyramid "Surplus CPU teams are dissolved").
	Protected bool
}

// GroupSlot names one group of a target structure.
type GroupSlot struct {
	Tier  int
	Group int
}

// RestructurePlan is where every team goes, how many new CPU teams each short group needs, and
// which teams are dissolved because the structure has fewer places than teams.
type RestructurePlan struct {
	Placement map[int64]GroupSlot
	Missing   map[GroupSlot]int
	Dissolve  []int64
}

// cutToPlaces keeps the first places teams of ordered and returns them with the rest, which are
// dissolved. A protected team beyond the cut takes the last place instead, and the lowest-ranked
// unprotected team inside the cut drops out in its place; one that finds no unprotected team to
// swap with is kept anyway (the bottom tier then takes it over its size). Pure.
func cutToPlaces(ordered []RankedTeam, places int) ([]RankedTeam, []RankedTeam) {
	if len(ordered) <= places {
		return ordered, nil
	}
	keep := append([]RankedTeam(nil), ordered[:places]...)
	var dropped []RankedTeam
	for _, team := range ordered[places:] {
		if !team.Protected {
			dropped = append(dropped, team)
			continue
		}
		swap := -1
		for i := len(keep) - 1; i >= 0; i-- {
			if !keep[i].Protected {
				swap = i
				break
			}
		}
		if swap >= 0 {
			dropped = append(dropped, keep[swap])
			keep = append(keep[:swap], keep[swap+1:]...)
		}
		keep = append(keep, team)
	}
	return keep, dropped
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
// has more places than teams. When it has fewer places than teams, the teams beyond the last
// place are dissolved, never placed (see cutToPlaces). Pure.
func PlanRestructure(teams []RankedTeam, target []common.TierConfig) RestructurePlan {
	tiers := append([]common.TierConfig(nil), target...)
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Tier < tiers[j].Tier })
	ordered := append([]RankedTeam(nil), teams...)
	sort.SliceStable(ordered, func(i, j int) bool { return rankedBefore(ordered[i], ordered[j]) })

	plan := RestructurePlan{Placement: make(map[int64]GroupSlot, len(ordered)), Missing: make(map[GroupSlot]int)}
	if len(tiers) == 0 {
		return plan
	}
	ordered, dropped := cutToPlaces(ordered, common.WorldConfigPlaces(tiers))
	for _, team := range dropped {
		plan.Dissolve = append(plan.Dissolve, team.TeamId)
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
	target := restructureTarget(dbCtx, country)
	if len(target) == 0 {
		return nil
	}
	if err := common.ValidateWorldConfig(target); err != nil {
		log.Errorf("not restructuring %s: its structure is invalid: %s", country, err)
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
	protected, err := common.GetProtectedTeams(dbCtx, teamIdsOf(teams))
	if err != nil {
		// Without knowing which clubs are users' or still in a cup, dissolving any is unsafe.
		log.Errorf("not restructuring %s: error reading its protected teams: %s", country, err)
		return nil
	}
	for i := range teams {
		teams[i].Protected = protected[teams[i].TeamId]
	}
	plan := PlanRestructure(teams, target)
	// Dissolve the surplus first, so a rollover resumed after a crash finds them out of the
	// leagues already and plans the same result for the rest.
	dissolved := 0
	for _, teamId := range plan.Dissolve {
		ok, returned, err := common.DissolveTeam(dbCtx, teamId, season)
		if err != nil {
			log.Errorf("error dissolving team %d in %s's restructure: %s", teamId, country, err)
			continue
		}
		if ok {
			dissolved++
			delete(currentLeague, teamId)
			InvalidateTeamCaches(dbCtx, teamId)
		}
		for _, loan := range returned {
			notifyLoanReturn(dbCtx, loan)
		}
	}
	if dissolved > 0 {
		log.Infof("restructure of %s dissolved %d CPU teams", country, dissolved)
	}

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

func teamIdsOf(teams []RankedTeam) []int64 {
	ids := make([]int64, len(teams))
	for i, t := range teams {
		ids[i] = t.TeamId
	}
	return ids
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
