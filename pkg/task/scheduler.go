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

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskName string
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

// NewScheduler creates a new scheduler for the given task graph
func NewScheduler(graph *TaskGraph, executor *Executor, logger *log.Logger) *Scheduler {
	minWorkers := runtime.NumCPU()
	maxWorkers := minWorkers * 4 // Allow up to 4x CPU cores

	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		graph:      graph,
		executor:   executor,
		logger:     logger,
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
		workerPool: make(chan struct{}, maxWorkers),
		metrics:    &SchedulerMetrics{},
		results:    make(chan TaskResult),
		errors:     make(chan error),
		status:     make(map[string]TaskStatus),
		done:       make(chan string),
		started:    make(chan string),
		failed:     make(chan string),
		cancel:     make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start begins executing tasks in the graph
func (s *Scheduler) Start() error {
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
	defer s.wg.Done()

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
	s.started <- task.Name()

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
		s.failed <- task.Name()
		s.metrics.mu.Lock()
		s.metrics.TasksFailed++
		s.metrics.mu.Unlock()
		s.errors <- fmt.Errorf("task %s failed: %w", task.Name(), err)
		return
	}

	// Send result
	s.results <- result

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TasksCompleted++
	s.metrics.mu.Unlock()

	// Notify completion
	s.done <- task.Name()

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
	return *s.metrics
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
