# GoQuest: A Concurrent, Programmable Task Orchestrator

## Overview
**GoQuest** is a standalone Go CLI tool and library that orchestrates concurrent task workflows defined in a simple Go-based script. It allows users to define tasks (e.g., HTTP requests, computations) and execute them in parallel with a configurable worker pool, leveraging Go's concurrency model (goroutines and channels). Designed as a lightweight alternative to tools like `make` or Apache Airflow, GoQuest is ideal for developers automating workflows, showcasing advanced Go features like concurrency, interfaces, and robust error handling.

This project is built by  me , **Abhinav Anand**, a third-year student at NIT Durgapur, to demonstrate proficiency in Go for backend and DevOps roles. The codebase emphasizes clean code, modularity, and testability, making it a standout portfolio piece for recruiters.

## Features
- **Concurrent Task Execution**: Runs multiple tasks in parallel using a worker pool with configurable concurrency.
- **Task Definition**: Supports tasks defined in a Go script with fields for name, execution logic, and dependencies (dependency management in progress).
- **CLI Interface**: Provides a user-friendly CLI with flags for script path (`--script`), concurrency (`--concurrency`), and verbose logging (`--verbose`).
- **Robust Error Handling**: Aggregates task errors and provides detailed, contextual error messages.
- **Thread-Safe Logging**: Outputs timestamped logs (INFO, DEBUG) with thread-safe operations for concurrent execution.
- **Extensible Design**: Uses interfaces (`TaskRunner`) for pluggable task types, preparing for future enhancements like HTTP or file tasks.
- **JSON Output**: Export task results as JSON for machine-readable results.

## Project Status
As of completion of Part 2 (of a 5-part development plan), GoQuest supports:
- Sequential and concurrent execution of multiple tasks defined in a Go script.
- A worker pool for parallel task execution with configurable concurrency.
- Thread-safe result collection and error aggregation.
- A CLI with flags for script loading, concurrency, and verbose debugging.
- Basic task validation and logging.

Future parts will add:
- Part 3: A DSL parser for dynamic script loading.
- Part 4: Task dependency management.
- Part 5: Advanced features like JSON output and built-in task types (e.g., HTTP requests).

## Architecture
GoQuest follows a modular, clean architecture with a clear separation of concerns:
- **CLI (`cmd/goquest`)**: Handles user input, parses flags, and orchestrates task execution.
- **Task Package (`pkg/task`)**: Defines task specifications (`TaskSpec`), interfaces (`TaskRunner`), and the concurrent executor (`Executor`).
- **Utility Package (`internal/util`)**: Provides thread-safe logging and error handling helpers.
- **Scripts (`scripts`)**: Contains user-defined Go scripts with task definitions.

The task execution pipeline:
1. The CLI loads a script (e.g., `scripts/example.go`) and registers tasks in a global registry.
2. The `Executor` creates a worker pool of goroutines, dispatching tasks via a channel.
3. Workers execute tasks concurrently, sending results to a result channel.
4. Results are collected thread-safely, and errors are aggregated for reporting.
5. The CLI displays task summaries and logs execution details.

## Folder Structure
```
goquest/
├── cmd/
│   └── goquest/
│       └── main.go           # CLI entry point
├── pkg/
│   └── task/
│       ├── task.go          # TaskSpec, TaskRunner, and registry logic
│       ├── executor.go      # Concurrent task executor and worker pool
│       └── task_test.go     # Tests for task logic
├── internal/
│   └── util/
│       ├── log.go           # Thread-safe logging functions
│       ├── error.go         # Error wrapping and exit helpers
│       ├── log_test.go      # Tests for logging
│       └── error_test.go    # Tests for error handling
├── scripts/
│   └── example.go           # Sample task script
├── .gitignore               # Git ignore file
├── go.mod                   # Go module file
├── go.sum                   # Dependency checksums
├── README.md                # Project documentation
```

## Prerequisites
- **Go**: Version 1.22 or higher (tested with 1.22).
- **Git**: For cloning the repository.
- **Optional Tools**:
  - `golangci-lint`: For linting (install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`).
  - `goimports`: For code formatting (install with `go install golang.org/x/tools/cmd/goimports@latest`).

## Setup Instructions
Follow these steps to set up and run GoQuest on your local machine.

### 1. Clone the Repository
```bash
git clone https://github.com/AbhinavAnand241201/goquest.git
cd goquest
```

### 2. Initialize Go Module
Ensure the module is initialized and dependencies are downloaded:
```bash
go mod tidy
```

### 3. Build the CLI
Compile the CLI binary:
```bash
go build -o bin/goquest cmd/goquest/main.go
```

### 4. Run Tests
Verify the codebase with unit tests:
```bash
go test ./pkg/task/... ./internal/util/...
```

### 5. Install Optional Tools (Recommended)
For linting and formatting:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

## Usage
GoQuest runs tasks defined in a Go script via the CLI. The sample script `scripts/example.go` defines three tasks for testing.

### Basic Command
Run all tasks in `scripts/example.go` with default concurrency (4):
```bash
./bin/goquest run --script scripts/example.go
```

**Example Output**:
```
2025-05-14 13:35:00 INFO: Loading script: scripts/example.go
2025-05-14 13:35:00 INFO: Running 3 tasks with concurrency 4
2025-05-14 13:35:01 INFO: Task task1: success, result: Hello from task1, duration: 0.1s
2025-05-14 13:35:01 INFO: Task task2: success, result: Delayed task, duration: 1.0s
2025-05-14 13:35:01 INFO: Task task3: success, result: 42, duration: 0.1s
2025-05-14 13:35:01 INFO: 3 tasks completed, 0 failed
```

### Custom Concurrency
Run tasks with a specific concurrency level (e.g., 8):
```bash
./bin/goquest run --script scripts/example.go --concurrency 8
```

### Verbose Mode
Enable debug logging for detailed execution info:
```bash
./bin/goquest run --script scripts/example.go --verbose
```

**Example Verbose Output**:
```
2025-05-14 13:35:00 INFO: Loading script: scripts/example.go
2025-05-14 13:35:00 DEBUG: Dispatching task: task1
2025-05-14 13:35:00 DEBUG: Dispatching task: task2
2025-05-14 13:35:00 DEBUG: Dispatching task: task3
2025-05-14 13:35:01 DEBUG: Received result for task: task1
2025-05-14 13:35:01 INFO: Task task1: success, result: Hello from task1, duration: 0.1s
...
```

### Writing Custom Scripts
Create a new script (e.g., `scripts/myscript.go`) to define tasks. Example:
```go
package main

import (
    "context"
    "github.com/AbhinavAnand241201/goquest/pkg/task"
)

func init() {
    task.RegisterTasks([]task.TaskSpec{
        {
            Name: "custom_task",
            Run: func(ctx context.Context) (any, error) {
                return "Custom task executed", nil
            },
            Depends: []string{},
        },
    })
}
```

Run the custom script:
```bash
./bin/goquest run --script scripts/myscript.go
```

## Task Scheduler Usage Guide

The task scheduler is a powerful component of GoQuest that allows you to define and execute tasks with dependencies. Here's a step-by-step guide on how to use it in your own projects:

### 1. Import the Package

First, import the task package in your Go code:

```go
import "github.com/AbhinavAnand241201/GoQuest/pkg/task"
```

### 2. Define Your Tasks

Define the tasks you want to execute. Each task needs a name, a function to run, and optional dependencies:

```go
// Create a simple task
task1 := task.TaskSpec{
    Name: "fetch-data",
    Run: func(ctx context.Context) (interface{}, error) {
        // Your task logic here
        return "Data fetched successfully", nil
    },
}

// Create a task that depends on the first task
task2 := task.TaskSpec{
    Name: "process-data",
    Run: func(ctx context.Context) (interface{}, error) {
        // Process the data
        return "Data processed successfully", nil
    },
    Dependencies: []string{"fetch-data"}, // This task depends on task1
}

// Create another task that depends on the second task
task3 := task.TaskSpec{
    Name: "save-results",
    Run: func(ctx context.Context) (interface{}, error) {
        // Save the processed data
        return "Results saved successfully", nil
    },
    Dependencies: []string{"process-data"}, // This task depends on task2
}
```

### 3. Create a Task Graph

Create a task graph and add your tasks to it:

```go
// Create a new task graph
graph := task.NewTaskGraph()

// Add tasks to the graph
if err := graph.AddTask(task1); err != nil {
    log.Fatalf("Failed to add task1: %v", err)
}
if err := graph.AddTask(task2); err != nil {
    log.Fatalf("Failed to add task2: %v", err)
}
if err := graph.AddTask(task3); err != nil {
    log.Fatalf("Failed to add task3: %v", err)
}

// Validate the graph to ensure there are no cycles or missing dependencies
if err := graph.Validate(); err != nil {
    log.Fatalf("Graph validation failed: %v", err)
}
```

### 4. Create an Executor

Create an executor to run the tasks:

```go
// Create a new executor
executor := task.NewExecutor()

// Set the maximum number of concurrent tasks (optional)
executor.SetConcurrency(4)

// Create a logger (optional)
logger := &CustomLogger{} // Implement the Logger interface
executor.SetLogger(logger)
```

### 5. Create a Scheduler

Create a scheduler to manage the execution order:

```go
// Create a scheduler with the graph and executor
scheduler := task.NewScheduler(graph, executor, logger)
```

### 6. Execute the Tasks

Execute the tasks with a context (which allows for cancellation):

```go
// Create a context with a timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Execute the tasks
results, err := scheduler.Schedule(ctx)
if err != nil {
    log.Fatalf("Task execution failed: %v", err)
}

// Process the results
for _, result := range results {
    if result.Error != nil {
        log.Printf("Task %s failed: %v", result.Name, result.Error)
    } else {
        log.Printf("Task %s succeeded with result: %v", result.Name, result.Value)
    }
}
```

### 7. Implementing a Custom Logger

You can implement a custom logger to track task execution:

```go
type CustomLogger struct{}

func (l *CustomLogger) Info(msg string, data map[string]interface{}) {
    log.Printf("[INFO] %s %v", msg, data)
}

func (l *CustomLogger) Error(msg string, data map[string]interface{}) {
    log.Printf("[ERROR] %s %v", msg, data)
}

func (l *CustomLogger) Debug(msg string, data map[string]interface{}) {
    log.Printf("[DEBUG] %s %v", msg, data)
}

func (l *CustomLogger) Warn(msg string, data map[string]interface{}) {
    log.Printf("[WARN] %s %v", msg, data)
}
```

### 8. Complete Example

Here's a complete example that puts everything together:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/AbhinavAnand241201/GoQuest/pkg/task"
)

// CustomLogger implements the task.Logger interface
type CustomLogger struct{}

func (l *CustomLogger) Info(msg string, data map[string]interface{})  { log.Printf("[INFO] %s %v", msg, data) }
func (l *CustomLogger) Error(msg string, data map[string]interface{}) { log.Printf("[ERROR] %s %v", msg, data) }
func (l *CustomLogger) Debug(msg string, data map[string]interface{}) { log.Printf("[DEBUG] %s %v", msg, data) }
func (l *CustomLogger) Warn(msg string, data map[string]interface{})  { log.Printf("[WARN] %s %v", msg, data) }

func main() {
    // Create tasks
    task1 := task.TaskSpec{
        Name: "fetch-data",
        Run: func(ctx context.Context) (interface{}, error) {
            log.Println("Fetching data...")
            time.Sleep(1 * time.Second) // Simulate work
            return "Raw data", nil
        },
    }

    task2 := task.TaskSpec{
        Name: "process-data",
        Run: func(ctx context.Context) (interface{}, error) {
            log.Println("Processing data...")
            time.Sleep(2 * time.Second) // Simulate work
            return "Processed data", nil
        },
        Dependencies: []string{"fetch-data"},
    }

    task3 := task.TaskSpec{
        Name: "save-results",
        Run: func(ctx context.Context) (interface{}, error) {
            log.Println("Saving results...")
            time.Sleep(1 * time.Second) // Simulate work
            return "Results saved", nil
        },
        Dependencies: []string{"process-data"},
    }

    // Create a task graph
    graph := task.NewTaskGraph()

    // Add tasks to the graph
    if err := graph.AddTask(task1); err != nil {
        log.Fatalf("Failed to add task1: %v", err)
    }
    if err := graph.AddTask(task2); err != nil {
        log.Fatalf("Failed to add task2: %v", err)
    }
    if err := graph.AddTask(task3); err != nil {
        log.Fatalf("Failed to add task3: %v", err)
    }

    // Validate the graph
    if err := graph.Validate(); err != nil {
        log.Fatalf("Graph validation failed: %v", err)
    }

    // Create an executor
    executor := task.NewExecutor()
    executor.SetConcurrency(2)

    // Create a logger
    logger := &CustomLogger{}

    // Create a scheduler
    scheduler := task.NewScheduler(graph, executor, logger)

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Execute the tasks
    log.Println("Starting task execution...")
    start := time.Now()
    results, err := scheduler.Schedule(ctx)
    duration := time.Since(start)

    if err != nil {
        log.Fatalf("Task execution failed: %v", err)
    }

    log.Printf("All tasks completed in %v", duration)

    // Process the results
    for _, result := range results {
        if result.Error != nil {
            log.Printf("Task %s failed: %v", result.Name, result.Error)
        } else {
            log.Printf("Task %s succeeded with result: %v", result.Name, result.Value)
        }
    }
}
```

### 9. Real-World Use Cases

Here are some practical use cases for the task scheduler:

1. **Data Processing Pipeline**: Fetch data from multiple sources, transform it, and save the results.
2. **Build System**: Compile code, run tests, generate documentation, and deploy the application.
3. **Workflow Automation**: Execute a series of steps in a business process with dependencies.
4. **Batch Processing**: Process a large number of items in parallel with controlled concurrency.
5. **Dependency Management**: Manage complex dependencies between tasks in any system.




### Key Go Features Used
- **Goroutines and Channels**: For concurrent task execution and result collection.
- **Interfaces**: `TaskRunner` for extensible task types.
- **Context**: For task cancellation and timeouts.
- **Mutexes**: For thread-safe logging and result aggregation.
- **Flag Package**: For CLI argument parsing.
- **Testing**: Comprehensive unit tests for task execution and utilities.

## Testing
The project includes unit tests to ensure reliability:
- **Task Tests (`pkg/task/task_test.go`)**: Verify task validation, registry operations, and executor behavior.
- **Executor Tests (`pkg/task/executor_test.go`)**: Test concurrent task execution with varying task counts and durations.
- **Utility Tests (`internal/util/log_test.go`, `error_test.go`)**: Validate thread-safe logging and error handling.

Run all tests:
```bash
go test ./...
```

## Linting and Formatting
Ensure code quality with linting and formatting:
```bash
golangci-lint run
goimports -w .
```

## Troubleshooting
- **Issue**: `go build` fails with missing dependencies.
  - **Fix**: Run `go mod tidy` to download dependencies.
- **Issue**: CLI output is interleaved or unreadable.
  - **Fix**: Enable `--verbose` to debug or check `internal/util/log.go` for mutex issues.
- **Issue**: Tasks hang or don't complete.
  - **Fix**: Verify the `--concurrency` value is reasonable (e.g., 4–8) and check script tasks respect context timeouts.
- **Issue**: Tests fail due to race conditions.
  - **Fix**: Run `go test -race` to detect and fix race conditions in `Executor` or logging.

## Contributing
Contributions are welcome! To contribute:
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Commit changes (`git commit -m "Add my feature"`).
4. Push to the branch (`git push origin feature/my-feature`).
5. Open a pull request with a detailed description.

Please follow Go best practices, run tests, and lint the code before submitting.

## Future Enhancements
- **Dynamic DSL**: Parse Go scripts dynamically without static registration (Part 3).
- **Dependency Graphs**: Execute tasks in the correct order based on dependencies (Part 4).
- **Built-in Tasks**: Support common tasks like HTTP requests or file operations (Part 5).
- **JSON Output**: Export task results as JSON for integration with other tools.
- **Benchmarks**: Add performance benchmarks to showcase scalability.

## Contact
For questions or feedback, contact Abhinav Anand via:
- **GitHub**: [AbhinavAnand241201](https://github.com/AbhinavAnand241201)

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details. 
