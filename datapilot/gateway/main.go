package main

import (
	"context"
	"net/http"
	"time"

	"datapilot/common/config"
	"datapilot/common/database"
	"datapilot/common/logger"
	"datapilot/common/middleware"
	"datapilot/gateway/handlers"
	"datapilot/gateway/models"
	"datapilot/gateway/proxy"

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

	db, err := database.InitDB(cfg.MySQLDSN, &models.User{})
	if err != nil {
		log.Fatal("failed to initialise database", zap.Error(err))
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Health check — no auth required.
	r.GET("/health", healthHandler(cfg.FileServiceURL, cfg.SchedulerServiceURL))

	// Auth routes — no JWT required.
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", handlers.Login(db, cfg.JWTSecret, log))
		auth.POST("/register", handlers.Register(db, log))
	}

	// File Service proxy — JWT required.
	r.Any("/api/v1/files/*path",
		middleware.JWTAuth(cfg.JWTSecret),
		proxy.NewProxy(cfg.FileServiceURL),
	)

	// Scheduler Service proxy — JWT required.
	r.Any("/api/v1/scheduler/*path",
		middleware.JWTAuth(cfg.JWTSecret),
		proxy.NewProxy(cfg.SchedulerServiceURL),
	)

	log.Info("starting API gateway", zap.String("port", cfg.HTTPPort))
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}

// healthHandler checks reachability of the upstream services and returns a
// JSON body describing each service's status.
func healthHandler(fileServiceURL, schedulerServiceURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileStatus := checkService(fileServiceURL + "/health")
		schedulerStatus := checkService(schedulerServiceURL + "/health")

		overallStatus := "ok"
		if fileStatus != "ok" || schedulerStatus != "ok" {
			overallStatus = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status": overallStatus,
			"services": gin.H{
				"file-service":      fileStatus,
				"scheduler-service": schedulerStatus,
			},
		})
	}
}

// checkService performs a GET request to url with a 3-second timeout and
// returns "ok" on HTTP 2xx, or "unreachable" otherwise.
func checkService(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "unreachable"
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok"
	}
	return "unreachable"
}
