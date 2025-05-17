package task

import (
	"time"
)

// TaskResult represents the result of a task execution
type TaskResult struct {
	Name     string
	Result   interface{}
	Duration time.Duration
	Error    error
	// Status is a backward compatibility field for tests
	Status   string
}

// GetStatus returns the status of the task based on its error field
func (r TaskResult) GetStatus() string {
	if r.Status != "" {
		return r.Status
	}
	if r.Error != nil {
		return "failed"
	}
	return "success"
}

// TaskStatus represents the execution status of a task
type TaskStatus struct {
	Started   time.Time
	Completed time.Time
	Duration  time.Duration
	Error     error
	Result    interface{}
}
