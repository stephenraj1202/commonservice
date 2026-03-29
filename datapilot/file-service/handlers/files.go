package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"datapilot/common/errors"
	"datapilot/common/pagination"
	"datapilot/file-service/models"
	"datapilot/file-service/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Upload handles POST /files/upload — parses a multipart form, stores the file,
// and inserts a FileRecord into the database.
func Upload(db *gorm.DB, store storage.Storage, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enforce 100 MB limit
		if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
			errors.RespondError(c, http.StatusRequestEntityTooLarge, "file_too_large", "file exceeds 100 MB limit")
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			errors.RespondError(c, http.StatusBadRequest, "bad_request", "missing file field")
			return
		}
		defer file.Close()

		// Reject files larger than 100 MB
		if header.Size > 100<<20 {
			errors.RespondError(c, http.StatusRequestEntityTooLarge, "file_too_large", "file exceeds 100 MB limit")
			return
		}

		// Generate UUID-based stored filename, preserving original extension
		ext := filepath.Ext(header.Filename)
		storedFilename := uuid.New().String() + ext

		// Detect MIME type
		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Write file to storage
		written, err := store.Save(storedFilename, file)
		if err != nil {
			logger.Error("failed to save file", zap.String("filename", storedFilename), zap.Error(err))
			errors.RespondError(c, http.StatusInternalServerError, "storage_error", "failed to write file to storage")
			return
		}

		// Extract uploader identity from JWT claims
		uploaderIdentity := extractUploaderIdentity(c)

		record := models.FileRecord{
			OriginalFilename: header.Filename,
			StoredFilename:   storedFilename,
			MIMEType:         mimeType,
			SizeBytes:        written,
			UploaderIdentity: uploaderIdentity,
			StoragePath:      storedFilename,
		}

		if result := db.Create(&record); result.Error != nil {
			logger.Error("failed to insert file record", zap.Error(result.Error))
			errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to save file record")
			return
		}

		c.JSON(http.StatusCreated, record)
	}
}

// Download handles GET /files/:id/download — streams the file to the client.
func Download(db *gorm.DB, store storage.Storage, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			errors.RespondError(c, http.StatusBadRequest, "bad_request", "invalid file ID")
			return
		}

		var record models.FileRecord
		if result := db.First(&record, id); result.Error != nil {
			errors.RespondError(c, http.StatusNotFound, "not_found", "file record not found")
			return
		}

		rc, err := store.Open(record.StoredFilename)
		if err != nil {
			logger.Error("physical file missing from storage",
				zap.Uint("id", uint(id)),
				zap.String("stored_filename", record.StoredFilename),
				zap.Error(err),
			)
			errors.RespondError(c, http.StatusInternalServerError, "storage_error", "file exists in database but is missing from storage")
			return
		}
		defer rc.Close()

		c.Header("Content-Type", record.MIMEType)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.OriginalFilename))
		c.Status(http.StatusOK)
		io.Copy(c.Writer, rc) //nolint:errcheck
	}
}

// List handles GET /files — returns a paginated list of FileRecords ordered by created_at DESC.
func List(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit := pagination.ParseParams(c)

		var total int64
		db.Model(&models.FileRecord{}).Count(&total)

		var records []models.FileRecord
		pagination.Paginate(db, page, limit).
			Order("created_at DESC").
			Find(&records)

		c.JSON(http.StatusOK, pagination.PagedResponse{
			Total: total,
			Page:  page,
			Limit: limit,
			Data:  records,
		})
	}
}

// Delete handles DELETE /files/:id — removes the physical file and the DB record.
func Delete(db *gorm.DB, store storage.Storage, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			errors.RespondError(c, http.StatusBadRequest, "bad_request", "invalid file ID")
			return
		}

		var record models.FileRecord
		if result := db.First(&record, id); result.Error != nil {
			errors.RespondError(c, http.StatusNotFound, "not_found", "file record not found")
			return
		}

		// Delete physical file; log error but continue to remove DB record
		if err := store.Delete(record.StoredFilename); err != nil {
			logger.Error("failed to delete physical file",
				zap.Uint("id", uint(id)),
				zap.String("stored_filename", record.StoredFilename),
				zap.Error(err),
			)
		}

		if result := db.Delete(&record); result.Error != nil {
			logger.Error("failed to delete file record", zap.Error(result.Error))
			errors.RespondError(c, http.StatusInternalServerError, "db_error", "failed to remove file record")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

// extractUploaderIdentity reads the uploader's identity from JWT claims stored
// in the Gin context. It prefers the "username" claim and falls back to "sub".
func extractUploaderIdentity(c *gin.Context) string {
	raw, exists := c.Get("claims")
	if !exists {
		return ""
	}
	claims, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	if username, ok := claims["username"].(string); ok && username != "" {
		return username
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}
