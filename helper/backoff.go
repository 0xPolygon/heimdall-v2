package helper

import (
	"time"
)

// ExponentialBackoff performs exponential backoff attempts on a given action
func ExponentialBackoff(action func() error, max uint, wait time.Duration) error {
	var err error

	if max > 10 {
		max = 10
	}

	for i := uint(0); i < max; i++ {
		if err = action(); err == nil {
			break
		}

		time.Sleep(wait)
		wait *= 2
	}

	return err
}
