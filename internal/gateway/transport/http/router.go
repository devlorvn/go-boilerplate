// Package http is the gateway's only layer: it turns REST calls into gRPC
// calls on the auth and file services. The gateway owns no data and no
// business rules; downstream services authorize every call themselves.
package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-boilerplate/internal/gateway/transport/http/middleware"
	"go-boilerplate/internal/golibs/redis"
	filev1 "go-boilerplate/pkg/genproto/file/v1"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type RateLimit struct {
	Requests int           `mapstructure:"requests"`
	Window   time.Duration `mapstructure:"window"`
}

type Router struct {
	Users     userv1.UserServiceClient
	Files     filev1.FileServiceClient
	Redis     *redis.RedisClient
	RateLimit RateLimit
}

func (rt *Router) Register(r *gin.Engine) {
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit(rt.Redis, rt.RateLimit.Requests, rt.RateLimit.Window))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", rt.register)
		auth.POST("/login", rt.login)
		auth.POST("/refresh-token", rt.refreshToken)
		auth.GET("/me", middleware.Auth(rt.Users), rt.me)

		files := v1.Group("/files", middleware.Auth(rt.Users))
		files.GET("", rt.listFiles)
		files.DELETE("", rt.deleteFile) // ?file_key=... (keys contain "/")
		files.GET("/presigned-url", rt.downloadURL)
		files.POST("/upload-presigned-url", rt.uploadURL)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorBody{Code: codes.NotFound.String(), Message: fmt.Sprintf("route %s %s not found", c.Request.Method, c.Request.URL.Path)})
	})
}

// downstream forwards the caller's Authorization header so downstream
// services verify the token themselves (zero-trust between services).
func downstream(c *gin.Context) context.Context {
	return metadata.AppendToOutgoingContext(c.Request.Context(), "authorization", c.GetHeader("Authorization"))
}

func (rt *Router) register(c *gin.Context) {
	var req userv1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortBadRequest(c, "invalid JSON body")
		return
	}
	resp, err := rt.Users.Register(c.Request.Context(), &req)
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (rt *Router) login(c *gin.Context) {
	var req userv1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortBadRequest(c, "invalid JSON body")
		return
	}
	resp, err := rt.Users.Login(c.Request.Context(), &req)
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (rt *Router) refreshToken(c *gin.Context) {
	var req userv1.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortBadRequest(c, "invalid JSON body")
		return
	}
	resp, err := rt.Users.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (rt *Router) me(c *gin.Context) {
	resp, err := rt.Users.GetProfile(downstream(c), &userv1.GetProfileRequest{}) // empty = caller
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (rt *Router) listFiles(c *gin.Context) {
	// Bad input parses as 0 and out-of-range input saturates at int32 bounds;
	// the file service applies its own defaults and caps.
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 32)
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)
	resp, err := rt.Files.GetAllFiles(downstream(c), &filev1.GetAllFilesRequest{
		Limit: int32(limit), Offset: int32(offset), Status: c.Query("status"),
	})
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (rt *Router) downloadURL(c *gin.Context) {
	key := c.Query("file_key")
	if key == "" {
		abortBadRequest(c, "file_key is required")
		return
	}
	expires, _ := strconv.ParseInt(c.Query("expires_in_seconds"), 10, 64)
	resp, err := rt.Files.GetPresignedUrl(downstream(c), &filev1.GetPresignedUrlRequest{FileKey: key, ExpiresInSeconds: expires})
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type uploadURLRequest struct {
	FileName         string `json:"file_name" binding:"required"`
	ContentType      string `json:"content_type"`
	Folder           string `json:"folder"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

func (rt *Router) uploadURL(c *gin.Context) {
	var req uploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortBadRequest(c, "file_name is required")
		return
	}
	resp, err := rt.Files.GetUploadPresignedUrl(downstream(c), &filev1.GetUploadPresignedUrlRequest{
		FileName: req.FileName, ContentType: req.ContentType, Folder: req.Folder, ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (rt *Router) deleteFile(c *gin.Context) {
	key := c.Query("file_key")
	if key == "" {
		abortBadRequest(c, "file_key is required")
		return
	}
	if _, err := rt.Files.DeleteFile(downstream(c), &filev1.DeleteFileRequest{FileKey: key}); err != nil {
		abortWithGRPCError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
