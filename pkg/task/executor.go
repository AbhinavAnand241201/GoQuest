package task

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Executor manages the execution of tasks
type Executor struct {
	// Channel for task results
	results chan TaskResult
	// Channel for task errors
	errors chan error
	// Wait group for task completion
	wg sync.WaitGroup
	// Mutex for concurrent access to status map
	statusMu sync.RWMutex
	// Map of task name to execution status
	status map[string]TaskStatus
	// Mutex for concurrent access to tasks slice
	tasksMu sync.RWMutex
	// List of tasks to execute
	tasks []Task
	// Number of concurrent workers
	concurrency int
	// Mutex for results
	resultMu sync.RWMutex
	// List of results
	resultList []TaskResult
}

// NewExecutor creates a new executor
// For backward compatibility, it accepts optional parameters:
// - tasks: a slice of TaskRunner or nil
// - concurrency: the number of concurrent workers or 0 for default
func NewExecutor(args ...interface{}) *Executor {
	executor := &Executor{
		results:     make(chan TaskResult),
		errors:      make(chan error),
		status:      make(map[string]TaskStatus),
		tasks:       make([]Task, 0),
		concurrency: runtime.NumCPU(),
		resultList:  make([]TaskResult, 0),
	}
	
	// Handle backward compatibility with old function signature
	if len(args) > 0 {
		// First argument might be a slice of TaskRunner
		if tasks, ok := args[0].([]TaskRunner); ok && tasks != nil {
			// Convert TaskRunner to Task
			for _, t := range tasks {
				if task, ok := t.(Task); ok {
					executor.tasks = append(executor.tasks, task)
				}
			}
		}
	}
	
	// Second argument might be concurrency
	if len(args) > 1 {
		if concurrency, ok := args[1].(int); ok && concurrency > 0 {
			executor.concurrency = concurrency
		}
	}
	
	return executor
}

// SetConcurrency sets the number of concurrent workers
func (e *Executor) SetConcurrency(concurrency int) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	e.tasksMu.Lock()
	defer e.tasksMu.Unlock()
	e.concurrency = concurrency
}

// AddTask adds a task to the executor
func (e *Executor) AddTask(task Task) {
	e.tasksMu.Lock()
	defer e.tasksMu.Unlock()
	e.tasks = append(e.tasks, task)
}

// GetResults returns the results of all executed tasks
func (e *Executor) GetResults() []TaskResult {
	e.resultMu.RLock()
	defer e.resultMu.RUnlock()
	
	// Return a copy to avoid concurrent access issues
	results := make([]TaskResult, len(e.resultList))
	copy(results, e.resultList)
	return results
}

// ExecuteTask executes a task and returns its result
func (e *Executor) ExecuteTask(ctx context.Context, task Task) (TaskResult, error) {
	// Record start time
	start := time.Now()

	// Execute task with proper context handling
	result, err := task.Run(ctx)
	duration := time.Since(start)

	// Record completion with proper mutex locking
	e.statusMu.Lock()
	e.status[task.Name()] = TaskStatus{
		Started:   start,
		Completed: time.Now(),
		Duration:  duration,
		Error:     err,
		Result:    result.Result,
	}
	e.statusMu.Unlock()

	if err != nil {
		return TaskResult{}, fmt.Errorf("task %s failed: %w", task.Name(), err)
	}

	return result, nil
}

// GetTaskStatus returns the execution status of a task
func (e *Executor) GetTaskStatus(taskName string) (TaskStatus, bool) {
	e.statusMu.RLock()
	defer e.statusMu.RUnlock()
	status, exists := e.status[taskName]
	return status, exists
}

// Run executes all tasks concurrently and returns their results.
func (e *Executor) Run(ctx context.Context) []TaskResult {
	// Get a thread-safe copy of tasks
	e.tasksMu.RLock()
	taskCount := len(e.tasks)
	tasks := make([]Task, taskCount)
	copy(tasks, e.tasks)
	e.tasksMu.RUnlock()

	taskCh := make(chan Task, taskCount)
	resultCh := make(chan TaskResult, taskCount)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < e.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				// Use ExecuteTask to ensure consistent status tracking
				result, err := e.ExecuteTask(ctx, task)
				if err != nil {
					result.Error = err
				}
				resultCh <- result
			}
		}()
	}

	// Send tasks to workers
	for _, task := range tasks {
		select {
		case taskCh <- task:
			// Task sent successfully
		case <-ctx.Done():
			// Context cancelled, stop sending tasks
			close(taskCh)
			goto ContextCancelled
		}
	}
	close(taskCh)

ContextCancelled:

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Process results
	results := make([]TaskResult, 0, taskCount)
	for result := range resultCh {
		e.resultMu.Lock()
		e.resultList = append(e.resultList, result)
		results = append(results, result)
		e.resultMu.Unlock()
	}

	return results
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
	
	// Format errors properly for better readability
	errMsgs := make([]string, 0, len(errors))
	for _, err := range errors {
		errMsgs = append(errMsgs, err.Error())
	}
	
	return fmt.Errorf("%d tasks failed:\n- %s", len(errors), strings.Join(errMsgs, "\n- "))
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
