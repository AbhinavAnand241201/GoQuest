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
	flag.Parse()

	// Set up logging
	util.SetVerbose(*verbose)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Clear any existing tasks and register new ones
	task.ClearRegistry()
	if err := task.RegisterTasks([]task.TaskSpec{
		{
			Name: "hello_task",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "Hello from GoQuest!", nil
			},
			Depends: []string{},
		},
		{
			Name: "delayed_task",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(500 * time.Millisecond)
				return "This task took longer to complete", nil
			},
			Depends: []string{"hello_task"},
		},
		{
			Name: "error_task",
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return nil, util.NewError("this task always fails")
			},
			Depends: []string{},
		},
	}); err != nil {
		util.ExitWithError(util.WrapError(err, "failed to register tasks"))
	}

	// Load and execute tasks
	util.LogInfo("Loading script: %s", *scriptPath)
	tasks := task.GetAllTasks()
	if len(tasks) == 0 {
		util.ExitWithError(util.NewError("no tasks registered"))
	}

	// Convert []*Task to []TaskRunner
	taskRunners := make([]task.TaskRunner, len(tasks))
	for i, t := range tasks {
		taskRunners[i] = t
	}

	util.LogInfo("Running %d tasks with concurrency %d", len(tasks), *concurrency)
	executor := task.NewExecutor(taskRunners, *concurrency)
	results := executor.Run(ctx)

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
