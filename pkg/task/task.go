package task

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	taskRegistry = make(map[string]TaskSpec)
	registryLock sync.RWMutex
)

// Task represents a unit of work that can be executed
type Task interface {
	// Name returns the unique identifier for this task
	Name() string
	// Run executes the task and returns its result
	Run(ctx context.Context) (TaskResult, error)
}

// TaskSpec defines the configuration for a task
type TaskSpec struct {
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Dir          string            `json:"dir,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
}

// TaskRunner defines the interface for executing tasks.
type TaskRunner interface {
	// Run executes the task and returns its result and any error
	Run(context.Context) (TaskResult, error)
	// GetName returns the task's name
	GetName() string
	// GetDependencies returns the task's dependencies
	GetDependencies() []string
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	Name     string
	Result   string
	Duration time.Duration
}

// task implements the Task interface
type task struct {
	spec TaskSpec
}

// NewTask creates a new task from a specification
func NewTask(spec TaskSpec) Task {
	return &task{
		spec: spec,
	}
}

// Name returns the task's name
func (t *task) Name() string {
	return t.spec.Name
}

// Run executes the task and returns its result
func (t *task) Run(ctx context.Context) (TaskResult, error) {
	start := time.Now()
	// TODO: Implement actual task execution
	result := fmt.Sprintf("Task %s completed", t.spec.Name)
	duration := time.Since(start)
	return TaskResult{
		Name:     t.spec.Name,
		Result:   result,
		Duration: duration,
	}, nil
}

// Validate ensures the TaskSpec is valid.
func (t TaskSpec) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if t.Command == "" {
		return fmt.Errorf("task command cannot be empty")
	}
	for _, dep := range t.Dependencies {
		if dep == "" {
			return fmt.Errorf("dependency name cannot be empty")
		}
	}
	return nil
}

// ClearRegistry resets the task registry.
func ClearRegistry() {
	registryLock.Lock()
	defer registryLock.Unlock()
	taskRegistry = make(map[string]TaskSpec)
}

// RegisterTasks registers multiple tasks in the task registry.
func RegisterTasks(specs []TaskSpec) error {
	registryLock.Lock()
	defer registryLock.Unlock()

	for _, spec := range specs {
		if err := spec.Validate(); err != nil {
			return fmt.Errorf("invalid task %s: %w", spec.Name, err)
		}
		if _, exists := taskRegistry[spec.Name]; exists {
			return fmt.Errorf("duplicate task name: %s", spec.Name)
		}
		taskRegistry[spec.Name] = spec
	}
	return nil
}

// GetTask returns a task by name from the registry.
func GetTask(name string) (Task, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	spec, exists := taskRegistry[name]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", name)
	}
	return NewTask(spec), nil
}

// GetAllTasks returns all registered tasks.
func GetAllTasks() []Task {
	registryLock.RLock()
	defer registryLock.RUnlock()

	tasks := make([]Task, 0, len(taskRegistry))
	for _, spec := range taskRegistry {
		tasks = append(tasks, NewTask(spec))
	}
	return tasks
}

// GetTaskNames returns a sorted list of registered task names.
func GetTaskNames() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	names := make([]string, 0, len(taskRegistry))
	for name := range taskRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
