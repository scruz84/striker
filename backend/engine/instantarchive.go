package engine

import (
	"context"
	"time"

	"github.com/reugn/go-quartz/quartz"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// instantArchiveJobName is the daily archive pass's slot in engine_job_runs (see
// openspec/specs/engine-replicas "Periodic jobs once per slot").
const instantArchiveJobName = "instant_archive"

// StartInstantArchiveScheduler runs the archive pass once a day at 04:41 UTC, clear of the other
// daily jobs. With the season archive off it only expires old friendlies' instants.
func StartInstantArchiveScheduler(job quartz.Job) {
	scheduleJob(job, "0 41 4 * * ?", instantArchiveJobName)
}

// InstantArchiveJob implements quartz.Job for the daily instant archive pass
// (openspec/specs/match-data-retention): one replica per day wins the slot, then deletes the
// instants of friendlies older than the days kept (common.ArchiveExpiredFriendlyInstants) and, with
// the season archive on, of every fixture older than the seasons kept (common.ArchiveOldInstants),
// batch by batch.
type InstantArchiveJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (j *InstantArchiveJob) Description() string { return j.desc }

func (j *InstantArchiveJob) Execute(context.Context) error {
	if !claimRun(j.dbCtx, instantArchiveJobName, 24*time.Hour) {
		return nil
	}
	RunInstantArchive(j.dbCtx)
	return nil
}

func NewInstantArchiveJob(dbCtx *common.DatabaseContext) *InstantArchiveJob {
	return &InstantArchiveJob{desc: "instant archive job", dbCtx: dbCtx}
}

// RunInstantArchive is one day's pass: friendlies older than the days kept lose their instants
// whatever the season archive setting, then, if the admin has the season archive on, every fixture
// older than the seasons kept.
func RunInstantArchive(dbCtx *common.DatabaseContext) {
	runFriendlyInstantExpiry(dbCtx)
	settings, err := common.GetInstantArchiveSettings(dbCtx)
	if err != nil {
		log.Errorf("error reading the instant archive setting: %s", err)
		return
	}
	if !settings.Enabled {
		return
	}
	started := time.Now()
	res, err := common.ArchiveOldInstants(dbCtx, settings.KeepSeasons)
	if err != nil {
		log.Errorf("instant archive stopped after %d league matches, %d cup ties and %d friendlies: %s",
			res.LeagueMatches, res.CupTies, res.Friendlies, err)
		return
	}
	log.Infof("instant archive (%d seasons kept): %d league matches, %d cup ties, %d friendlies, %d instant rows deleted in %s",
		settings.KeepSeasons, res.LeagueMatches, res.CupTies, res.Friendlies, res.Instants, time.Since(started).Round(time.Second))
}

// runFriendlyInstantExpiry deletes the instants of friendlies older than the days kept (none while
// that is 0).
func runFriendlyInstantExpiry(dbCtx *common.DatabaseContext) {
	keepDays, err := common.GetFriendlyInstantKeepDays(dbCtx)
	if err != nil {
		log.Errorf("error reading the friendly instant retention: %s", err)
		return
	}
	if keepDays == 0 {
		return
	}
	started := time.Now()
	res, err := common.ArchiveExpiredFriendlyInstants(dbCtx, keepDays)
	if err != nil {
		log.Errorf("friendly instant expiry stopped after %d friendlies: %s", res.Friendlies, err)
		return
	}
	log.Infof("friendly instant expiry (%d days kept): %d friendlies, %d instant rows deleted in %s",
		keepDays, res.Friendlies, res.Instants, time.Since(started).Round(time.Second))
}
