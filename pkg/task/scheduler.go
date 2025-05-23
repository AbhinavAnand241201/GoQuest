package task

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/log"
)

// SchedulerMetrics tracks performance metrics for the scheduler
type SchedulerMetrics struct {
	mu sync.RWMutex
	// Task execution metrics
	TasksStarted   int64
	TasksCompleted int64
	TasksFailed    int64
	TotalDuration  time.Duration
	// Worker pool metrics
	ActiveWorkers     int
	MaxWorkers        int
	WorkerUtilization float64
	// Memory metrics
	AllocBytes      uint64
	TotalAllocBytes uint64
	NumGC           uint32
}

// Scheduler manages the execution of tasks based on their dependencies
type Scheduler struct {
	graph      *TaskGraph
	executor   *Executor
	logger     *log.Logger
	minWorkers int
	maxWorkers int
	// Worker pool semaphore to limit concurrent tasks
	workerPool chan struct{}
	metrics    *SchedulerMetrics
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
	// Channel for task completion notifications
	done chan string
	// Channel for task start notifications
	started chan string
	// Channel for task error notifications
	failed chan string
	// Channel for task cancellation
	cancel chan struct{}
	// Context for cancellation
	ctx context.Context
	// Cancel function for context
	cancelFunc context.CancelFunc
}

// NewScheduler creates a new scheduler for the given task graph
func NewScheduler(graph *TaskGraph, executor *Executor, logger *log.Logger) *Scheduler {
	// For backward compatibility with tests that don't provide a logger
	if logger == nil {
		logger = log.NewLogger(false)
	}
	minWorkers := runtime.NumCPU()
	maxWorkers := minWorkers * 4 // Allow up to 4x CPU cores

	// Calculate buffer size based on expected number of tasks
	taskCount := graph.Size()
	bufferSize := taskCount * 2 // Buffer size should be at least 2x the number of tasks
	if bufferSize < 10 {
		bufferSize = 10 // Minimum buffer size
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		graph:      graph,
		executor:   executor,
		logger:     logger,
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
		workerPool: make(chan struct{}, maxWorkers),
		metrics:    &SchedulerMetrics{},
		results:    make(chan TaskResult, bufferSize),
		errors:     make(chan error, bufferSize),
		status:     make(map[string]TaskStatus),
		done:       make(chan string, bufferSize),
		started:    make(chan string, bufferSize),
		failed:     make(chan string, bufferSize),
		cancel:     make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start begins executing tasks in the graph
func (s *Scheduler) Start() error {
	return s.startExecution(context.Background())
}

// Schedule is a backward compatibility method that wraps Start
// It's used by tests that expect the old API
func (s *Scheduler) Schedule(ctx context.Context) ([]TaskResult, error) {
	// Use the scheduler's context instead of creating a new one
	err := s.startExecution(s.ctx)
	// Return the results from the executor
	return s.executor.GetResults(), err
}

// startExecution is the internal implementation of task execution
func (s *Scheduler) startExecution(ctx context.Context) error {
	// Validate the graph
	if err := s.graph.Validate(); err != nil {
		return fmt.Errorf("invalid task graph: %w", err)
	}

	// Start monitoring goroutine with the same context
	monitorDone := make(chan struct{})
	go func() {
		s.monitor(ctx)
		close(monitorDone)
	}()

	// Get initial runnable tasks
	runnable := s.graph.GetRunnableTasks()
	if len(runnable) == 0 {
		return fmt.Errorf("no runnable tasks found")
	}

	// Start executing tasks
	for _, task := range runnable {
		s.wg.Add(1)
		go s.executeTask(task)
	}

	// Wait for all tasks to complete or context cancellation
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// Create a channel for errors
	errCh := make(chan error, 1)

	// Wait for completion or cancellation
	select {
	case <-done:
		// All tasks completed
		// Check for errors
		select {
		case err := <-s.errors:
			errCh <- err
		default:
			errCh <- nil
		}
	case <-ctx.Done():
		// Context cancelled
		errCh <- ctx.Err()
		// Cancel our own context to stop all tasks
		s.cancelFunc()
	}

	// Wait for the monitor to finish
	<-monitorDone

	// Return any error
	return <-errCh
}

// executeTask executes a single task
func (s *Scheduler) executeTask(task Task) {
	// Check if task's dependencies are completed
	deps := s.graph.GetDependencies(task.Name())
	for _, dep := range deps {
		s.mu.RLock()
		status, exists := s.status[dep]
		s.mu.RUnlock()
		if !exists || status.Error != nil {
			// Dependency not completed or failed
			s.wg.Done()
			return
		}
	}

	// Acquire a worker slot from the pool
	select {
	case s.workerPool <- struct{}{}:
		// Got a worker slot
	case <-s.ctx.Done():
		// Context cancelled
		s.wg.Done()
		return
	}
	defer func() {
		// Release the worker slot when done
		<-s.workerPool
		s.wg.Done()
	}()

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TasksStarted++
	s.metrics.ActiveWorkers++
	if s.metrics.ActiveWorkers > s.metrics.MaxWorkers {
		s.metrics.MaxWorkers = s.metrics.ActiveWorkers
	}
	s.metrics.mu.Unlock()

	// Record start time
	start := time.Now()

	// Send started notification with non-blocking select
	select {
	case s.started <- task.Name():
		// Successfully sent
	case <-s.ctx.Done():
		// Context cancelled, exit early
		return
	default:
		// Channel full, log and continue
		s.logger.Debug("Started channel full, skipping notification", map[string]interface{}{
			"task": task.Name(),
		})
	}

	// Execute task
	result, err := task.Run(s.ctx)
	duration := time.Since(start)

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TotalDuration += duration
	s.metrics.ActiveWorkers--
	s.metrics.mu.Unlock()

	// Record completion
	s.mu.Lock()
	s.status[task.Name()] = TaskStatus{
		Started:   start,
		Completed: time.Now(),
		Duration:  duration,
		Error:     err,
		Result:    result,
	}
	s.mu.Unlock()

	if err != nil {
		// Send failed notification with non-blocking select
		select {
		case s.failed <- task.Name():
			// Successfully sent
		case <-s.ctx.Done():
			// Context cancelled, exit early
			return
		default:
			// Channel full, log and continue
			s.logger.Debug("Failed channel full, skipping notification", map[string]interface{}{
				"task":  task.Name(),
				"error": err.Error(),
			})
		}

		s.metrics.mu.Lock()
		s.metrics.TasksFailed++
		s.metrics.mu.Unlock()

		// Send error with non-blocking select
		select {
		case s.errors <- fmt.Errorf("task %s failed: %w", task.Name(), err):
			// Successfully sent
		case <-s.ctx.Done():
			// Context cancelled, exit early
			return
		default:
			// Channel full, log and continue
			s.logger.Error("Error channel full, dropping error", map[string]interface{}{
				"task":  task.Name(),
				"error": err.Error(),
			})
		}
		return
	}

	// Send result with non-blocking select
	select {
	case s.results <- result:
		// Successfully sent
	case <-s.ctx.Done():
		// Context cancelled, exit early
		return
	default:
		// Channel full, log and continue
		s.logger.Debug("Results channel full, dropping result", map[string]interface{}{
			"task": task.Name(),
		})
	}

	// Send done notification with non-blocking select
	select {
	case s.done <- task.Name():
		// Successfully sent
	case <-s.ctx.Done():
		// Context cancelled, exit early
		return
	default:
		// Channel full, log and continue
		s.logger.Debug("Done channel full, skipping notification", map[string]interface{}{
			"task": task.Name(),
		})
	}

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TasksCompleted++
	s.metrics.mu.Unlock()

	// Start any dependent tasks that are now runnable
	dependentNames := s.graph.GetDependents(task.Name())
	for _, depName := range dependentNames {
		// Get the task specification
		spec, exists := s.graph.GetTaskSpec(depName)
		if !exists {
			s.logger.Error("Dependent task not found in graph", map[string]interface{}{
				"task": depName,
			})
			continue
		}

		// Check if all dependencies of the dependent task are completed
		allDepsCompleted := true
		deps := s.graph.GetDependencies(depName)
		for _, dep := range deps {
			s.mu.RLock()
			status, exists := s.status[dep]
			s.mu.RUnlock()
			if !exists || status.Error != nil {
				allDepsCompleted = false
				break
			}
		}
		if allDepsCompleted {
			s.wg.Add(1)
			go s.executeTask(NewTask(spec))
		}
	}
}

// monitor handles task completion and error notifications
func (s *Scheduler) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
			return
		case <-s.cancel:
			s.cancelFunc()
			return
		case taskName := <-s.started:
			// Process task started notification
			s.logger.Debug("Task started", map[string]interface{}{
				"task": taskName,
			})
		case taskName := <-s.done:
			// Process task completion notification
			s.logger.Debug("Task completed", map[string]interface{}{
				"task": taskName,
			})
		case taskName := <-s.failed:
			// Process task failure notification
			s.logger.Debug("Task failed", map[string]interface{}{
				"task": taskName,
			})
		case <-ticker.C:
			// Update metrics periodically
			s.updateMetrics()
		}
	}
}

// updateMetrics updates the scheduler metrics
func (s *Scheduler) updateMetrics() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	// Update memory metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.metrics.AllocBytes = m.Alloc
	s.metrics.TotalAllocBytes = m.TotalAlloc
	s.metrics.NumGC = m.NumGC

	// Calculate worker utilization
	if s.metrics.MaxWorkers > 0 {
		s.metrics.WorkerUtilization = float64(s.metrics.ActiveWorkers) / float64(s.metrics.MaxWorkers)
	}
}

// GetMetrics returns the current scheduler metrics
func (s *Scheduler) GetMetrics() SchedulerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	return *s.metrics
}

// GetTaskStatus returns the status of a specific task
func (s *Scheduler) GetTaskStatus(taskName string) (TaskStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.status[taskName]
	return status, ok
}

// Cancel cancels all running tasks
func (s *Scheduler) Cancel() {
	select {
	case s.cancel <- struct{}{}:
		// Successfully sent
	default:
		// Channel full, log and continue
		s.logger.Debug("Cancel channel full, skipping notification", map[string]interface{}{})
	}
}

// GetExecutionOrder returns the order in which tasks were executed
func (s *Scheduler) GetExecutionOrder() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a slice of task names
	order := make([]string, 0, len(s.status))
	for name := range s.status {
		order = append(order, name)
	}

	// Sort by start time
	sort.Slice(order, func(i, j int) bool {
		return s.status[order[i]].Started.Before(s.status[order[j]].Started)
	})

	return order, nil
}
