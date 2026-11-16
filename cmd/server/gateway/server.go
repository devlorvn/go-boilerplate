// Package gateway wires the HTTP API gateway into the bootstrap: `server gserver gateway`.
package gateway

import (
	"context"
	"errors"
	"fmt"

	"go-boilerplate/internal/gateway/app"
	"go-boilerplate/internal/gateway/configurations"
	"go-boilerplate/internal/golibs/bootstrap"
	filev1 "go-boilerplate/pkg/genproto/file/v1"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	bootstrap.Register[configurations.Config]("gateway", &server{})
}

type server struct {
	app   *app.GatewayApp
	conns []*grpc.ClientConn
}

func (s *server) InitDependencies(ctx context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	if c.Upstreams.AuthAddr == "" || c.Upstreams.FileAddr == "" {
		return errors.New("upstreams.auth_addr and upstreams.file_addr are required")
	}
	rdb, err := rsc.Redis(ctx)
	if err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	authConn, err := s.dial(c.Upstreams.AuthAddr)
	if err != nil {
		return err
	}
	fileConn, err := s.dial(c.Upstreams.FileAddr)
	if err != nil {
		return err
	}

	s.app = app.NewGatewayApp(rsc.Logger(), rdb,
		userv1.NewUserServiceClient(authConn),
		filev1.NewFileServiceClient(fileConn),
		c.RateLimit,
	)
	return nil
}

// dial is lazy (grpc.NewClient does not connect), so upstreams may start later.
func (s *server) dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc client %s: %w", addr, err)
	}
	s.conns = append(s.conns, conn)
	return conn, nil
}

func (s *server) RegisterHTTP(_ configurations.Config, r *gin.Engine) {
	s.app.RegisterHTTP(r)
}

func (s *server) Shutdown(context.Context) {
	for _, c := range s.conns {
		_ = c.Close()
	}
}
