package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// maybeCompleteBestXi picks the team of the week of a league match's matchday once that matchday
// is complete - every match played and past its replay (see common.IsLeagueWeekComplete) - and
// tells the owners of the clubs whose players made it (openspec/specs/best-xi). Called after each
// league match's post-match effects, so the last of the matchday triggers it; replicas finishing
// matches of the same matchday at once store it once (see common.SaveBestXi).
func maybeCompleteBestXi(dbCtx *common.DatabaseContext, matchId int64) {
	leagueId, season, week, err := common.GetLeagueWeekOfMatch(dbCtx, matchId)
	if err != nil {
		log.Errorf("error reading the matchday of match %d for its team of the week: %s", matchId, err)
		return
	}
	if week <= common.BestXiSeasonWeek {
		return
	}
	complete, err := common.IsLeagueWeekComplete(dbCtx, leagueId, season, week)
	if err != nil {
		log.Errorf("error checking whether matchday %d of league %d is complete: %s", week, leagueId, err)
		return
	}
	if !complete {
		return
	}
	candidates, err := common.GetWeekBestXiCandidates(dbCtx, leagueId, season, week)
	if err != nil {
		log.Errorf("error reading the team of the week candidates of league %d matchday %d: %s", leagueId, week, err)
		return
	}
	storeBestXi(dbCtx, leagueId, season, week, candidates)
}

// awardSeasonBestXi picks a league's team of the season at its rollover, alongside the season
// awards.
func awardSeasonBestXi(dbCtx *common.DatabaseContext, leagueId int64, season int) {
	candidates, err := common.GetSeasonBestXiCandidates(dbCtx, leagueId, season)
	if err != nil {
		log.Errorf("error reading the team of the season candidates of league %d season %d: %s", leagueId, season, err)
		return
	}
	storeBestXi(dbCtx, leagueId, season, common.BestXiSeasonWeek, candidates)
}

func storeBestXi(dbCtx *common.DatabaseContext, leagueId int64, season int, week int, candidates []common.BestXiCandidate) {
	picks := common.SelectBestXi(candidates)
	if len(picks) == 0 {
		return
	}
	stored, err := common.SaveBestXi(dbCtx, leagueId, season, week, picks)
	if err != nil {
		log.Errorf("error storing the best XI of league %d season %d week %d: %s", leagueId, season, week, err)
		return
	}
	if stored {
		notifyBestXi(dbCtx, leagueId, season, week, picks)
	}
}

// notifyBestXi tells each club's owner, once per best XI, which of his players made it.
func notifyBestXi(dbCtx *common.DatabaseContext, leagueId int64, season int, week int, picks []common.BestXiPick) {
	namesByTeam := map[int64][]string{}
	for _, p := range picks {
		player, err := common.GetPlayerById(dbCtx, p.PlayerId)
		if err != nil {
			continue
		}
		namesByTeam[p.TeamId] = append(namesByTeam[p.TeamId], player.FirstName+" "+player.LastName)
	}
	teamIds := make([]int64, 0, len(namesByTeam))
	for teamId := range namesByTeam {
		teamIds = append(teamIds, teamId)
	}
	sort.Slice(teamIds, func(i, j int) bool { return teamIds[i] < teamIds[j] })
	for _, teamId := range teamIds {
		names := strings.Join(namesByTeam[teamId], ", ")
		var msg common.NotificationMessage
		var link string
		if week == common.BestXiSeasonWeek {
			msg = common.NotificationMessage{
				Text:   fmt.Sprintf("Team of the season %d: %s", season, names),
				Key:    "notifications.messages.teamOfTheSeason",
				Params: map[string]any{"season": season, "names": names, "count": len(namesByTeam[teamId])},
			}
			link = fmt.Sprintf("/division/%d?season=%d&xi=0", leagueId, season)
		} else {
			msg = common.NotificationMessage{
				Text:   fmt.Sprintf("Team of the week (matchday %d): %s", week, names),
				Key:    "notifications.messages.teamOfTheWeek",
				Params: map[string]any{"week": week, "names": names, "count": len(namesByTeam[teamId])},
			}
			link = fmt.Sprintf("/division/%d?xi=%d", leagueId, week)
		}
		if err := common.NotifyTeamOwnerWithLink(dbCtx, teamId, common.NOTIFICATION_AWARD, msg, link); err != nil {
			log.Errorf("error notifying team %d of its best XI players: %s", teamId, err)
		}
	}
}
