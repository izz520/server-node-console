package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"server-sing-box-2/backend/internal/auth"
	"server-sing-box-2/backend/internal/domain"
	"server-sing-box-2/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	jwtSecret string
}

func New(db *gorm.DB, jwtSecret string) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret}
}

func (h *Handler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": err.Error()})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) NotImplemented(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"resource": resource,
			"message":  "endpoint scaffolded; implementation pending",
		})
	}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=2,max=64"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type loginRequest struct {
	Account  string `json:"account" binding:"required,max=255"`
	Password string `json:"password" binding:"required,max=128"`
}

type userResponse struct {
	ID        uint            `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      domain.UserRole `json:"role"`
	CreatedAt time.Time       `json:"createdAt"`
}

type authResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expiresAt"`
	User      userResponse `json:"user"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if username == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and email are required"})
		return
	}

	var existing domain.User
	err := h.db.Where("username = ? OR email = ?", username, email).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check user failed"})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}

	user := domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.UserRoleUser,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed"})
		return
	}

	response, err := h.buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token failed"})
		return
	}
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	account := strings.ToLower(strings.TrimSpace(req.Account))
	var user domain.User
	err := h.db.Where("LOWER(username) = ? OR LOWER(email) = ?", account, account).First(&user).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid account or password"})
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid account or password"})
		return
	}

	response, err := h.buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) Me(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *Handler) RefreshToken(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	response, err := h.buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) currentUser(c *gin.Context) (domain.User, bool) {
	userIDValue, ok := c.Get(middleware.ContextUserID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return domain.User{}, false
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return domain.User{}, false
	}

	var user domain.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return domain.User{}, false
	}

	return user, true
}

func (h *Handler) buildAuthResponse(user domain.User) (authResponse, error) {
	token, expiresAt, err := auth.GenerateToken(h.jwtSecret, user.ID, string(user.Role), time.Now())
	if err != nil {
		return authResponse{}, err
	}
	return authResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      toUserResponse(user),
	}, nil
}

func toUserResponse(user domain.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}
