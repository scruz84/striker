package engine

import (
	"strconv"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// awardSeasonHonors notifies leagueId's Golden Boot (top scorer) and Player of the Season
// winners' team owners for the season just finished - called from applyPromotionRelegation,
// right alongside that season's prize money and season-ticket payouts, while the classification
// and match_events this reads are still the just-finished season's (classification is reset in
// place right after - see ResetClassificationForSeason - but match_events, keyed by season, are
// never touched by rollover). Either award is skipped (not everyone wins every season) if nobody
// qualifies - see common.GetTopScorers/GetPlayerOfTheSeason.
func awardSeasonHonors(dbCtx *common.DatabaseContext, leagueId int64, season int) {
	scorers, err := common.GetTopScorers(dbCtx, leagueId, season, 1)
	if err != nil {
		log.Errorf("error getting golden boot candidate for league %d season %d: %s", leagueId, season, err)
	} else if len(scorers) > 0 && scorers[0].Goals > 0 {
		notifyGoldenBoot(dbCtx, scorers[0])
	}

	playerOfSeason, err := common.GetPlayerOfTheSeason(dbCtx, leagueId, season)
	if err != nil {
		log.Errorf("error getting player of the season for league %d season %d: %s", leagueId, season, err)
	} else if playerOfSeason != nil {
		notifyPlayerOfTheSeason(dbCtx, *playerOfSeason)
	}
}

func notifyGoldenBoot(dbCtx *common.DatabaseContext, scorer common.ScorerStanding) {
	name := scorer.FirstName + " " + scorer.LastName
	msg := common.NotificationMessage{
		Text:   name + " won the Golden Boot with " + strconv.Itoa(scorer.Goals) + " goals!",
		Key:    "notifications.messages.goldenBoot",
		Params: map[string]any{"name": name, "goals": scorer.Goals},
	}
	if err := common.NotifyTeamOwnerWithLink(dbCtx, scorer.TeamId, common.NOTIFICATION_AWARD, msg, "/player/"+strconv.FormatInt(scorer.PlayerId, 10)); err != nil {
		log.Errorf("error notifying team %d of golden boot award: %s", scorer.TeamId, err)
	}
}

func notifyPlayerOfTheSeason(dbCtx *common.DatabaseContext, award common.SeasonAward) {
	name := award.FirstName + " " + award.LastName
	msg := common.NotificationMessage{
		Text:   name + " was named Player of the Season!",
		Key:    "notifications.messages.playerOfSeason",
		Params: map[string]any{"name": name},
	}
	if err := common.NotifyTeamOwnerWithLink(dbCtx, award.TeamId, common.NOTIFICATION_AWARD, msg, "/player/"+strconv.FormatInt(award.PlayerId, 10)); err != nil {
		log.Errorf("error notifying team %d of player of the season award: %s", award.TeamId, err)
	}
}
