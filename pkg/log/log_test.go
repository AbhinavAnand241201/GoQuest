package log

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLogger_Log(t *testing.T) {
	tests := []struct {
		name       string
		jsonOutput bool
		level      LogLevel
		message    string
		fields     map[string]interface{}
	}{
		{
			name:       "JSON output",
			jsonOutput: true,
			level:      LevelInfo,
			message:    "test message",
			fields: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:       "Human-readable output",
			jsonOutput: false,
			level:      LevelError,
			message:    "error message",
			fields: map[string]interface{}{
				"error": "test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.jsonOutput)
			logger.log(tt.level, tt.message, tt.fields)
		})
	}
}

func TestLogEntry_MarshalJSON(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Level:     LevelInfo,
		Message:   "test message",
		Fields: map[string]interface{}{
			"key": "value",
		},
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Errorf("Error marshaling LogEntry: %v", err)
	}

	var unmarshaled LogEntry
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Errorf("Error unmarshaling LogEntry: %v", err)
	}

	if unmarshaled.Level != entry.Level {
		t.Errorf("Expected level %s, got %s", entry.Level, unmarshaled.Level)
	}
	if unmarshaled.Message != entry.Message {
		t.Errorf("Expected message %s, got %s", entry.Message, unmarshaled.Message)
	}
	if unmarshaled.Fields["key"] != entry.Fields["key"] {
		t.Errorf("Expected field value %v, got %v", entry.Fields["key"], unmarshaled.Fields["key"])
	}
}

func TestLogger_SetJSONOutput(t *testing.T) {
	logger := NewLogger(false)
	if logger.jsonOutput {
		t.Error("Expected jsonOutput to be false")
	}

	logger.SetJSONOutput(true)
	if !logger.jsonOutput {
		t.Error("Expected jsonOutput to be true")
	}
}
