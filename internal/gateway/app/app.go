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

	v1 := router.Group("/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", a.handleRegister)
			auth.POST("/login", a.handleLogin)
			auth.POST("/refresh-token", a.handleRefreshToken)
		}

		protectedAuth := v1.Group("/auth")
		protectedAuth.Use(middleware.AuthMiddleware(a.UserClient))
		{
			protectedAuth.GET("/me", a.handleGetProfile)
		}

		files := v1.Group("/files")
		files.Use(middleware.AuthMiddleware(a.UserClient))
		{
			files.GET("/presigned-url", a.handleGetPresignedURL)
			files.GET("/upload-presigned-url", a.handleGetUploadPresignedURL)
		}
	}
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

func (a *GatewayApp) handleGetUploadPresignedURL(c *gin.Context) {
	fileName := c.Query("file_name")
	contentType := c.Query("content_type")
	folder := c.DefaultQuery("folder", "uploads")
	expiresIn, _ := strconv.ParseInt(c.DefaultQuery("expires_in_seconds", "3600"), 10, 64)

	resp, err := a.FileClient.GetUploadPresignedUrl(c.Request.Context(), &filev1.GetUploadPresignedUrlRequest{
		FileName:         fileName,
		ContentType:      contentType,
		Folder:           folder,
		ExpiresInSeconds: expiresIn,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
