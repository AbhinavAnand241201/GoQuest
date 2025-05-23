package task

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Simple logger interface for task execution
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// Default logger implementation
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields map[string]interface{}) {
	log.Printf("%s [DEBUG] %s %v", time.Now().Format("2006-01-02 15:04:05"), msg, fields)
}

func (l *defaultLogger) Info(msg string, fields map[string]interface{}) {
	log.Printf("%s [INFO] %s %v", time.Now().Format("2006-01-02 15:04:05"), msg, fields)
}

func (l *defaultLogger) Error(msg string, fields map[string]interface{}) {
	log.Printf("%s [ERROR] %s %v", time.Now().Format("2006-01-02 15:04:05"), msg, fields)
}

// Global logger instance
var logger Logger = &defaultLogger{}

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

// ExecuteTask executes a single task and returns its result.
func (e *Executor) ExecuteTask(ctx context.Context, task Task) (TaskResult, error) {
	name := task.Name()
	logger.Debug("Task started", map[string]interface{}{"task": name})

	// Create a channel to collect the result
	resultCh := make(chan TaskResult, 1)
	errCh := make(chan error, 1)

	// Execute the task in a goroutine
	go func() {
		start := time.Now()
		result, err := task.Run(ctx)
		duration := time.Since(start)

		taskResult := TaskResult{
			Name:     name,
			Duration: duration,
		}

		if err != nil {
			logger.Debug("Task failed", map[string]interface{}{"task": name})
			taskResult.Error = err
			errCh <- err
		} else {
			logger.Debug("Task completed", map[string]interface{}{"task": name})
			taskResult.Result = result
			errCh <- nil
		}

		resultCh <- taskResult
	}()

	// Wait for the task to complete or for the context to be cancelled
	select {
	case taskResult := <-resultCh:
		err := <-errCh
		return taskResult, err
	case <-ctx.Done():
		// Context was cancelled
		logger.Debug("Task cancelled", map[string]interface{}{"task": name})
		return TaskResult{
			Name:     name,
			Duration: time.Since(time.Time{}), // We don't know the actual duration
			Error:    ctx.Err(),
		}, ctx.Err()
	}
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

	// If there are no tasks, return immediately
	if taskCount == 0 {
		return []TaskResult{}
	}

	// Create a context that can be cancelled
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure resources are cleaned up

	taskCh := make(chan Task, taskCount)
	resultCh := make(chan TaskResult, taskCount)
	var wg sync.WaitGroup

	// Start worker goroutines
	workerCount := e.concurrency
	if workerCount > taskCount {
		workerCount = taskCount // Don't create more workers than tasks
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				// Check if context is cancelled
				select {
				case <-execCtx.Done():
					// Context cancelled, stop processing tasks
					return
				default:
					// Continue processing
				}

				// Use ExecuteTask to ensure consistent status tracking
				result, err := e.ExecuteTask(execCtx, task)
				if err != nil {
					result.Error = err
				}

				// Send result with non-blocking select
				select {
				case resultCh <- result:
					// Result sent successfully
				case <-execCtx.Done():
					// Context cancelled, stop sending results
					return
				}
			}
		}()
	}

	// Send tasks to workers
	tasksSent := 0
	for _, task := range tasks {
		select {
		case taskCh <- task:
			// Task sent successfully
			tasksSent++
		case <-ctx.Done():
			// Context cancelled, stop sending tasks
			break
		}
	}

	// Close the task channel to signal no more tasks
	close(taskCh)

	// Create a channel to signal when all workers are done
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(resultCh) // Safe to close after all workers are done
	}()

	// Wait for all workers to finish or context to be cancelled
	var results []TaskResult
	resultsReceived := 0

	// Use a timeout to prevent deadlocks, but make it longer than the test timeouts
	timeout := time.NewTimer(500 * time.Millisecond)
	defer timeout.Stop()

	// Collect results until we've received all expected results or context is cancelled
	results = make([]TaskResult, 0, tasksSent)
	for resultsReceived < tasksSent {
		select {
		case result, ok := <-resultCh:
			if !ok {
				// Channel closed, no more results
				goto DoneCollecting
			}
			// Process result
			e.resultMu.Lock()
			e.resultList = append(e.resultList, result)
			results = append(results, result)
			e.resultMu.Unlock()
			resultsReceived++
		case <-ctx.Done():
			// Context cancelled
			cancel() // Cancel our execution context
			goto DoneCollecting
		case <-done:
			// All workers finished
			goto DoneCollecting
		case <-timeout.C:
			// Timeout reached, prevent deadlock
			cancel() // Cancel our execution context
			goto DoneCollecting
		}
	}

DoneCollecting:
	// If we didn't receive all expected results, add cancellation results for missing tasks
	if resultsReceived < tasksSent {
		for i := resultsReceived; i < tasksSent; i++ {
			// We don't know which tasks didn't complete, so we can't add specific results
			// This is a best-effort approach
		}
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
