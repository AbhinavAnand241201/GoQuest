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
	timeout := flag.Int("timeout", 1800, "Timeout in seconds for the entire execution")
	flag.Parse()

	// Initialize logger
	logger := log.NewLogger(*jsonOutput)

	// Validate script file
	if *scriptFile == "" {
		logger.Error("No script file specified", nil)
		os.Exit(1)
	}

	// Create a context with timeout for potential future use
	_, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
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
	
	// Validate the task graph
	if err := graph.Validate(); err != nil {
		if *jsonOutput {
			output.PrintError(err)
		} else {
			logger.Error("Invalid task graph", map[string]interface{}{"error": err})
		}
		os.Exit(1)
	}

	// Create executor with proper concurrency
	executor := task.NewExecutor()
	executor.SetConcurrency(*concurrency)
	
	// Add tasks to the executor
	for _, taskName := range task.GetTaskNames() {
		taskObj, err := task.GetTask(taskName)
		if err != nil {
			if *jsonOutput {
				output.PrintError(err)
			} else {
				logger.Error("Failed to get task", map[string]interface{}{"error": err})
			}
			os.Exit(1)
		}
		executor.AddTask(taskObj)
	}
	
	// Create scheduler with logger
	scheduler := task.NewScheduler(graph, executor, logger)

	// Print task information
	if !*jsonOutput {
		logger.Info("Tasks to execute:", nil)
		for _, taskName := range task.GetTaskNames() {
			taskObj, _ := task.GetTask(taskName)
			logger.Info(fmt.Sprintf("- %s", taskObj.Name()), nil)
		}
		logger.Info(fmt.Sprintf("Concurrency: %d", *concurrency), nil)
	}

	// Execute tasks
	if !*jsonOutput {
		logger.Info("Starting task execution...", nil)
	}
	
	// Start the scheduler with context
	err = scheduler.Start()
	if err != nil {
		if *jsonOutput {
			output.PrintError(err)
		} else {
			logger.Error("Task execution failed", map[string]interface{}{"error": err})
		}
		os.Exit(1)
	}
	
	// Get results
	results := executor.GetResults()

	// Print results
	if *jsonOutput {
		output.PrintTaskResults(results)
	} else {
		logger.Info("Task execution completed", nil)
		
		// Print summary
		successful := executor.GetSuccessfulTasks(results)
		failed := executor.GetFailedTasks(results)
		
		logger.Info(fmt.Sprintf("Summary: %d tasks completed, %d tasks failed", len(successful), len(failed)), nil)
		
		// Print details for each task
		for _, result := range results {
			if result.Error != nil {
				logger.Error(fmt.Sprintf("Task %s failed", result.Name), map[string]interface{}{
					"error": result.Error,
					"duration": result.Duration.String(),
				})
			} else {
				logger.Info(fmt.Sprintf("Task %s completed in %s", result.Name, result.Duration), map[string]interface{}{
					"result": result.Result,
				})
			}
		}
		
		// Print metrics
		metrics := scheduler.GetMetrics()
		logger.Info("Execution metrics", map[string]interface{}{
			"total_duration": metrics.TotalDuration.String(),
			"worker_utilization": fmt.Sprintf("%.2f%%", metrics.WorkerUtilization*100),
			"max_workers": metrics.MaxWorkers,
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
