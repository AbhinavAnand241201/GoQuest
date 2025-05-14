package task

import (
	"fmt"
	"sync"
)

// TaskGraph represents a directed acyclic graph of tasks
type TaskGraph struct {
	mu sync.RWMutex
	// Map of task name to task specification
	tasks map[string]TaskSpec
	// Map of task name to its dependencies
	dependencies map[string][]string
	// Map of task name to tasks that depend on it
	dependents map[string][]string
}

// NewTaskGraph creates a new empty task graph
func NewTaskGraph() *TaskGraph {
	return &TaskGraph{
		tasks:        make(map[string]TaskSpec),
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
	}
}

// AddTask adds a task to the graph
func (g *TaskGraph) AddTask(spec TaskSpec) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if task already exists
	if _, exists := g.tasks[spec.Name]; exists {
		return fmt.Errorf("task %s already exists", spec.Name)
	}

	// Add task
	g.tasks[spec.Name] = spec
	g.dependencies[spec.Name] = spec.Dependencies

	// Add to dependents map
	for _, dep := range spec.Dependencies {
		g.dependents[dep] = append(g.dependents[dep], spec.Name)
	}

	return nil
}

// GetTaskSpec returns the specification for a task
func (g *TaskGraph) GetTaskSpec(name string) (TaskSpec, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	spec, exists := g.tasks[name]
	return spec, exists
}

// GetDependencies returns the dependencies of a task
func (g *TaskGraph) GetDependencies(name string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dependencies[name]
}

// GetDependents returns the tasks that depend on the given task
func (g *TaskGraph) GetDependents(name string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dependents[name]
}

// GetAllTasks returns all task specifications
func (g *TaskGraph) GetAllTasks() []TaskSpec {
	g.mu.RLock()
	defer g.mu.RUnlock()
	tasks := make([]TaskSpec, 0, len(g.tasks))
	for _, spec := range g.tasks {
		tasks = append(tasks, spec)
	}
	return tasks
}

// Size returns the number of tasks in the graph
func (g *TaskGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.tasks)
}

// GetRunnableTasks returns tasks that have no dependencies
func (g *TaskGraph) GetRunnableTasks() []Task {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var runnable []Task
	for name, spec := range g.tasks {
		if len(g.dependencies[name]) == 0 {
			runnable = append(runnable, NewTask(spec))
		}
	}
	return runnable
}

// GetNextTasks returns tasks that can be run after the given task
func (g *TaskGraph) GetNextTasks(name string) []Task {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var next []Task
	for _, depName := range g.dependents[name] {
		// Check if all dependencies are satisfied
		allDepsSatisfied := true
		for _, dep := range g.dependencies[depName] {
			if dep != name {
				allDepsSatisfied = false
				break
			}
		}
		if allDepsSatisfied {
			next = append(next, NewTask(g.tasks[depName]))
		}
	}
	return next
}

// Validate checks if the graph is valid (no cycles)
func (g *TaskGraph) Validate() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) error
	dfs = func(name string) error {
		visited[name] = true
		recStack[name] = true

		for _, dep := range g.dependencies[name] {
			if !visited[dep] {
				if err := dfs(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("cycle detected: %s -> %s", name, dep)
			}
		}

		recStack[name] = false
		return nil
	}

	for name := range g.tasks {
		if !visited[name] {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}
