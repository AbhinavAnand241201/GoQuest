package main

import (
	"context"
	"flag"
	"runtime"
	"time"

	"github.com/AbhinavAnand241201/goquest/internal/util"
	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

func main() {
	// Parse command line flags
	scriptPath := flag.String("script", "scripts/example.go", "path to the task script")
	timeout := flag.Duration("timeout", 10*time.Second, "task execution timeout")
	concurrency := flag.Int("concurrency", runtime.NumCPU(), "number of concurrent tasks")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	listTasks := flag.Bool("list", false, "list tasks and their dependencies")
	flag.Parse()

	// Set up logging
	util.SetVerbose(*verbose)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Clear any existing tasks
	task.ClearRegistry()

	// Load tasks from script
	util.LogInfo("Loading script: %s", *scriptPath)
	tasks := task.GetAllTasks()
	if len(tasks) == 0 {
		util.ExitWithError(util.NewError("no tasks registered"))
	}

	// Create task graph
	graph := task.NewTaskGraph()
	for _, t := range tasks {
		if err := graph.AddTask(t.Spec); err != nil {
			util.ExitWithError(util.WrapError(err, "failed to add task to graph"))
		}
	}

	// Create executor and scheduler
	executor := task.NewExecutor(nil, *concurrency)
	scheduler := task.NewScheduler(graph, executor)

	// List tasks if requested
	if *listTasks {
		if err := graph.Validate(); err != nil {
			util.ExitWithError(util.WrapError(err, "invalid task dependencies"))
		}

		// Get execution order
		order, err := scheduler.GetExecutionOrder()
		if err != nil {
			util.ExitWithError(util.WrapError(err, "failed to get execution order"))
		}

		// Print tasks and their dependencies
		for _, name := range order {
			deps := graph.GetDependencies(name)
			if len(deps) == 0 {
				util.LogInfo("Task: %s, Depends: []", name)
			} else {
				util.LogInfo("Task: %s, Depends: %v", name, deps)
			}
		}
		return
	}

	// Execute tasks
	util.LogInfo("Running %d tasks with concurrency %d", len(tasks), *concurrency)
	results, err := scheduler.Schedule(ctx)
	if err != nil {
		util.ExitWithError(util.WrapError(err, "failed to schedule tasks"))
	}

	// Print results
	for _, result := range results {
		if result.Error != nil {
			util.LogInfo("Task %s: %s, error: %v, duration: %v",
				result.Name, result.Status, result.Error, result.Duration)
		} else {
			util.LogInfo("Task %s: %s, result: %v, duration: %v",
				result.Name, result.Status, result.Result, result.Duration)
		}
	}

	// Print summary
	failed := executor.GetFailedTasks(results)
	successful := executor.GetSuccessfulTasks(results)
	util.LogInfo("%d tasks completed, %d failed", len(successful), len(failed))

	// Exit with error if any tasks failed
	if err := executor.AggregateErrors(results); err != nil {
		util.ExitWithError(err)
	}
}
