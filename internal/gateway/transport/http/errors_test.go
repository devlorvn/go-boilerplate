package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAbortWithGRPCError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		err      error
		wantHTTP int
		wantMsg  string
	}{
		{status.Error(codes.InvalidArgument, "bad email"), http.StatusBadRequest, "bad email"},
		{status.Error(codes.Unauthenticated, "nope"), http.StatusUnauthorized, "nope"},
		{status.Error(codes.PermissionDenied, "no"), http.StatusForbidden, "no"},
		{status.Error(codes.NotFound, "file not found"), http.StatusNotFound, "file not found"},
		{status.Error(codes.AlreadyExists, "email already exists"), http.StatusConflict, "email already exists"},
		{status.Error(codes.Unavailable, "dial tcp 10.0.0.5:50051: refused"), http.StatusServiceUnavailable, "upstream service unavailable"},
		{status.Error(codes.Internal, "pq: secret detail"), http.StatusInternalServerError, "internal error"},
		{errors.New("not a status"), http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		abortWithGRPCError(c, tc.err)

		var body ErrorBody
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != tc.wantHTTP || body.Message != tc.wantMsg {
			t.Errorf("%v -> %d %q, want %d %q", tc.err, w.Code, body.Message, tc.wantHTTP, tc.wantMsg)
		}
	}
}
