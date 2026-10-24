package engine

import (
	"math/rand"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// WorldCupFrequencySeasons is how often (in completed seasons) a new World Cup starts - real
// tournaments come around every four years, so this mirrors that against this game's own season
// counter rather than running one every single season the way the regional international cup
// does (see StartInternationalCupsForNewSeason).
const WorldCupFrequencySeasons = 4

// worldCupGroupMatchdayGap mirrors cupRoundGap's own spacing logic: a group's first matchday
// kicks off on the nearest Saturday (see common.NextLocalSaturdayKickoff), and each subsequent
// matchday a full week after that, so the group stage visibly plays out "once a week" the same
// way ordinary national-team friendlies do rather than all at once.
const worldCupGroupMatchdayGap = 7 * 24 * time.Hour

// StartWorldCupForNewSeason starts a fresh World Cup for season if it's actually time for one
// (see WorldCupFrequencySeasons), none is already running (see common.HasActiveWorldCup), and
// enough countries are active to form at least two groups (see common.MinWorldCupCountries).
// Every active country enters, including one that still has an ordinary weekly friendly pending
// from before the tournament started - common.CancelPendingNationalTeamFriendlies clears any such
// fixture first, so that country doesn't end up double-booked into both it and its first group
// match. Called from StartInternationalCupsIfDue, alongside StartInternationalCupsForNewSeason,
// on that job's own decoupled cadence rather than any one country's season rollover.
//
// Holds the national-team scheduling lock (see common.WithNationalTeamSchedulingLock) for the
// whole start, waiting for it rather than skipping: common.ScheduleNationalTeamFriendlies takes
// the same lock, so a friendlies pass can never read "no World Cup running" and then pair a
// country into a friendly the same week this creates its first group match.
func StartWorldCupForNewSeason(dbCtx *common.DatabaseContext, season int) {
	if season%WorldCupFrequencySeasons != 0 {
		return
	}
	if err := common.WithNationalTeamSchedulingLock(dbCtx, func() error {
		startWorldCupLocked(dbCtx, season)
		return nil
	}); err != nil {
		log.Errorf("error taking the national-team scheduling lock to start season %d's World Cup: %s", season, err)
	}
}

func startWorldCupLocked(dbCtx *common.DatabaseContext, season int) {
	hasActive, err := common.HasActiveWorldCup(dbCtx)
	if err != nil {
		log.Errorf("error checking for an active World Cup before starting season %d's: %s", season, err)
		return
	}
	if hasActive {
		return
	}

	active, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		log.Errorf("error listing active countries for season %d's World Cup: %s", season, err)
		return
	}
	if len(active) < common.MinWorldCupCountries {
		return
	}

	if err := common.CancelPendingNationalTeamFriendlies(dbCtx, active); err != nil {
		log.Errorf("error cancelling pending national-team friendlies before starting season %d's World Cup: %s", season, err)
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	groups := common.AssignWorldCupGroups(active, r)

	worldCupId, err := common.CreateWorldCup(dbCtx, season, "FIFA World Cup")
	if err != nil {
		log.Errorf("error creating World Cup for season %d: %s", season, err)
		return
	}
	log.Infof("World Cup %d started for season %d with %d countries in %d groups", worldCupId, season, len(active), len(groups))

	for i, countries := range groups {
		if err := startWorldCupGroup(dbCtx, worldCupId, groupLetter(i), countries); err != nil {
			log.Errorf("error starting World Cup %d group %s: %s", worldCupId, groupLetter(i), err)
		}
	}
}

// groupLetter names a World Cup group by index (0 -> "A", 1 -> "B", ...) - a realistic group
// count (at most a couple dozen active countries, so well under ten groups of four) never comes
// close to running out of letters.
func groupLetter(i int) string {
	return string(rune('A' + i))
}

// startWorldCupGroup creates one group, enters its countries' national teams, and schedules its
// full round-robin fixture list (see common.BuildRoundRobinRounds), one matchday per week -
// immediately setting both sides' lineups for every fixture (see common.SetNationalTeamLineup),
// same as a weekly national-team friendly.
func startWorldCupGroup(dbCtx *common.DatabaseContext, worldCupId int64, groupName string, countries []string) error {
	groupId, err := common.CreateWorldCupGroup(dbCtx, worldCupId, groupName)
	if err != nil {
		return err
	}

	teamIds := make([]int64, len(countries))
	for i, country := range countries {
		teamId, err := common.GetNationalTeamId(dbCtx, country)
		if err != nil {
			return err
		}
		teamIds[i] = teamId
		if err := common.AddWorldCupGroupTeam(dbCtx, groupId, teamId); err != nil {
			return err
		}
	}

	rounds := common.BuildRoundRobinRounds(len(teamIds))
	for roundIndex, pairs := range rounds {
		for _, pair := range pairs {
			homeTeamId, awayTeamId := teamIds[pair[0]], teamIds[pair[1]]
			homeCountry, awayCountry := countries[pair[0]], countries[pair[1]]
			// Local to the home side, same as an ordinary weekly friendly (see
			// common.ScheduleNationalTeamFriendlies) - each round lands on the Saturday
			// roundIndex weeks out, so a team only ever plays one group match per round.
			matchDate := common.NextLocalSaturdayKickoff(homeCountry, common.NationalTeamFriendlyKickoffHour, common.NationalTeamFriendlyKickoffMinute).
				Add(time.Duration(roundIndex) * worldCupGroupMatchdayGap)

			var matchId int64
			if err := dbCtx.Db.QueryRow(common.InsertNationalTeamFriendlyMatchStmtSql, homeTeamId, awayTeamId, matchDate).Scan(&matchId); err != nil {
				return err
			}
			if err := common.LinkWorldCupGroupMatch(dbCtx, groupId, roundIndex+1, matchId); err != nil {
				return err
			}
			if err := common.SetNationalTeamLineup(dbCtx, matchId, homeTeamId, homeCountry); err != nil {
				log.Errorf("error setting World Cup group lineup for %s (match %d): %s", homeCountry, matchId, err)
			}
			if err := common.SetNationalTeamLineup(dbCtx, matchId, awayTeamId, awayCountry); err != nil {
				log.Errorf("error setting World Cup group lineup for %s (match %d): %s", awayCountry, matchId, err)
			}
		}
	}
	return nil
}

// AdvanceWorldCups checks every active World Cup: one still in its group stage moves to the
// knockout stage once every group match has been played (see
// common.IsWorldCupGroupStageComplete/startWorldCupKnockout); one already in the knockout stage
// is marked completed once its underlying Cup (see common.CreateInternationalCup, region
// "world") itself is. Called every tick alongside AdvanceCups - see CupJob.
func AdvanceWorldCups(dbCtx *common.DatabaseContext) {
	ids, err := common.GetActiveWorldCupIds(dbCtx)
	if err != nil {
		log.Errorf("error listing active World Cups: %s", err)
		return
	}

	for _, worldCupId := range ids {
		if err := advanceWorldCup(dbCtx, worldCupId); err != nil {
			log.Errorf("error advancing World Cup %d: %s", worldCupId, err)
		}
	}
}

// advanceWorldCup is a check-then-act (is the group stage complete? then create the knockout cup)
// with no transaction of its own, so it runs under a session-level advisory lock keyed to
// worldCupId, the same shape as advanceCup: two --engine replicas ticking CupJob at the same
// instant could otherwise both see the group stage complete and both create a knockout cup,
// sending every qualifier into two brackets at once. The World Cup is re-read inside the lock, so
// the replica that runs second sees the knockout stage already started and does nothing.
func advanceWorldCup(dbCtx *common.DatabaseContext, worldCupId int64) error {
	_, err := common.TryWithSessionAdvisoryLock(dbCtx, common.NamespacedLockKey(common.LockNamespaceWorldCupAdvance, worldCupId), func() error {
		return advanceWorldCupLocked(dbCtx, worldCupId)
	})
	return err
}

func advanceWorldCupLocked(dbCtx *common.DatabaseContext, worldCupId int64) error {
	worldCup, err := common.GetWorldCupById(dbCtx, worldCupId)
	if err != nil {
		return err
	}

	if worldCup.Stage != "knockout" {
		complete, err := common.IsWorldCupGroupStageComplete(dbCtx, worldCupId)
		if err != nil || !complete {
			return err
		}
		return startWorldCupKnockout(dbCtx, worldCup)
	}

	if worldCup.KnockoutCupId == nil {
		return nil
	}
	cup, err := common.GetCup(dbCtx, *worldCup.KnockoutCupId)
	if err != nil || !cup.Completed || cup.ChampionTeamId == nil {
		return err
	}
	log.Infof("World Cup %d complete, champion is team %d", worldCupId, *cup.ChampionTeamId)
	return common.SetWorldCupCompleted(dbCtx, worldCupId, *cup.ChampionTeamId)
}

// startWorldCupKnockout seeds the knockout bracket from every group's top two (see
// common.GetWorldCupGroupQualifiers) as an ordinary Cup (region "world") - from here on,
// AdvanceCups/SimulateCupTies drive it exactly like any other cup, with createRoundTies' own
// national-team check (see setNationalTeamLineupForTieIfNeeded) covering every round's lineups.
// Qualifiers are shuffled without trying to keep a group's two qualifiers apart in round one the
// way a real World Cup's seeded bracket does - this game has no strength ranking to seed by in
// the first place, so a plain shuffle is no less fair than any other draw it could make.
func startWorldCupKnockout(dbCtx *common.DatabaseContext, worldCup *common.WorldCup) error {
	qualifiers, err := common.GetWorldCupGroupQualifiers(dbCtx, worldCup.WorldCupId)
	if err != nil {
		return err
	}
	if len(qualifiers) < 2 {
		log.Errorf("World Cup %d group stage complete but only %d qualifiers found", worldCup.WorldCupId, len(qualifiers))
		return nil
	}

	cupId, err := common.CreateInternationalCup(dbCtx, "world", worldCup.Season, worldCup.Name)
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pairs := BuildFirstRoundPairings(qualifiers, r, classificationPointsForSeeding(dbCtx, qualifiers))
	if err := createRoundTies(dbCtx, cupId, 1, pairs); err != nil {
		return err
	}

	log.Infof("World Cup %d knockout stage started as cup %d with %d qualifiers", worldCup.WorldCupId, cupId, len(qualifiers))
	return common.SetWorldCupKnockoutStarted(dbCtx, worldCup.WorldCupId, cupId)
}
