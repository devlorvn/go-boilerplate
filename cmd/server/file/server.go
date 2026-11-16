// Package file wires the file service into the bootstrap: `server gserver file`.
package file

import (
	"context"
	"errors"
	"fmt"

	fileapp "go-boilerplate/internal/file/app"
	"go-boilerplate/internal/file/configurations"
	"go-boilerplate/internal/file/infras"
	"go-boilerplate/internal/file/usecase"
	"go-boilerplate/internal/file/worker"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"
	s3lib "go-boilerplate/internal/golibs/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/grpc"
)

func init() {
	bootstrap.Register[configurations.Config]("file", &server{})
}

type server struct {
	app      *fileapp.App
	usecase  *usecase.FileUsecase
	verifier *jwtutil.HS256
}

func (s *server) InitDependencies(_ context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	verifier, err := jwtutil.NewHS256(c.JWT.Secret)
	if err != nil {
		return fmt.Errorf("jwt.secret: %w", err)
	}
	if c.Storage.Bucket == "" {
		return errors.New("storage.bucket is required")
	}
	db, err := rsc.DB()
	if err != nil {
		return err
	}
	fileRepo, err := infras.NewGORMFileRepository(db)
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
	storage := infras.NewS3StorageRepository(s3lib.NewS3Client(s3.New(s3Opts), c.Storage.Bucket))

	s.verifier = verifier
	s.usecase = usecase.NewFileUsecase(fileRepo, storage)
	s.app = fileapp.NewApp(s.usecase)
	return nil
}

func (s *server) RegisterGRPC(_ configurations.Config, srv *grpc.Server) {
	s.app.RegisterGRPC(srv)
}

func (s *server) UnaryInterceptors(_ configurations.Config, _ *bootstrap.Resources) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{interceptors.AuthInterceptor(s.verifier)}
}

// RunBackground runs the expired-pending-files cleanup until shutdown.
func (s *server) RunBackground(ctx context.Context, c configurations.Config, rsc *bootstrap.Resources) error {
	w := worker.NewCleanupWorker(s.usecase, rsc.Logger(), c.Cleanup.Interval)
	w.Start(ctx)
	<-ctx.Done()
	return nil
}
