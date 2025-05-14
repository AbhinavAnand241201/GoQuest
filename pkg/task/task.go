package task

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

var (
	taskRegistry = make(map[string]TaskSpec)
	registryLock sync.RWMutex
)

// TaskSpec defines a task's metadata and execution behavior.
type TaskSpec struct {
	// Name is a unique identifier for the task
	Name string
	// Run is the function that executes the task
	Run func(context.Context) (interface{}, error)
	// Depends lists the names of tasks that must complete before this task runs
	Depends []string
}

// TaskRunner defines the interface for executing tasks.
type TaskRunner interface {
	// Run executes the task and returns its result and any error
	Run(context.Context) (interface{}, error)
	// GetName returns the task's name
	GetName() string
	// GetDependencies returns the task's dependencies
	GetDependencies() []string
}

// Task wraps a TaskSpec and implements TaskRunner.
type Task struct {
	Spec TaskSpec
}

// NewTask creates a new Task from a TaskSpec.
func NewTask(spec TaskSpec) *Task {
	return &Task{Spec: spec}
}

// Run executes the task's Run function.
func (t *Task) Run(ctx context.Context) (interface{}, error) {
	return t.Spec.Run(ctx)
}

// GetName returns the task's name.
func (t *Task) GetName() string {
	return t.Spec.Name
}

// GetDependencies returns the task's dependencies.
func (t *Task) GetDependencies() []string {
	return t.Spec.Depends
}

// Validate ensures the TaskSpec is valid.
func (t TaskSpec) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if t.Run == nil {
		return fmt.Errorf("task run function cannot be nil")
	}
	for _, dep := range t.Depends {
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
func GetTask(name string) (*Task, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	spec, exists := taskRegistry[name]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", name)
	}
	return NewTask(spec), nil
}

// GetAllTasks returns all registered tasks.
func GetAllTasks() []*Task {
	registryLock.RLock()
	defer registryLock.RUnlock()

	tasks := make([]*Task, 0, len(taskRegistry))
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
