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

	"main/common"
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

type fakeConfigGetter struct {
	config common.ConfigData
}

func (g fakeConfigGetter) Get() common.ConfigData {
	return g.config
}

func testConfig() ConfigGetter {
	return fakeConfigGetter{config: common.ConfigData{MaxRetries: 2}}
}

type fakeJobStore struct {
	job               *db.Job
	claimErr          error
	updateErrByStatus map[string]error
	updateHook        func(fakeStatusUpdate)
	updates           []fakeStatusUpdate
	logs              []fakeJobLog
}

type fakeStatusUpdate struct {
	ID           int64
	Status       string
	Retries      *int
	Progress     *string
	ErrorMessage *string
	FinishedAt   *time.Time
}

type fakeJobLog struct {
	jobID   int64
	level   string
	message string
}

func (s *fakeJobStore) ClaimNextJob(context.Context) (*db.Job, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	job := *s.job
	return &job, nil
}

func (s *fakeJobStore) applyStatus(status fakeStatusUpdate) (*db.Job, error) {
	s.updates = append(s.updates, status)
	if err := s.updateErrByStatus[status.Status]; err != nil {
		return nil, err
	}

	s.job.Status = status.Status
	if status.Retries != nil {
		s.job.Retries = *status.Retries
	}
	if status.Progress != nil {
		s.job.Progress = *status.Progress
	}
	s.job.ErrorMessage = status.ErrorMessage
	s.job.FinishedAt = status.FinishedAt
	if s.updateHook != nil {
		s.updateHook(status)
	}

	job := *s.job
	return &job, nil
}

func (s *fakeJobStore) MarkJobAsPaused(_ context.Context, id int64) (*db.Job, error) {
	return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobPaused})
}

func (s *fakeJobStore) MarkJobAsRetrying(_ context.Context, id int64, retries int) (*db.Job, error) {
	if _, ok := s.updateErrByStatus[JobRetrying]; ok {
		return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobRetrying, Retries: &retries})
	}

	wantRetries := s.job.Retries + 1
	if retries != wantRetries {
		return nil, fmt.Errorf("retry count = %d, want %d", retries, wantRetries)
	}
	return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobRetrying, Retries: &retries})
}

func (s *fakeJobStore) MarkJobAsCompleted(_ context.Context, id int64, progress string, errMessage *string, finishedAt time.Time) (*db.Job, error) {
	return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobCompleted, Progress: &progress, ErrorMessage: errMessage, FinishedAt: &finishedAt})
}

func (s *fakeJobStore) MarkJobAsFailed(_ context.Context, id int64, errMessage string) (*db.Job, error) {
	return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobFailed, ErrorMessage: &errMessage})
}

func (s *fakeJobStore) MarkJobAsCancelled(_ context.Context, id int64) (*db.Job, error) {
	return s.applyStatus(fakeStatusUpdate{ID: id, Status: JobCancelled})
}

func (s *fakeJobStore) AddJobLog(_ context.Context, jobID int64, level, message string) error {
	s.logs = append(s.logs, fakeJobLog{jobID: jobID, level: level, message: message})
	return nil
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

func newTestWorker(t *testing.T, executor Executor) (*Worker, *db.Store, *sqlx.DB) {
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
	return NewWorker(store, executor, testConfig()), store, database
}

func insertJob(t *testing.T, database *sqlx.DB, jobType, status string, retries int, createdAt time.Time) int64 {
	t.Helper()

	result, err := database.ExecContext(context.Background(), `
		INSERT INTO jobs (job_type, status, retries, created_at)
		VALUES (?, ?, ?, ?)
	`, jobType, status, retries, createdAt)
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

func assertStatusSequence(t *testing.T, updates []fakeStatusUpdate, want ...string) {
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

func assertHasLog(t *testing.T, logs []fakeJobLog, level, message string) {
	t.Helper()
	for _, log := range logs {
		if log.level == level && log.message == message {
			return
		}
	}
	t.Fatalf("logs do not contain %s %q; logs = %+v", level, message, logs)
}

func waitForRetrying(t *testing.T, result <-chan error, retrying <-chan struct{}) {
	t.Helper()
	select {
	case <-retrying:
	case err := <-result:
		t.Fatalf("process next returned before retry wait: %v", err)
	case <-time.After(time.Second):
		t.Fatal("process next did not enter retry wait")
	}
}

func TestClaimNextJobClaimsOldestQueuedJobAndSkipsNonQueuedJobs(t *testing.T) {
	_, store, database := newTestWorker(t, nil)
	now := time.Now().UTC()

	insertJob(t, database, RefreshTitle, JobRunning, 0, now.Add(-3*time.Minute))
	newerQueuedID := insertJob(t, database, RefreshTitle, JobQueued, 0, now.Add(-1*time.Minute))
	oldestQueuedID := insertJob(t, database, RefreshTitle, JobQueued, 0, now.Add(-2*time.Minute))

	claimed, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim next job: %v", err)
	}

	if claimed.ID != oldestQueuedID {
		t.Fatalf("claimed job ID = %d, want oldest queued job ID %d", claimed.ID, oldestQueuedID)
	}
	if claimed.Status != JobRunning {
		t.Fatalf("claimed status = %q, want %q", claimed.Status, JobRunning)
	}
	if claimed.StartedAt == nil {
		t.Fatal("claimed job should record started_at")
	}

	newerQueued := getJob(t, store, newerQueuedID)
	if newerQueued.Status != JobQueued {
		t.Fatalf("newer queued status = %q, want it left queued", newerQueued.Status)
	}
}

func TestCreateJobPersistsCreatedAtProvidedByGoInUTC(t *testing.T) {
	_, store, _ := newTestWorker(t, nil)
	createdAt := time.Date(2026, 8, 28, 19, 53, 36, 0, time.UTC)

	jobID, err := store.CreateJob(context.Background(), db.Job{
		JobType:   RefreshTitle,
		Status:    JobQueued,
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
	firstID := insertJob(t, database, RefreshTitle, JobQueued, 0, createdAt)
	insertJob(t, database, RefreshTitle, JobQueued, 0, createdAt)

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
	insertJob(t, database, RefreshTitle, JobRunning, 0, time.Now().UTC())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process next with no queued jobs: %v", err)
	}
}

func TestProcessNextMarksJobCompletedWhenExecutorSucceeds(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 10, Status: JobRunning, Retries: 0})
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
	}), testConfig())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process successful job: %v", err)
	}

	if !executed {
		t.Fatal("executor was not called")
	}
	assertStatusSequence(t, store.updates, JobCompleted)
	if store.job.Status != JobCompleted {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobCompleted)
	}
	if store.job.Progress != "100" {
		t.Fatalf("job progress = %q, want 100", store.job.Progress)
	}
	if store.job.FinishedAt == nil {
		t.Fatal("completed job should record finished_at")
	}
	assertHasLog(t, store.logs, Info, "job executed successfully")
}

func TestProcessNextRetriesRetryableFailuresUntilSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := newFakeJobStore(db.Job{ID: 11, Status: JobRunning, Retries: 0})
		calls := 0
		worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
			calls++
			if calls < 3 {
				return errors.New("temporary failure")
			}
			return nil
		}), testConfig())

		if err := worker.processNext(context.Background()); err != nil {
			t.Fatalf("process eventually successful job: %v", err)
		}

		if calls != 3 {
			t.Fatalf("executor calls = %d, want 3", calls)
		}
		assertStatusSequence(t, store.updates, JobRetrying, JobRetrying, JobCompleted)
		if store.job.Retries != 2 {
			t.Fatalf("job retries = %d, want 2", store.job.Retries)
		}
		if store.job.Status != JobCompleted {
			t.Fatalf("job status = %q, want %q", store.job.Status, JobCompleted)
		}
	})
}

func TestProcessNextDoesNotRetryUnknownJobType(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 12, JobType: "unknown", Status: JobRunning, Retries: 0})
	calls := 0
	worker := NewWorker(store, fakeExecutor(func(_ context.Context, job *db.Job) error {
		calls++
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}), testConfig())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process unknown job type: %v", err)
	}

	if calls != 1 {
		t.Fatalf("executor calls = %d, want 1", calls)
	}
	assertStatusSequence(t, store.updates, JobFailed)
	if store.job.Retries != 0 {
		t.Fatalf("job retries = %d, want no retry", store.job.Retries)
	}
}

func TestProcessNextMarksJobCancelledWhenExecutorObservesCancellation(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 13, Status: JobRunning, Retries: 0})
	ctx, cancel := context.WithCancelCause(context.Background())
	worker := NewWorker(store, fakeExecutor(func(execCtx context.Context, _ *db.Job) error {
		cancel(ErrJobCancelled)
		<-execCtx.Done()
		return execCtx.Err()
	}), testConfig())

	if err := worker.processNext(ctx); err != nil {
		t.Fatalf("process cancelled job: %v", err)
	}

	assertStatusSequence(t, store.updates, JobCancelled)
	if store.job.Status != JobCancelled {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobCancelled)
	}
	assertHasLog(t, store.logs, Info, "cancelling job")
}

func TestProcessNextMarksJobPausedWhenExecutorObservesPause(t *testing.T) {
	store := newFakeJobStore(db.Job{ID: 15, Status: JobRunning, Progress: "42", Retries: 0})
	ctx, cancel := context.WithCancelCause(context.Background())
	worker := NewWorker(store, fakeExecutor(func(execCtx context.Context, _ *db.Job) error {
		cancel(ErrJobPaused)
		<-execCtx.Done()
		return execCtx.Err()
	}), testConfig())

	if err := worker.processNext(ctx); err != nil {
		t.Fatalf("process paused job: %v", err)
	}

	assertStatusSequence(t, store.updates, JobPaused)
	if store.job.Status != JobPaused {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobPaused)
	}
	if store.job.Progress != "42" {
		t.Fatalf("job progress = %q, want current progress preserved", store.job.Progress)
	}
	assertHasLog(t, store.logs, Info, "pausing job")
}

func TestProcessNextReturnsStatusUpdateErrorAndDoesNotContinue(t *testing.T) {
	updateErr := errors.New("store update failed")
	store := newFakeJobStore(db.Job{ID: 14, Status: JobRunning, Retries: 0})
	store.updateErrByStatus[JobRetrying] = updateErr
	calls := 0
	worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
		calls++
		return errors.New("temporary failure")
	}), testConfig())

	err := worker.processNext(context.Background())
	if !errors.Is(err, updateErr) {
		t.Fatalf("process next error = %v, want %v", err, updateErr)
	}
	if calls != 1 {
		t.Fatalf("executor calls = %d, want stop after first failed status update", calls)
	}
	assertStatusSequence(t, store.updates, JobRetrying)
}

func TestCancelJobMarksQueuedJobAsCancelled(t *testing.T) {
	worker, store, database := newTestWorker(t, nil)
	jobID := insertJob(t, database, RefreshTitle, JobQueued, 0, time.Now().UTC())

	if err := worker.CancelJob(context.Background(), jobID); err != nil {
		t.Fatalf("cancel queued job: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != JobCancelled {
		t.Fatalf("job status = %q, want %q", job.Status, JobCancelled)
	}
}

func TestCancelJobCancelsRegisteredRunningJobWithoutOverwritingStatus(t *testing.T) {
	worker, store, database := newTestWorker(t, nil)
	jobID := insertJob(t, database, RefreshTitle, JobRunning, 0, time.Now().UTC())
	ctx, cancel := context.WithCancelCause(context.Background())
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
	if job.Status != JobRunning {
		t.Fatalf("running job status = %q, want it left running until worker observes cancellation", job.Status)
	}
}

func TestPauseJobPausesRegisteredRunningJobWithoutOverwritingStatus(t *testing.T) {
	worker, store, database := newTestWorker(t, nil)
	timeNow := time.Now().UTC()
	jobID := insertJob(t, database, RefreshTitle, JobRunning, 0, timeNow)
	ctx, cancel := context.WithCancelCause(context.Background())
	worker.registerRunning(jobID, cancel)
	t.Cleanup(func() { worker.unregisterRunning(jobID) })

	if err := worker.PauseJob(context.Background(), jobID); err != nil {
		t.Fatalf("pause registered running job: %v", err)
	}

	select {
	case <-ctx.Done():
		if !errors.Is(context.Cause(ctx), ErrJobPaused) {
			t.Fatalf("pause cause = %v, want %v", context.Cause(ctx), ErrJobPaused)
		}
	case <-time.After(time.Second):
		t.Fatal("registered running job context was not paused")
	}

	job := getJob(t, store, jobID)
	if job.Status != JobRunning {
		t.Fatalf("running job status = %q, want it left running until worker observes pause", job.Status)
	}
}

func TestProcessNextMarksJobFailedWhenExecutionCannotSucceed(t *testing.T) {
	worker, store, database := newTestWorker(t, fakeExecutor(func(_ context.Context, job *db.Job) error {
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}))
	jobID := insertJob(t, database, "unknown_job_type", JobQueued, 2, time.Now().UTC())

	err := worker.processNext(context.Background())
	if err != nil {
		t.Fatalf("process next should persist terminal failure instead of returning execution error: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != JobFailed {
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

func TestProcessNextMarksKnownJobFailedAfterLastRetryFailure(t *testing.T) {
	worker, store, database := newTestWorker(t, fakeExecutor(func(context.Context, *db.Job) error {
		return errors.New("refresh_title job missing saved_title_id")
	}))
	worker.configManager = fakeConfigGetter{config: common.ConfigData{MaxRetries: 1}}
	jobID := insertJob(t, database, RefreshTitle, JobQueued, 1, time.Now().UTC())

	if err := worker.processNext(context.Background()); err != nil {
		t.Fatalf("process final failed retry: %v", err)
	}

	job := getJob(t, store, jobID)
	if job.Status != JobFailed {
		t.Fatalf("job status = %q, want %q", job.Status, JobFailed)
	}
	if job.Retries != 1 {
		t.Fatalf("job retries = %d, want claimed retry recorded", job.Retries)
	}
	if job.ErrorMessage == nil || *job.ErrorMessage != "refresh_title job missing saved_title_id" {
		t.Fatalf("job error message = %v, want missing saved_title_id details", job.ErrorMessage)
	}
}

func TestProcessNextCancelsJobWhileWaitingForRetry(t *testing.T) {
	retrying := make(chan struct{})
	store := newFakeJobStore(db.Job{ID: 16, Status: JobRunning, Retries: 0})
	store.updateHook = func(status fakeStatusUpdate) {
		if status.Status == JobRetrying {
			close(retrying)
		}
	}
	worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
		return errors.New("temporary failure")
	}), testConfig())
	ctx, cancel := context.WithCancelCause(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- worker.processNext(ctx)
	}()

	waitForRetrying(t, result, retrying)
	cancel(ErrJobCancelled)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("process next error = %v, want nil after cancelling job", err)
		}
	case <-time.After(time.Second):
		t.Fatal("process next did not stop after retry wait context cancellation")
	}

	if store.job.Status != JobCancelled {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobCancelled)
	}
	if store.job.Retries != 1 {
		t.Fatalf("job retry = %d, want retry recorded before cancellation", store.job.Retries)
	}
	assertStatusSequence(t, store.updates, JobRetrying, JobCancelled)
	assertHasLog(t, store.logs, Info, "cancelling job")
}

func TestProcessNextPausesJobWhileWaitingForRetry(t *testing.T) {
	retrying := make(chan struct{})
	store := newFakeJobStore(db.Job{ID: 17, Status: JobRunning, Progress: "19", Retries: 0})
	store.updateHook = func(status fakeStatusUpdate) {
		if status.Status == JobRetrying {
			close(retrying)
		}
	}
	worker := NewWorker(store, fakeExecutor(func(context.Context, *db.Job) error {
		return errors.New("temporary failure")
	}), testConfig())
	ctx, cancel := context.WithCancelCause(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- worker.processNext(ctx)
	}()

	waitForRetrying(t, result, retrying)
	cancel(ErrJobPaused)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("process next error = %v, want nil after pausing job", err)
		}
	case <-time.After(time.Second):
		t.Fatal("process next did not stop after retry wait context pause")
	}

	if store.job.Status != JobPaused {
		t.Fatalf("job status = %q, want %q", store.job.Status, JobPaused)
	}
	if store.job.Retries != 1 {
		t.Fatalf("job retries = %d, want retry recorded before pause", store.job.Retries)
	}
	if store.job.Progress != "19" {
		t.Fatalf("job progress = %q, want current progress preserved", store.job.Progress)
	}
	assertStatusSequence(t, store.updates, JobRetrying, JobPaused)
	assertHasLog(t, store.logs, Info, "pausing job")
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
