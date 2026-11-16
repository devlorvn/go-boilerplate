// Package auth wires the auth service into the bootstrap: `server gserver auth`.
package auth

import (
	"context"
	"errors"
	"fmt"

	authapp "go-boilerplate/internal/auth/app"
	"go-boilerplate/internal/auth/configurations"
	"go-boilerplate/internal/auth/infras"
	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"

	"google.golang.org/grpc"
)

func init() {
	bootstrap.Register[configurations.Config]("auth", &server{})
}

type server struct {
	app    *authapp.App
	signer *jwtutil.HS256
}

func (s *server) InitDependencies(ctx context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
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
	km, err := rsc.Kafka()
	switch {
	case errors.Is(err, bootstrap.ErrKafkaDisabled):
		rsc.Logger().Warn("kafka disabled, user events will not be published")
	case err != nil:
		return err
	}

	userRepo, err := infras.NewGORMUserRepository(db)
	if err != nil {
		return err
	}
	uc := usecase.NewAuthUsecase(
		userRepo,
		infras.NewRedisTokenStore(rdb),
		infras.NewJWTManager(signer),
		infras.NewKafkaEventPublisher(km), // no-op when kafka is disabled (km == nil)
	)

	s.signer = signer
	s.app = authapp.NewApp(uc)
	return nil
}

func (s *server) RegisterGRPC(_ configurations.Config, srv *grpc.Server) {
	s.app.RegisterGRPC(srv)
}

func (s *server) UnaryInterceptors(_ configurations.Config, _ *bootstrap.Resources) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{interceptors.AuthInterceptor(s.signer, authapp.PublicMethods...)}
}
