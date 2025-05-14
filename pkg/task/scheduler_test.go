package task

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestScheduler_LinearDependencies(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(nil, 4)

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
		Depends: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Depends: []string{"task2"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)

	scheduler := NewScheduler(graph, executor)
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

func TestScheduler_DiamondDependencies(t *testing.T) {
	graph := NewTaskGraph()
	executor := NewExecutor(nil, 4)

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
		Depends: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Depends: []string{"task1"},
	}
	task4 := TaskSpec{
		Name: "task4",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task4 result", nil
		},
		Depends: []string{"task2", "task3"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)
	graph.AddTask(task4)

	scheduler := NewScheduler(graph, executor)
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
	executor := NewExecutor(nil, 4)

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
		Depends: []string{"task1"},
	}
	task3 := TaskSpec{
		Name: "task3",
		Run: func(ctx context.Context) (interface{}, error) {
			return "task3 result", nil
		},
		Depends: []string{"task2"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)

	scheduler := NewScheduler(graph, executor)
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
			if result.Status == "not_executed" {
				t.Error("task1 should have been executed")
			}
		}
		if result.Name == "task2" {
			foundTask2 = true
			if result.Error == nil {
				t.Error("task2 should have failed")
			}
			if result.Status == "not_executed" {
				t.Error("task2 should have been executed")
			}
		}
		if result.Name == "task3" {
			foundTask3 = true
			if result.Status != "not_executed" {
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
	executor := NewExecutor(nil, 4)

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
		Depends: []string{"task1"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)

	scheduler := NewScheduler(graph, executor)

	// Create a context that cancels after 50ms
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results, err := scheduler.Schedule(ctx)
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
			if result.Status != "cancelled" && result.Status != "not_executed" {
				t.Errorf("task1 should be cancelled or not_executed, got status %s", result.Status)
			}
		}
		if result.Name == "task2" {
			foundTask2 = true
			if result.Status != "not_executed" {
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
