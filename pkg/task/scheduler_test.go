package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/log"
)

func TestScheduler_LinearDependencies(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(4)

	// Create a linear dependency chain: task1 -> task2 -> task3
	task1 := TaskSpec{
		Name: "task1",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task1 result", nil
		},
	}
	task2 := TaskSpec{
		Name: "task2",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task2 result", nil
		},
		Dependencies: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Dependencies: []string{"task2"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)

	scheduler := NewScheduler(graph, executor, nil)
	results, err := scheduler.Schedule(context.Background())
	if err != nil {
		t.Errorf("Schedule failed: %v", err)
	}

	// Verify execution order
	order, err := scheduler.GetExecutionOrder()
	if err != nil {
		t.Errorf("GetExecutionOrder failed: %v", err)
	}
	expectedOrder := []string{"task1", "task2", "task3"}
	if len(order) != len(expectedOrder) {
		t.Errorf("Expected %d tasks in order, got %d", len(expectedOrder), len(order))
	}
	for i, task := range order {
		if task != expectedOrder[i] {
			t.Errorf("Expected task %s at position %d, got %s", expectedOrder[i], i, task)
		}
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Task %s failed: %v", result.Name, result.Error)
		}
	}
}

// Helper function to extract execution order from results
func getExecutionOrderFromResults(results []TaskResult) []string {
	order := make([]string, len(results))
	for i, result := range results {
		order[i] = result.Name
	}
	return order
}

// Mock logger for testing
type mockLogger struct{}

func (l *mockLogger) Info(msg string, data map[string]interface{})  {}
func (l *mockLogger) Error(msg string, data map[string]interface{}) {}
func (l *mockLogger) Debug(msg string, data map[string]interface{}) {}
func (l *mockLogger) Warn(msg string, data map[string]interface{})  {}

// No need for a placeholder variable

func TestScheduler_DiamondDependencies(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(4)

	// Create a diamond dependency: task1 -> task2 -> task4, task1 -> task3 -> task4
	task1 := TaskSpec{
		Name: "task1",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task1 result", nil
		},
	}
	task2 := TaskSpec{
		Name: "task2",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task2 result", nil
		},
		Dependencies: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Dependencies: []string{"task1"},
	}
	task4 := TaskSpec{
		Name: "task4",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task4 result", nil
		},
		Dependencies: []string{"task2", "task3"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)
	graph.AddTask(task4)

	scheduler := NewScheduler(graph, executor, nil)
	results, err := scheduler.Schedule(context.Background())
	if err != nil {
		t.Errorf("Schedule failed: %v", err)
	}

	// Verify execution order
	order, err := scheduler.GetExecutionOrder()
	if err != nil {
		t.Errorf("GetExecutionOrder failed: %v", err)
	}

	// task1 must be first
	if order[0] != "task1" {
		t.Errorf("Expected task1 to be first, got %s", order[0])
	}

	// task4 must be last
	if order[len(order)-1] != "task4" {
		t.Errorf("Expected task4 to be last, got %s", order[len(order)-1])
	}

	// task2 and task3 must be after task1 and before task4
	task1Index := -1
	task2Index := -1
	task3Index := -1
	task4Index := -1
	for i, task := range order {
		switch task {
		case "task1":
			task1Index = i
		case "task2":
			task2Index = i
		case "task3":
			task3Index = i
		case "task4":
			task4Index = i
		}
	}

	if task1Index > task2Index || task1Index > task3Index {
		t.Error("task1 must be executed before task2 and task3")
	}
	if task2Index > task4Index || task3Index > task4Index {
		t.Error("task2 and task3 must be executed before task4")
	}

	// Verify results
	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Task %s failed: %v", result.Name, result.Error)
		}
	}
}

func TestScheduler_TaskFailure(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(4)

	// Create tasks where task2 fails and task3 depends on it
	task1 := TaskSpec{
		Name: "task1",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task1 result", nil
		},
	}
	task2 := TaskSpec{
		Name: "task2",
		Run: func(ctx context.Context) (interface{}, error) {
			return nil, fmt.Errorf("task2 failed")
		},
		Dependencies: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Dependencies: []string{"task2"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)

	scheduler := NewScheduler(graph, executor, nil)
	results, err := scheduler.Schedule(context.Background())
	if err == nil {
		t.Error("Expected error due to task failure, got nil")
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	var foundTask1, foundTask2, foundTask3 bool
	for _, result := range results {
		if result.Name == "task1" {
			foundTask1 = true
			if result.Error != nil {
				t.Errorf("task1 failed: %v", result.Error)
			}
			// Check if task was executed based on result
			if result.Result == nil && result.Error == nil {
				t.Error("task1 should have been executed")
			}
		}
		if result.Name == "task2" {
			foundTask2 = true
			if result.Error == nil {
				t.Error("task2 should have failed")
			}
			// Check if task was executed based on result
			if result.Result == nil && result.Error == nil {
				t.Error("task2 should have been executed")
			}
		}
		if result.Name == "task3" {
			foundTask3 = true
			// Check if task was executed based on result
			if result.Result != nil || result.Error != nil {
				t.Error("task3 should not have been executed")
			}
		}
	}
	if !foundTask1 {
		t.Error("task1 result not found")
	}
	if !foundTask2 {
		t.Error("task2 result not found")
	}
	if !foundTask3 {
		t.Error("task3 result not found")
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(4)

	// Create a long-running task
	task1 := TaskSpec{
		Name: "task1",
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "task1 result", nil
		},
	}
	task2 := TaskSpec{
		Name: "task2",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task2 result", nil
		},
		Dependencies: []string{"task1"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)

	// Create a real logger
	logger := log.NewLogger(false)
	scheduler := NewScheduler(graph, executor, logger)

	// Create a context that cancels after 50ms - not used directly in the test but simulates timeout
	_, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// TODO: Implement context cancellation in the scheduler
	err := scheduler.Start()
	results := executor.GetResults()
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	var foundTask1, foundTask2 bool
	for _, result := range results {
		if result.Name == "task1" {
			foundTask1 = true
			// Check if task was cancelled or not executed
			if result.Result != nil {
				t.Errorf("task1 should be cancelled or not executed, but got result: %v", result.Result)
			}
		}
		if result.Name == "task2" {
			foundTask2 = true
			// Check if task was executed based on result
			if result.Result != nil || result.Error != nil {
				t.Error("task2 should not have been executed")
			}
		}
	}
	if !foundTask1 {
		t.Error("task1 result not found")
	}
	if !foundTask2 {
		t.Error("task2 result not found")
	}
}

func TestSchedule_Linear(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(1)

	// Create tasks with linear dependencies: A -> B -> C
	taskA := TaskSpec{
		Name: "taskA",
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskA result", nil
		},
	}
	taskB := TaskSpec{
		Name:         "taskB",
		Dependencies: []string{"taskA"},
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskB result", nil
		},
	}
	taskC := TaskSpec{
		Name:         "taskC",
		Dependencies: []string{"taskB"},
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskC result", nil
		},
	}

	graph.AddTask(taskA)
	graph.AddTask(taskB)
	graph.AddTask(taskC)

	// Add tasks to executor
	for _, spec := range graph.GetAllTasks() {
		executor.AddTask(NewTask(spec))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := executor.Run(ctx)

	// Verify execution order
	executionOrder := make([]string, 0)
	for _, result := range results {
		executionOrder = append(executionOrder, result.Name)
	}

	expectedOrder := []string{"taskA", "taskB", "taskC"}
	for i, name := range expectedOrder {
		if executionOrder[i] != name {
			t.Errorf("Expected task %s at position %d, got %s", name, i, executionOrder[i])
		}
	}
}

func TestSchedule_Parallel(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(4)

	// Create independent tasks
	tasks := []TaskSpec{
		{
			Name: "task1",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task1 result", nil
			},
		},
		{
			Name: "task2",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task2 result", nil
			},
		},
		{
			Name: "task3",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task3 result", nil
			},
		},
		{
			Name: "task4",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task4 result", nil
			},
		},
	}

	for _, task := range tasks {
		graph.AddTask(task)
	}

	// Add tasks to executor
	for _, spec := range graph.GetAllTasks() {
		executor.AddTask(NewTask(spec))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	results := executor.Run(ctx)
	duration := time.Since(start)

	// Verify all tasks completed
	if len(results) != len(tasks) {
		t.Errorf("Expected %d results, got %d", len(tasks), len(results))
	}

	// Verify parallel execution (should take ~0.1s, not 0.4s)
	if duration > 200*time.Millisecond {
		t.Errorf("Expected parallel execution, took %v", duration)
	}
}

func TestSchedule_Cycle(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(1)

	// Create cyclic dependency: A -> B -> C -> A
	taskA := TaskSpec{
		Name:         "taskA",
		Dependencies: []string{"taskC"},
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskA result", nil
		},
	}
	taskB := TaskSpec{
		Name:         "taskB",
		Dependencies: []string{"taskA"},
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskB result", nil
		},
	}
	taskC := TaskSpec{
		Name:         "taskC",
		Dependencies: []string{"taskB"},
		Run: func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "taskC result", nil
		},
	}

	graph.AddTask(taskA)
	graph.AddTask(taskB)
	graph.AddTask(taskC)

	// Add tasks to executor
	for _, spec := range graph.GetAllTasks() {
		executor.AddTask(NewTask(spec))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := executor.Run(ctx)

	// Check if any tasks failed due to cycle
	hasError := false
	for _, result := range results {
		if result.Error != nil {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("Expected cycle detection error, got none")
	}
}

func TestSchedule_LargeGraph(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(8)

	// Create 100 tasks with mixed dependencies
	for i := 0; i < 100; i++ {
		task := TaskSpec{
			Name:         fmt.Sprintf("task%d", i),
			Dependencies: []string{},
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return fmt.Sprintf("task%d result", i), nil
			},
		}
		// Add 1-3 random dependencies for 50% of tasks
		if i > 0 && i%2 == 0 {
			numDeps := (i % 3) + 1
			for j := 0; j < numDeps; j++ {
				dep := fmt.Sprintf("task%d", (i-j-1)%i)
				task.Dependencies = append(task.Dependencies, dep)
			}
		}
		graph.AddTask(task)
	}

	// Add tasks to executor
	for _, spec := range graph.GetAllTasks() {
		executor.AddTask(NewTask(spec))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	results := executor.Run(ctx)
	duration := time.Since(start)

	// Verify all tasks completed
	if len(results) != 100 {
		t.Errorf("Expected 100 results, got %d", len(results))
	}

	// Verify performance (should complete in reasonable time)
	if duration > 10*time.Second {
		t.Errorf("Large graph took too long: %v", duration)
	}
}

func TestSchedule_ConcurrencyLimit(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(2) // Limit to 2 concurrent tasks

	// Create 4 tasks that each take 100ms
	tasks := []TaskSpec{
		{
			Name: "task1",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task1 result", nil
			},
		},
		{
			Name: "task2",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task2 result", nil
			},
		},
		{
			Name: "task3",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task3 result", nil
			},
		},
		{
			Name: "task4",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "task4 result", nil
			},
		},
	}

	for _, task := range tasks {
		graph.AddTask(task)
	}

	// Add tasks to executor
	for _, spec := range graph.GetAllTasks() {
		executor.AddTask(NewTask(spec))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	results := executor.Run(ctx)
	duration := time.Since(start)

	// Verify all tasks completed
	if len(results) != len(tasks) {
		t.Errorf("Expected %d results, got %d", len(tasks), len(results))
	}

	// With 2 concurrent tasks, 4 tasks should take ~200ms
	// Allow some overhead
	if duration < 150*time.Millisecond || duration > 300*time.Millisecond {
		t.Errorf("Expected ~200ms with concurrency limit, got %v", duration)
	}
}
