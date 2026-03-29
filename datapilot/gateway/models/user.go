package models

import "gorm.io/gorm"

// User represents a platform user stored in the users table.
type User struct {
	ID        uint           `gorm:"primaryKey"       json:"id"`
	Username  string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"not null"         json:"-"`
	CreatedAt int64          `                        json:"created_at"`
	UpdatedAt int64          `                        json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"            json:"-"`
}

// TableName overrides the default GORM table name.
func (User) TableName() string {
	return "users"
}
