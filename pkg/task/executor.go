package task

import (
	"context"
	"fmt"
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

// Executor manages concurrent task execution.
type Executor struct {
	tasks       []TaskRunner
	concurrency int
	resultMutex sync.Mutex
	results     []TaskResult
}

// NewExecutor creates a new Executor with the given tasks and concurrency level.
func NewExecutor(tasks []TaskRunner, concurrency int) *Executor {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Executor{
		tasks:       tasks,
		concurrency: concurrency,
		results:     make([]TaskResult, 0, len(tasks)),
	}
}

// Run executes all tasks concurrently and returns their results.
func (e *Executor) Run(ctx context.Context) []TaskResult {
	taskCh := make(chan TaskRunner, len(e.tasks))
	resultCh := make(chan TaskResult, len(e.tasks))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < e.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				start := time.Now()
				result, err := task.Run(ctx)
				duration := time.Since(start)

				status := "success"
				if err != nil {
					status = "failed"
				} else if ctx.Err() != nil {
					status = "cancelled"
				}

				resultCh <- TaskResult{
					Name:     task.GetName(),
					Result:   result,
					Error:    err,
					Duration: duration,
					Status:   status,
				}
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
		e.results = append(e.results, result)
		e.resultMutex.Unlock()
	}

	return e.results
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
