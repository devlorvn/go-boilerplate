// Package auth wires the auth service into the bootstrap: `server gserver auth`.
// This is the composition root: the only place that knows every layer.
package auth

import (
	"context"
	"errors"
	"fmt"

	"go-boilerplate/internal/auth/application/commands"
	"go-boilerplate/internal/auth/application/queries"
	"go-boilerplate/internal/auth/configurations"
	"go-boilerplate/internal/auth/infrastructure/kafka"
	"go-boilerplate/internal/auth/infrastructure/postgres"
	"go-boilerplate/internal/auth/infrastructure/redis"
	authgrpc "go-boilerplate/internal/auth/transport/grpc"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"google.golang.org/grpc"
)

func init() {
	bootstrap.Register[configurations.Config]("auth", &server{})
}

type server struct {
	handler *authgrpc.Handler
	signer  *jwtutil.HS256
}

func (s *server) InitDependencies(ctx context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	if err := authgrpc.Policy.Validate(userv1.UserService_ServiceDesc); err != nil {
		return err
	}
	if c.Tokens.AccessTTL <= 0 || c.Tokens.RefreshTTL <= 0 {
		return errors.New("tokens.access_ttl and tokens.refresh_ttl must be > 0")
	}
	signer, err := jwtutil.NewHS256(c.JWT.Secret)
	if err != nil {
		return fmt.Errorf("jwt.secret: %w", err)
	}
	db, err := rsc.DB()
	if err != nil {
		return err
	}
	rdb, err := rsc.Redis(ctx)
	if err != nil {
		return err
	}

	users := postgres.NewUserRepository(db)
	cmds := &commands.AuthCommandHandler{
		Users:   users,
		Refresh: redis.NewRefreshTokenStore(rdb),
		Signer:  signer,
		TTL:     commands.TokenTTL{Access: c.Tokens.AccessTTL, Refresh: c.Tokens.RefreshTTL},
		Logger:  rsc.Logger(),
	}
	switch km, err := rsc.Kafka(); {
	case errors.Is(err, bootstrap.ErrKafkaDisabled):
		rsc.Logger().Warn("kafka disabled, user events will not be published")
	case err != nil:
		return err
	default:
		cmds.Events = kafka.NewUserEventPublisher(km)
	}

	s.signer = signer
	s.handler = &authgrpc.Handler{
		Commands: cmds,
		Queries:  &queries.UserQueryHandler{Users: users, Signer: signer},
	}
	return nil
}

func (s *server) RegisterGRPC(_ configurations.Config, srv *grpc.Server) {
	userv1.RegisterUserServiceServer(srv, s.handler)
}

func (s *server) UnaryInterceptors(_ configurations.Config, _ *bootstrap.Resources) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{interceptors.AuthInterceptor(s.signer, authgrpc.Policy)}
}
