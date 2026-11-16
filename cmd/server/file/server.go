// Package file wires the file service into the bootstrap: `server gserver file`.
// This is the composition root: the only place that knows every layer.
package file

import (
	"context"
	"errors"
	"fmt"

	"go-boilerplate/internal/file/application/commands"
	"go-boilerplate/internal/file/application/queries"
	"go-boilerplate/internal/file/configurations"
	"go-boilerplate/internal/file/infrastructure/postgres"
	filestorage "go-boilerplate/internal/file/infrastructure/s3"
	filegrpc "go-boilerplate/internal/file/transport/grpc"
	"go-boilerplate/internal/file/transport/worker"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"
	s3lib "go-boilerplate/internal/golibs/s3"
	filev1 "go-boilerplate/pkg/genproto/file/v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/grpc"
)

func init() {
	bootstrap.Register[configurations.Config]("file", &server{})
}

type server struct {
	handler  *filegrpc.Handler
	verifier *jwtutil.HS256
}

func (s *server) InitDependencies(_ context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	if err := filegrpc.Policy.Validate(filev1.FileService_ServiceDesc); err != nil {
		return err
	}
	if c.Storage.Bucket == "" {
		return errors.New("storage.bucket is required")
	}
	if c.Cleanup.Interval <= 0 || c.Cleanup.PendingTimeout <= 0 {
		return errors.New("cleanup.interval and cleanup.pending_timeout must be > 0")
	}
	verifier, err := jwtutil.NewHS256(c.JWT.Secret)
	if err != nil {
		return fmt.Errorf("jwt.secret: %w", err)
	}
	db, err := rsc.DB()
	if err != nil {
		return err
	}

	s3Opts := s3.Options{
		Region:       c.Storage.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(c.Storage.AccessKeyID, c.Storage.SecretAccessKey, ""),
		UsePathStyle: true,
	}
	if c.Storage.Endpoint != "" {
		s3Opts.BaseEndpoint = aws.String(c.Storage.Endpoint)
	}

	files := postgres.NewFileRepository(db)
	storage := filestorage.NewStorage(s3lib.NewS3Client(s3.New(s3Opts), c.Storage.Bucket))

	s.verifier = verifier
	s.handler = &filegrpc.Handler{
		Commands: &commands.FileCommandHandler{Files: files, Storage: storage, Logger: rsc.Logger()},
		Queries:  &queries.FileQueryHandler{Files: files, Storage: storage},
	}
	return nil
}

func (s *server) RegisterGRPC(_ configurations.Config, srv *grpc.Server) {
	filev1.RegisterFileServiceServer(srv, s.handler)
}

func (s *server) UnaryInterceptors(_ configurations.Config, _ *bootstrap.Resources) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{interceptors.AuthInterceptor(s.verifier, filegrpc.Policy)}
}

func (s *server) RunBackground(ctx context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	(&worker.Cleanup{
		Expirer:   s.handler.Commands,
		Logger:    rsc.Logger(),
		Interval:  c.Cleanup.Interval,
		OlderThan: c.Cleanup.PendingTimeout,
	}).Run(ctx)
	return nil
}
