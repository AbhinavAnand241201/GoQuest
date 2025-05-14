package util

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// osExit is a variable that holds the function to exit the program.
// It is used for testing purposes.
var osExit = os.Exit

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
	osExit(1)
}

// NewError creates a new error with the given message.
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// IsTimeoutError checks if the error is a timeout error.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// JoinErrors combines multiple errors into a single error.
func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var errMsgs []string
	for _, err := range errs {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}
	if len(errMsgs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errMsgs, "; "))
}

// FormatScriptError formats a script error with file and line information.
func FormatScriptError(err error, file string, line int) string {
	if line > 0 {
		return fmt.Sprintf("%s:%d: %v", file, line, err)
	}
	return fmt.Sprintf("%s: %v", file, err)
}
