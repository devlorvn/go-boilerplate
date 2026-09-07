package app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-boilerplate/internal/gateway/app/middleware"
	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/redis"
	filev1 "go-boilerplate/pkg/genproto/file/v1"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GatewayApp struct {
	Config     *configs.Config
	Logger     *zap.Logger
	Redis      *redis.RedisClient
	UserClient userv1.UserServiceClient
	FileClient filev1.FileServiceClient
}

func NewGatewayApp(
	cfg *configs.Config,
	log *zap.Logger,
	rClient *redis.RedisClient,
	userClient userv1.UserServiceClient,
	fileClient filev1.FileServiceClient,
) *GatewayApp {
	return &GatewayApp{
		Config:     cfg,
		Logger:     log,
		Redis:      rClient,
		UserClient: userClient,
		FileClient: fileClient,
	}
}

func (a *GatewayApp) RegisterHTTP(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit(a.Redis, 100, time.Minute))

	registerRoutes := func(rg *gin.RouterGroup) {
		auth := rg.Group("/auth")
		{
			auth.POST("/register", a.handleRegister)
			auth.POST("/login", a.handleLogin)
			auth.POST("/refresh-token", a.handleRefreshToken)
		}

		protectedAuth := rg.Group("/auth")
		protectedAuth.Use(middleware.AuthMiddleware(a.UserClient))
		{
			protectedAuth.GET("/me", a.handleGetProfile)
		}

		files := rg.Group("/files")
		files.Use(middleware.AuthMiddleware(a.UserClient))
		{
			files.GET("", a.handleGetAllFiles)
			files.GET("/presigned-url", a.handleGetPresignedURL)
			files.POST("/upload-presigned-url", a.handleGetUploadPresignedURL)
		}
	}

	registerRoutes(router.Group("/v1"))
	registerRoutes(router.Group("/api/v1"))

	// Catch-all 404 handler for undefined routes
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      http.StatusNotFound,
			"error":     "ROUTE_NOT_FOUND",
			"message":   fmt.Sprintf("Route %s %s is not defined", c.Request.Method, c.Request.URL.Path),
			"path":      c.Request.URL.Path,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Catch-all 405 handler for unsupported HTTP methods
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"code":      http.StatusMethodNotAllowed,
			"error":     "METHOD_NOT_ALLOWED",
			"message":   fmt.Sprintf("Method %s is not allowed for route %s", c.Request.Method, c.Request.URL.Path),
			"path":      c.Request.URL.Path,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})
}

func (a *GatewayApp) handleRegister(c *gin.Context) {
	var req userv1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := a.UserClient.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (a *GatewayApp) handleLogin(c *gin.Context) {
	var req userv1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := a.UserClient.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (a *GatewayApp) handleRefreshToken(c *gin.Context) {
	var req userv1.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := a.UserClient.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (a *GatewayApp) handleGetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	resp, err := a.UserClient.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: fmt.Sprintf("%v", userID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (a *GatewayApp) handleGetPresignedURL(c *gin.Context) {
	fileKey := c.Query("file_key")
	expiresIn, _ := strconv.ParseInt(c.DefaultQuery("expires_in_seconds", "3600"), 10, 64)

	resp, err := a.FileClient.GetPresignedUrl(c.Request.Context(), &filev1.GetPresignedUrlRequest{
		FileKey:          fileKey,
		ExpiresInSeconds: expiresIn,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

type GetUploadPresignedURLRequestDTO struct {
	FileName         string `json:"file_name" binding:"required"`
	ContentType      string `json:"content_type"`
	Folder           string `json:"folder"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

func (a *GatewayApp) handleGetUploadPresignedURL(c *gin.Context) {
	var dto GetUploadPresignedURLRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dto.Folder == "" {
		dto.Folder = "uploads"
	}
	if dto.ExpiresInSeconds <= 0 {
		dto.ExpiresInSeconds = 3600
	}

	resp, err := a.FileClient.GetUploadPresignedUrl(c.Request.Context(), &filev1.GetUploadPresignedUrlRequest{
		FileName:         dto.FileName,
		ContentType:      dto.ContentType,
		Folder:           dto.Folder,
		ExpiresInSeconds: dto.ExpiresInSeconds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (a *GatewayApp) handleGetAllFiles(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")

	resp, err := a.FileClient.GetAllFiles(c.Request.Context(), &filev1.GetAllFilesRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
		Status: status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
