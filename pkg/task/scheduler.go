package task

import (
	"context"
	"fmt"
	"sync"
)

// Scheduler manages the execution of tasks based on their dependencies.
type Scheduler struct {
	graph    *TaskGraph
	executor *Executor
}

// NewScheduler creates a new Scheduler with the given graph and executor.
func NewScheduler(graph *TaskGraph, executor *Executor) *Scheduler {
	return &Scheduler{
		graph:    graph,
		executor: executor,
	}
}

// Schedule executes tasks in topological order while respecting dependencies.
func (s *Scheduler) Schedule(ctx context.Context) ([]TaskResult, error) {
	// Validate the graph first
	if err := s.graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task graph: %w", err)
	}

	// Calculate in-degree for each task
	inDegree := make(map[string]int)
	for _, taskName := range s.graph.GetAllTasks() {
		deps := s.graph.GetDependencies(taskName)
		inDegree[taskName] = len(deps)
	}

	// Initialize queue with tasks that have no dependencies
	queue := make([]string, 0)
	for taskName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskName)
		}
	}

	var results []TaskResult
	var mu sync.Mutex        // Protect results slice from concurrent writes
	var failedTasks []string // Track failed tasks for better error reporting

	// Process tasks in batches
	processed := make(map[string]bool)
	for len(queue) > 0 {
		// Debug: print current queue
		// fmt.Printf("Scheduler queue: %v\n", queue)
		// Get current batch of tasks
		batch := make([]TaskRunner, 0, len(queue))
		for _, taskName := range queue {
			if spec, exists := s.graph.GetTaskSpec(taskName); exists {
				batch = append(batch, NewTask(spec))
			}
		}
		queue = queue[:0] // Clear queue for next batch

		// Create a new executor for this batch
		batchExecutor := NewExecutor(batch, s.executor.concurrency)

		// If context is already cancelled, mark all tasks in batch as cancelled
		if ctx.Err() != nil {
			mu.Lock()
			for _, runner := range batch {
				results = append(results, TaskResult{
					Name:   runner.GetName(),
					Status: "cancelled",
					Error:  ctx.Err(),
				})
				failedTasks = append(failedTasks, runner.GetName())
				processed[runner.GetName()] = true
			}
			mu.Unlock()
			break
		}

		// Execute current batch concurrently
		batchResults := batchExecutor.Run(ctx)
		mu.Lock()
		results = append(results, batchResults...)
		for _, r := range batchResults {
			processed[r.Name] = true
		}
		mu.Unlock()

		// Track which tasks failed in this batch
		failedOrCancelled := make(map[string]bool)
		for _, result := range batchResults {
			if result.Error != nil || result.Status == "cancelled" {
				failedTasks = append(failedTasks, result.Name)
				failedOrCancelled[result.Name] = true
			}
		}

		// Update in-degree and add new tasks to queue
		for _, result := range batchResults {
			if failedOrCancelled[result.Name] {
				// Do not schedule dependents of failed/cancelled tasks
				continue
			}
			for _, dependent := range s.graph.GetDependents(result.Name) {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// After processing, if any tasks were not processed, mark them as failed/cancelled
	for _, taskName := range s.graph.GetAllTasks() {
		if !processed[taskName] {
			results = append(results, TaskResult{
				Name:   taskName,
				Status: "not_executed",
				Error:  fmt.Errorf("task was not executed due to failed or cancelled dependency"),
			})
		}
	}

	// Check if all tasks were executed
	hasNotExecuted := false
	hasCancelled := false
	for _, r := range results {
		if r.Status == "not_executed" {
			hasNotExecuted = true
		}
		if r.Status == "cancelled" {
			hasCancelled = true
		}
	}
	if hasNotExecuted || hasCancelled {
		return results, fmt.Errorf("some tasks were not executed or cancelled")
	}

	if len(results) != len(s.graph.GetAllTasks()) {
		return nil, fmt.Errorf("incomplete execution: %d tasks failed (%v), possible cycle or missing tasks",
			len(failedTasks), failedTasks)
	}

	return results, nil
}

// GetExecutionOrder returns the tasks in their execution order.
func (s *Scheduler) GetExecutionOrder() ([]string, error) {
	if err := s.graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task graph: %w", err)
	}

	// Calculate in-degree for each task
	inDegree := make(map[string]int)
	for _, taskName := range s.graph.GetAllTasks() {
		deps := s.graph.GetDependencies(taskName)
		inDegree[taskName] = len(deps)
	}

	// Initialize queue with tasks that have no dependencies
	queue := make([]string, 0)
	for taskName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskName)
		}
	}

	// Process tasks in order
	order := make([]string, 0)
	for len(queue) > 0 {
		taskName := queue[0]
		queue = queue[1:]
		order = append(order, taskName)

		// Update in-degree for dependents
		for _, dependent := range s.graph.GetDependents(taskName) {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check if all tasks were ordered
	if len(order) != len(s.graph.GetAllTasks()) {
		return nil, fmt.Errorf("incomplete ordering: possible cycle or missing tasks")
	}

	return order, nil
}
