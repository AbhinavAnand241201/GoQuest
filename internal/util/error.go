package util

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

// WrapError wraps an error with additional context.
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return errors.Wrap(err, context)
}

// ExitWithError logs the error and exits with status code 1.
func ExitWithError(err error) {
	if err != nil {
		LogFatal("%v", err)
	}
}

// NewError creates a new error with the given message.
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// IsTimeoutError checks if the error is a timeout error.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
