// Package database provides a shared MySQL connection helper for all DataPilot services.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// requiredParams are the DSN parameters that must be present for correct operation.
var requiredParams = "charset=utf8mb4&parseTime=True&loc=Local"

// ensureDSNParams appends the required DSN parameters if they are not already present.
func ensureDSNParams(dsn string) string {
	if strings.Contains(dsn, "?") {
		// Already has query params — append missing ones
		if !strings.Contains(dsn, "charset=utf8mb4") {
			dsn += "&charset=utf8mb4"
		}
		if !strings.Contains(dsn, "parseTime=True") {
			dsn += "&parseTime=True"
		}
		if !strings.Contains(dsn, "loc=Local") {
			dsn += "&loc=Local"
		}
		return dsn
	}
	return dsn + "?" + requiredParams
}

// InitDB opens a GORM MySQL connection, configures the connection pool, pings the
// database within a 10-second deadline, and runs AutoMigrate for all supplied models.
//
// Requirements: 3.1, 3.2, 3.3, 3.4
func InitDB(dsn string, models ...interface{}) (*gorm.DB, error) {
	dsn = ensureDSNParams(dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("database: failed to open connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: failed to get underlying *sql.DB: %w", err)
	}

	// Requirement 3.2 — connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Requirement 3.3 — ping within 10-second deadline
	if err := pingWithTimeout(sqlDB, 10*time.Second); err != nil {
		return nil, fmt.Errorf("database: ping failed: %w", err)
	}

	// Requirement 3.4 — auto-migrate all registered models
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			return nil, fmt.Errorf("database: auto-migrate failed: %w", err)
		}
	}

	return db, nil
}

// pingWithTimeout pings the database using a context with the given deadline.
func pingWithTimeout(sqlDB *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return sqlDB.PingContext(ctx)
}
