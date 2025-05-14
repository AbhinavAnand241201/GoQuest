package task

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// TaskResult represents the result of a task execution.
type TaskResult struct {
	Name     string
	Result   interface{}
	Error    error
	Duration time.Duration
	Status   string // "success", "failed", "cancelled"
}

// Executor manages the execution of tasks
type Executor struct {
	// Channel for task results
	results chan TaskResult
	// Channel for task errors
	errors chan error
	// Wait group for task completion
	wg sync.WaitGroup
	// Mutex for concurrent access
	mu sync.RWMutex
	// Map of task name to execution status
	status map[string]TaskStatus
	// List of tasks to execute
	tasks []Task
	// Number of concurrent workers
	concurrency int
	// Mutex for results
	resultMutex sync.Mutex
	// List of results
	resultList []TaskResult
}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{
		results:     make(chan TaskResult),
		errors:      make(chan error),
		status:      make(map[string]TaskStatus),
		tasks:       make([]Task, 0),
		concurrency: runtime.NumCPU(),
		resultList:  make([]TaskResult, 0),
	}
}

// ExecuteTask executes a task and returns its result
func (e *Executor) ExecuteTask(ctx context.Context, task Task) (TaskResult, error) {
	// Record start time
	start := time.Now()

	// Execute task
	result, err := task.Run(ctx)
	duration := time.Since(start)

	// Record completion
	e.mu.Lock()
	e.status[task.Name()] = TaskStatus{
		Started:   start,
		Completed: time.Now(),
		Duration:  duration,
		Error:     err,
		Result:    result.Result,
	}
	e.mu.Unlock()

	if err != nil {
		return TaskResult{}, fmt.Errorf("task %s failed: %w", task.Name(), err)
	}

	return result, nil
}

// GetTaskStatus returns the execution status of a task
func (e *Executor) GetTaskStatus(taskName string) (TaskStatus, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	status, exists := e.status[taskName]
	return status, exists
}

// Run executes all tasks concurrently and returns their results.
func (e *Executor) Run(ctx context.Context) []TaskResult {
	taskCh := make(chan Task, len(e.tasks))
	resultCh := make(chan TaskResult, len(e.tasks))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < e.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				result, err := task.Run(ctx)
				if err != nil {
					result.Error = err
				}
				resultCh <- result
			}
		}()
	}

	// Send tasks to workers
	for _, task := range e.tasks {
		taskCh <- task
	}
	close(taskCh)

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Process results
	for result := range resultCh {
		e.resultMutex.Lock()
		e.resultList = append(e.resultList, result)
		e.resultMutex.Unlock()
	}

	return e.resultList
}

// AggregateErrors combines task errors into a single error.
func (e *Executor) AggregateErrors(results []TaskResult) error {
	var errors []error
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("%s: %w", result.Name, result.Error))
		}
	}
	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("%d tasks failed: %v", len(errors), errors)
}

// GetFailedTasks returns a list of task names that failed.
func (e *Executor) GetFailedTasks(results []TaskResult) []string {
	var failed []string
	for _, result := range results {
		if result.Error != nil {
			failed = append(failed, result.Name)
		}
	}
	return failed
}

// GetSuccessfulTasks returns a list of task names that succeeded.
func (e *Executor) GetSuccessfulTasks(results []TaskResult) []string {
	var successful []string
	for _, result := range results {
		if result.Error == nil {
			successful = append(successful, result.Name)
		}
	}
	return successful
}
