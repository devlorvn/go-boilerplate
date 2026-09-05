package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	t.Parallel()
	dummyErr := errors.New("dummy error")
	retryNum := 5

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		successfulFn := func() (int, error) {
			return 1, nil
		}
		data, err := Retry(retryNum, 1*time.Millisecond, successfulFn)
		assert.NoError(t, err)
		assert.Equal(t, 1, data)
	})

	t.Run("success after some retries", func(t *testing.T) {
		t.Parallel()
		failCount := 2
		successfulFn := func() (int, error) {
			if failCount > 0 {
				failCount--
				return 0, dummyErr
			}
			return 1, nil
		}
		data, err := Retry(retryNum, 1*time.Millisecond, successfulFn)
		assert.NoError(t, err)
		assert.Equal(t, 1, data)
	})

	t.Run("failure: exceeding retry limit", func(t *testing.T) {
		t.Parallel()
		alwaysFailedFn := func() (int, error) {
			return 0, dummyErr
		}
		_, err := Retry(retryNum, 1*time.Millisecond, alwaysFailedFn)
		assert.Error(t, err)
	})

	t.Run("failure: stop", func(t *testing.T) {
		t.Parallel()
		stopFn := func() (int, error) {
			return 0, NewStop(dummyErr)
		}
		_, err := Retry(retryNum, 1*time.Millisecond, stopFn)
		assert.Error(t, err)
	})
}
