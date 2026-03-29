package models

import "time"

// FileRecord stores metadata for an uploaded file in the file_records table.
type FileRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	OriginalFilename string    `json:"original_filename"`
	StoredFilename   string    `gorm:"unique" json:"stored_filename"`
	MIMEType         string    `json:"mime_type"`
	SizeBytes        int64     `json:"size_bytes"`
	UploaderIdentity string    `json:"uploader_identity"`
	StoragePath      string    `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (FileRecord) TableName() string {
	return "file_records"
}
