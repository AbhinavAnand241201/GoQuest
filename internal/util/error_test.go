package util

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestWrapError_Context(t *testing.T) {
	original := errors.New("original error")
	context := "test context"
	wrapped := WrapError(original, context)

	if wrapped == nil {
		t.Error("WrapError() returned nil")
	}
	if wrapped.Error() != context+": "+original.Error() {
		t.Errorf("WrapError() = %v, want %v", wrapped.Error(), context+": "+original.Error())
	}
}

func TestWrapError_Nil(t *testing.T) {
	wrapped := WrapError(nil, "context")
	if wrapped != nil {
		t.Errorf("WrapError(nil) = %v, want nil", wrapped)
	}
}

func TestJoinErrors_Multiple(t *testing.T) {
	errs := []error{
		errors.New("error 1"),
		errors.New("error 2"),
		errors.New("error 3"),
	}
	joined := JoinErrors(errs)

	if joined == nil {
		t.Error("JoinErrors() returned nil")
	}
	joinedMsg := joined.Error()
	for _, err := range errs {
		if !strings.Contains(joinedMsg, err.Error()) {
			t.Errorf("JoinErrors() message does not contain %v", err.Error())
		}
	}
}

func TestJoinErrors_Empty(t *testing.T) {
	joined := JoinErrors(nil)
	if joined != nil {
		t.Errorf("JoinErrors(nil) = %v, want nil", joined)
	}

	joined = JoinErrors([]error{})
	if joined != nil {
		t.Errorf("JoinErrors([]) = %v, want nil", joined)
	}
}

func TestFormatScriptError_WithLine(t *testing.T) {
	err := errors.New("parse error")
	file := "test.go"
	line := 42
	formatted := FormatScriptError(err, file, line)

	expected := fmt.Sprintf("%s:%d: %s", file, line, err.Error())
	if formatted != expected {
		t.Errorf("FormatScriptError() = %v, want %v", formatted, expected)
	}
}

func TestFormatScriptError_NoLine(t *testing.T) {
	err := errors.New("parse error")
	file := "test.go"
	formatted := FormatScriptError(err, file, 0)

	expected := file + ": " + err.Error()
	if formatted != expected {
		t.Errorf("FormatScriptError() = %v, want %v", formatted, expected)
	}
}

func TestExitWithError_Logs(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Mock os.Exit
	originalExit := os.Exit
	var exitCode int
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = originalExit }()

	// Test error
	err := errors.New("test error")
	ExitWithError(err)

	// Verify output
	output := buf.String()
	if !strings.Contains(output, err.Error()) {
		t.Errorf("ExitWithError() output = %v, want to contain %v", output, err.Error())
	}
	if exitCode != 1 {
		t.Errorf("ExitWithError() exit code = %v, want 1", exitCode)
	}
}

func TestExitWithError_Concurrent(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Mock osExit to a no-op (set once before goroutines)
	originalExit := osExit
	osExit = func(code int) {}
	defer func() { osExit = originalExit }()

	// Run concurrent exits
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ExitWithError(errors.New("error " + fmt.Sprintf("%d", i)))
		}(i)
	}
	wg.Wait()

	// Verify output
	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Errorf("ExitWithError() concurrent output = %v, want to contain error", output)
	}
}
