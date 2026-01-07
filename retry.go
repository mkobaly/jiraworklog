package jiraworklog

import (
	"errors"
	"fmt"
	"time"
)

// Retry returns a value of type T or the last error encountered.
// fn is executed up to `attempts` times with `delay` between attempts.
func Retry[T any](attempts int, delay time.Duration, fn func() (T, error)) (T, error) {
	var zero T

	if attempts <= 0 {
		return zero, errors.New("attempts must be > 0")
	}

	var (
		val T
		err error
	)

	for i := 0; i < attempts; i++ {
		val, err = fn()
		if err == nil {
			return val, nil
		}

		if i < attempts-1 {
			fmt.Println("retrying", err)
			time.Sleep(delay)
		}
	}

	return zero, err
}
