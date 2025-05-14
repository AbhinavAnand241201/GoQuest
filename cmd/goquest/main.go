package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/log"
	"github.com/AbhinavAnand241201/goquest/pkg/output"
	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

func main() {
	// Parse command line flags
	scriptFile := flag.String("script", "", "Path to the Go script containing task definitions")
	concurrency := flag.Int("concurrency", 4, "Maximum number of concurrent tasks")
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	flag.Parse()

	// Initialize logger
	logger := log.NewLogger(*jsonOutput)

	// Validate script file
	if *scriptFile == "" {
		logger.Error("No script file specified", nil)
		os.Exit(1)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Parse script
	parser := task.NewParser()
	tasks, err := parser.ParseScript(*scriptFile)
	if err != nil {
		if *jsonOutput {
			output.PrintError(err)
		} else {
			logger.Error("Failed to parse script", map[string]interface{}{"error": err})
		}
		os.Exit(1)
	}

	// Create task graph
	graph := task.NewTaskGraph()
	for _, task := range tasks {
		if err := graph.AddTask(task); err != nil {
			if *jsonOutput {
				output.PrintError(err)
			} else {
				logger.Error("Failed to add task to graph", map[string]interface{}{"error": err})
			}
			os.Exit(1)
		}
	}

	// Create executor and scheduler
	executor := task.NewExecutor(nil, *concurrency)
	scheduler := task.NewScheduler(graph, executor)

	// Log start of execution
	if !*jsonOutput {
		logger.Info("Running tasks", map[string]interface{}{
			"script":      *scriptFile,
			"concurrency": *concurrency,
			"taskCount":   len(tasks),
		})
	}

	// Schedule and run tasks
	results, err := scheduler.Schedule(ctx)
	if err != nil {
		if *jsonOutput {
			output.PrintError(err)
		} else {
			logger.Error("Failed to schedule tasks", map[string]interface{}{"error": err})
		}
		os.Exit(1)
	}

	// Print results
	if *jsonOutput {
		if err := output.PrintTaskResults(results); err != nil {
			output.PrintError(err)
			os.Exit(1)
		}
	} else {
		// Print human-readable results
		for _, result := range results {
			fields := map[string]interface{}{
				"status":   result.Status,
				"duration": result.Duration,
			}
			if result.Error != nil {
				fields["error"] = result.Error
			} else {
				fields["result"] = result.Result
			}
			logger.Info(fmt.Sprintf("Task %s completed", result.Name), fields)
		}

		// Print summary
		failed := executor.GetFailedTasks(results)
		successful := executor.GetSuccessfulTasks(results)
		logger.Info("Execution summary", map[string]interface{}{
			"successful": len(successful),
			"failed":     len(failed),
		})
	}

	// Exit with error if any tasks failed
	if err := executor.AggregateErrors(results); err != nil {
		if *jsonOutput {
			output.PrintError(err)
		} else {
			logger.Error("Task execution failed", map[string]interface{}{"error": err})
		}
		os.Exit(1)
	}
}
