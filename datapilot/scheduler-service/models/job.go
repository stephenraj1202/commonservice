package models

import (
	"time"

	"gorm.io/gorm"
)

// Job represents a scheduled cron job.
// Status values: "active", "paused", "deleted"
type Job struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `json:"name"`
	CronExpression string         `json:"cron_expression"`
	TargetURL      string         `json:"target_url"`
	HTTPMethod     string         `json:"http_method"`
	Description    string         `json:"description"`
	Status         string         `gorm:"index" json:"status"` // active | paused | deleted
	CronEntryID    int            `json:"-"`                   // robfig/cron entry ID
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName sets the GORM table name.
func (Job) TableName() string {
	return "jobs"
}
