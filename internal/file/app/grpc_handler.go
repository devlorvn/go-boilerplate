package app

import (
	"context"
	"time"

	"go-boilerplate/internal/file/usecase"
	filev1 "go-boilerplate/pkg/genproto/file/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	filev1.UnimplementedFileServiceServer
	usecase *usecase.FileUsecase
}

func NewGRPCHandler(u *usecase.FileUsecase) *GRPCHandler {
	return &GRPCHandler{usecase: u}
}

func (h *GRPCHandler) UploadFile(ctx context.Context, req *filev1.UploadFileRequest) (*filev1.UploadFileResponse, error) {
	file, err := h.usecase.UploadFile(ctx, req.FileName, req.Content, req.ContentType, req.Folder)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &filev1.UploadFileResponse{
		FileUrl: file.URL,
		FileKey: file.FileKey,
	}, nil
}

func (h *GRPCHandler) GetPresignedUrl(ctx context.Context, req *filev1.GetPresignedUrlRequest) (*filev1.GetPresignedUrlResponse, error) {
	url, err := h.usecase.GetPresignedUrl(ctx, req.FileKey, req.ExpiresInSeconds)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &filev1.GetPresignedUrlResponse{
		PresignedUrl: url,
	}, nil
}

func (h *GRPCHandler) GetUploadPresignedUrl(ctx context.Context, req *filev1.GetUploadPresignedUrlRequest) (*filev1.GetUploadPresignedUrlResponse, error) {
	uploadURL, fileKey, err := h.usecase.GetUploadPresignedUrl(ctx, req.FileName, req.ContentType, req.Folder, req.ExpiresInSeconds)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &filev1.GetUploadPresignedUrlResponse{
		UploadUrl: uploadURL,
		FileKey:   fileKey,
	}, nil
}

func (h *GRPCHandler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	if err := h.usecase.DeleteFile(ctx, req.FileKey); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &filev1.DeleteFileResponse{
		Success: true,
	}, nil
}

func (h *GRPCHandler) GetAllFiles(ctx context.Context, req *filev1.GetAllFilesRequest) (*filev1.GetAllFilesResponse, error) {
	files, total, err := h.usecase.GetAllFiles(ctx, int(req.Limit), int(req.Offset), req.Status)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	records := make([]*filev1.FileRecord, len(files))
	for i, f := range files {
		records[i] = &filev1.FileRecord{
			Id:          f.ID,
			FileKey:     f.FileKey,
			FileName:    f.FileName,
			FileSize:    f.FileSize,
			ContentType: f.ContentType,
			Status:      string(f.Status),
			Url:         f.URL,
			CreatedAt:   f.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &filev1.GetAllFilesResponse{
		Files: records,
		Total: total,
	}, nil
}
