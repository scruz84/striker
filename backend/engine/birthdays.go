package engine

import (
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// BirthdayCycleDays is how many in-game days pass between a player's birthdays. At the default
// match schedule (one round per day - see common.DefaultMatchScheduleWeekdaysMask) a season runs
// well over BirthdayCycleDays days itself, so most players see one or two birthdays a season
// rather than the old fixed "once per season" cadence.
const BirthdayCycleDays = 20

// ProcessBirthdays ages by one year every player - rostered, free agent, or academy prospect
// alike - whose birth_day anchor has reached BirthdayCycleDays in the past, and rolls birth_day
// forward by exactly that many days per year applied, so it's correctly positioned for their
// next birthday. Looping over however many cycles have actually elapsed (rather than only ever
// applying one) keeps a player correctly aged even if this job missed several runs in a row -
// e.g. the server was down for a while.
//
// Deliberately independent of season rollover: ProgressPlayers and ProgressFreeAgents check
// retirement and drift skills against a player's current age, but no longer change age
// themselves - a season can span zero, one, or several birthdays depending on how many matchdays
// it runs, not always exactly one.
func ProcessBirthdays(dbCtx *common.DatabaseContext) {
	now := time.Now()
	players, err := common.GetPlayersDueBirthday(dbCtx, now.AddDate(0, 0, -BirthdayCycleDays))
	if err != nil {
		log.Errorf("error listing players due a birthday: %s", err)
		return
	}

	for _, player := range players {
		years := int(now.Sub(player.BirthDay).Hours() / 24 / BirthdayCycleDays)
		if years < 1 {
			continue
		}
		newAge := player.Age + years
		newBirthDay := player.BirthDay.AddDate(0, 0, years*BirthdayCycleDays)
		if err := common.UpdatePlayerBirthday(dbCtx, player.Id, newAge, newBirthDay); err != nil {
			log.Errorf("error applying birthday for player %d: %s", player.Id, err)
		}
	}
}
