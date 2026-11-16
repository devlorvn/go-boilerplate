package retry

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"
)

type Stop struct {
	error
}

func NewStop(err error) Stop {
	return Stop{err}
}

func Retry[T any](attempts int, sleep time.Duration, f func() (T, error)) (T, error) {
	data, err := f()
	if err != nil {
		var s Stop
		if errors.As(err, &s) {
			return zero[T](), s.error
		}

		if attempts--; attempts > 0 {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(sleep)))
			jitter := time.Duration(n.Int64()) / 2
			sleep += jitter

			time.Sleep(sleep)
			return Retry(attempts, 2*sleep, f)
		}
		return zero[T](), err
	}

	return data, nil
}

func zero[T any]() T {
	var z T
	return z
}
