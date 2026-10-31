package engine

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/reugn/go-quartz/quartz"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// Account lifecycle jobs (OpenSpec user-account-lifecycle): a daily pass that warns, releases the
// clubs of inactive owners and erases long-idle accounts, and a 15-minute pass that resets
// released clubs into fresh CPU clubs. The release is cheap and happens in the daily pass; the
// squad swap waits for the reset pass so it never runs under a match being simulated.

const accountLifecycleJobName = "account_lifecycle_daily"

func daysAgo(now time.Time, days int) time.Time { return now.AddDate(0, 0, -days) }

// daysLeft is how many whole days remain until `limit` days of inactivity are reached.
func daysLeft(now, lastSeen time.Time, limit int) int {
	left := limit - int(now.Sub(lastSeen).Hours()/24)
	if left < 1 {
		left = 1
	}
	return left
}

// RunAccountLifecycle is the daily pass, in order: club warnings (the 7-day one before the 14-day
// one so a user idle past both only hears the latest), releases, erasure warnings, erasures.
// Every step re-checks or atomically claims its row, so a re-run or a second replica is harmless.
func RunAccountLifecycle(dbCtx *common.DatabaseContext, now time.Time) {
	warnClubOwners(dbCtx, now)
	releaseInactiveOwners(dbCtx, now)
	warnAccountErasure(dbCtx, now)
	eraseIdleAccounts(dbCtx, now)
}

func warnClubOwners(dbCtx *common.DatabaseContext, now time.Time) {
	stages := []struct{ stage, days int }{
		{common.WarningStageClubSecond, common.InactivityWarningDays[1]},
		{common.WarningStageClubFirst, common.InactivityWarningDays[0]},
	}
	for _, st := range stages {
		cutoff := daysAgo(now, st.days)
		owners, err := common.GetOwnersDueWarning(dbCtx, cutoff, st.stage)
		if err != nil {
			log.Errorf("error listing owners due inactivity warning stage %d: %s", st.stage, err)
			continue
		}
		for _, o := range owners {
			claimed, err := common.ClaimWarningStage(dbCtx, o.UserId, st.stage, cutoff)
			if err != nil {
				log.Errorf("error claiming warning stage %d for user %d: %s", st.stage, o.UserId, err)
				continue
			}
			if !claimed {
				continue
			}
			msg := common.NotificationMessage{
				Text: "You have not played for a long time: " + o.TeamName + " will be released in " +
					strconv.Itoa(daysLeft(now, o.LastSeenAt, common.InactivityReleaseDays)) + " days unless you come back",
				Key: "notifications.account.releaseWarning",
				Params: map[string]any{
					"days": daysLeft(now, o.LastSeenAt, common.InactivityReleaseDays),
					"team": o.TeamName,
				},
			}
			if err := common.NotifyUser(dbCtx, o.UserId, common.NOTIFICATION_ACCOUNT, msg); err != nil {
				log.Errorf("error sending inactivity warning to user %d: %s", o.UserId, err)
			}
		}
	}
}

func releaseInactiveOwners(dbCtx *common.DatabaseContext, now time.Time) {
	cutoff := daysAgo(now, common.InactivityReleaseDays)
	owners, err := common.GetOwnersDueRelease(dbCtx, cutoff)
	if err != nil {
		log.Errorf("error listing owners due club release: %s", err)
		return
	}
	for _, userId := range owners {
		teamId, err := common.ReleaseInactiveOwner(dbCtx, userId, cutoff)
		if err != nil {
			log.Errorf("error releasing the club of inactive user %d: %s", userId, err)
			continue
		}
		if teamId != 0 {
			InvalidateTeamCaches(dbCtx, teamId)
			log.Infof("released club %d of user %d after %d days of inactivity", teamId, userId, common.InactivityReleaseDays)
		}
	}
}

func warnAccountErasure(dbCtx *common.DatabaseContext, now time.Time) {
	cutoff := daysAgo(now, common.AccountErasureWarningDays)
	accounts, err := common.GetAccountsDueErasureWarning(dbCtx, cutoff)
	if err != nil {
		log.Errorf("error listing accounts due an erasure warning: %s", err)
		return
	}
	for _, a := range accounts {
		claimed, err := common.ClaimWarningStage(dbCtx, a.UserId, common.WarningStageErasure, cutoff)
		if err != nil {
			log.Errorf("error claiming erasure warning for user %d: %s", a.UserId, err)
			continue
		}
		if !claimed {
			continue
		}
		days := daysLeft(now, a.LastSeenAt, common.AccountErasureDays)
		msg := common.NotificationMessage{
			Text:   "Your account will be erased automatically in " + strconv.Itoa(days) + " days because of inactivity - sign in to keep it",
			Key:    "notifications.account.erasureWarning",
			Params: map[string]any{"days": days},
		}
		if err := common.NotifyUser(dbCtx, a.UserId, common.NOTIFICATION_ACCOUNT, msg); err != nil {
			log.Errorf("error sending erasure warning to user %d: %s", a.UserId, err)
		}
	}
}

func eraseIdleAccounts(dbCtx *common.DatabaseContext, now time.Time) {
	cutoff := daysAgo(now, common.AccountErasureDays)
	accounts, err := common.GetAccountsDueErasure(dbCtx, cutoff)
	if err != nil {
		log.Errorf("error listing accounts due erasure: %s", err)
		return
	}
	for _, userId := range accounts {
		teamId, err := common.DeleteAccountIfIdle(dbCtx, userId, common.ReleaseReasonErasure, cutoff)
		if errors.Is(err, common.ErrUserActive) || errors.Is(err, common.ErrUserNotFound) {
			continue
		}
		if err != nil {
			log.Errorf("error erasing idle account %d: %s", userId, err)
			continue
		}
		if teamId != 0 {
			InvalidateTeamCaches(dbCtx, teamId)
		}
		log.Infof("erased account %d after %d days of inactivity", userId, common.AccountErasureDays)
	}
}

// ResetReleasedTeams is the 15-minute pass: it resets every club waiting in the queue
// (teams.reset_pending_since). Several replicas may run it at once - ResetReleasedTeam locks the
// club and re-checks it, so each club is reset exactly once.
func ResetReleasedTeams(dbCtx *common.DatabaseContext) {
	teamIds, err := common.GetPendingResetTeams(dbCtx)
	if err != nil {
		log.Errorf("error listing clubs pending reset: %s", err)
		return
	}
	for _, teamId := range teamIds {
		reset, returned, err := common.ResetReleasedTeam(dbCtx, teamId)
		if err != nil {
			log.Errorf("error resetting released club %d: %s", teamId, err)
			continue
		}
		if !reset {
			continue
		}
		InvalidateTeamCaches(dbCtx, teamId)
		for _, loan := range returned {
			notifyLoanReturn(dbCtx, loan)
		}
		log.Infof("released club %d reset into a fresh CPU club", teamId)
	}
}

// AccountLifecycleJob implements quartz.Job for the daily pass.
type AccountLifecycleJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (j *AccountLifecycleJob) Description() string { return j.desc }

func (j *AccountLifecycleJob) Execute(context.Context) error {
	if !claimRun(j.dbCtx, accountLifecycleJobName, 24*time.Hour) {
		return nil
	}
	log.Info("Account lifecycle job")
	RunAccountLifecycle(j.dbCtx, time.Now())
	return nil
}

func NewAccountLifecycleJob(dbCtx *common.DatabaseContext) *AccountLifecycleJob {
	return &AccountLifecycleJob{desc: "account lifecycle job", dbCtx: dbCtx}
}

// ResetReleasedTeamsJob implements quartz.Job for the 15-minute reset pass. No job-slot claim:
// per-club safety comes from ResetReleasedTeam's lock.
type ResetReleasedTeamsJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (j *ResetReleasedTeamsJob) Description() string { return j.desc }

func (j *ResetReleasedTeamsJob) Execute(context.Context) error {
	ResetReleasedTeams(j.dbCtx)
	return nil
}

func NewResetReleasedTeamsJob(dbCtx *common.DatabaseContext) *ResetReleasedTeamsJob {
	return &ResetReleasedTeamsJob{desc: "reset released teams job", dbCtx: dbCtx}
}

var _ quartz.Job = (*AccountLifecycleJob)(nil)
var _ quartz.Job = (*ResetReleasedTeamsJob)(nil)
