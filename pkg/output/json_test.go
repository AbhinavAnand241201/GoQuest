package output

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

func TestConvertTaskResults(t *testing.T) {
	results := []task.TaskResult{
		{
			Name:     "task1",
			Result:   "Hello",
			Error:    nil,
			Duration: 100 * time.Millisecond,
		},
		{
			Name:     "task2",
			Result:   nil,
			Error:    errors.New("task failed"),
			Duration: 200 * time.Millisecond,
		},
	}

	jsonResults := ConvertTaskResults(results)
	if len(jsonResults) != 2 {
		t.Errorf("Expected 2 results, got %d", len(jsonResults))
	}

	// Test first result
	if jsonResults[0].Name != "task1" {
		t.Errorf("Expected name 'task1', got %s", jsonResults[0].Name)
	}
	if jsonResults[0].Status != "success" {
		t.Errorf("Expected status 'success', got %s", jsonResults[0].Status)
	}
	if jsonResults[0].Result != "Hello" {
		t.Errorf("Expected result 'Hello', got %v", jsonResults[0].Result)
	}
	if jsonResults[0].Error != "" {
		t.Errorf("Expected empty error, got %s", jsonResults[0].Error)
	}
	if jsonResults[0].Duration != "100ms" {
		t.Errorf("Expected duration '100ms', got %s", jsonResults[0].Duration)
	}

	// Test second result
	if jsonResults[1].Name != "task2" {
		t.Errorf("Expected name 'task2', got %s", jsonResults[1].Name)
	}
	if jsonResults[1].Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", jsonResults[1].Status)
	}
	if jsonResults[1].Result != nil {
		t.Errorf("Expected nil result, got %v", jsonResults[1].Result)
	}
	if jsonResults[1].Error != "task failed" {
		t.Errorf("Expected error 'task failed', got %s", jsonResults[1].Error)
	}
	if jsonResults[1].Duration != "200ms" {
		t.Errorf("Expected duration '200ms', got %s", jsonResults[1].Duration)
	}
}

func TestPrintError(t *testing.T) {
	err := errors.New("test error")
	jsonData, err := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
	if err != nil {
		t.Errorf("Error marshaling error: %v", err)
	}

	// We expect the output to be this JSON string plus a newline
	_ = string(jsonData) + "\n"
	// Note: We can't easily test the actual output since it writes to stderr
	// This test just ensures the function doesn't panic
	PrintError(err)
}
