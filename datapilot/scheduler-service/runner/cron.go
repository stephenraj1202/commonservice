package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"datapilot/scheduler-service/models"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Runner manages the cron scheduler and job execution.
type Runner struct {
	cron   *cron.Cron
	db     *gorm.DB
	logger *zap.Logger
}

// NewRunner initialises a new cron Runner with second-level precision.
func NewRunner(db *gorm.DB, logger *zap.Logger) *Runner {
	return &Runner{
		cron:   cron.New(cron.WithSeconds()),
		db:     db,
		logger: logger,
	}
}

// Register adds a job to the cron scheduler and persists the assigned EntryID back to the DB.
func (r *Runner) Register(job *models.Job) error {
	entryID, err := r.cron.AddFunc(job.CronExpression, r.tickFunc(job))
	if err != nil {
		return fmt.Errorf("failed to register job %d: %w", job.ID, err)
	}

	job.CronEntryID = int(entryID)

	if result := r.db.Model(job).Update("cron_entry_id", job.CronEntryID); result.Error != nil {
		// Non-fatal: log but don't fail — the job is already scheduled in memory.
		r.logger.Error("failed to persist cron_entry_id",
			zap.Uint("job_id", job.ID),
			zap.Error(result.Error),
		)
	}

	r.logger.Info("job registered",
		zap.Uint("job_id", job.ID),
		zap.String("cron_expression", job.CronExpression),
		zap.Int("entry_id", job.CronEntryID),
	)

	return nil
}

// Remove removes a cron entry by its EntryID.
func (r *Runner) Remove(entryID int) {
	r.cron.Remove(cron.EntryID(entryID))
}

// Start begins the cron scheduler in a background goroutine.
func (r *Runner) Start() {
	r.cron.Start()
}

// Stop gracefully stops the cron scheduler.
func (r *Runner) Stop() {
	r.cron.Stop()
}

// tickFunc returns the function executed on each cron tick for the given job.
func (r *Runner) tickFunc(job *models.Job) func() {
	// Capture immutable fields at registration time to avoid data races.
	jobID := job.ID
	targetURL := job.TargetURL
	httpMethod := job.HTTPMethod

	return func() {
		r.executeJob(jobID, targetURL, httpMethod)
	}
}

// executeJob performs a single job execution: sends the HTTP request and saves the log.
// Exported for testing purposes.
func (r *Runner) ExecuteJob(jobID uint, targetURL, httpMethod string) {
	r.executeJob(jobID, targetURL, httpMethod)
}

func (r *Runner) executeJob(jobID uint, targetURL, httpMethod string) {
	start := time.Now()

	log := &models.JobExecutionLog{
		JobID:      jobID,
		ExecutedAt: start,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, httpMethod, targetURL, nil)
	if err != nil {
		log.Status = "failed"
		log.ErrorDetail = fmt.Sprintf("failed to build request: %s", err.Error())
		log.DurationMS = time.Since(start).Milliseconds()
		r.saveLog(log)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	log.DurationMS = time.Since(start).Milliseconds()

	if err != nil {
		log.Status = "failed"
		log.ResponseCode = 0
		log.ErrorDetail = err.Error()
	} else {
		defer resp.Body.Close()
		log.ResponseCode = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Status = "success"
		} else {
			log.Status = "failed"
			log.ErrorDetail = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		}
	}

	r.saveLog(log)
}

// saveLog persists a JobExecutionLog entry and logs any DB error.
func (r *Runner) saveLog(log *models.JobExecutionLog) {
	if result := r.db.Create(log); result.Error != nil {
		r.logger.Error("failed to save execution log",
			zap.Uint("job_id", log.JobID),
			zap.Error(result.Error),
		)
	}
}
