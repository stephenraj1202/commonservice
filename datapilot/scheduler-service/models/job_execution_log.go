package models

import "time"

// JobExecutionLog records the outcome of a single job execution.
// Status values: "success", "failed"
type JobExecutionLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	JobID        uint      `gorm:"index" json:"job_id"`
	Status       string    `json:"status"` // success | failed
	ResponseCode int       `json:"response_code"`
	DurationMS   int64     `json:"duration_ms"`
	ErrorDetail  string    `json:"error_detail"`
	ExecutedAt   time.Time `gorm:"index" json:"executed_at"`
}

// TableName sets the GORM table name.
func (JobExecutionLog) TableName() string {
	return "job_execution_logs"
}
