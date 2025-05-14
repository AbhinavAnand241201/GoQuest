package log

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	LevelInfo  LogLevel = "INFO"
	LevelError LogLevel = "ERROR"
	LevelDebug LogLevel = "DEBUG"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger handles structured logging with JSON support
type Logger struct {
	jsonOutput bool
}

// NewLogger creates a new Logger instance
func NewLogger(jsonOutput bool) *Logger {
	return &Logger{
		jsonOutput: jsonOutput,
	}
}

// log writes a log entry with the given level and message
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	if l.jsonOutput {
		// Output as JSON
		jsonData, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		// Output as human-readable format
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("%s [%s] %s", timestamp, level, message)
		if len(fields) > 0 {
			fmt.Printf(" %v", fields)
		}
		fmt.Println()
	}
}

// Info logs an informational message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(LevelInfo, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(LevelError, message, fields)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(LevelDebug, message, fields)
}

// SetJSONOutput enables or disables JSON output
func (l *Logger) SetJSONOutput(jsonOutput bool) {
	l.jsonOutput = jsonOutput
}
