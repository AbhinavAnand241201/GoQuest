package task

import (
	"context"
	"strings"
	"testing"
)

func TestTaskGraph_AddTask(t *testing.T) {
	graph := NewTaskGraph()

	// Test adding a task with no dependencies
	task1 := TaskSpec{
		Name: "task1",
		Run:  func(ctx context.Context) (interface{}, error) { return nil, nil },
	}
	if err := graph.AddTask(task1); err != nil {
		t.Errorf("AddTask failed: %v", err)
	}

	// Test adding a task with dependencies
	task2 := TaskSpec{
		Name:    "task2",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task1"},
	}
	if err := graph.AddTask(task2); err != nil {
		t.Errorf("AddTask failed: %v", err)
	}

	// Verify task1 has task2 as dependent
	dependents := graph.GetDependents("task1")
	if len(dependents) != 1 || dependents[0] != "task2" {
		t.Errorf("Expected task1 to have task2 as dependent, got %v", dependents)
	}

	// Verify task2 has task1 as dependency
	dependencies := graph.GetDependencies("task2")
	if len(dependencies) != 1 || dependencies[0] != "task1" {
		t.Errorf("Expected task2 to have task1 as dependency, got %v", dependencies)
	}
}

func TestTaskGraph_Validate_Cycle(t *testing.T) {
	graph := NewTaskGraph()

	// Create a cyclic dependency: task1 -> task2 -> task3 -> task1
	task1 := TaskSpec{
		Name:    "task1",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task3"},
	}
	task2 := TaskSpec{
		Name:    "task2",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task1"},
	}
	task3 := TaskSpec{
		Name:    "task3",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task2"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)

	err := graph.Validate()
	if err == nil {
		t.Error("Expected cycle detection error, got nil")
	}
	if err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "cycle detected:") ||
			!strings.Contains(errStr, "task1") ||
			!strings.Contains(errStr, "task2") ||
			!strings.Contains(errStr, "task3") {
			t.Errorf("Expected cycle error message with all tasks, got: %v", errStr)
		}
	}
}

func TestTaskGraph_Validate_MissingDependency(t *testing.T) {
	graph := NewTaskGraph()

	// Create a task with a non-existent dependency
	task1 := TaskSpec{
		Name:    "task1",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"non_existent"},
	}

	graph.AddTask(task1)

	err := graph.Validate()
	if err == nil {
		t.Error("Expected missing dependency error, got nil")
	}
	if err != nil && err.Error() != "invalid dependency: task task1 depends on non-existent task non_existent" {
		t.Errorf("Expected missing dependency error message, got: %v", err)
	}
}

func TestTaskGraph_Validate_ValidDAG(t *testing.T) {
	graph := NewTaskGraph()

	// Create a valid DAG: task1 -> task2 -> task4, task1 -> task3 -> task4
	task1 := TaskSpec{
		Name: "task1",
		Run:  func(ctx context.Context) (interface{}, error) { return nil, nil },
	}
	task2 := TaskSpec{
		Name:    "task2",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task1"},
	}
	task3 := TaskSpec{
		Name:    "task3",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task1"},
	}
	task4 := TaskSpec{
		Name:    "task4",
		Run:     func(ctx context.Context) (interface{}, error) { return nil, nil },
		Depends: []string{"task2", "task3"},
	}

	graph.AddTask(task1)
	graph.AddTask(task2)
	graph.AddTask(task3)
	graph.AddTask(task4)

	err := graph.Validate()
	if err != nil {
		t.Errorf("Expected valid DAG, got error: %v", err)
	}
}

func TestTaskGraph_GetTaskSpec(t *testing.T) {
	graph := NewTaskGraph()

	task1 := TaskSpec{
		Name: "task1",
		Run:  func(ctx context.Context) (interface{}, error) { return nil, nil },
	}
	graph.AddTask(task1)

	// Test getting existing task
	spec, exists := graph.GetTaskSpec("task1")
	if !exists {
		t.Error("Expected task1 to exist")
	}
	if spec.Name != "task1" {
		t.Errorf("Expected task1, got %s", spec.Name)
	}

	// Test getting non-existent task
	_, exists = graph.GetTaskSpec("non_existent")
	if exists {
		t.Error("Expected non_existent task to not exist")
	}
}

func TestTaskGraph_GetAllTasks(t *testing.T) {
	graph := NewTaskGraph()

	task1 := TaskSpec{
		Name: "task1",
		Run:  func(ctx context.Context) (interface{}, error) { return nil, nil },
	}
	task2 := TaskSpec{
		Name: "task2",
		Run:  func(ctx context.Context) (interface{}, error) { return nil, nil },
	}

	graph.AddTask(task1)
	graph.AddTask(task2)

	tasks := graph.GetAllTasks()
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Check that both tasks are present
	taskMap := make(map[string]bool)
	for _, task := range tasks {
		taskMap[task] = true
	}
	if !taskMap["task1"] || !taskMap["task2"] {
		t.Error("Expected both task1 and task2 to be present")
	}
}
