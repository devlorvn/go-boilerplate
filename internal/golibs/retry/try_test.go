package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	t.Parallel()
	dummyErr := errors.New("dummy error")
	retryNum := 5

	assert.Greater(t, retryNum, 0)
	assert.LessOrEqual(t, retryNum, maxRetries)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		successfulFn := func(int) (bool, error) {
			return true, nil
		}
		assert.NoError(t, Do(successfulFn))
	})

	t.Run("success after some retries", func(t *testing.T) {
		t.Parallel()
		succeededAfterRetries := func(attempt int) (bool, error) {
			if attempt < retryNum {
				return true, dummyErr
			}
			return true, nil
		}

		assert.NoError(t, Do(succeededAfterRetries))
	})

	t.Run("failure: exceeding retry limit", func(t *testing.T) {
		t.Parallel()
		alwaysFailedFn := func(attempt int) (bool, error) {
			return true, dummyErr
		}
		assert.EqualError(t, Do(alwaysFailedFn), "exceeded retry limit")
	})

	t.Run("failure: function choosing to exit", func(t *testing.T) {
		t.Parallel()
		failAfterRetries := func(attempt int) (bool, error) {
			if attempt < retryNum {
				return true, dummyErr
			}
			return false, dummyErr
		}
		assert.EqualError(t, Do(failAfterRetries), "dummy error")
	})
}

func TestDoWithCtx(t *testing.T) {
	t.Parallel()
	dummyErr := errors.New("dummy error")
	retryNum := 5

	// retryNum must be greater than 0 and less than maxRetries for these tests
	assert.Greater(t, retryNum, 0)
	assert.LessOrEqual(t, retryNum, maxRetries)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		successfulFunc := func(context.Context, int) (bool, error) {
			return true, nil
		}

		ctx := context.Background()
		assert.NoError(t, DoWithCtx(ctx, successfulFunc))
	})

	t.Run("success after some retries", func(t *testing.T) {
		t.Parallel()
		succeedAfterRetries := func(_ context.Context, attempt int) (bool, error) {
			if attempt < retryNum {
				return true, dummyErr
			}
			return true, nil
		}
		ctx := context.Background()
		assert.NoError(t, DoWithCtx(ctx, succeedAfterRetries))
	})

	t.Run("failure: exceeding retry limit", func(t *testing.T) {
		t.Parallel()
		alwaysFailedFunc := func(_ context.Context, attempt int) (bool, error) {
			return true, errors.New("dummy error")
		}
		ctx := context.Background()
		assert.EqualError(t, DoWithCtx(ctx, alwaysFailedFunc), "exceeded retry limit")
	})

	t.Run("failure: function choosing to exit", func(t *testing.T) {
		t.Parallel()
		failAfterRetries := func(_ context.Context, attempt int) (bool, error) {
			if attempt < retryNum {
				return true, dummyErr
			}
			return false, dummyErr
		}
		ctx := context.Background()
		assert.EqualError(t, DoWithCtx(ctx, failAfterRetries), "dummy error")
	})

	t.Run("cancel", func(t *testing.T) {
		t.Parallel()
		retriedNum := 0
		fn := func(_ context.Context, attempt int) (bool, error) {
			if attempt < retryNum {
				time.Sleep(500 * time.Millisecond)
				retriedNum++
				return true, dummyErr
			}
			return true, nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(1200 * time.Millisecond)
			cancel()
		}()
		DoWithCtx(ctx, fn)
		assert.Equal(t, 3, retriedNum)
	})
}
