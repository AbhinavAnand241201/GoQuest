package task

import (
	"context"
	"fmt"
	"testing"
	"time"
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

	// Add tasks in order
	if err := graph.AddTask(task1); err != nil {
		t.Fatalf("Failed to add task1: %v", err)
	}
	if err := graph.AddTask(task2); err != nil {
		t.Fatalf("Failed to add task2: %v", err)
	}
	if err := graph.AddTask(task3); err != nil {
		t.Fatalf("Failed to add task3: %v", err)
	}

	// Validate the graph
	if err := graph.Validate(); err != nil {
		t.Fatalf("Graph validation failed: %v", err)
	}

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
		if i < len(expectedOrder) && task != expectedOrder[i] {
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

	// Add tasks in order
	if err := graph.AddTask(task1); err != nil {
		t.Fatalf("Failed to add task1: %v", err)
	}
	if err := graph.AddTask(task2); err != nil {
		t.Fatalf("Failed to add task2: %v", err)
	}
	if err := graph.AddTask(task3); err != nil {
		t.Fatalf("Failed to add task3: %v", err)
	}
	if err := graph.AddTask(task4); err != nil {
		t.Fatalf("Failed to add task4: %v", err)
	}

	// Validate the graph
	if err := graph.Validate(); err != nil {
		t.Fatalf("Graph validation failed: %v", err)
	}

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
	if len(order) == 0 {
		t.Fatalf("Execution order is empty")
	}
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

	// Create tasks with a failure in the middle: task1 -> task2 (fails) -> task3
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

	// Add tasks in order
	if err := graph.AddTask(task1); err != nil {
		t.Fatalf("Failed to add task1: %v", err)
	}
	if err := graph.AddTask(task2); err != nil {
		t.Fatalf("Failed to add task2: %v", err)
	}
	if err := graph.AddTask(task3); err != nil {
		t.Fatalf("Failed to add task3: %v", err)
	}

	// Validate the graph
	if err := graph.Validate(); err != nil {
		t.Fatalf("Graph validation failed: %v", err)
	}

	scheduler := NewScheduler(graph, executor, nil)
	results, err := scheduler.Schedule(context.Background())

	// Verify execution order
	order, err := scheduler.GetExecutionOrder()
	if err != nil {
		t.Errorf("GetExecutionOrder failed: %v", err)
	}

	// task1 and task2 should be executed, but task3 should not
	expectedTasks := map[string]bool{"task1": true, "task2": true}
	for _, task := range order {
		if task == "task3" {
			t.Errorf("Task3 should not have been executed")
		}
		expectedTasks[task] = false
	}

	// Check that all expected tasks were executed
	for task, notExecuted := range expectedTasks {
		if notExecuted {
			t.Errorf("Task %s was not executed", task)
		}
	}

	// Verify results
	task2Failed := false
	for _, result := range results {
		if result.Name == "task2" && result.Error != nil {
			task2Failed = true
		}
		if result.Name == "task3" {
			t.Errorf("Task3 should not have a result")
		}
	}

	if !task2Failed {
		t.Errorf("Task2 should have failed")
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(4)

	// Create a task that takes longer than the context timeout
	task1 := TaskSpec{
		Name: "task1",
		Run: func(ctx context.Context) (interface{}, error) {
			// This task should complete before cancellation
			return "task1 result", nil
		},
	}
	task2 := TaskSpec{
		Name: "task2",
		Run: func(ctx context.Context) (interface{}, error) {
			// This task should be cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
				return "task2 result", nil
			}
		},
		Dependencies: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			// This task should never start due to cancellation
			return "task3 result", nil
		},
		Dependencies: []string{"task2"},
	}

	// Add tasks in order
	if err := graph.AddTask(task1); err != nil {
		t.Fatalf("Failed to add task1: %v", err)
	}
	if err := graph.AddTask(task2); err != nil {
		t.Fatalf("Failed to add task2: %v", err)
	}
	if err := graph.AddTask(task3); err != nil {
		t.Fatalf("Failed to add task3: %v", err)
	}

	// Validate the graph
	if err := graph.Validate(); err != nil {
		t.Fatalf("Graph validation failed: %v", err)
	}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	scheduler := NewScheduler(graph, executor, nil)
	results, _ := scheduler.Schedule(ctx)

	// Verify results
	var foundTask1 bool
	for _, result := range results {
		if result.Name == "task1" {
			foundTask1 = true
			// task1 should have completed successfully
			if result.Error != nil {
				t.Errorf("task1 failed: %v", result.Error)
			}
		}
		if result.Name == "task2" {
			// task2 should have been cancelled
			if result.Error == nil {
				t.Errorf("task2 should have been cancelled")
			}
		}
		if result.Name == "task3" {
			// task3 should not have been executed
			t.Error("task3 should not have been executed")
		}
	}

	// task1 should have been executed
	if !foundTask1 {
		t.Error("task1 should have been executed")
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
	executor := NewExecutor()
	executor.SetConcurrency(4)

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

	// Add tasks to graph - should detect cycle during validation
	graph.AddTask(taskA)
	graph.AddTask(taskB)
	graph.AddTask(taskC)

	// Validate should detect the cycle
	err := graph.Validate()
	if err == nil {
		// If validation doesn't catch it, try running the tasks
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
}

func TestSchedule_LargeGraph(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor()
	executor.SetConcurrency(8)

	// Create 100 tasks with mixed dependencies
	for i := 0; i < 100; i++ {
		// Use a closure to capture the correct i value for each task
		taskIndex := i
		task := TaskSpec{
			Name:         fmt.Sprintf("task%d", taskIndex),
			Dependencies: []string{},
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return fmt.Sprintf("task%d result", taskIndex), nil
			},
		}
		// Add 1-3 random dependencies for 50% of tasks
		if taskIndex > 0 && taskIndex%2 == 0 {
			numDeps := (taskIndex % 3) + 1
			for j := 0; j < numDeps; j++ {
				// Make sure we don't create invalid dependencies
				depIndex := (taskIndex - j - 1) % taskIndex
				dep := fmt.Sprintf("task%d", depIndex)
				task.Dependencies = append(task.Dependencies, dep)
			}
		}
		if err := graph.AddTask(task); err != nil {
			t.Fatalf("Failed to add task%d: %v", taskIndex, err)
		}
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
	executor := NewExecutor()
	executor.SetConcurrency(2) // Limit to 2 concurrent tasks

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
		if err := graph.AddTask(task); err != nil {
			t.Fatalf("Failed to add task %s: %v", task.Name, err)
		}
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
