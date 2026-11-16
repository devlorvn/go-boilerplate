// Package errorx gives domain and application errors a transport-neutral
// Kind. Domain code declares errors once:
//
//	var ErrUserNotFound = errorx.New(errorx.NotFound, "user not found")
//
// and transports translate the Kind (gRPC codes here, HTTP in the gateway).
// Errors without a Kind are Internal: their message is never sent to clients.
package errorx

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Kind int

const (
	Internal Kind = iota
	InvalidArgument
	NotFound
	AlreadyExists
	Unauthenticated
	PermissionDenied
	FailedPrecondition
)

// Error is a client-safe error: Msg may be shown to the caller.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func New(kind Kind, msg string) *Error { return &Error{Kind: kind, Msg: msg} }

// Invalid builds an InvalidArgument error, e.g. for request validation.
func Invalid(format string, args ...any) *Error {
	return &Error{Kind: InvalidArgument, Msg: fmt.Sprintf(format, args...)}
}

// KindOf returns the Kind of the first *Error in err's chain, or Internal.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return Internal
}

var grpcCodes = map[Kind]codes.Code{
	InvalidArgument:    codes.InvalidArgument,
	NotFound:           codes.NotFound,
	AlreadyExists:      codes.AlreadyExists,
	Unauthenticated:    codes.Unauthenticated,
	PermissionDenied:   codes.PermissionDenied,
	FailedPrecondition: codes.FailedPrecondition,
}

// ToGRPC converts err to a gRPC status. Existing statuses pass through;
// Internal errors become a generic message so DB/driver details never leak.
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	var e *Error
	if errors.As(err, &e) {
		if code, ok := grpcCodes[e.Kind]; ok {
			return status.Error(code, e.Msg)
		}
	}
	return status.Error(codes.Internal, "internal error")
}
