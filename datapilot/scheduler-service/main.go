package main

import (
	"datapilot/common/config"
	"datapilot/common/database"
	"datapilot/common/logger"
	"datapilot/common/middleware"
	"datapilot/scheduler-service/handlers"
	"datapilot/scheduler-service/models"
	"datapilot/scheduler-service/runner"
	"net/http"

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

	db, err := database.InitDB(cfg.MySQLDSN, &models.Job{}, &models.JobExecutionLog{})
	if err != nil {
		log.Fatal("failed to initialise database", zap.Error(err))
	}

	r := runner.NewRunner(db, log)

	// Load all active jobs from DB and register them with the cron runner.
	var activeJobs []models.Job
	if result := db.Where("status = ?", "active").Find(&activeJobs); result.Error != nil {
		log.Error("failed to load active jobs", zap.Error(result.Error))
	} else {
		for i := range activeJobs {
			if err := r.Register(&activeJobs[i]); err != nil {
				log.Error("failed to register job on startup",
					zap.Uint("job_id", activeJobs[i].ID),
					zap.Error(err),
				)
			}
		}
		log.Info("loaded active jobs", zap.Int("count", len(activeJobs)))
	}

	r.Start()
	defer r.Stop()

	h := handlers.NewHandler(db, r, log)

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery(log))

	// Health check — no auth required.
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Scheduler routes — JWT required.
	sched := engine.Group("/scheduler", middleware.JWTAuth(cfg.JWTSecret))
	{
		sched.POST("/jobs", h.CreateJob)
		sched.GET("/jobs", h.ListJobs)
		sched.PUT("/jobs/:id", h.UpdateJob)
		sched.POST("/jobs/:id/pause", h.PauseJob)
		sched.POST("/jobs/:id/resume", h.ResumeJob)
		sched.DELETE("/jobs/:id", h.DeleteJob)
		sched.GET("/jobs/:id/logs", h.GetLogs)
	}

	log.Info("starting scheduler service", zap.String("port", cfg.HTTPPort))
	if err := engine.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
