package task

import (
	"fmt"
	"strings"
)

// TaskGraph represents a directed acyclic graph of tasks and their dependencies.
type TaskGraph struct {
	// nodes maps task names to their TaskSpec objects
	nodes map[string]TaskSpec
	// edges maps task names to their dependent task names (outgoing edges)
	edges map[string][]string
}

// NewTaskGraph creates a new empty TaskGraph.
func NewTaskGraph() *TaskGraph {
	return &TaskGraph{
		nodes: make(map[string]TaskSpec),
		edges: make(map[string][]string),
	}
}

// AddTask adds a task and its dependencies to the graph.
func (g *TaskGraph) AddTask(spec TaskSpec) error {
	// Add the task to nodes
	g.nodes[spec.Name] = spec

	// Initialize edges for the task if not exists
	if _, exists := g.edges[spec.Name]; !exists {
		g.edges[spec.Name] = make([]string, 0)
	}

	// Add edges for each dependency
	for _, dep := range spec.Depends {
		// Initialize edges for the dependency if not exists
		if _, exists := g.edges[dep]; !exists {
			g.edges[dep] = make([]string, 0)
		}
		// Add the task as a dependent of its dependency
		g.edges[dep] = append(g.edges[dep], spec.Name)
	}

	return nil
}

// GetDependencies returns the list of tasks that a given task depends on.
func (g *TaskGraph) GetDependencies(taskName string) []string {
	if spec, exists := g.nodes[taskName]; exists {
		return spec.Depends
	}
	return nil
}

// GetDependents returns the list of tasks that depend on a given task.
func (g *TaskGraph) GetDependents(taskName string) []string {
	if deps, exists := g.edges[taskName]; exists {
		return deps
	}
	return nil
}

// Validate checks if the graph is a valid DAG by detecting cycles and invalid dependencies.
func (g *TaskGraph) Validate() error {
	visited := make(map[string]bool)
	path := make(map[string]bool)
	cyclePath := make([]string, 0)

	var dfs func(node string) error
	dfs = func(node string) error {
		if path[node] {
			// Found a cycle, build the cycle path
			cyclePath = append(cyclePath, node)
			start := 0
			for i, n := range cyclePath {
				if n == node {
					start = i
					break
				}
			}
			cycle := cyclePath[start:]
			return fmt.Errorf("cycle detected: %s", strings.Join(cycle, " -> "))
		}
		if visited[node] {
			return nil
		}

		visited[node] = true
		path[node] = true
		cyclePath = append(cyclePath, node)

		// Check all dependents
		for _, dependent := range g.edges[node] {
			if err := dfs(dependent); err != nil {
				return err
			}
		}

		path[node] = false
		cyclePath = cyclePath[:len(cyclePath)-1]
		return nil
	}

	// Check for cycles starting from each node
	for node := range g.nodes {
		if !visited[node] {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}

	// Validate that all dependencies exist
	for node, spec := range g.nodes {
		for _, dep := range spec.Depends {
			if _, exists := g.nodes[dep]; !exists {
				return fmt.Errorf("invalid dependency: task %s depends on non-existent task %s", node, dep)
			}
		}
	}

	return nil
}

// GetTaskSpec returns the TaskSpec for a given task name.
func (g *TaskGraph) GetTaskSpec(taskName string) (TaskSpec, bool) {
	spec, exists := g.nodes[taskName]
	return spec, exists
}

// GetAllTasks returns all task names in the graph.
func (g *TaskGraph) GetAllTasks() []string {
	tasks := make([]string, 0, len(g.nodes))
	for name := range g.nodes {
		tasks = append(tasks, name)
	}
	return tasks
}
