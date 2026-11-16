package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorBody is the single error shape returned by the gateway.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var grpcToHTTP = map[codes.Code]int{
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.FailedPrecondition: http.StatusPreconditionFailed,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.Unavailable:        http.StatusServiceUnavailable,
}

// abortWithGRPCError maps a downstream gRPC error to an HTTP response.
// Messages of unmapped codes are replaced so upstream internals never leak.
func abortWithGRPCError(c *gin.Context, err error) {
	st := status.Convert(err)
	httpStatus, ok := grpcToHTTP[st.Code()]
	msg := st.Message()
	if !ok {
		httpStatus, msg = http.StatusInternalServerError, "internal error"
	}
	if st.Code() == codes.Unavailable {
		msg = "upstream service unavailable"
	}
	_ = c.Error(err) // keep the original for access logs
	c.AbortWithStatusJSON(httpStatus, ErrorBody{Code: st.Code().String(), Message: msg})
}

func abortBadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorBody{Code: codes.InvalidArgument.String(), Message: msg})
}
