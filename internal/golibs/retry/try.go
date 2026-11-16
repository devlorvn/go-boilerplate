package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var maxRetries = 10

var ErrMaxRetriesReached = errors.New("exceeded retry limit")

type retryableFn func(attempt int) (retry bool, err error)

type retryableFnWithCtx func(ctx context.Context, attempt int) (retry bool, err error)

func Do(fn retryableFn) error {
	var err error
	var cont bool

	attempt := 1

	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++

		if attempt > maxRetries {
			return ErrMaxRetriesReached
		}
	}
	return err
}

func DoWithCtx(ctx context.Context, fn retryableFnWithCtx) error {
	var err error
	var cont bool
	attempt := 1
	for {
		select {
		case <-ctx.Done():
			return err
		default:
			cont, err = fn(ctx, attempt)
			if !cont || err == nil {
				return err
			}
			attempt++
			if attempt > maxRetries {
				return ErrMaxRetriesReached
			}
		}
	}
}

func DoBackOff(fn retryableFn, dur time.Duration) error {
	var err error
	var cont bool
	attempt := 1
	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++
		if attempt > maxRetries {
			return fmt.Errorf("%w: %w", ErrMaxRetriesReached, err)
		}
		time.Sleep(dur)
	}
	return err
}
