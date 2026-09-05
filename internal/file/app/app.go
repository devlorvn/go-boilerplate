package app

import (
	"google.golang.org/grpc"

	"go-boilerplate/internal/file/usecase"
	filev1 "go-boilerplate/pkg/genproto/file/v1"
)

type App struct {
	fileUsecase *usecase.FileUsecase
	grpcHandler *GRPCHandler
}

func NewApp(fileUsecase *usecase.FileUsecase) *App {
	return &App{
		fileUsecase: fileUsecase,
		grpcHandler: NewGRPCHandler(fileUsecase),
	}
}

func (a *App) RegisterGRPC(server *grpc.Server) {
	filev1.RegisterFileServiceServer(server, a.grpcHandler)
}
