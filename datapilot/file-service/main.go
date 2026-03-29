package main

import (
	"net/http"

	"datapilot/common/config"
	"datapilot/common/database"
	"datapilot/common/logger"
	"datapilot/common/middleware"
	"datapilot/file-service/handlers"
	"datapilot/file-service/models"
	"datapilot/file-service/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.NewLogger(cfg.ServiceName, cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	db, err := database.InitDB(cfg.MySQLDSN, &models.FileRecord{})
	if err != nil {
		log.Fatal("failed to initialise database", zap.Error(err))
	}

	store := &storage.LocalStorage{BasePath: cfg.FileStoragePath}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(log))

	// Health check — no auth required.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// File routes — JWT required.
	files := r.Group("/files", middleware.JWTAuth(cfg.JWTSecret))
	{
		files.POST("/upload", handlers.Upload(db, store, log))
		files.GET("/:id/download", handlers.Download(db, store, log))
		files.GET("", handlers.List(db))
		files.DELETE("/:id", handlers.Delete(db, store, log))
	}

	log.Info("starting file service", zap.String("port", cfg.HTTPPort))
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
