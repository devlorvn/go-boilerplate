// Package grpc adapts the file use cases to file.v1.FileService.
package grpc

import (
	"context"
	"time"

	"go-boilerplate/internal/file/application/commands"
	"go-boilerplate/internal/file/application/queries"
	"go-boilerplate/internal/file/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/interceptors"
	filev1 "go-boilerplate/pkg/genproto/file/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Policy is the RBAC table of this service. Per-file ownership is enforced in
// the application layer, so every RPC only requires a valid token here.
var Policy = authz.Policy{
	filev1.FileService_UploadFile_FullMethodName:            authz.Authenticated,
	filev1.FileService_GetPresignedUrl_FullMethodName:       authz.Authenticated,
	filev1.FileService_GetUploadPresignedUrl_FullMethodName: authz.Authenticated,
	filev1.FileService_DeleteFile_FullMethodName:            authz.Authenticated,
	filev1.FileService_GetAllFiles_FullMethodName:           authz.Authenticated,
}

type Handler struct {
	filev1.UnimplementedFileServiceServer
	Commands *commands.FileCommandHandler
	Queries  *queries.FileQueryHandler
}

func actor(ctx context.Context) (authz.Actor, error) {
	a, ok := interceptors.ActorFromContext(ctx)
	if !ok {
		return a, status.Error(codes.Unauthenticated, "missing caller")
	}
	return a, nil
}

func (h *Handler) UploadFile(ctx context.Context, req *filev1.UploadFileRequest) (*filev1.UploadFileResponse, error) {
	a, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	f, err := h.Commands.Upload(ctx, a, commands.UploadCommand{
		FileName: req.FileName, ContentType: req.ContentType, Folder: req.Folder, Content: req.Content,
	})
	if err != nil {
		return nil, err
	}
	return &filev1.UploadFileResponse{FileUrl: f.URL, FileKey: f.Key}, nil
}

func (h *Handler) GetUploadPresignedUrl(ctx context.Context, req *filev1.GetUploadPresignedUrlRequest) (*filev1.GetUploadPresignedUrlResponse, error) {
	a, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.Commands.RequestUpload(ctx, a, commands.RequestUploadCommand{
		FileName: req.FileName, ContentType: req.ContentType, Folder: req.Folder, TTLSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		return nil, err
	}
	return &filev1.GetUploadPresignedUrlResponse{UploadUrl: t.UploadURL, FileKey: t.Key}, nil
}

func (h *Handler) GetPresignedUrl(ctx context.Context, req *filev1.GetPresignedUrlRequest) (*filev1.GetPresignedUrlResponse, error) {
	a, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	url, err := h.Queries.DownloadURL(ctx, a, req.FileKey, req.ExpiresInSeconds)
	if err != nil {
		return nil, err
	}
	return &filev1.GetPresignedUrlResponse{PresignedUrl: url}, nil
}

func (h *Handler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	a, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.Commands.Delete(ctx, a, req.FileKey); err != nil {
		return nil, err
	}
	return &filev1.DeleteFileResponse{Success: true}, nil
}

func (h *Handler) GetAllFiles(ctx context.Context, req *filev1.GetAllFilesRequest) (*filev1.GetAllFilesResponse, error) {
	a, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	files, total, err := h.Queries.List(ctx, a, queries.ListQuery{Status: req.Status, Limit: int(req.Limit), Offset: int(req.Offset)})
	if err != nil {
		return nil, err
	}
	out := make([]*filev1.FileRecord, len(files))
	for i, f := range files {
		out[i] = toRecord(f)
	}
	return &filev1.GetAllFilesResponse{Files: out, Total: total}, nil
}

func toRecord(f *domain.File) *filev1.FileRecord {
	return &filev1.FileRecord{
		Id:          f.ID,
		FileKey:     f.Key,
		FileName:    f.Name,
		FileSize:    f.Size,
		ContentType: f.ContentType,
		Status:      string(f.Status),
		Url:         f.URL,
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
	}
}
