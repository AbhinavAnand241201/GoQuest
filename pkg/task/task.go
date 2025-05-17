package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	// TaskRunner provides the basic task execution interface
	TaskRunner
	// Name returns the unique identifier for this task (deprecated, use GetName instead)
	Name() string
}

// TaskSpec defines the configuration for a task
type TaskSpec struct {
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Dir          string            `json:"dir,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	// Run is a backward compatibility field for tests
	Run          func(context.Context) (interface{}, error) `json:"-"`
	// Depends is a backward compatibility field for tests (alias for Dependencies)
	Depends      []string          `json:"depends,omitempty"`
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

// task implements the Task and TaskRunner interfaces
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

// GetName returns the task's name (for TaskRunner interface)
func (t *task) GetName() string {
	return t.spec.Name
}

// GetDependencies returns the task's dependencies (for TaskRunner interface)
func (t *task) GetDependencies() []string {
	// Check both Dependencies and Depends (backward compatibility)
	if len(t.spec.Dependencies) > 0 {
		return t.spec.Dependencies
	}
	return t.spec.Depends
}

// Run executes the task and returns its result
func (t *task) Run(ctx context.Context) (TaskResult, error) {
	start := time.Now()
	
	// Create a channel to signal task completion
	done := make(chan struct{})
	
	// Create channels for result and error
	resultCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)
	
	// Execute the task in a goroutine
	go func() {
		defer close(done)
		
		// Check if we have a Run function (backward compatibility)
		if t.spec.Run != nil {
			// Execute the Run function
			result, err := t.spec.Run(ctx)
			resultCh <- result
			errCh <- err
		} else if t.spec.Command != "" {
			// Execute the command
			result, err := executeCommand(ctx, t.spec)
			resultCh <- result
			errCh <- err
		} else {
			// Default implementation for tasks without commands
			result := fmt.Sprintf("Task %s completed", t.spec.Name)
			resultCh <- result
			errCh <- nil
		}
	}()
	
	// Wait for task completion or context cancellation
	select {
	case <-done:
		// Task completed normally
		duration := time.Since(start)
		return TaskResult{
			Name:     t.spec.Name,
			Result:   <-resultCh,
			Duration: duration,
			Error:    <-errCh,
		}, <-errCh
		
	case <-ctx.Done():
		// Context was cancelled or timed out
		duration := time.Since(start)
		return TaskResult{
			Name:     t.spec.Name,
			Result:   nil,
			Duration: duration,
			Error:    ctx.Err(),
		}, ctx.Err()
	}
}

// Validate ensures the TaskSpec is valid.
func (t TaskSpec) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	// Command can be empty for tasks that don't need to execute a command
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

// executeCommand executes a command as specified in the TaskSpec
func executeCommand(ctx context.Context, spec TaskSpec) (interface{}, error) {
	// Create a command with the given context
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
	
	// Set the working directory if specified
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	
	// Set environment variables if specified
	if len(spec.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range spec.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	
	// Read stdout and stderr
	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}
	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}
	
	// Wait for the command to complete
	err = cmd.Wait()
	
	// Prepare the result
	result := map[string]interface{}{
		"stdout": string(stdoutBytes),
		"stderr": string(stderrBytes),
		"exit_code": cmd.ProcessState.ExitCode(),
	}
	
	// Return an error if the command failed
	if err != nil {
		return result, fmt.Errorf("command failed: %w\nstderr: %s", err, stderrBytes)
	}
	
	return result, nil
}
