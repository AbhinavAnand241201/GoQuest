package task

import "time"

// TaskResult represents the result of a task execution
type TaskResult struct {
	Name     string
	Result   string
	Duration time.Duration
}

// TaskStatus represents the execution status of a task
type TaskStatus struct {
	Started   time.Time
	Completed time.Time
	Duration  time.Duration
	Error     error
	Result    string
}
