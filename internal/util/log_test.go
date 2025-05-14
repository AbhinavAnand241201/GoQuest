package util

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestLogInfo_Format(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Test message
	msg := "test message"
	LogInfo(msg)

	// Verify output format
	output := buf.String()
	if !strings.Contains(output, msg) {
		t.Errorf("LogInfo() output = %v, want to contain %v", output, msg)
	}
	if !strings.Contains(output, "INFO:") {
		t.Errorf("LogInfo() output = %v, want to contain INFO:", output)
	}
}

func TestLogInfo_Concurrent(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Run concurrent logging
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			LogInfo("test message " + fmt.Sprintf("%d", i))
		}(i)
	}
	wg.Wait()

	// Verify output
	output := buf.String()
	lines := strings.Split(output, "\n")
	if len(lines) < 100 {
		t.Errorf("LogInfo() concurrent output has %d lines, want at least 100", len(lines))
	}
}

func TestLogDebug_VerboseOn(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Enable verbose mode
	SetVerbose(true)
	defer SetVerbose(false)

	// Test message
	msg := "debug message"
	LogDebug(msg)

	// Verify output
	output := buf.String()
	if !strings.Contains(output, msg) {
		t.Errorf("LogDebug() output = %v, want to contain %v", output, msg)
	}
	if !strings.Contains(output, "DEBUG:") {
		t.Errorf("LogDebug() output = %v, want to contain DEBUG:", output)
	}
}

func TestLogDebug_VerboseOff(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Disable verbose mode
	SetVerbose(false)

	// Test message
	msg := "debug message"
	LogDebug(msg)

	// Verify no output
	output := buf.String()
	if output != "" {
		t.Errorf("LogDebug() output = %v, want empty string", output)
	}
}

func TestLogInfo_EmptyMessage(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Test empty message
	LogInfo("")

	// Verify output
	output := buf.String()
	if !strings.Contains(output, "INFO:") {
		t.Errorf("LogInfo() output = %v, want to contain INFO:", output)
	}
}

func TestLogInfo_LongMessage(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Create long message (10KB)
	longMsg := strings.Repeat("a", 10*1024)
	LogInfo(longMsg)

	// Verify output
	output := buf.String()
	if !strings.Contains(output, longMsg) {
		t.Errorf("LogInfo() output does not contain long message")
	}
}

func TestSetVerbose_Toggle(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	// Test verbose on
	SetVerbose(true)
	LogDebug("debug1")
	output1 := buf.String()
	if !strings.Contains(output1, "debug1") {
		t.Errorf("LogDebug() with verbose=true: output = %v, want to contain debug1", output1)
	}

	// Clear buffer
	buf.Reset()

	// Test verbose off
	SetVerbose(false)
	LogDebug("debug2")
	output2 := buf.String()
	if output2 != "" {
		t.Errorf("LogDebug() with verbose=false: output = %v, want empty string", output2)
	}
}
