package task

import (
	"context"
	"fmt"
	"runtime"
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
	err := s.startExecution(ctx)
	// Return the results from the executor
	return s.executor.GetResults(), err
}

// startExecution is the internal implementation of task execution
func (s *Scheduler) startExecution(ctx context.Context) error {
	// Validate the graph
	if err := s.graph.Validate(); err != nil {
		return fmt.Errorf("invalid task graph: %w", err)
	}

	// Start monitoring goroutine
	go s.monitor()

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

	// Wait for all tasks to complete
	s.wg.Wait()

	// Check for errors
	select {
	case err := <-s.errors:
		return err
	default:
		return nil
	}
}

// executeTask executes a single task
func (s *Scheduler) executeTask(task Task) {
	// Acquire a worker slot from the pool
	s.workerPool <- struct{}{}
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
		Result:    result.Result,
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
				"task": task.Name(),
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
				"task": task.Name(),
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

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TasksCompleted++
	s.metrics.mu.Unlock()

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

	// Get next runnable tasks
	nextTasks := s.graph.GetNextTasks(task.Name())
	for _, nextTask := range nextTasks {
		s.wg.Add(1)
		go s.executeTask(nextTask)
	}
}

// monitor handles task completion and error notifications
func (s *Scheduler) monitor() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
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
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			s.metrics.mu.Lock()
			s.metrics.AllocBytes = m.Alloc
			s.metrics.TotalAllocBytes = m.TotalAlloc
			s.metrics.NumGC = m.NumGC
			if s.metrics.TasksCompleted > 0 {
				s.metrics.WorkerUtilization = float64(s.metrics.ActiveWorkers) / float64(s.metrics.MaxWorkers)
			}
			s.metrics.mu.Unlock()
		}
	}
}

// GetMetrics returns the current scheduler metrics
func (s *Scheduler) GetMetrics() SchedulerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	
	// Create a copy without the mutex to avoid copying the mutex
	return SchedulerMetrics{
		TasksStarted:      s.metrics.TasksStarted,
		TasksCompleted:    s.metrics.TasksCompleted,
		TasksFailed:       s.metrics.TasksFailed,
		TotalDuration:     s.metrics.TotalDuration,
		ActiveWorkers:     s.metrics.ActiveWorkers,
		MaxWorkers:        s.metrics.MaxWorkers,
		WorkerUtilization: s.metrics.WorkerUtilization,
		AllocBytes:        s.metrics.AllocBytes,
		TotalAllocBytes:   s.metrics.TotalAllocBytes,
		NumGC:             s.metrics.NumGC,
	}
}

// GetTaskStatus returns the execution status of a task
func (s *Scheduler) GetTaskStatus(taskName string) (TaskStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, exists := s.status[taskName]
	return status, exists
}

// Cancel stops the scheduler and all running tasks
func (s *Scheduler) Cancel() {
	close(s.cancel)
}

// GetExecutionOrder is a backward compatibility method for tests
// It returns the order in which tasks were executed
func (s *Scheduler) GetExecutionOrder() ([]string, error) {
	results := s.executor.GetResults()
	order := make([]string, len(results))
	for i, result := range results {
		order[i] = result.Name
	}
	return order, nil
}
