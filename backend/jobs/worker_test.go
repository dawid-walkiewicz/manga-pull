package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"main/db"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

func TestMain(m *testing.M) {
	goose.SetLogger(goose.NopLogger())
	os.Exit(m.Run())
}

type fakeExecutor func(ctx context.Context, job *db.Job) error

func (f fakeExecutor) Execute(ctx context.Context, job *db.Job) error {
	return f(ctx, job)
}

type fakeJobStore struct {
	job               *db.Job
	claimErr          error
	updateErrByStatus map[string]error
	updates           []db.JobStatusUpdate
}

func (s *fakeJobStore) ClaimNextJob(context.Context) (*db.Job, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	job := *s.job
	return &job, nil
}

func (s *fakeJobStore) UpdateJobStatus(_ context.Context, status db.JobStatusUpdate) (*db.Job, error) {
	s.updates = append(s.updates, status)
	if err := s.updateErrByStatus[status.Status]; err != nil {
		return nil, err
	}

	s.job.Status = status.Status
	if status.Attempt != nil {
		s.job.Attempt = *status.Attempt
	}
	if status.Progress != nil {
		s.job.Progress = *status.Progress
	}
	s.job.ErrorMessage = status.ErrorMessage
	s.job.StartedAt = status.StartedAt
	s.job.FinishedAt = status.FinishedAt

	job := *s.job
	return &job, nil
}

func (s *fakeJobStore) GetSavedTitle(context.Context, int64) (db.SavedTitle, error) {
	panic("GetSavedTitle should not be called by Worker tests")
}

func (s *fakeJobStore) CreateJob(context.Context, db.Job) (int64, error) {
	panic("CreateJob should not be called by Worker tests")
}

func newFakeJobStore(job db.Job) *fakeJobStore {
	return &fakeJobStore{
		job:               &job,
		updateErrByStatus: make(map[string]error),
	}
}

func newTestWorker(t *testing.T, executor JobExecutor) (*Worker, *db.Store, *sqlx.DB) {
	t.Helper()
	if executor == nil {
		executor = fakeExecutor(func(context.Context, *db.Job) error { return nil })
	}

	database, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	store := db.NewStore(database)
	return NewWorker(store, executor), store, database
}

func insertJob(t *testing.T, database *sqlx.DB, jobType, status string, attempt int, createdAt time.Time) int64 {
	t.Helper()

	result, err := database.ExecContext(context.Background(), `
		INSERT INTO jobs (job_type, status, attempt, created_at)
		VALUES (?, ?, ?, ?)
	`, jobType, status, attempt, createdAt)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read inserted job id: %v", err)
	}

	return id
}

func getJob(t *testing.T, store *db.Store, id int64) db.Job {
	t.Helper()

	job, err := store.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("get job %d: %v", id, err)
	}
	return job
}

func assertStatusSequence(t *testing.T, updates []db.JobStatusUpdate, want ...string) {
	t.Helper()
	if len(updates) != len(want) {
		t.Fatalf("status update count = %d, want %d; updates = %+v", len(updates), len(want), updates)
	}
	for i := range want {
		if updates[i].Status != want[i] {
			t.Fatalf("status update %d = %q, want %q; updates = %+v", i, updates[i].Status, want[i], updates)
		}
	}
}

func TestClaimNextJobClaimsOldestQueuedJobAndSkipsNonQueuedJobs(t *testing.T) {
	_, store, database := newTestWorker(t, nil)
	now := time.Now().UTC()

	insertJob(t, database, string(JobRefreshTitle), string(JobRunning), 0, now.Add(-3*time.Minute))
	newerQueuedID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, now.Add(-1*time.Minute))
	oldestQueuedID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, now.Add(-2*time.Minute))

	claimed, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim next job: %v", err)
	}

	if claimed.ID != oldestQueuedID {
		t.Fatalf("claimed job ID = %d, want oldest queued job ID %d", claimed.ID, oldestQueuedID)
	}
	if claimed.Status != string(JobRunning) {
		t.Fatalf("claimed status = %q, want %q", claimed.Status, JobRunning)
	}
	if claimed.StartedAt == nil {
		t.Fatal("claimed job should record started_at")
	}

	newerQueued := getJob(t, store, newerQueuedID)
	if newerQueued.Status != string(JobQueued) {
		t.Fatalf("newer queued status = %q, want it left queued", newerQueued.Status)
	}
}

func TestCreateJobPersistsCreatedAtProvidedByGoInUTC(t *testing.T) {
	_, store, _ := newTestWorker(t, nil)
	createdAt := time.Date(2026, 8, 28, 19, 53, 36, 0, time.UTC)

	jobID, err := store.CreateJob(context.Background(), db.Job{
		JobType:   string(JobRefreshTitle),
		Status:    string(JobQueued),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	job := getJob(t, store, jobID)
	if !job.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", job.CreatedAt, createdAt)
	}
	if job.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at location = %v, want UTC", job.CreatedAt.Location())
	}
}

func TestClaimNextJobUsesIDAsTieBreakerForSameCreatedAt(t *testing.T) {
	_, store, database := newTestWorker(t, nil)
	createdAt := time.Now().UTC()
	firstID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, createdAt)
	insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, createdAt)

	claimed, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim next job: %v", err)
	}

	if claimed.ID != firstID {
		t.Fatalf("claimed job ID = %d, want lowest queued job ID %d for same created_at", claimed.ID, firstID)
	}
}

func TestProcessNextReturnsNilWhenThereAreNoQueuedJobs(t *testing.T) {
	worker, _, database := newTestWorker(t, nil)
	insertJob(t, database, string(JobRefreshTitle), string(JobRunning), 0, time.Now().UTC())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process next with no queued jobs: %v", err)
	}
}

func TestProcessNextMarksJobCompletedWhenExecutorSucceeds(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 10, Status: string(JobRunning), Attempt: 0})
	executed := false
	worker := NewWorker(store, fakeExecutor(func(ctx context.Context, job *db.Job) error {
		executed = true
		if ctx.Err() != nil {
			t.Fatalf("executor context is already cancelled: %v", ctx.Err())
		}
		if job.ID != 10 {
			t.Fatalf("executor job ID = %d, want 10", job.ID)
		}
		return nil
	}))

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process successful job: %v", err)
	}

	if !executed {
		t.Fatal("executor was not called")
	}
	assertStatusSequence(t, store.updates, string(JobCompleted))
	if store.job.Status != string(JobCompleted) {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobCompleted)
	}
	if store.job.Progress != "100" {
		t.Fatalf("job progress = %q, want 100", store.job.Progress)
	}
	if store.job.FinishedAt == nil {
		t.Fatal("completed job should record finished_at")
	}
}

func TestProcessNextRetriesRetryableFailuresUntilSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := newFakeJobStore(db.Job{ID: 11, Status: string(JobRunning), Attempt: 0})
		calls := 0
		worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
			calls++
			if calls < 3 {
				return errors.New("temporary failure")
			}
			return nil
		}))

		if err := worker.processNext(context.Background()); err != nil {
			t.Fatalf("process eventually successful job: %v", err)
		}

		if calls != 3 {
			t.Fatalf("executor calls = %d, want 3", calls)
		}
		assertStatusSequence(t, store.updates, string(JobRetrying), string(JobRetrying), string(JobCompleted))
		if store.job.Attempt != 2 {
			t.Fatalf("job attempt = %d, want 2", store.job.Attempt)
		}
		if store.job.Status != string(JobCompleted) {
			t.Fatalf("job status = %q, want %q", store.job.Status, JobCompleted)
		}
	})
}

func TestProcessNextDoesNotRetryUnknownJobType(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 12, JobType: "unknown", Status: string(JobRunning), Attempt: 0})
	calls := 0
	worker := NewWorker(store, fakeExecutor(func(_ context.Context, job *db.Job) error {
		calls++
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}))

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process unknown job type: %v", err)
	}

	if calls != 1 {
		t.Fatalf("executor calls = %d, want 1", calls)
	}
	assertStatusSequence(t, store.updates, string(JobFailed))
	if store.job.Attempt != 0 {
		t.Fatalf("job attempt = %d, want no retry", store.job.Attempt)
	}
}

func TestProcessNextMarksJobCancelledWhenExecutorObservesCancellation(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 13, Status: string(JobRunning), Attempt: 0})
	worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
		return context.Canceled
	}))

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process cancelled job: %v", err)
	}

	assertStatusSequence(t, store.updates, string(JobCancelled))
	if store.job.Status != string(JobCancelled) {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobCancelled)
	}
}

func TestProcessNextReturnsStatusUpdateErrorAndDoesNotContinue(t *testing.T) {
	updateErr := errors.New("store update failed")
	store := newFakeJobStore(db.Job{ID: 14, Status: string(JobRunning), Attempt: 0})
	store.updateErrByStatus[string(JobRetrying)] = updateErr
	calls := 0
	worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
		calls++
		return errors.New("temporary failure")
	}))

	err := worker.processNext(context.Background())
	if !errors.Is(err, updateErr) {
		t.Fatalf("process next error = %v, want %v", err, updateErr)
	}
	if calls != 1 {
		t.Fatalf("executor calls = %d, want stop after first failed status update", calls)
	}
	assertStatusSequence(t, store.updates, string(JobRetrying))
}

func TestCancelJobMarksQueuedJobAsCancelled(t *testing.T) {
	worker, store, database := newTestWorker(t, nil)
	jobID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, time.Now().UTC())

	if err := worker.CancelJob(context.Background(), jobID); err != nil {
		t.Fatalf("cancel queued job: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != string(JobCancelled) {
		t.Fatalf("job status = %q, want %q", job.Status, JobCancelled)
	}
}

func TestCancelJobCancelsRegisteredRunningJobWithoutOverwritingStatus(t *testing.T) {
	worker, store, database := newTestWorker(t, nil)
	jobID := insertJob(t, database, string(JobRefreshTitle), string(JobRunning), 0, time.Now().UTC())
	ctx, cancel := context.WithCancel(context.Background())
	worker.registerRunning(jobID, cancel)
	t.Cleanup(func() { worker.unregisterRunning(jobID) })

	if err := worker.CancelJob(context.Background(), jobID); err != nil {
		t.Fatalf("cancel registered running job: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("registered running job context was not cancelled")
	}

	job := getJob(t, store, jobID)
	if job.Status != string(JobRunning) {
		t.Fatalf("running job status = %q, want it left running until worker observes cancellation", job.Status)
	}
}

func TestProcessNextMarksJobFailedWhenExecutionCannotSucceed(t *testing.T) {
	worker, store, database := newTestWorker(t, fakeExecutor(func(_ context.Context, job *db.Job) error {
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}))
	jobID := insertJob(t, database, "unknown_job_type", string(JobQueued), 2, time.Now().UTC())

	err := worker.processNext(context.Background())
	if err != nil {
		t.Fatalf("process next should persist terminal failure instead of returning execution error: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != string(JobFailed) {
		t.Fatalf("job status = %q, want %q", job.Status, JobFailed)
	}
	if job.ErrorMessage == nil || !strings.Contains(*job.ErrorMessage, `unknown job type: "unknown_job_type"`) {
		if job.ErrorMessage == nil {
			t.Fatalf("job error message = <nil>, want unknown job type details")
		} else {
			t.Fatalf("job error message = %q, want unknown job type: \"unknown_job_type\"", *job.ErrorMessage)
		}
	}
}

func TestProcessNextMarksKnownJobFailedAfterLastAttemptFailure(t *testing.T) {
	worker, store, database := newTestWorker(t, fakeExecutor(func(context.Context, *db.Job) error {
		return errors.New("refresh_title job missing saved_title_id")
	}))
	jobID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 2, time.Now().UTC())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process final failed attempt: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != string(JobFailed) {
		t.Fatalf("job status = %q, want %q", job.Status, JobFailed)
	}
	if job.Attempt != 2 {
		t.Fatalf("job attempt = %d, want final attempt not incremented", job.Attempt)
	}
	if job.ErrorMessage == nil || *job.ErrorMessage != "refresh_title job missing saved_title_id" {
		t.Fatalf("job error message = %v, want missing saved_title_id details", job.ErrorMessage)
	}
}

func TestProcessNextCancelsJobWhileWaitingForRetry(t *testing.T) {
	worker, store, database := newTestWorker(t, fakeExecutor(func(context.Context, *db.Job) error {
		return errors.New("temporary failure")
	}))
	jobID := insertJob(t, database, string(JobRefreshTitle), string(JobQueued), 0, time.Now().UTC())
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- worker.processNext(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("process next error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("process next did not stop after retry wait context cancellation")
	}

	job := getJob(t, store, jobID)
	if job.Status != string(JobCancelled) {
		t.Fatalf("job status = %q, want %q", job.Status, JobCancelled)
	}
	if job.Attempt != 1 {
		t.Fatalf("job attempt = %d, want retry attempt recorded before cancellation", job.Attempt)
	}
}

func TestJobStatusUpdateBuildersDefineTerminalAndRetryTransitions(t *testing.T) {
	completed := markJobAsCompleted(10)
	if completed.ID != 10 || completed.Status != string(JobCompleted) || completed.Progress == nil || *completed.Progress != "100" || completed.FinishedAt == nil || completed.ErrorMessage != nil {
		t.Fatalf("completed status update = %+v, want completed with 100%% progress, finish time, and cleared error", completed)
	}

	failed := markJobAsFailed(11, assertErr("boom"))
	if failed.ID != 11 || failed.Status != string(JobFailed) || failed.ErrorMessage == nil || *failed.ErrorMessage != "boom" {
		t.Fatalf("failed status update = %+v, want failed with error message", failed)
	}

	retrying := markJobAsRetrying(&db.Job{ID: 12, Attempt: 1})
	if retrying.ID != 12 || retrying.Status != string(JobRetrying) || retrying.Attempt == nil || *retrying.Attempt != 2 {
		t.Fatalf("retrying status update = %+v, want attempt incremented to 2", retrying)
	}

	cancelled := markJobAsCancelled(13)
	if cancelled.ID != 13 || cancelled.Status != string(JobCancelled) {
		t.Fatalf("cancelled status update = %+v, want cancelled", cancelled)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
