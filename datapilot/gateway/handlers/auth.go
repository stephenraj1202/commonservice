package handlers

import (
	"fmt"
	"net/http"
	"time"

	"datapilot/common/errors"
	"datapilot/gateway/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
// It looks up the user by username, compares the bcrypt hash, and on success
// returns a signed JWT with a 24-hour expiry.
func Login(db *gorm.DB, jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.RespondError(c, http.StatusBadRequest, "bad_request", "username and password are required")
			return
		}

		var user models.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			// User not found — return 401 to avoid username enumeration
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
			return
		}

		now := time.Now()
		claims := jwt.MapClaims{
			"sub":      fmt.Sprintf("%d", user.ID),
			"username": user.Username,
			"exp":      now.Add(24 * time.Hour).Unix(),
			"iat":      now.Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			logger.Error("failed to sign JWT", zap.Error(err))
			errors.RespondError(c, http.StatusInternalServerError, "internal_error", "could not generate token")
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": signed})
	}
}

// Register handles POST /api/v1/auth/register.
// It hashes the supplied password with bcrypt (cost 12) and inserts a new User row.
func Register(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.RespondError(c, http.StatusBadRequest, "bad_request", "username and password are required")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			logger.Error("failed to hash password", zap.Error(err))
			errors.RespondError(c, http.StatusInternalServerError, "internal_error", "could not process request")
			return
		}

		user := models.User{
			Username: req.Username,
			Password: string(hashed),
		}

		if err := db.Create(&user).Error; err != nil {
			// Duplicate username — unique constraint violation
			errors.RespondError(c, http.StatusConflict, "conflict", "username already exists")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "user created"})
	}
}
