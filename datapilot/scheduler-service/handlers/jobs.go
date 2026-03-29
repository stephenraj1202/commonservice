package handlers

import (
	"net/http"
	"strconv"

	"datapilot/common/errors"
	"datapilot/common/pagination"
	"datapilot/scheduler-service/models"
	"datapilot/scheduler-service/runner"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler holds shared dependencies for all job handlers.
type Handler struct {
	db     *gorm.DB
	runner *runner.Runner
	logger *zap.Logger
}

// NewHandler creates a Handler with the given dependencies.
func NewHandler(db *gorm.DB, r *runner.Runner, logger *zap.Logger) *Handler {
	return &Handler{db: db, runner: r, logger: logger}
}

// createJobRequest is the expected JSON body for CreateJob and UpdateJob.
type createJobRequest struct {
	Name           string `json:"name"`
	CronExpression string `json:"cron_expression"`
	TargetURL      string `json:"target_url"`
	HTTPMethod     string `json:"http_method"`
	Description    string `json:"description"`
}

// validateCronExpression returns an error if the expression is not a valid
// standard 5-field or 6-field (with seconds) cron expression.
func validateCronExpression(expr string) error {
	parser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	_, err := parser.Parse(expr)
	return err
}

// CreateJob handles POST /scheduler/jobs.
// Validates the cron expression (HTTP 422 if invalid), persists the job with
// status="active", registers it with the runner, and returns HTTP 201.
func (h *Handler) CreateJob(c *gin.Context) {
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if err := validateCronExpression(req.CronExpression); err != nil {
		errors.RespondError(c, http.StatusUnprocessableEntity, "invalid_cron", "invalid cron expression: "+err.Error())
		return
	}

	job := models.Job{
		Name:           req.Name,
		CronExpression: req.CronExpression,
		TargetURL:      req.TargetURL,
		HTTPMethod:     req.HTTPMethod,
		Description:    req.Description,
		Status:         "active",
	}

	if result := h.db.Create(&job); result.Error != nil {
		h.logger.Error("failed to create job", zap.Error(result.Error))
		errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to create job")
		return
	}

	if err := h.runner.Register(&job); err != nil {
		h.logger.Error("failed to register job with runner", zap.Uint("job_id", job.ID), zap.Error(err))
		// Job is persisted; runner registration failure is non-fatal for the HTTP response.
	}

	c.JSON(http.StatusCreated, job)
}

// ListJobs handles GET /scheduler/jobs.
// Returns a paginated list ordered by created_at DESC. Supports optional
// ?status= query param to filter by job status.
func (h *Handler) ListJobs(c *gin.Context) {
	page, limit := pagination.ParseParams(c)
	statusFilter := c.Query("status")

	query := h.db.Model(&models.Job{})
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var total int64
	query.Count(&total)

	var jobs []models.Job
	pagination.Paginate(query, page, limit).
		Order("created_at DESC").
		Find(&jobs)

	c.JSON(http.StatusOK, pagination.PagedResponse{
		Total: total,
		Page:  page,
		Limit: limit,
		Data:  jobs,
	})
}

// UpdateJob handles PUT /scheduler/jobs/:id.
// Looks up the job (404 if missing), validates the new cron expression (422 if
// invalid), updates the DB record, and re-registers with the runner.
func (h *Handler) UpdateJob(c *gin.Context) {
	job, ok := h.findJob(c)
	if !ok {
		return
	}

	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if err := validateCronExpression(req.CronExpression); err != nil {
		errors.RespondError(c, http.StatusUnprocessableEntity, "invalid_cron", "invalid cron expression: "+err.Error())
		return
	}

	// Remove old cron entry before re-registering.
	h.runner.Remove(job.CronEntryID)

	job.Name = req.Name
	job.CronExpression = req.CronExpression
	job.TargetURL = req.TargetURL
	job.HTTPMethod = req.HTTPMethod
	job.Description = req.Description

	if result := h.db.Save(&job); result.Error != nil {
		h.logger.Error("failed to update job", zap.Uint("job_id", job.ID), zap.Error(result.Error))
		errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to update job")
		return
	}

	if job.Status == "active" {
		if err := h.runner.Register(&job); err != nil {
			h.logger.Error("failed to re-register job with runner", zap.Uint("job_id", job.ID), zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, job)
}

// PauseJob handles POST /scheduler/jobs/:id/pause.
// Sets status="paused" and removes the job from the runner.
func (h *Handler) PauseJob(c *gin.Context) {
	job, ok := h.findJob(c)
	if !ok {
		return
	}

	h.runner.Remove(job.CronEntryID)

	job.Status = "paused"
	if result := h.db.Save(&job); result.Error != nil {
		h.logger.Error("failed to pause job", zap.Uint("job_id", job.ID), zap.Error(result.Error))
		errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to pause job")
		return
	}

	c.JSON(http.StatusOK, job)
}

// ResumeJob handles POST /scheduler/jobs/:id/resume.
// Sets status="active" and re-registers the job with the runner.
func (h *Handler) ResumeJob(c *gin.Context) {
	job, ok := h.findJob(c)
	if !ok {
		return
	}

	job.Status = "active"
	if result := h.db.Save(&job); result.Error != nil {
		h.logger.Error("failed to resume job", zap.Uint("job_id", job.ID), zap.Error(result.Error))
		errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to resume job")
		return
	}

	if err := h.runner.Register(&job); err != nil {
		h.logger.Error("failed to re-register job with runner", zap.Uint("job_id", job.ID), zap.Error(err))
	}

	c.JSON(http.StatusOK, job)
}

// DeleteJob handles DELETE /scheduler/jobs/:id.
// Soft-deletes the job (sets deleted_at) and removes it from the runner.
func (h *Handler) DeleteJob(c *gin.Context) {
	job, ok := h.findJob(c)
	if !ok {
		return
	}

	h.runner.Remove(job.CronEntryID)

	if result := h.db.Delete(&job); result.Error != nil {
		h.logger.Error("failed to delete job", zap.Uint("job_id", job.ID), zap.Error(result.Error))
		errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to delete job")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetLogs handles GET /scheduler/jobs/:id/logs.
// Looks up the job (404 if missing), then returns paginated execution logs
// ordered by executed_at DESC.
func (h *Handler) GetLogs(c *gin.Context) {
	job, ok := h.findJob(c)
	if !ok {
		return
	}

	page, limit := pagination.ParseParams(c)

	var total int64
	h.db.Model(&models.JobExecutionLog{}).Where("job_id = ?", job.ID).Count(&total)

	var logs []models.JobExecutionLog
	pagination.Paginate(h.db, page, limit).
		Where("job_id = ?", job.ID).
		Order("executed_at DESC").
		Find(&logs)

	c.JSON(http.StatusOK, pagination.PagedResponse{
		Total: total,
		Page:  page,
		Limit: limit,
		Data:  logs,
	})
}

// findJob is a helper that parses the :id param and fetches the Job from DB.
// It writes the appropriate error response and returns false if the job is not found.
func (h *Handler) findJob(c *gin.Context) (models.Job, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errors.RespondError(c, http.StatusBadRequest, "bad_request", "invalid job ID")
		return models.Job{}, false
	}

	var job models.Job
	if result := h.db.First(&job, id); result.Error != nil {
		errors.RespondError(c, http.StatusNotFound, "not_found", "job not found")
		return models.Job{}, false
	}

	return job, true
}
