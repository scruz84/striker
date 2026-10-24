package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/common/metrictest"
)

type fakeJob struct {
	err error
}

func (f fakeJob) Description() string { return "fake job" }

func (f fakeJob) Execute(context.Context) error {
	time.Sleep(5 * time.Millisecond)
	return f.err
}

// TestInstrumentedJob_RecordsRunDurationAndLastRun: one Execute adds one run under the job's
// label, records one duration and stamps last_run with a time that is now; the wrapped job's
// error is passed through.
func TestInstrumentedJob_RecordsRunDurationAndLastRun(t *testing.T) {
	metrictest.Install()
	job := common.EngineJobAttribute("fake_job")
	runsBefore := metrictest.Sum(t, "striker.engine.job.runs", job)
	durationsBefore := metrictest.HistogramCount(t, "striker.engine.job.duration", job)

	wrapped := instrumentedJob{inner: fakeJob{}, name: "fake_job"}
	if wrapped.Description() != "fake job" {
		t.Errorf("Description() = %q, want the wrapped job's", wrapped.Description())
	}
	if err := wrapped.Execute(context.Background()); err != nil {
		t.Fatalf("Execute: %s", err)
	}

	if got := metrictest.Sum(t, "striker.engine.job.runs", job) - runsBefore; got != 1 {
		t.Errorf("runs grew by %v, want 1", got)
	}
	if got := metrictest.HistogramCount(t, "striker.engine.job.duration", job) - durationsBefore; got != 1 {
		t.Errorf("durations grew by %d, want 1", got)
	}
	lastRun := metrictest.Sum(t, "striker.engine.job.last_run", job)
	if age := time.Since(time.Unix(int64(lastRun), 0)); age < 0 || age > time.Minute {
		t.Errorf("last_run is %s old, want a time that is now", age)
	}

	boom := errors.New("boom")
	if err := (instrumentedJob{inner: fakeJob{err: boom}, name: "fake_job"}).Execute(context.Background()); !errors.Is(err, boom) {
		t.Errorf("Execute error = %v, want the wrapped job's", err)
	}
}
